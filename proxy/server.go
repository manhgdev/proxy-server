package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"proxy/utils"

	"github.com/elazarl/goproxy"
)

var (
	logger = utils.NewLogger()
)

// ProxyTransport handles HTTP requests through a proxy
type ProxyTransport struct {
	proxyManager *ProxyManager
}

// RoundTrip implements goproxy.RoundTripper
func (t *ProxyTransport) RoundTrip(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Response, error) {
	logger.StartRequest()
	logger.Request("Method: %s", req.Method)
	logger.Request("URL: %s", req.URL.String())
	logger.Request("Host: %s", req.Host)
	logger.Header("Headers:")
	for k, v := range req.Header {
		logger.Header("  %s: %v", k, v)
	}

	// Track already tried proxies to avoid using them again in retries
	triedProxies := make(map[string]bool)
	var lastError error
	var lastProxy *Proxy

	// Try up to maxRetries times
	for retry := 0; retry <= t.proxyManager.maxRetries; retry++ {
		// Get a proxy, excluding ones we've already tried
		var proxy *Proxy
		if retry == 0 {
			proxy = t.proxyManager.GetRandomProxy()
		} else {
			var excludeURL string
			if lastProxy != nil {
				excludeURL = lastProxy.URL
			}
			proxy = t.proxyManager.GetNextWorkingProxy(excludeURL)
			logger.Info("Retry %d/%d with proxy %s", retry, t.proxyManager.maxRetries, proxy.URL)
		}

		if proxy == nil {
			logger.Error("No more available proxies to try after %d attempts", retry)
			if lastError != nil {
				return nil, fmt.Errorf("all proxies failed, last error: %v", lastError)
			}
			return nil, fmt.Errorf("no proxy available")
		}

		// Skip if we've already tried this proxy
		if triedProxies[proxy.URL] {
			continue
		}

		// Mark this proxy as tried
		triedProxies[proxy.URL] = true
		lastProxy = proxy

		var proxyURL *url.URL
		var err error

		// Parse proxy URL based on format
		if strings.Contains(proxy.URL, "http://") {
			// URL already has scheme
			proxyURL, err = url.Parse(proxy.URL)
		} else {
			// Add scheme
			proxyURL, err = url.Parse("http://" + proxy.URL)
		}

		if err != nil {
			logger.Error("Invalid proxy URL: %v", err)
			lastError = err
			t.proxyManager.MarkProxyFailed(proxy)
			continue // Try next proxy
		}

		// Set proxy authentication in URL if credentials exist
		if proxy.Username != "" && proxy.Password != "" {
			proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
		}

		// Create a new request to forward
		forwardReq := req.Clone(req.Context())
		forwardReq.URL.Host = req.Host
		forwardReq.URL.Scheme = req.URL.Scheme
		if req.URL.Path != "" {
			forwardReq.URL.Path = req.URL.Path
		}
		if req.URL.RawQuery != "" {
			forwardReq.URL.RawQuery = req.URL.RawQuery
		}

		// Add proxy authentication header
		if proxy.Username != "" && proxy.Password != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Username, proxy.Password)))
			forwardReq.Header.Set("Proxy-Authorization", "Basic "+auth)
		}

		// Create transport with proxy settings
		transport := &http.Transport{
			Proxy: func(_ *http.Request) (*url.URL, error) {
				return proxyURL, nil
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ProxyConnectHeader: http.Header{},
		}

		// Add proxy authentication header if credentials exist
		if proxy.Username != "" && proxy.Password != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Username, proxy.Password)))
			transport.ProxyConnectHeader.Set("Proxy-Authorization", "Basic "+auth)
		}

		// Set a shorter timeout for faster failure detection
		clientTimeout := 20 * time.Second

		// Create a custom client for HTTP requests
		if req.URL.Scheme == "http" {
			client := &http.Client{
				Transport: transport,
				Timeout:   clientTimeout,
			}

			logger.Proxy("Forwarding HTTP request to: %s via proxy %s", forwardReq.URL.String(), proxyURL.String())
			resp, err := client.Do(forwardReq)
			if err != nil {
				logger.Error("Error forwarding HTTP request: %v", err)
				lastError = err
				t.proxyManager.MarkProxyFailed(proxy)
				continue // Try next proxy
			}

			// Success - return the response
			return resp, nil
		}

		logger.Proxy("Forwarding request to: %s via proxy %s", forwardReq.URL.String(), proxyURL.String())

		// Set custom timeout for the request context
		ctx, cancel := context.WithTimeout(forwardReq.Context(), clientTimeout)
		forwardReq = forwardReq.WithContext(ctx)
		defer cancel()

		resp, err := transport.RoundTrip(forwardReq)
		if err != nil {
			logger.Error("Error forwarding request: %v", err)
			lastError = err
			t.proxyManager.MarkProxyFailed(proxy)
			continue // Try next proxy
		}

		// Create a new response with the same status and headers
		newResp := &http.Response{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Proto:      resp.Proto,
			ProtoMajor: resp.ProtoMajor,
			ProtoMinor: resp.ProtoMinor,
			Header:     resp.Header,
		}

		// Copy the body
		if resp.Body != nil {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error("Error reading response body: %v", err)
				lastError = err
				t.proxyManager.MarkProxyFailed(proxy)
				continue // Try next proxy
			}
			resp.Body.Close()

			newResp.Body = io.NopCloser(strings.NewReader(string(body)))
			newResp.ContentLength = int64(len(body))
			logger.Response("Response body: %s", string(body))
		}

		logger.EndRequest()
		return newResp, nil
	}

	// If we get here, all retries failed
	return nil, fmt.Errorf("all proxy attempts failed after %d retries, last error: %v",
		t.proxyManager.maxRetries, lastError)
}

// handleConnect handles CONNECT requests
func (t *ProxyTransport) handleConnect(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	logger.StartRequest()
	logger.Request("CONNECT request to: %s", host)

	proxy := t.proxyManager.GetRandomProxy()
	if proxy == nil {
		logger.Error("No proxy available for CONNECT")
		return goproxy.RejectConnect, "no proxy available"
	}

	logger.Proxy("Using proxy: %s", proxy.URL)
	logger.EndRequest()
	return goproxy.OkConnect, host
}

// NewProxyServer creates a new proxy server
func NewProxyServer(proxyManager *ProxyManager) http.Handler {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	transport := &ProxyTransport{
		proxyManager: proxyManager,
	}

	proxy.OnRequest().Do(goproxy.FuncReqHandler(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		ctx.RoundTripper = transport
		return req, nil
	}))

	proxy.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		return transport.handleConnect(host, ctx)
	}))

	return proxy
}

// StartProxyServer khởi động proxy server lắng nghe kết nối
func StartProxyServer(pm *ProxyManager, addr string) error {
	// Cấu hình SOCKS5
	ConfigureSOCKS5(&SOCKS5Config{
		SkipVerify: true, // Bỏ qua xác thực SSL
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", addr, err)
	}

	logger.Info("Starting proxy server on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Failed to accept connection: %v", err)
			continue
		}

		go handleProxyConnection(conn, pm)
	}
}

// handleProxyConnection xử lý kết nối mới
func handleProxyConnection(clientConn net.Conn, pm *ProxyManager) {
	defer clientConn.Close()

	// Đọc byte đầu tiên để xác định protocol
	firstByte := make([]byte, 1)
	if _, err := clientConn.Read(firstByte); err != nil {
		logger.Error("Failed to read first byte: %v", err)
		return
	}

	// Kiểm tra nếu là SOCKS5 (byte đầu tiên là 0x05)
	if firstByte[0] == SOCKS5_VERSION {
		// Đẩy byte đầu tiên trở lại kết nối
		tempReader := io.MultiReader(bytes.NewReader(firstByte), clientConn)

		// Sử dụng io.TeeReader và bufio.NewReader để xử lý SOCKS5
		readerConn := &readConn{
			Reader: tempReader,
			Conn:   clientConn,
		}

		handleSOCKS5(readerConn, pm)
		return
	}

	// Nếu không phải SOCKS5, tiếp tục xử lý HTTP/HTTPS
	reader := bufio.NewReader(io.MultiReader(bytes.NewReader(firstByte), clientConn))
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		logger.Error("Failed to read first line: %v", err)
		return
	}

	// Xác định nếu là CONNECT (HTTPS) hoặc HTTP thông thường
	if strings.HasPrefix(firstLine, "CONNECT") {
		handleHTTPSProxy(clientConn, reader, firstLine, pm)
	} else {
		handleHTTPProxy(clientConn, reader, firstLine, pm)
	}
}

// readConn kết hợp io.Reader và net.Conn để xử lý protocol SOCKS5
type readConn struct {
	io.Reader
	net.Conn
}

// Read ghi đè phương thức Read để sử dụng Reader đã nhúng
func (r *readConn) Read(b []byte) (int, error) {
	return r.Reader.Read(b)
}
