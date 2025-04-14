# REST API cho Proxy Server (MongoDB)

API endpoints cho hệ thống proxy server sử dụng MongoDB.

## Authentication

### Đăng nhập

```
POST /api/auth/login
```

**Request:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "user": {
      "_id": "507f1f77bcf86cd799439011",
      "username": "admin",
      "email": "admin@example.com"
    }
  }
}
```

### Đăng ký

```
POST /api/auth/register
```

**Request:**
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "password123",
  "fullname": "Nguyễn Văn A",
  "phone": "0901234567"
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Đăng ký thành công",
  "data": {
    "user_id": "507f1f77bcf86cd799439012",
    "username": "newuser",
    "api_key": "uk_123456789abcdef"
  }
}
```

## Quản lý Người dùng

### Lấy thông tin người dùng

```
GET /api/users/:id
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "_id": "507f1f77bcf86cd799439011",
    "username": "admin",
    "email": "admin@example.com",
    "fullname": "Admin User",
    "phone": "0901234567",
    "created_at": "2023-01-01T00:00:00Z",
    "active": true
  }
}
```

### Cập nhật thông tin người dùng

```
PUT /api/users/:id
```

**Request:**
```json
{
  "email": "newemail@example.com",
  "password": "newpassword",
  "fullname": "Admin Updated",
  "phone": "0909876543"
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Cập nhật thông tin thành công"
}
```

## Quản lý Loại Proxy

### Lấy danh sách loại proxy

```
GET /api/proxy-types
```

**Response:**
```json
{
  "status": "success",
  "data": [
    {
      "_id": "507f1f77bcf86cd799439013",
      "name": "Static Datacenter",
      "protocol": "http",
      "category": "datacenter",
      "isp": "AWS",
      "is_rotating": false,
      "price_per_ip": 1.99,
      "price_per_day": 0.5,
      "active": true
    },
    {
      "_id": "507f1f77bcf86cd799439014",
      "name": "Rotating Residential",
      "protocol": "socks5",
      "category": "residential",
      "isp": "Viettel",
      "is_rotating": true,
      "rotation_type": "time-based",
      "price_per_ip": 3.99,
      "price_per_day": 1.2,
      "active": true
    }
  ]
}
```

### Tạo loại proxy mới

```
POST /api/proxy-types
```

**Request:**
```json
{
  "name": "Premium Residential",
  "protocol": "http",
  "category": "residential",
  "isp": "FPT",
  "is_rotating": false,
  "price_per_ip": 2.99,
  "price_per_day": 0.8
}
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "_id": "507f1f77bcf86cd799439015",
    "name": "Premium Residential",
    "protocol": "http",
    "category": "residential",
    "isp": "FPT",
    "is_rotating": false,
    "price_per_ip": 2.99,
    "price_per_day": 0.8,
    "active": true
  }
}
```

## Quản lý Proxy

### Lấy danh sách proxy

```
GET /api/proxies
```

**Query Parameters:**
- `protocol`: Lọc theo giao thức (http, socks5)
- `category`: Lọc theo danh mục (residential, datacenter)
- `isp`: Lọc theo nhà mạng
- `status`: Lọc theo trạng thái (active, inactive, error)
- `assigned`: Lọc theo trạng thái gán (true, false)
- `limit`: Số lượng trả về (mặc định: 20)
- `skip`: Số lượng bỏ qua (mặc định: 0)

**Response:**
```json
{
  "status": "success",
  "data": {
    "proxies": [
      {
        "_id": "507f1f77bcf86cd799439016",
        "ip": "192.168.1.1",
        "port": 8080,
        "protocol": "http",
        "category": "datacenter",
        "isp": "AWS",
        "country": "US",
        "city": "New York",
        "status": "active",
        "success_count": 150,
        "fail_count": 2,
        "assigned": true
      },
      {
        "_id": "507f1f77bcf86cd799439017",
        "ip": "192.168.1.2",
        "port": 8080,
        "protocol": "http",
        "category": "datacenter",
        "isp": "AWS",
        "country": "US",
        "city": "New York",
        "status": "active",
        "success_count": 120,
        "fail_count": 1,
        "assigned": false
      }
    ],
    "pagination": {
      "total": 145,
      "limit": 20,
      "skip": 0
    }
  }
}
```

### Thêm proxy mới

```
POST /api/proxies
```

**Request:**
```json
{
  "ip": "192.168.1.5",
  "port": 8080,
  "username": "proxyuser",
  "password": "proxypass",
  "protocol": "http",
  "category": "datacenter",
  "isp": "AWS",
  "country": "US",
  "city": "New York",
  "proxy_type_id": "507f1f77bcf86cd799439013"
}
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "_id": "507f1f77bcf86cd799439018",
    "ip": "192.168.1.5",
    "port": 8080,
    "protocol": "http",
    "category": "datacenter",
    "isp": "AWS",
    "country": "US",
    "city": "New York",
    "status": "active",
    "assigned": false
  }
}
```

### Nhập danh sách proxy từ file

```
POST /api/proxies/import
```

**Request:**
Multipart form data với file text mỗi dòng có định dạng:
- `ip:port`
- `ip:port:username:password`
- `username:password@ip:port`

Và các trường:
- `proxy_type_id`: ID của loại proxy
- `protocol`: Giao thức (http/socks5)
- `category`: Danh mục (residential/datacenter)
- `isp`: Nhà mạng

**Response:**
```json
{
  "status": "success",
  "message": "Nhập 56 proxy thành công",
  "data": {
    "succeeded": 56,
    "failed": 4,
    "errors": [
      "Dòng 3: Định dạng không hợp lệ",
      "Dòng 10: IP/port đã tồn tại"
    ]
  }
}
```

### Kiểm tra trạng thái proxy

```
GET /api/proxies/:id/check
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "_id": "507f1f77bcf86cd799439016",
    "is_working": true,
    "response_time": 235,
    "external_ip": "203.0.113.1",
    "checked_at": "2023-10-15T14:30:00Z"
  }
}
```

## Quản lý Đơn hàng

### Tạo đơn hàng mới

```
POST /api/orders
```

**Request:**
```json
{
  "user_id": "507f1f77bcf86cd799439011",
  "payment_method": "bank_transfer",
  "items": [
    {
      "proxy_type_id": "507f1f77bcf86cd799439013",
      "protocol": "http",
      "isp": "AWS",
      "category": "datacenter",
      "is_rotating": false,
      "quantity": 5,
      "duration_days": 30,
      "username": "customuser1",
      "password": "custompass1"
    },
    {
      "proxy_type_id": "507f1f77bcf86cd799439014",
      "protocol": "socks5",
      "isp": "Viettel",
      "category": "residential",
      "is_rotating": true,
      "quantity": 10,
      "duration_days": 30,
      "username": "customuser2",
      "password": "custompass2"
    }
  ]
}
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "_id": "507f1f77bcf86cd799439020",
    "order_number": "ORD123456",
    "user_id": "507f1f77bcf86cd799439011",
    "total_amount": 549.5,
    "status": "pending",
    "payment_method": "bank_transfer",
    "payment_status": "pending",
    "created_at": "2023-10-15T10:30:00Z",
    "items": [
      {
        "proxy_type_id": "507f1f77bcf86cd799439013",
        "protocol": "http",
        "isp": "AWS",
        "category": "datacenter",
        "is_rotating": false,
        "quantity": 5,
        "duration_days": 30,
        "username": "customuser1",
        "password": "custompass1",
        "price": 1.99,
        "subtotal": 299.5
      },
      {
        "proxy_type_id": "507f1f77bcf86cd799439014",
        "protocol": "socks5",
        "isp": "Viettel",
        "category": "residential",
        "is_rotating": true,
        "quantity": 10,
        "duration_days": 30,
        "username": "customuser2",
        "password": "custompass2",
        "price": 3.99,
        "subtotal": 250.0
      }
    ]
  }
}
```

### Cập nhật trạng thái đơn hàng

```
PUT /api/orders/:id
```

**Request:**
```json
{
  "status": "completed",
  "payment_status": "paid"
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Cập nhật đơn hàng thành công"
}
```

## Quản lý UserProxies

### Danh sách proxy của người dùng

```
GET /api/users/:id/proxies
```

**Response:**
```json
{
  "status": "success",
  "data": [
    {
      "_id": "507f1f77bcf86cd799439030",
      "name": "Datacenter Package",
      "protocol": "http",
      "category": "datacenter",
      "is_rotating": false,
      "isp": "AWS",
      "username": "customuser1",
      "start_date": "2023-10-15T00:00:00Z",
      "end_date": "2023-11-14T23:59:59Z",
      "active": true,
      "api_key": "pk_dc123456789abcdef",
      "proxy_count": 5
    },
    {
      "_id": "507f1f77bcf86cd799439031",
      "name": "Rotating Residential",
      "protocol": "socks5",
      "category": "residential",
      "is_rotating": true,
      "isp": "Viettel",
      "username": "customuser2",
      "start_date": "2023-10-15T00:00:00Z",
      "end_date": "2023-11-14T23:59:59Z",
      "active": true,
      "api_key": "pk_rr987654321zyxwvu",
      "rotation_interval": 300,
      "rotation_url": "https://example.com/rotate/user1_abc123",
      "proxy_count": 10
    }
  ]
}
```

### Lấy chi tiết gói proxy

```
GET /api/user-proxies/:id
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "_id": "507f1f77bcf86cd799439030",
    "name": "Datacenter Package",
    "user_id": "507f1f77bcf86cd799439011",
    "order_id": "507f1f77bcf86cd799439020",
    "protocol": "http",
    "category": "datacenter",
    "is_rotating": false,
    "isp": "AWS",
    "username": "customuser1",
    "password": "custompass1",
    "api_key": "pk_dc123456789abcdef",
    "start_date": "2023-10-15T00:00:00Z",
    "end_date": "2023-11-14T23:59:59Z",
    "active": true,
    "proxies": [
      {
        "_id": "507f1f77bcf86cd799439016",
        "ip": "192.168.1.1",
        "port": 8080,
        "protocol": "http"
      },
      {
        "_id": "507f1f77bcf86cd799439017",
        "ip": "192.168.1.2",
        "port": 8080,
        "protocol": "http"
      }
    ]
  }
}
```

## Sử dụng Proxy

### Xoay proxy thủ công

```
POST /api/user-proxies/:id/rotate
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "new_proxy": {
      "ip": "192.168.2.1",
      "port": 1080,
      "protocol": "socks5"
    },
    "rotated_at": "2023-10-15T15:30:00Z"
  }
}
```

### Lấy proxy hiện tại bằng API key

```
GET /api/proxy
```

**Headers:**
```
X-API-Key: pk_rr987654321zyxwvu
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "type": "rotating",
    "proxy": {
      "host": "192.168.2.1",
      "port": 1080,
      "username": "customuser2",
      "password": "custompass2",
      "protocol": "socks5",
      "last_rotation": "2023-10-15T15:30:00Z",
      "next_rotation": "2023-10-15T15:35:00Z"
    }
  }
}
```

hoặc nếu là proxy tĩnh:

```json
{
  "status": "success",
  "data": {
    "type": "static",
    "proxies": [
      {
        "host": "192.168.1.1",
        "port": 8080,
        "username": "customuser1",
        "password": "custompass1",
        "protocol": "http"
      },
      {
        "host": "192.168.1.2",
        "port": 8080,
        "username": "customuser1",
        "password": "custompass1",
        "protocol": "http"
      }
    ]
  }
}
```

### Xoay proxy thủ công bằng API key

```
POST /api/proxy/rotate
```

**Headers:**
```
X-API-Key: pk_rr987654321zyxwvu
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "new_proxy": {
      "host": "192.168.2.3",
      "port": 1080,
      "username": "customuser2",
      "password": "custompass2",
      "protocol": "socks5"
    },
    "rotated_at": "2023-10-15T15:40:00Z"
  }
}
```

## Thống kê

### Thống kê sử dụng gói proxy

```
GET /api/user-proxies/:id/stats
```

**Query Parameters:**
- `start_date`: Ngày bắt đầu (YYYY-MM-DD)
- `end_date`: Ngày kết thúc (YYYY-MM-DD)

**Response:**
```json
{
  "status": "success",
  "data": {
    "total_requests": 1250,
    "success_rate": 98.5,
    "avg_response_time": 156,
    "total_bytes": 240568234,
    "daily_stats": [
      {
        "date": "2023-10-10",
        "requests": 120,
        "success_rate": 99.1,
        "avg_response_time": 145
      },
      {
        "date": "2023-10-11",
        "requests": 134,
        "success_rate": 98.2,
        "avg_response_time": 160
      }
    ],
    "most_used_proxy": {
      "_id": "507f1f77bcf86cd799439016",
      "ip": "192.168.1.1",
      "port": 8080,
      "requests": 562
    }
  }
}
```

### Thống kê sử dụng người dùng

```
GET /api/users/:id/stats
```

**Query Parameters:**
- `start_date`: Ngày bắt đầu (YYYY-MM-DD)
- `end_date`: Ngày kết thúc (YYYY-MM-DD)

**Response:**
```json
{
  "status": "success",
  "data": {
    "total_requests": 5648,
    "active_packages": 2,
    "total_proxies": 15,
    "usage_by_package": [
      {
        "package_id": "507f1f77bcf86cd799439030",
        "package_name": "Datacenter Package",
        "requests": 2345,
        "percentage": 41.5
      },
      {
        "package_id": "507f1f77bcf86cd799439031",
        "package_name": "Rotating Residential",
        "requests": 3303,
        "percentage": 58.5
      }
    ],
    "usage_by_protocol": [
      {
        "protocol": "http",
        "requests": 2345,
        "percentage": 41.5
      },
      {
        "protocol": "socks5",
        "requests": 3303,
        "percentage": 58.5
      }
    ]
  }
}
```

## Xử lý lỗi

Tất cả các lỗi sẽ trả về định dạng sau:

```json
{
  "status": "error",
  "error": {
    "code": "NOT_FOUND",
    "message": "Không tìm thấy proxy với ID 507f1f77bcf86cd799439999"
  }
}
```

### Mã lỗi phổ biến

- `BAD_REQUEST`: Yêu cầu không hợp lệ
- `UNAUTHORIZED`: Cần xác thực
- `FORBIDDEN`: Không có quyền truy cập
- `NOT_FOUND`: Không tìm thấy tài nguyên
- `VALIDATION_ERROR`: Lỗi xác thực dữ liệu
- `PROXY_ERROR`: Lỗi kết nối với proxy
- `SERVER_ERROR`: Lỗi máy chủ 