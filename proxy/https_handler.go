package proxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

// handleHTTPSProxy xử lý các request HTTPS (CONNECT) proxy với tự động thử lại
func handleHTTPSProxy(clientConn net.Conn, reader *bufio.Reader, firstLine string, pm *ProxyManager) {
	logger.Info("Handling HTTPS proxy request: %s", firstLine)

	// Trích xuất host từ dòng lệnh CONNECT
	parts := strings.Split(firstLine, " ")
	if len(parts) != 3 {
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	// Format: CONNECT example.com:443 HTTP/1.1
	hostPort := parts[1]

	// Đọc tất cả headers (chúng ta cần dừng ở dòng trống)
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			logger.Error("Failed to read header: %v", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break // Kết thúc headers
		}

		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) == 2 {
			headers[strings.TrimSpace(headerParts[0])] = strings.TrimSpace(headerParts[1])
		}
	}

	// Theo dõi các proxy đã thử để tránh dùng lại chúng khi thử lại
	triedProxies := make(map[string]bool)
	var lastError error
	var lastProxy *Proxy

	// Chỉ chọn proxy HTTP cho HTTPS tunnel
	// (SOCKS5 sẽ được xử lý khác, HTTP proxy vẫn hỗ trợ CONNECT cho HTTPS)
	httpOnlySelector := func(p *Proxy) bool {
		return p.Type == ProxyTypeHTTP || p.Type == ProxyTypeUnknown
	}

	// Thử tối đa maxRetries lần
	for retry := 0; retry <= pm.maxRetries; retry++ {
		// Lấy một proxy, loại trừ những proxy đã thử
		var proxy *Proxy
		if retry == 0 {
			proxy = pm.GetRandomProxyWithFilter(httpOnlySelector)
		} else {
			var excludeURL string
			if lastProxy != nil {
				excludeURL = lastProxy.URL
			}
			proxy = pm.GetNextWorkingProxyWithFilter(excludeURL, httpOnlySelector)
			if proxy == nil {
				logger.Error("No more available HTTP proxies to try after %d attempts", retry)
				clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
				return
			}
			logger.Info("HTTPS Retry %d/%d with proxy %s", retry, pm.maxRetries, proxy.URL)
		}

		if proxy == nil {
			logger.Error("No more available HTTP proxies to try after %d attempts", retry)
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}

		// Bỏ qua nếu đã thử proxy này
		if triedProxies[proxy.URL] {
			continue
		}

		// Đánh dấu proxy này đã được thử
		triedProxies[proxy.URL] = true
		lastProxy = proxy

		proxyURL, err := url.Parse(proxy.URL)
		if err != nil {
			logger.Error("Failed to parse proxy URL: %v", err)
			lastError = err
			pm.MarkProxyFailed(proxy)
			continue // Thử proxy tiếp theo
		}

		// Kết nối tới proxy với timeout
		proxyConn, err := net.DialTimeout("tcp", proxyURL.Host, 10*time.Second)
		if err != nil {
			logger.Error("Failed to connect to proxy: %v", err)
			lastError = err
			pm.MarkProxyFailed(proxy)
			continue // Thử proxy tiếp theo
		}

		// Sử dụng defer trong một hàm để đảm bảo kết nối này được đóng trước khi thử proxy khác
		tunnelEstablished := false
		func() {
			defer func() {
				if !tunnelEstablished {
					proxyConn.Close()
				}
			}()

			// Xây dựng request CONNECT
			connectRequest := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", hostPort, hostPort)

			// Thêm xác thực proxy nếu cần
			if proxy.Username != "" && proxy.Password != "" {
				auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Username, proxy.Password)))
				connectRequest += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
			}

			// Kết thúc request
			connectRequest += "\r\n"

			// Gửi request CONNECT tới proxy
			if err := proxyConn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				logger.Error("Failed to set write deadline: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return
			}

			if _, err := proxyConn.Write([]byte(connectRequest)); err != nil {
				logger.Error("Failed to send CONNECT request: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return
			}

			// Đặt lại deadline
			if err := proxyConn.SetWriteDeadline(time.Time{}); err != nil {
				logger.Error("Failed to reset write deadline: %v", err)
			}

			// Đọc phản hồi từ proxy
			if err := proxyConn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
				logger.Error("Failed to set read deadline: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return
			}

			// Đọc phản hồi
			proxyReader := bufio.NewReader(proxyConn)
			responseLine, err := proxyReader.ReadString('\n')
			if err != nil {
				logger.Error("Failed to read response line: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return
			}

			// Kiểm tra xem phản hồi có phải là 200 OK không
			if !strings.Contains(responseLine, "200") {
				logger.Error("Proxy did not return 200: %s", responseLine)
				lastError = fmt.Errorf("proxy returned: %s", responseLine)
				pm.MarkProxyFailed(proxy)
				return
			}

			// Đọc hết headers để đến body
			for {
				line, err := proxyReader.ReadString('\n')
				if err != nil {
					logger.Error("Failed to read header: %v", err)
					lastError = err
					pm.MarkProxyFailed(proxy)
					return
				}

				if strings.TrimSpace(line) == "" {
					break // End of headers
				}
			}

			// Gửi thông báo thành công (200) cho client
			clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

			// Đặt lại deadline
			if err := proxyConn.SetReadDeadline(time.Time{}); err != nil {
				logger.Error("Failed to reset read deadline: %v", err)
			}

			// Tunnel đã được thiết lập
			tunnelEstablished = true
		}()

		// Nếu tunnel được thiết lập, tiếp tục truyền dữ liệu giữa client và server
		if tunnelEstablished {
			// Đánh dấu proxy này thành công
			pm.MarkProxySuccess(proxy)

			// Tạo tunnel giữa client và upstream server
			logger.Info("HTTPS tunnel established via proxy %s to %s", proxy.URL, hostPort)

			// Xử lý truyền dữ liệu hai chiều
			go copyData(clientConn, proxyConn)
			copyData(proxyConn, clientConn)

			// Đóng kết nối sau khi kết thúc
			proxyConn.Close()
			return
		}
	}

	// Nếu đến đây, tất cả các lần thử đều thất bại
	logger.Error("All HTTPS proxy attempts failed after %d retries, last error: %v", pm.maxRetries, lastError)
	clientConn.Write([]byte(fmt.Sprintf("HTTP/1.1 502 Bad Gateway\r\n\r\nAll proxy attempts failed: %v\r\n", lastError)))
}

// copyData là hàm tiện ích để truyền dữ liệu giữa hai kết nối
func copyData(dst, src net.Conn) {
	errChan := make(chan error, 2)

	// Tạo goroutine để copy dữ liệu theo hai hướng
	go func() {
		_, err := io.Copy(dst, src)
		errChan <- err
	}()

	// Hướng ngược lại
	go func() {
		_, err := io.Copy(src, dst)
		errChan <- err
	}()

	// Đợi một trong hai hướng kết thúc
	err := <-errChan
	if err != nil && err != io.EOF {
		logger.Error("Tunnel error: %v", err)
	}

	// Đóng kết nối sau khi hoàn thành
	dst.Close()
	src.Close()
}
