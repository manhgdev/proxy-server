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

// handleHTTPProxy xử lý các yêu cầu HTTP proxy với tự động thử lại
func handleHTTPProxy(clientConn net.Conn, reader *bufio.Reader, firstLine string, pm *ProxyManager) {
	logger.Info("Handling HTTP proxy request: %s", firstLine)

	// Lưu trữ tất cả headers để tái sử dụng khi thử lại
	headers := make(map[string]string)
	var host string

	// Đọc tất cả headers từ client
	headerLines := []string{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			logger.Error("Failed to read header: %v", err)
			return
		}

		headerLines = append(headerLines, line)

		line = strings.TrimSpace(line)
		if line == "" {
			break // Kết thúc headers
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		headers[key] = value
		if strings.ToLower(key) == "host" {
			host = value
		}
	}

	// Trích xuất URL đích từ dòng đầu tiên
	parts := strings.Split(firstLine, " ")
	if len(parts) != 3 {
		logger.Error("Invalid HTTP request")
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	method := parts[0]
	targetURL := parts[1]

	// Nếu không tìm thấy header host trong request gốc
	if host == "" {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			logger.Error("Failed to parse target URL: %v", err)
			clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			return
		}
		host = parsedURL.Host
	}

	// Theo dõi các proxy đã thử để tránh dùng lại chúng khi thử lại
	triedProxies := make(map[string]bool)
	var lastError error
	var lastProxy *Proxy

	// Chỉ chọn proxy HTTP
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
			logger.Info("HTTP Retry %d/%d with proxy %s", retry, pm.maxRetries, proxy.URL)
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

		// Thiết lập xác thực proxy trong URL nếu có thông tin đăng nhập
		if proxy.Username != "" && proxy.Password != "" {
			proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
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
		func() {
			defer proxyConn.Close()

			// Xây dựng request
			var request strings.Builder
			request.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", method, targetURL))
			request.WriteString(fmt.Sprintf("Host: %s\r\n", host))

			// Thêm xác thực proxy
			if proxy.Username != "" && proxy.Password != "" {
				auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Username, proxy.Password)))
				request.WriteString(fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth))
			}

			// Thêm các header còn lại
			for key, value := range headers {
				if key != "Host" && key != "Proxy-Authorization" {
					request.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
				}
			}

			// Thêm header Connection
			request.WriteString("Connection: Keep-Alive\r\n\r\n")

			logger.Info("Sending request to proxy: %s", request.String())

			// Gửi request tới proxy với timeout
			if err := proxyConn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				logger.Error("Failed to set write deadline: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Thử proxy tiếp theo
			}

			if _, err := proxyConn.Write([]byte(request.String())); err != nil {
				logger.Error("Failed to send request to proxy: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Thử proxy tiếp theo
			}

			// Đặt lại deadline sau khi ghi
			if err := proxyConn.SetWriteDeadline(time.Time{}); err != nil {
				logger.Error("Failed to reset write deadline: %v", err)
			}

			// Đặt read timeout
			if err := proxyConn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
				logger.Error("Failed to set read deadline: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Thử proxy tiếp theo
			}

			// Tạo buffer để đọc phản hồi
			respBuf := make([]byte, 32*1024)

			// Đọc phần đầu tiên để kiểm tra nếu proxy có phản hồi
			n, err := proxyConn.Read(respBuf)
			if err != nil {
				logger.Error("Failed to read response from proxy: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Thử proxy tiếp theo
			}

			// Đặt lại deadline sau khi đọc ban đầu
			if err := proxyConn.SetReadDeadline(time.Time{}); err != nil {
				logger.Error("Failed to reset read deadline: %v", err)
			}

			// Kiểm tra nếu phản hồi hợp lệ
			respStr := string(respBuf[:n])
			if !strings.Contains(respStr, "HTTP/1.") {
				logger.Error("Invalid response from proxy: %s", respStr)
				lastError = fmt.Errorf("invalid response from proxy")
				pm.MarkProxyFailed(proxy)
				return // Thử proxy tiếp theo
			}

			// Phần đầu tiên của phản hồi trông tốt, gửi nó cho client
			if _, err := clientConn.Write(respBuf[:n]); err != nil {
				logger.Error("Failed to write to client: %v", err)
				return // Thoát hàm này nhưng không thử proxy khác
			}

			var totalBytes int64 = int64(n)

			// Tiếp tục đọc và chuyển tiếp phản hồi
			for {
				n, err := proxyConn.Read(respBuf)
				if n > 0 {
					// Chuyển tiếp tới client
					if _, err := clientConn.Write(respBuf[:n]); err != nil {
						logger.Error("Failed to write to client: %v", err)
						return
					}
					totalBytes += int64(n)
				}
				if err != nil {
					if err != io.EOF {
						logger.Error("Error reading from proxy: %v", err)
					}
					break
				}
			}

			logger.Info("HTTP request completed successfully. Total bytes: %d", totalBytes)

			// Đánh dấu proxy này là thành công
			pm.MarkProxySuccess(proxy)

			return
		}()

		// Nếu không có lỗi, nhảy ra khỏi vòng lặp
		if lastError == nil {
			return
		}
	}

	// Nếu đến đây, tất cả các lần thử đều thất bại
	logger.Error("All HTTP proxy attempts failed after %d retries, last error: %v", pm.maxRetries, lastError)
	clientConn.Write([]byte(fmt.Sprintf("HTTP/1.1 502 Bad Gateway\r\n\r\nAll proxy attempts failed: %v\r\n", lastError)))
}
