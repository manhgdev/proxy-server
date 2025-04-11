package proxy

import (
	"bufio"
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

// StartProxyServer starts a proxy server that listens for incoming connections
func StartProxyServer(pm *ProxyManager, addr string) error {
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

// handleProxyConnection handles a new client connection
func handleProxyConnection(clientConn net.Conn, pm *ProxyManager) {
	defer clientConn.Close()

	// Read the first line to determine the protocol
	reader := bufio.NewReader(clientConn)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		logger.Error("Failed to read first line: %v", err)
		return
	}

	// Determine if this is a CONNECT request (HTTPS) or regular HTTP request
	if strings.HasPrefix(firstLine, "CONNECT") {
		handleHTTPSProxy(clientConn, reader, firstLine, pm)
	} else {
		handleHTTPProxy(clientConn, reader, firstLine, pm)
	}
}

// handleHTTPProxy handles HTTP proxy requests with auto retry
func handleHTTPProxy(clientConn net.Conn, reader *bufio.Reader, firstLine string, pm *ProxyManager) {
	// Store all headers for reuse in retries
	headers := make(map[string]string)
	var host string

	// Read all headers from client
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
			break // End of headers
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

	// Extract target URL from first line
	parts := strings.Split(firstLine, " ")
	if len(parts) != 3 {
		logger.Error("Invalid HTTP request")
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	method := parts[0]
	targetURL := parts[1]

	// If no host header found in the original request
	if host == "" {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			logger.Error("Failed to parse target URL: %v", err)
			clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			return
		}
		host = parsedURL.Host
	}

	// Track already tried proxies to avoid using them again in retries
	triedProxies := make(map[string]bool)
	var lastError error
	var lastProxy *Proxy

	// Try up to maxRetries times
	for retry := 0; retry <= pm.maxRetries; retry++ {
		// Get a proxy, excluding ones we've already tried
		var proxy *Proxy
		if retry == 0 {
			proxy = pm.GetRandomProxy()
		} else {
			var excludeURL string
			if lastProxy != nil {
				excludeURL = lastProxy.URL
			}
			proxy = pm.GetNextWorkingProxy(excludeURL)
			logger.Info("HTTP Retry %d/%d with proxy %s", retry, pm.maxRetries, proxy.URL)
		}

		if proxy == nil {
			logger.Error("No more available proxies to try after %d attempts", retry)
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}

		// Skip if we've already tried this proxy
		if triedProxies[proxy.URL] {
			continue
		}

		// Mark this proxy as tried
		triedProxies[proxy.URL] = true
		lastProxy = proxy

		proxyURL, err := url.Parse(proxy.URL)
		if err != nil {
			logger.Error("Failed to parse proxy URL: %v", err)
			lastError = err
			pm.MarkProxyFailed(proxy)
			continue // Try next proxy
		}

		// Set proxy authentication in URL if credentials exist
		if proxy.Username != "" && proxy.Password != "" {
			proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
		}

		// Connect to proxy with timeout
		proxyConn, err := net.DialTimeout("tcp", proxyURL.Host, 10*time.Second)
		if err != nil {
			logger.Error("Failed to connect to proxy: %v", err)
			lastError = err
			pm.MarkProxyFailed(proxy)
			continue // Try next proxy
		}

		// Use defer in a function to ensure this connection is closed before trying another proxy
		func() {
			defer proxyConn.Close()

			// Construct request
			var request strings.Builder
			request.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", method, targetURL))
			request.WriteString(fmt.Sprintf("Host: %s\r\n", host))

			// Add proxy authentication
			if proxy.Username != "" && proxy.Password != "" {
				auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Username, proxy.Password)))
				request.WriteString(fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth))
			}

			// Add remaining headers
			for key, value := range headers {
				if key != "Host" && key != "Proxy-Authorization" {
					request.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
				}
			}

			// Add connection header
			request.WriteString("Connection: Keep-Alive\r\n\r\n")

			logger.Info("Sending request to proxy: %s", request.String())

			// Send request to proxy with timeout
			if err := proxyConn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				logger.Error("Failed to set write deadline: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			if _, err := proxyConn.Write([]byte(request.String())); err != nil {
				logger.Error("Failed to send request to proxy: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			// Reset deadline after write
			if err := proxyConn.SetWriteDeadline(time.Time{}); err != nil {
				logger.Error("Failed to reset write deadline: %v", err)
			}

			// Set read timeout
			if err := proxyConn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
				logger.Error("Failed to set read deadline: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			// Create a buffer to read response
			respBuf := make([]byte, 32*1024)

			// Read first chunk to check if proxy is responsive
			n, err := proxyConn.Read(respBuf)
			if err != nil {
				logger.Error("Failed to read response from proxy: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			// Reset deadline after initial read
			if err := proxyConn.SetReadDeadline(time.Time{}); err != nil {
				logger.Error("Failed to reset read deadline: %v", err)
			}

			// Check if response is valid
			respStr := string(respBuf[:n])
			if !strings.Contains(respStr, "HTTP/1.") {
				logger.Error("Invalid response from proxy: %s", respStr)
				lastError = fmt.Errorf("invalid response from proxy")
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			// First response chunk looks good, send it to client
			if _, err := clientConn.Write(respBuf[:n]); err != nil {
				logger.Error("Failed to write to client: %v", err)
				return // Exit this function but don't try another proxy
			}

			var totalBytes int64 = int64(n)

			// Continue reading and forwarding response
			for {
				n, err := proxyConn.Read(respBuf)
				if n > 0 {
					// Forward to client
					if _, err := clientConn.Write(respBuf[:n]); err != nil {
						logger.Error("Failed to write to client: %v", err)
						return // Exit this function but don't try another proxy
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

			logger.Info("Request completed successfully. Total bytes transferred: %d", totalBytes)
			return // Successfully completed
		}()

		// If we got here and there's no error, we successfully completed the request
		if lastError == nil {
			return
		}
	}

	// If we reach here, all retries failed
	logger.Error("All proxy attempts failed after %d retries, last error: %v",
		pm.maxRetries, lastError)
	clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
}

// handleHTTPSProxy handles HTTPS proxy requests (CONNECT) with auto retry
func handleHTTPSProxy(clientConn net.Conn, reader *bufio.Reader, firstLine string, pm *ProxyManager) {
	parts := strings.Split(firstLine, " ")
	if len(parts) != 3 {
		logger.Error("Invalid CONNECT request")
		clientConn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	target := parts[1]

	// Read all headers from client
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			logger.Error("Failed to read header: %v", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break // End of headers
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		headers[key] = value
	}

	// Track already tried proxies to avoid using them again in retries
	triedProxies := make(map[string]bool)
	var lastError error
	var lastProxy *Proxy

	// Try up to maxRetries times
	for retry := 0; retry <= pm.maxRetries; retry++ {
		// Get a proxy, excluding ones we've already tried
		var proxy *Proxy
		if retry == 0 {
			proxy = pm.GetRandomProxy()
		} else {
			var excludeURL string
			if lastProxy != nil {
				excludeURL = lastProxy.URL
			}
			proxy = pm.GetNextWorkingProxy(excludeURL)
			logger.Info("HTTPS Retry %d/%d with proxy %s", retry, pm.maxRetries, proxy.URL)
		}

		if proxy == nil {
			logger.Error("No more available proxies to try after %d attempts", retry)
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}

		// Skip if we've already tried this proxy
		if triedProxies[proxy.URL] {
			continue
		}

		// Mark this proxy as tried
		triedProxies[proxy.URL] = true
		lastProxy = proxy

		// Parse proxy URL and extract host:port
		proxyURL, err := url.Parse(proxy.URL)
		if err != nil {
			logger.Error("Failed to parse proxy URL: %v", err)
			lastError = err
			pm.MarkProxyFailed(proxy)
			continue // Try next proxy
		}

		// Set proxy authentication in URL if credentials exist
		if proxy.Username != "" && proxy.Password != "" {
			proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
		}

		// Connect to proxy with timeout
		proxyConn, err := net.DialTimeout("tcp", proxyURL.Host, 10*time.Second)
		if err != nil {
			logger.Error("Failed to connect to proxy: %v", err)
			lastError = err
			pm.MarkProxyFailed(proxy)
			continue // Try next proxy
		}

		// Use defer in a function to ensure this connection is closed before trying another proxy
		func() {
			defer proxyConn.Close()

			// Send CONNECT request to proxy with timeout
			connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", target, target)
			if proxy.Username != "" && proxy.Password != "" {
				auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Username, proxy.Password)))
				connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
			}

			// Add any other relevant headers from the original request
			for key, value := range headers {
				if key != "Host" && key != "Proxy-Authorization" &&
					key != "Connection" && key != "Proxy-Connection" {
					connectReq += fmt.Sprintf("%s: %s\r\n", key, value)
				}
			}

			connectReq += "Connection: Keep-Alive\r\n\r\n"

			// Set write deadline
			if err := proxyConn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				logger.Error("Failed to set write deadline: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			if _, err := proxyConn.Write([]byte(connectReq)); err != nil {
				logger.Error("Failed to send CONNECT request: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			// Reset deadline after write
			if err := proxyConn.SetWriteDeadline(time.Time{}); err != nil {
				logger.Error("Failed to reset write deadline: %v", err)
			}

			// Set read deadline
			if err := proxyConn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
				logger.Error("Failed to set read deadline: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			// Read proxy response
			respBuf := make([]byte, 1024)
			n, err := proxyConn.Read(respBuf)
			if err != nil {
				logger.Error("Failed to read proxy response: %v", err)
				lastError = err
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			// Reset read deadline
			if err := proxyConn.SetReadDeadline(time.Time{}); err != nil {
				logger.Error("Failed to reset read deadline: %v", err)
			}

			// Check if connection was successful
			resp := string(respBuf[:n])
			if !strings.Contains(resp, "200") {
				logger.Error("Proxy connection failed: %s", resp)
				lastError = fmt.Errorf("proxy connection failed: %s", resp)
				pm.MarkProxyFailed(proxy)
				return // Try next proxy
			}

			// Send success response to client
			if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
				logger.Error("Failed to send success response to client: %v", err)
				return // No need to retry another proxy as the client connection failed
			}

			// Start bidirectional copy with timeout handling
			clientToProxy := make(chan error, 1)
			proxyToClient := make(chan error, 1)

			// Copy from client to proxy
			go func() {
				_, err := io.Copy(proxyConn, clientConn)
				clientToProxy <- err
			}()

			// Copy from proxy to client
			go func() {
				_, err := io.Copy(clientConn, proxyConn)
				proxyToClient <- err
			}()

			// Wait for either direction to finish or timeout
			select {
			case err := <-clientToProxy:
				if err != nil && err != io.EOF {
					logger.Error("Error in client->proxy copy: %v", err)
				}
			case err := <-proxyToClient:
				if err != nil && err != io.EOF {
					logger.Error("Error in proxy->client copy: %v", err)
				}
			}

			logger.Info("HTTPS connection completed successfully")
			lastError = nil // Mark as successful
			return
		}()

		// If we got here and there's no error, we successfully completed the request
		if lastError == nil {
			return
		}
	}

	// If we reach here, all retries failed
	logger.Error("All HTTPS proxy attempts failed after %d retries, last error: %v",
		pm.maxRetries, lastError)
	clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
}
