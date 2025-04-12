# Proxy Server

Hệ thống proxy server với khả năng tự động luân chuyển proxy và hỗ trợ xác thực.

## Tính năng

- Tự động luân chuyển proxy
- Hỗ trợ cả HTTP và HTTPS
- Hỗ trợ SOCKS5 protocol
- Hỗ trợ xác thực proxy
- Mã nguồn sạch và hiệu quả
- Ghi nhật ký chi tiết

## Yêu cầu

- Go 1.24 trở lên
- Danh sách proxy server có sẵn

## Cài đặt

1. Clone repository:
```bash
git clone https://github.com/manhgdev/proxy-server.git
cd proxy-server
```

2. Cài đặt dependencies:
```bash
go mod download
```

## Cấu hình

1. Tạo file chứa danh sách proxy:
   - `proxy_http.txt` chứa danh sách HTTP proxy
   - `proxy_sockets5.txt` chứa danh sách SOCKS5 proxy

Mỗi proxy phải có định dạng:
```
ip:port
ip:port:username:password
username:password@ip:port
```

Ví dụ:
```
1.2.3.4:8080
1.2.3.4:8080:admin:123456
admin:123456@1.2.3.4:8080
proxy.example.com:8080
admin:123456@proxy.example.com:8080 
```

## Sử dụng

1. Khởi động server:
```bash
go run main.go
```

2. Sử dụng proxy server:

### HTTP/HTTPS qua HTTP Proxy
```bash
# Truy cập website HTTPS qua HTTP proxy
curl -x localhost:8081 https://api.zm.io.vn/check-ip/

# Truy cập website HTTP thông thường
curl -x localhost:8081 ip4.me/api/
```

### HTTP/HTTPS qua SOCKS5 Proxy
```bash
# Truy cập website HTTPS qua SOCKS5 proxy (cần flag -k nếu có vấn đề với SSL)
curl -x socks5://localhost:8081 https://api.zm.io.vn/check-ip/
# Hoặc bỏ qua xác thực SSL nếu cần
curl -k -x socks5://localhost:8081 https://api.zm.io.vn/check-ip/

# Truy cập website HTTP thông thường qua SOCKS5
curl -x socks5://localhost:8081 ip4.me/api/
```

## Lưu ý khi sử dụng SOCKS5 với HTTPS

Khi sử dụng SOCKS5 proxy với kết nối HTTPS, SSL handshake được thực hiện trực tiếp giữa client (curl) và server đích, không phải qua proxy. Do đó:

1. Nếu gặp lỗi SSL certificate, thêm tùy chọn `-k` vào lệnh curl:
   ```bash
   curl -k -x socks5://localhost:8081 https://api.zm.io.vn/check-ip/
   ```

2. Đối với ứng dụng khác, có thể cần cấu hình bỏ qua xác thực SSL tương tự
   
3. Lý do: SOCKS5 hoạt động ở tầng mạng (layer 4/5), khác với HTTP proxy hoạt động ở tầng ứng dụng (layer 7). SOCKS5 chỉ tạo tunnel nên không thể kiểm soát việc xác thực SSL.

## Cấu trúc dự án

```
.
├── main.go                  # Điểm khởi đầu chương trình
├── proxy_http.txt           # Danh sách HTTP proxies
├── proxy_sockets5.txt       # Danh sách SOCKS5 proxies
├── proxy/
│   ├── server.go            # Triển khai proxy server
│   ├── manager.go           # Quản lý danh sách proxy
│   ├── https_handler.go     # Xử lý kết nối HTTPS
│   └── socks5_handler.go    # Xử lý kết nối SOCKS5
└── utils/
    └── logger.go            # Tiện ích ghi log
```

## Giấy phép

MIT License 