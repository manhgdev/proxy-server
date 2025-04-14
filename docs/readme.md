# Hệ thống Proxy Server - Tài liệu Tổng Quan

## Giới thiệu

Hệ thống Proxy Server là giải pháp trung gian cho phép người dùng kết nối internet thông qua các địa chỉ IP khác nhau. Hệ thống hỗ trợ nhiều loại proxy khác nhau, bao gồm proxy tĩnh, proxy xoay và proxy residential với mô hình tính phí theo băng thông, đáp ứng đa dạng nhu cầu của người dùng.

## Các loại Proxy hỗ trợ

### Proxy Tĩnh (Static Proxy)
- **Đặc điểm**: Địa chỉ IP cố định không thay đổi
- **Ứng dụng**: Phù hợp cho SEO, quản lý nhiều tài khoản, duyệt web ổn định
- **Loại proxy**: Datacenter hoặc Residential

### Proxy Xoay (Rotating Proxy)
- **Đặc điểm**: Tự động thay đổi địa chỉ IP theo chu kỳ hoặc yêu cầu
- **Tùy chỉnh**: Người dùng có thể thiết lập khoảng thời gian xoay
- **Ứng dụng**: Thu thập dữ liệu, scraping, tự động hóa, tránh giới hạn truy cập

### Proxy Residential theo Băng Thông (Bandwidth Model)
- **Đặc điểm**: Sử dụng IP từ các kết nối internet thực của người dùng thông thường
- **Tính phí**: Dựa trên lượng dữ liệu sử dụng (GB)
- **Ứng dụng**: Truy cập nội dung bị giới hạn địa lý, xác thực cao, thân thiện với các trang web lớn

## Hệ thống Ví điện tử

Hệ thống bao gồm một cơ chế ví tiền điện tử tích hợp giúp người dùng dễ dàng nạp tiền và thanh toán:

### Đặc điểm chính
- **Ví cá nhân**: Mỗi người dùng có một ví riêng để quản lý số dư
- **Nạp tiền**: Hỗ trợ nhiều phương thức nạp tiền (chuyển khoản, thẻ tín dụng, Paypal...)
- **Lịch sử giao dịch**: Ghi lại chi tiết tất cả hoạt động nạp và sử dụng
- **Thanh toán tự động**: Mua proxy hoặc nạp thêm băng thông trực tiếp từ số dư trong ví

### Quy trình thanh toán
1. Người dùng nạp tiền vào ví
2. Khi mua proxy hoặc nạp thêm băng thông, tiền sẽ được trừ trực tiếp từ ví
3. Hệ thống cảnh báo khi số dư thấp
4. Lịch sử giao dịch chi tiết hiển thị trong trang cá nhân

## Kiến trúc Hệ thống

Hệ thống được xây dựng dựa trên kiến trúc microservices với các thành phần chính:

1. **API Server**: Xử lý yêu cầu của người dùng qua RESTful API
2. **Proxy Manager**: Quản lý và phân phối proxy cho người dùng
3. **Auth Service**: Xác thực và ủy quyền
4. **Billing System**: Quản lý đơn hàng, gói dịch vụ và thanh toán
5. **Wallet Service**: Quản lý ví điện tử và các giao dịch
6. **Monitoring Service**: Giám sát trạng thái và hiệu suất của proxy
7. **Analytics Engine**: Phân tích và báo cáo việc sử dụng

## Quy trình mua và sử dụng Proxy

### Đăng ký và nạp tiền
1. Người dùng đăng ký tài khoản
2. Nạp tiền vào ví thông qua các phương thức được hỗ trợ
3. Theo dõi số dư và lịch sử giao dịch trong trang cá nhân

### Mua và sử dụng Proxy
1. Chọn loại proxy (tĩnh, xoay, residential) và gói dịch vụ phù hợp
2. Tùy chỉnh các thông số (quốc gia, nhà mạng, giao thức, số lượng IP, etc.)
3. Thanh toán đơn hàng sử dụng số dư trong ví
4. Hệ thống cấp proxy và thông tin xác thực
5. Sử dụng proxy thông qua các phương thức được cung cấp

### Sử dụng Proxy
- **Xác thực**: Sử dụng username/password hoặc API key
- **Kết nối**: Qua HTTP, SOCKS5 hoặc API endpoints
- **Quản lý**: Theo dõi sử dụng, điều chỉnh cài đặt thông qua dashboard

### Đặc điểm theo loại Proxy

#### Proxy Tĩnh và Xoay:
- Được cấp một số lượng IP cố định trong thời gian sử dụng
- Mỗi proxy chỉ được bán cho một người dùng (không tái sử dụng)
- Proxy xoay có thể thay đổi IP theo khoảng thời gian hoặc qua API endpoint

#### Proxy Residential theo Băng Thông:
- Không giới hạn số lượng IP có thể sử dụng
- Thanh toán dựa trên dữ liệu đã sử dụng (GB)
- Có thể nạp thêm băng thông khi sắp hết (thanh toán từ ví)

## Cấu trúc Cơ sở Dữ liệu

Hệ thống sử dụng MongoDB với các collections chính như sau:
- **Users**: Thông tin người dùng
- **Wallets**: Ví tiền của người dùng
- **WalletTransactions**: Lịch sử giao dịch ví tiền
- **ProductPackages**: Các gói dịch vụ proxy
- **Proxies**: Thông tin chi tiết về từng proxy
- **ProxyPools**: Nhóm proxy theo quốc gia, nhà mạng, v.v.
- **Orders**: Đơn hàng và thanh toán
- **UserPlans**: Gói dịch vụ của người dùng
- **StaticProxyPlans/BandwidthPlans**: Chi tiết từng loại gói dịch vụ
- **ProxyUsageLogs**: Nhật ký sử dụng proxy

## Tài liệu Tham khảo

- [Sơ đồ Mối quan hệ](diagram.md): Sơ đồ mối quan hệ giữa các collections
- [Cấu trúc Dữ liệu](database-schema.md): Chi tiết schema MongoDB
- [API Endpoints](rest-api.md): Tài liệu API của hệ thống
- [Cấu trúc mã nguồn](code-structure.md): Chi tiết cấu trúc mã nguồn

## Quy trình cài đặt và triển khai

1. Cài đặt các dependency:
```bash
bun install
```

2. Cấu hình môi trường:
```bash
cp .env.example .env
# Chỉnh sửa các biến môi trường
```

3. Khởi tạo cơ sở dữ liệu:
```bash
bun run init-db
```

4. Khởi chạy ứng dụng:
```bash
bun run dev    # Môi trường phát triển
bun run start  # Môi trường sản xuất
```

## Các tính năng đặc biệt

- **Ví điện tử tích hợp**: Nạp tiền và thanh toán dễ dàng
- **Địa chỉ IP Sticky**: Giữ nguyên địa chỉ IP trong một phiên làm việc
- **Định tuyến theo Quốc gia**: Chọn IP từ quốc gia cụ thể
- **Xác thực IP**: Xác thực địa chỉ IP thật của proxy
- **Cảnh báo**: Thông báo khi sắp hết băng thông, gói sắp hết hạn, số dư thấp
- **Dashboard**: Giao diện quản lý trực quan
- **API**: RESTful API cho tích hợp với các ứng dụng khác 