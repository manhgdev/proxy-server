# Cấu trúc Code - Proxy Server System

## Tổng quan

Hệ thống Proxy Server được tổ chức theo kiến trúc microservices, hỗ trợ mở rộng và bảo trì dễ dàng. Mã nguồn được tổ chức như sau:

```
proxy-server/
├── config/                     # Cấu hình hệ thống
├── src/
│   ├── api/                    # API server 
│   │   ├── controllers/        # Điều khiển API
│   │   ├── middlewares/        # Middlewares
│   │   ├── routes/             # Định nghĩa routes
│   │   ├── validators/         # Xác thực đầu vào
│   │   └── server.js           # Khởi tạo server
│   │
│   ├── proxy-manager/          # Quản lý proxy
│   │   ├── services/           # Dịch vụ proxy
│   │   ├── handlers/           # Xử lý yêu cầu proxy
│   │   ├── rotator/            # Xoay proxy
│   │   └── pools/              # Quản lý pool proxy
│   │
│   ├── auth/                   # Xác thực & Ủy quyền
│   │   ├── strategies/         # Chiến lược xác thực
│   │   ├── services/           # Dịch vụ xác thực
│   │   └── helpers/            # Tiện ích
│   │
│   ├── wallet/                 # Quản lý ví tiền
│   │   ├── services/           # Dịch vụ ví
│   │   ├── transactions/       # Quản lý giao dịch
│   │   ├── providers/          # Nhà cung cấp thanh toán
│   │   └── validators/         # Xác thực giao dịch
│   │
│   ├── billing/                # Hệ thống thanh toán
│   │   ├── services/           # Dịch vụ thanh toán
│   │   ├── providers/          # Nhà cung cấp (Stripe, PayPal...)
│   │   └── plans/              # Quản lý gói dịch vụ
│   │
│   ├── monitoring/             # Giám sát hệ thống
│   │   ├── services/           # Dịch vụ giám sát
│   │   ├── checkers/           # Kiểm tra tình trạng proxy
│   │   └── alerts/             # Hệ thống cảnh báo
│   │
│   ├── analytics/              # Phân tích dữ liệu
│   │   ├── collectors/         # Thu thập dữ liệu
│   │   ├── processors/         # Xử lý dữ liệu
│   │   └── reporters/          # Tạo báo cáo
│   │
│   ├── database/               # Quản lý dữ liệu
│   │   ├── models/             # Mô hình dữ liệu
│   │   ├── repositories/       # Repository pattern
│   │   └── connections/        # Kết nối DB
│   │
│   ├── common/                 # Tiện ích chung
│   │   ├── utils/              # Công cụ trợ giúp
│   │   ├── errors/             # Quản lý lỗi
│   │   ├── logger/             # Ghi log
│   │   └── constants/          # Hằng số
│   │
│   └── web/                    # Giao diện người dùng
│       ├── dashboard/          # Dashboard
│       ├── auth/               # Xác thực
│       ├── wallet/             # Quản lý ví
│       ├── billing/            # Thanh toán
│       └── account/            # Quản lý tài khoản
│
├── scripts/                    # Scripts tiện ích
├── tests/                      # Unit & integration tests
├── docs/                       # Tài liệu
└── .env                        # Biến môi trường
```

## Thành phần chính

### 1. API Server (`src/api`)

API Server cung cấp RESTful API cho người dùng và các hệ thống tích hợp. Các thành phần chính:

- **Controllers**: Xử lý yêu cầu API và trả về phản hồi
- **Middlewares**: Bộ lọc, xác thực, xử lý lỗi
- **Routes**: Định nghĩa endpoint và liên kết với controllers
- **Validators**: Xác thực và làm sạch dữ liệu đầu vào

Ví dụ route định nghĩa:

```javascript
// src/api/routes/proxy.routes.js
import { Router } from 'express';
import { proxyController } from '../controllers';
import { authMiddleware } from '../middlewares';
import { validateProxyRequest } from '../validators';

const router = Router();

router.get(
  '/proxies',
  authMiddleware.authenticate, 
  proxyController.listProxies
);

router.post(
  '/proxies/rotate',
  authMiddleware.authenticate,
  validateProxyRequest,
  proxyController.rotateProxy
);

export default router;
```

### 2. Proxy Manager (`src/proxy-manager`)

Quản lý các proxy, bao gồm:

- **Services**: Dịch vụ cấp, xoay và quản lý proxy
- **Handlers**: Xử lý kết nối và chuyển tiếp yêu cầu
- **Rotator**: Quản lý xoay proxy theo chu kỳ hoặc yêu cầu
- **Pools**: Quản lý nhóm proxy theo loại, quốc gia...

Ví dụ về Rotating Proxy Service:

```javascript
// src/proxy-manager/services/rotating-proxy.service.js
import { ProxyRepository } from '../../database/repositories';
import { LogService } from '../../common/logger';

class RotatingProxyService {
  async rotateProxy(userPlanId) {
    try {
      const plan = await StaticProxyPlanRepository.findByUserPlanId(userPlanId);
      
      if (!plan || !plan.is_rotating) {
        throw new Error('Invalid plan or not a rotating proxy plan');
      }
      
      // Lấy proxy mới từ danh sách
      const availableProxies = plan.proxies.filter(p => p._id !== plan.current_proxy_id);
      
      if (availableProxies.length === 0) {
        throw new Error('No available proxies for rotation');
      }
      
      // Chọn proxy ngẫu nhiên
      const randomIndex = Math.floor(Math.random() * availableProxies.length);
      const newProxy = availableProxies[randomIndex];
      
      // Cập nhật proxy hiện tại
      await StaticProxyPlanRepository.updateCurrentProxy(plan._id, newProxy._id);
      
      return {
        success: true,
        proxy: {
          ip: newProxy.ip,
          port: newProxy.port,
          username: plan.custom_username || newProxy.username,
          password: plan.custom_password || newProxy.password
        }
      };
    } catch (error) {
      LogService.error('Proxy rotation failed', error);
      throw error;
    }
  }
}

export default new RotatingProxyService();
```

### 3. Auth Service (`src/auth`)

Xác thực và ủy quyền, bao gồm:

- **Strategies**: JWT, API Key, Basic Auth
- **Services**: Đăng nhập, đăng ký, quản lý token
- **Helpers**: Mã hóa, tạo token, xác thực password

### 4. Wallet Service (`src/wallet`)

Quản lý ví điện tử và giao dịch tài chính:

- **Services**: Dịch vụ quản lý ví và giao dịch
- **Transactions**: Xử lý và quản lý các loại giao dịch 
- **Providers**: Tích hợp với các nhà cung cấp dịch vụ thanh toán
- **Validators**: Xác thực giao dịch và phòng chống gian lận

Ví dụ về Wallet Service:

```javascript
// src/wallet/services/wallet.service.js
import { WalletRepository, TransactionRepository } from '../../database/repositories';
import { NotFoundError, InsufficientFundsError } from '../../common/errors';

class WalletService {
  async getWalletByUserId(userId) {
    const wallet = await WalletRepository.findByUserId(userId);
    if (!wallet) {
      throw new NotFoundError('Wallet not found');
    }
    return wallet;
  }
  
  async deposit(userId, amount, metadata = {}) {
    const wallet = await this.getWalletByUserId(userId);
    const balanceBefore = wallet.balance;
    const balanceAfter = balanceBefore + amount;
    
    // Cập nhật số dư trong ví
    await WalletRepository.updateBalance(wallet._id, balanceAfter);
    
    // Tạo giao dịch
    const transaction = await TransactionRepository.create({
      wallet_id: wallet._id,
      user_id: userId,
      type: 'deposit',
      amount,
      balance_before: balanceBefore,
      balance_after: balanceAfter,
      currency: wallet.currency,
      status: 'completed',
      description: 'Deposit to wallet',
      metadata
    });
    
    return {
      wallet: await this.getWalletByUserId(userId),
      transaction
    };
  }
  
  async withdraw(userId, amount, description = '', metadata = {}) {
    const wallet = await this.getWalletByUserId(userId);
    
    if (wallet.balance < amount) {
      throw new InsufficientFundsError('Insufficient funds');
    }
    
    const balanceBefore = wallet.balance;
    const balanceAfter = balanceBefore - amount;
    
    // Cập nhật số dư trong ví
    await WalletRepository.updateBalance(wallet._id, balanceAfter);
    
    // Tạo giao dịch
    const transaction = await TransactionRepository.create({
      wallet_id: wallet._id,
      user_id: userId,
      type: 'withdrawal',
      amount,
      balance_before: balanceBefore,
      balance_after: balanceAfter,
      currency: wallet.currency,
      status: 'completed',
      description: description || 'Withdrawal from wallet',
      metadata
    });
    
    return {
      wallet: await this.getWalletByUserId(userId),
      transaction
    };
  }
  
  async processPayment(userId, amount, orderId, description) {
    try {
      const result = await this.withdraw(userId, amount, description, { order_id: orderId });
      return {
        success: true,
        transaction_id: result.transaction._id,
        ...result
      };
    } catch (error) {
      if (error instanceof InsufficientFundsError) {
        return {
          success: false,
          error: 'insufficient_funds',
          message: error.message
        };
      }
      throw error;
    }
  }
}

export default new WalletService();
```

### 5. Billing System (`src/billing`)

Quản lý đơn hàng, gói dịch vụ và thanh toán:

- **Services**: Quản lý đơn hàng, gói dịch vụ
- **Providers**: Tích hợp cổng thanh toán (Stripe, PayPal...)
- **Plans**: Quản lý các loại gói dịch vụ và giá

### 6. Monitoring (`src/monitoring`)

Giám sát và theo dõi trạng thái hệ thống:

- **Services**: Quản lý việc giám sát
- **Checkers**: Kiểm tra tình trạng và hiệu suất proxy
- **Alerts**: Cảnh báo khi phát hiện vấn đề

### 7. Analytics (`src/analytics`)

Thu thập và phân tích dữ liệu sử dụng:

- **Collectors**: Thu thập dữ liệu sử dụng proxy
- **Processors**: Xử lý dữ liệu thô
- **Reporters**: Tạo báo cáo và thống kê

### 8. Database Layer (`src/database`)

Quản lý dữ liệu và tương tác với cơ sở dữ liệu:

- **Models**: Định nghĩa schema MongoDB
- **Repositories**: Truy vấn và thao tác với dữ liệu
- **Connections**: Kết nối và quản lý pool kết nối DB

Ví dụ Repository:

```javascript
// src/database/repositories/wallet.repository.js
import { Wallet } from '../models';

class WalletRepository {
  async findByUserId(userId) {
    return Wallet.findOne({ user_id: userId });
  }
  
  async create(userId, currency = 'USD') {
    return Wallet.create({
      user_id: userId,
      balance: 0,
      currency,
      locked_amount: 0,
      created_at: new Date(),
      updated_at: new Date(),
      is_active: true
    });
  }
  
  async updateBalance(walletId, newBalance) {
    return Wallet.updateOne(
      { _id: walletId },
      { 
        $set: { 
          balance: newBalance,
          updated_at: new Date(),
          last_usage_at: new Date()
        } 
      }
    );
  }
  
  async updateLockedAmount(walletId, lockedAmount) {
    return Wallet.updateOne(
      { _id: walletId },
      { 
        $set: { 
          locked_amount: lockedAmount,
          updated_at: new Date()
        } 
      }
    );
  }
}

export default new WalletRepository();
```

### 9. Web Dashboard (`src/web`)

Giao diện người dùng, gồm:

- **Dashboard**: Tổng quan và quản lý proxy
- **Auth**: Đăng nhập, đăng ký
- **Wallet**: Quản lý giao dịch ví tiền
- **Billing**: Quản lý gói dịch vụ và thanh toán
- **Account**: Quản lý thông tin tài khoản

## Quy trình xử lý chính

### Quy trình nạp tiền vào ví

1. Người dùng chọn phương thức nạp tiền (chuyển khoản, thẻ tín dụng, PayPal...)
2. Hệ thống tạo một giao dịch nạp tiền với trạng thái `pending`
3. Người dùng hoàn thành thanh toán qua cổng thanh toán hoặc upload bằng chứng chuyển khoản
4. Admin xác nhận (với chuyển khoản) hoặc hệ thống xác nhận tự động (với cổng thanh toán)
5. `WalletService` cập nhật số dư và tạo giao dịch với trạng thái `completed`

### Cấp proxy cho người dùng

1. Người dùng mua gói proxy (tĩnh, xoay hoặc bandwidth)
2. Hệ thống xử lý đơn hàng qua `OrderController → BillingService`
3. `WalletService` kiểm tra và trừ tiền từ ví người dùng
4. Sau khi thanh toán thành công, `ProxyService` sẽ:
   - Với proxy tĩnh/xoay: Tìm và cấp các proxy còn trống
   - Với bandwidth: Cấp quyền truy cập vào pool proxy

### Xử lý yêu cầu proxy

1. Người dùng gửi yêu cầu HTTP qua proxy với thông tin xác thực
2. `AuthMiddleware` xác thực yêu cầu
3. `ProxyHandler` xác định proxy cần sử dụng:
   - Với proxy tĩnh: Sử dụng proxy đã cấp
   - Với proxy xoay: Sử dụng proxy hiện tại hoặc xoay nếu đến chu kỳ
   - Với bandwidth: Chọn proxy từ pool
4. Chuyển tiếp yêu cầu qua proxy đã chọn
5. `UsageLogger` ghi lại thông tin sử dụng

### Xoay proxy

1. Người dùng gọi API `POST /proxies/rotate`
2. `AuthMiddleware` xác thực yêu cầu
3. `ProxyController` gọi `RotatingProxyService.rotateProxy()`
4. Hệ thống chọn proxy mới và cập nhật `current_proxy_id`
5. Trả về thông tin proxy mới cho người dùng

### Nạp thêm băng thông

1. Người dùng chọn gói nạp thêm băng thông
2. Tạo đơn hàng với loại `bandwidth_topup`
3. `WalletService` xử lý thanh toán từ ví người dùng
4. `BandwidthService` cập nhật `gb_remaining` trong `BandwidthPlan`
5. Tạo ghi chú trong `BandwidthTopups`

## Công nghệ sử dụng

- **Runtime**: Node.js với Bun
- **API Framework**: Express.js
- **Database**: MongoDB
- **Authentication**: JWT, API Keys
- **Logging**: Winston, Pino
- **Testing**: Jest, Supertest
- **Documentation**: Swagger, JSDoc
- **Frontend**: React, Material UI (Dashboard)
- **DevOps**: Docker, Docker Compose, GitHub Actions 