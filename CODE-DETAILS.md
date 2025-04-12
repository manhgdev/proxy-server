# Chi tiết mã nguồn Proxy Server

## Tổng quan

Proxy Server này là một hệ thống chuyển tiếp yêu cầu HTTP, HTTPS và SOCKS5 với tính năng tự động luân chuyển proxy, hỗ trợ xác thực, theo dõi proxy sống/chết và xử lý lỗi một cách mạnh mẽ.

## Cấu trúc mã nguồn

### 1. Main (main.go)

File chính khởi tạo ứng dụng:
- Khởi tạo ProxyManager
- Tải danh sách proxy từ các file
- Chạy server proxy trên cổng cấu hình
- Xử lý signal để tắt server một cách graceful

```go
func main() {
    // Tạo proxy manager
    pm := proxy.NewProxyManager()
    
    // Tải proxy từ nhiều file
    proxy.LoadProxiesFromMultipleFiles(httpProxyFile, socks5ProxyFile, pm)
    
    // Khởi động proxy server
    proxy.StartProxyServer(pm, serverPort)
}
```

### 2. Proxy Manager (proxy/manager.go)

Quản lý danh sách proxy và trạng thái của chúng:
- Lưu trữ danh sách proxy
- Theo dõi proxy sống/chết
- Cung cấp các phương thức chọn proxy theo tiêu chí khác nhau
- Tự động đánh dấu proxy thành công/thất bại

```go
// Cấu trúc Proxy
type Proxy struct {
    URL       string     // URL đầy đủ của proxy
    Host      string     // Host của proxy
    Port      string     // Port của proxy
    Username  string     // Tên đăng nhập (nếu có)
    Password  string     // Mật khẩu (nếu có)
    Type      ProxyType  // Loại proxy (HTTP, SOCKS5)
    FailCount int        // Số lần thất bại liên tiếp
    LastUsed  time.Time  // Thời điểm sử dụng gần nhất
}

// ProxyManager quản lý danh sách proxy
type ProxyManager struct {
    proxies       []*Proxy
    maxRetries    int
    failThreshold int
    mu            sync.RWMutex
}
```

### 3. Server và Transport (proxy/server.go)

Triển khai server proxy HTTP và HTTPS:
- Sử dụng goproxy để xử lý request HTTP
- Custom transport để chuyển tiếp request qua proxy
- Điều hướng giữa HTTP, HTTPS và SOCKS5

```go
// ProxyTransport triển khai http.RoundTripper
type ProxyTransport struct {
    proxyManager *ProxyManager
}

// RoundTrip triển khai phương thức chuyển tiếp HTTP request
func (t *ProxyTransport) RoundTrip(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Response, error) {
    // Thử vài proxy khác nhau cho đến khi thành công
    for retry := 0; retry <= t.proxyManager.maxRetries; retry++ {
        proxy := t.proxyManager.GetRandomProxy()
        
        // Cấu hình proxy và thực hiện request
        // ...
        
        // Nếu thành công, trả về response
        // Nếu thất bại, thử proxy khác
    }
}
```

### 4. HTTPS Handler (proxy/https_handler.go)

Xử lý các kết nối HTTPS qua HTTP CONNECT:
- Xử lý lệnh CONNECT từ client
- Thiết lập tunnel giữa client và target server
- Hỗ trợ xác thực proxy
- Tự động thử lại với proxy khác nếu thất bại

```go
// handleHTTPSProxy xử lý các request HTTPS (CONNECT)
func handleHTTPSProxy(clientConn net.Conn, reader *bufio.Reader, firstLine string, pm *ProxyManager) {
    // Trích xuất host và port từ lệnh CONNECT
    // ...
    
    // Thử kết nối qua nhiều proxy khác nhau
    for retry := 0; retry <= pm.maxRetries; retry++ {
        // Chọn một proxy HTTP hoặc HTTPS
        proxy := pm.GetRandomProxyWithFilter(httpOnlySelector)
        
        // Thiết lập tunnel với proxy
        // ...
        
        // Nếu thành công, tạo tunnel hai chiều
        if tunnelEstablished {
            // Đánh dấu proxy thành công
            pm.MarkProxySuccess(proxy)
            
            // Tạo tunnel giữa client và server
            go copyData(clientConn, proxyConn)
            copyData(proxyConn, clientConn)
            return
        }
    }
}

// copyData truyền dữ liệu hai chiều giữa hai kết nối
func copyData(dst, src net.Conn) {
    errChan := make(chan error, 2)
    
    // Tạo goroutine copy dữ liệu theo hai hướng
    go func() {
        _, err := io.Copy(dst, src)
        errChan <- err
    }()
    
    go func() {
        _, err := io.Copy(src, dst)
        errChan <- err
    }()
    
    // Đợi một trong hai hướng kết thúc
    err := <-errChan
    if err != nil && err != io.EOF {
        logger.Error("Tunnel error: %v", err)
    }
    
    // Đóng kết nối
    dst.Close()
    src.Close()
}
```

### 5. SOCKS5 Handler (proxy/socks5_handler.go)

Xử lý các kết nối qua SOCKS5 protocol:
- Xử lý bắt tay SOCKS5
- Hỗ trợ cả địa chỉ IPv4, IPv6 và domain
- Tạo tunnel để chuyển tiếp dữ liệu
- Hỗ trợ kết nối TLS qua SOCKS5

```go
// handleSOCKS5 xử lý các request SOCKS5
func handleSOCKS5(clientConn net.Conn, pm *ProxyManager) {
    // Xử lý bắt tay SOCKS5 (negotiation)
    // ...
    
    // Đọc và xử lý request
    // ...
    
    // Trích xuất địa chỉ đích (IPv4, IPv6, domain)
    // ...
    
    // Chọn proxy SOCKS5 phù hợp
    proxy := pm.GetRandomProxyWithFilter(socks5Selector)
    
    // Kết nối tới proxy
    // ...
    
    // Thực hiện bắt tay SOCKS5 với proxy
    // ...
    
    // Gửi request kết nối tới địa chỉ đích
    // ...
    
    // Tạo tunnel hai chiều
    handleTLSOverSOCKS5(clientConn, targetAddr, proxyConn)
}

// handleTLSOverSOCKS5 xử lý kết nối TLS qua SOCKS5
func handleTLSOverSOCKS5(clientConn net.Conn, targetAddr string, proxyConn net.Conn) {
    // Tạo tunnel hai chiều giữa client và proxy
    errChan := make(chan error, 2)
    
    // Client -> Proxy
    go func() {
        _, err := io.Copy(proxyConn, clientConn)
        errChan <- err
    }()
    
    // Proxy -> Client
    go func() {
        _, err := io.Copy(clientConn, proxyConn)
        errChan <- err
    }()
    
    // Đợi một bên kết thúc
    err := <-errChan
    if err != nil && err != io.EOF {
        logger.Error("SOCKS5 tunnel error: %v", err)
    }
}
```

## Luồng xử lý request

### 1. HTTP Request

1. Client gửi request HTTP đến proxy server
2. Proxy server chọn một proxy từ danh sách
3. Proxy server chuyển tiếp request đến proxy đã chọn
4. Proxy đã chọn chuyển tiếp request đến server đích
5. Proxy server nhận phản hồi và trả về cho client
6. Nếu thất bại, proxy server thử lại với proxy khác

### 2. HTTPS Request (qua HTTP Proxy)

1. Client gửi CONNECT request đến proxy server
2. Proxy server chọn một HTTP proxy từ danh sách
3. Proxy server gửi CONNECT request đến proxy đã chọn
4. Proxy server thiết lập tunnel giữa client và server đích
5. Client tiến hành TLS handshake trực tiếp với server đích
6. Dữ liệu được truyền qua tunnel đã thiết lập

### 3. SOCKS5 Request

1. Client thực hiện bắt tay SOCKS5 với proxy server
2. Proxy server chọn một SOCKS5 proxy từ danh sách
3. Proxy server thực hiện bắt tay SOCKS5 với proxy đã chọn
4. Proxy server thiết lập tunnel giữa client và server đích
5. Client giao tiếp trực tiếp với server đích qua tunnel

## Lưu ý khi sử dụng SOCKS5 với HTTPS

Khi client sử dụng SOCKS5 để truy cập HTTPS, cần chú ý:

1. SSL handshake diễn ra trực tiếp giữa client và server đích (không qua proxy)
2. Client phải tự xử lý xác thực chứng chỉ SSL
3. Với cURL, cần thêm cờ `-k` để bỏ qua xác thực SSL nếu cần thiết

Lý do là vì SOCKS5 hoạt động ở tầng TCP/IP (layer 4/5), trong khi SSL/TLS hoạt động ở tầng ứng dụng (layer 7). SOCKS5 chỉ chuyển tiếp dữ liệu TCP nguyên bản mà không can thiệp vào nội dung.

HTTP proxy thì ngược lại, có thể can thiệp vào nội dung của kết nối (do hoạt động ở layer 7), nên có thể xử lý SSL/TLS ở phía proxy server. 