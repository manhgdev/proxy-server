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
│   │   ├── replacer/           # Thay thế proxy chết
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
│   │   ├── bonus/              # Quản lý hoa hồng
│   │   └── validators/         # Xác thực giao dịch
│   │
│   ├── renewal/                # Quản lý gia hạn proxy
│   │   ├── services/           # Dịch vụ gia hạn
│   │   ├── schedulers/         # Lập lịch gia hạn tự động
│   │   ├── notifiers/          # Gửi thông báo
│   │   └── processors/         # Xử lý gia hạn
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
│       ├── renewal/            # Quản lý gia hạn
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
- **Replacer**: Thay thế proxy chết hoặc kém hiệu suất
- **Pools**: Quản lý nhóm proxy theo loại, quốc gia...

Ví dụ về Proxy Replacement Service:

```javascript
// src/proxy-manager/replacer/proxy-replacement.service.js
import { ProxyRepository, StaticProxyPlanRepository, ProxyReplacementRepository } from '../../database/repositories';
import { NotificationService } from '../../common/services';
import { LogService } from '../../common/logger';

class ProxyReplacementService {
  async replaceProxy(userId, proxyId, reason = 'user_request', preferredCountry = null, requestId = null) {
    try {
      // Tìm proxy cần thay thế
      const originalProxy = await ProxyRepository.findById(proxyId);
      if (!originalProxy) {
        throw new Error('Proxy not found');
      }
      
      // Tìm gói proxy của người dùng có chứa proxy này
      const userProxyPlan = await StaticProxyPlanRepository.findByUserIdAndProxyId(userId, proxyId);
      if (!userProxyPlan) {
        throw new Error('Proxy does not belong to user');
      }
      
      // Tìm proxy mới phù hợp
      const criteria = {
        type: originalProxy.type,
        protocol: originalProxy.protocol,
        category: originalProxy.category,
        country: preferredCountry || originalProxy.country,
        sold: false,
        assigned: false
      };
      
      const newProxy = await ProxyRepository.findOne(criteria);
      if (!newProxy) {
        throw new Error('No suitable replacement proxy found');
      }
      
      // Tạo bản ghi thay thế
      const replacement = await ProxyReplacementRepository.create({
        user_id: userId,
        plan_id: userProxyPlan.user_plan_id,
        static_plan_id: userProxyPlan._id,
        original_proxy_id: proxyId,
        original_proxy_ip: originalProxy.ip,
        new_proxy_id: newProxy._id,
        new_proxy_ip: newProxy.ip,
        reason,
        requested_at: new Date(),
        processed_at: new Date(),
        status: 'completed',
        auto_replaced: !requestId,
        api_request_id: requestId
      });
      
      // Cập nhật proxy trong gói của người dùng
      await StaticProxyPlanRepository.replaceProxy(userProxyPlan._id, proxyId, newProxy._id);
      
      // Cập nhật trạng thái của các proxy
      await ProxyRepository.markAsNotAssigned(proxyId);
      await ProxyRepository.markAsSoldAndAssigned(newProxy._id, userId);
      
      // Gửi thông báo cho người dùng
      await NotificationService.sendProxyReplacementNotification(
        userId, 
        originalProxy.ip, 
        newProxy.ip
      );
      
      return {
        success: true,
        replacement_id: replacement._id,
        original_proxy: {
          id: originalProxy._id,
          ip: originalProxy.ip
        },
        new_proxy: {
          id: newProxy._id,
          ip: newProxy.ip,
          port: newProxy.port,
          username: userProxyPlan.custom_username || newProxy.username,
          password: userProxyPlan.custom_password || newProxy.password,
          protocol: newProxy.protocol,
          country: newProxy.country
        }
      };
    } catch (error) {
      LogService.error('Proxy replacement failed', error);
      throw error;
    }
  }
  
  async findDeadProxies() {
    return ProxyRepository.findByStatus('dead');
  }
  
  async processAutoReplacements() {
    try {
      // Tìm các proxy chết đang được gán cho người dùng
      const deadProxies = await ProxyRepository.findByStatusAndAssigned('dead', true);
      let replacedCount = 0;
      
      for (const proxy of deadProxies) {
        try {
          // Tìm gói proxy chứa proxy này
          const userProxyPlan = await StaticProxyPlanRepository.findByProxyId(proxy._id);
          if (!userProxyPlan) continue;
          
          // Thực hiện thay thế
          await this.replaceProxy(
            userProxyPlan.user_id, 
            proxy._id, 
            'dead', 
            proxy.country, 
            null
          );
          
          replacedCount++;
        } catch (error) {
          LogService.error(`Auto replacement failed for proxy ${proxy._id}`, error);
        }
      }
      
      return {
        total_dead_proxies: deadProxies.length,
        replaced_count: replacedCount
      };
    } catch (error) {
      LogService.error('Error processing auto replacements', error);
      throw error;
    }
  }
}

export default new ProxyReplacementService();
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
- **Bonus**: Quản lý hoa hồng theo mức nạp tiền
- **Validators**: Xác thực giao dịch và phòng chống gian lận

Ví dụ về xử lý hoa hồng trong Wallet Service:

```javascript
// src/wallet/services/wallet.service.js
import { WalletRepository, TransactionRepository, BonusTierRepository } from '../../database/repositories';
import { NotFoundError, InsufficientFundsError } from '../../common/errors';
import { LogService } from '../../common/logger';

class WalletService {
  async getWalletByUserId(userId) {
    const wallet = await WalletRepository.findByUserId(userId);
    if (!wallet) {
      throw new NotFoundError('Wallet not found');
    }
    return wallet;
  }
  
  async calculateBonus(amount, currency = 'VND') {
    try {
      // Tìm mức hoa hồng phù hợp
      const bonusTier = await BonusTierRepository.findApplicableTier(amount, currency);
      
      if (!bonusTier || !bonusTier.active) {
        return {
          bonus_amount: 0,
          bonus_percent: 0
        };
      }
      
      // Tính toán số tiền thưởng
      let bonusAmount = amount * (bonusTier.bonus_percent / 100);
      
      // Kiểm tra hạn mức thưởng tối đa nếu có
      if (bonusTier.bonus_max && bonusAmount > bonusTier.bonus_max) {
        bonusAmount = bonusTier.bonus_max;
      }
      
      return {
        bonus_amount: bonusAmount,
        bonus_percent: bonusTier.bonus_percent,
        tier_name: bonusTier.name
      };
    } catch (error) {
      LogService.error('Error calculating bonus', error);
      return {
        bonus_amount: 0,
        bonus_percent: 0
      };
    }
  }
  
  async processDeposit(userId, amount, paymentMethod, metadata = {}) {
    try {
      const wallet = await this.getWalletByUserId(userId);
      
      // Tính toán hoa hồng
      const bonus = await this.calculateBonus(amount, wallet.currency);
      const totalAmount = amount + bonus.bonus_amount;
      
      const balanceBefore = wallet.balance;
      const balanceAfter = balanceBefore + totalAmount;
      
      // Cập nhật số dư trong ví
      await WalletRepository.updateBalance(wallet._id, balanceAfter);
      
      // Tạo giao dịch
      const transaction = await TransactionRepository.create({
        wallet_id: wallet._id,
        user_id: userId,
        type: 'deposit',
        amount,
        bonus_amount: bonus.bonus_amount,
        bonus_percent: bonus.bonus_percent,
        balance_before: balanceBefore,
        balance_after: balanceAfter,
        currency: wallet.currency,
        status: 'completed',
        description: bonus.bonus_amount > 0 ? 
          `Nạp tiền vào ví (${amount} ${wallet.currency}) + Thưởng ${bonus.bonus_percent}% (${bonus.bonus_amount} ${wallet.currency})` :
          `Nạp tiền vào ví (${amount} ${wallet.currency})`,
        metadata: {
          ...metadata,
          payment_method: paymentMethod,
          bonus_tier: bonus.tier_name
        }
      });
      
      // Cập nhật thời gian nạp tiền gần nhất
      await WalletRepository.updateLastDepositTime(wallet._id);
      
      return {
        wallet: await this.getWalletByUserId(userId),
        transaction,
        bonus: bonus.bonus_amount > 0 ? {
          amount: bonus.bonus_amount,
          percent: bonus.bonus_percent,
          tier: bonus.tier_name
        } : null
      };
    } catch (error) {
      LogService.error('Error processing deposit', error);
      throw error;
    }
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

### 5. Renewal Service (`src/renewal`)

Quản lý gia hạn proxy tự động và thủ công:

- **Services**: Dịch vụ chính cho gia hạn
- **Schedulers**: Lập lịch gia hạn tự động
- **Notifiers**: Gửi thông báo cho người dùng
- **Processors**: Xử lý quy trình gia hạn

Ví dụ về Renewal Service:

```javascript
// src/renewal/services/renewal.service.js
import { UserPlanRepository, RenewalRecordRepository } from '../../database/repositories';
import { WalletService } from '../../wallet/services';
import { OrderService } from '../../billing/services';
import { NotificationService } from '../../common/services';
import { LogService } from '../../common/logger';

class RenewalService {
  async createRenewalOrder(userId, planId, duration = null) {
    try {
      // Lấy thông tin gói proxy
      const plan = await UserPlanRepository.findById(planId);
      
      if (!plan || plan.user_id.toString() !== userId.toString()) {
        throw new Error('Plan not found or does not belong to user');
      }
      
      // Lấy thông tin gói và giá gia hạn
      const packageInfo = await PackageRepository.findById(plan.package_id);
      const renewalDuration = duration || packageInfo.duration_days;
      const renewalPrice = plan.renewal_price || (packageInfo.renewal_price || packageInfo.price);
      
      // Tạo đơn hàng gia hạn
      const order = await OrderService.createOrder({
        user_id: userId,
        order_type: 'renewal',
        renewal_for: planId,
        items: [
          {
            package_id: plan.package_id,
            quantity: 1,
            price: renewalPrice,
            subtotal: renewalPrice
          }
        ],
        total_amount: renewalPrice
      });
      
      return {
        success: true,
        order_id: order._id,
        renewal_price: renewalPrice,
        renewal_duration: renewalDuration
      };
    } catch (error) {
      LogService.error('Error creating renewal order', error);
      throw error;
    }
  }
  
  async processRenewal(userId, orderId) {
    try {
      // Lấy thông tin đơn hàng
      const order = await OrderService.getOrderById(orderId);
      
      if (!order || order.user_id.toString() !== userId.toString()) {
        throw new Error('Order not found or does not belong to user');
      }
      
      if (order.order_type !== 'renewal') {
        throw new Error('Not a renewal order');
      }
      
      // Kiểm tra và thực hiện thanh toán
      const payment = await WalletService.processPayment(
        userId, 
        order.total_amount, 
        orderId, 
        `Proxy renewal: ${order.order_number}`
      );
      
      if (!payment.success) {
        return {
          success: false,
          error: payment.error,
          message: payment.message
        };
      }
      
      // Lấy thông tin gói đang gia hạn
      const currentPlan = await UserPlanRepository.findById(order.renewal_for);
      const packageInfo = await PackageRepository.findById(currentPlan.package_id);
      
      // Tính toán ngày hết hạn mới
      const currentEndDate = new Date(currentPlan.end_date);
      const newEndDate = new Date(currentEndDate);
      newEndDate.setDate(newEndDate.getDate() + packageInfo.duration_days);
      
      // Cập nhật gói hiện tại
      await UserPlanRepository.update(currentPlan._id, {
        end_date: newEndDate,
        renewal_count: (currentPlan.renewal_count || 0) + 1,
        updated_at: new Date()
      });
      
      // Tạo bản ghi gia hạn
      const renewalRecord = await RenewalRecordRepository.create({
        user_id: userId,
        original_plan_id: currentPlan._id,
        order_id: orderId,
        wallet_transaction_id: payment.transaction_id,
        renewal_date: new Date(),
        renewal_price: order.total_amount,
        renewal_duration_days: packageInfo.duration_days,
        auto_renewal: false,
        status: 'success'
      });
      
      // Cập nhật đơn hàng
      await OrderService.updateOrderStatus(orderId, 'completed', 'paid');
      
      // Gửi thông báo gia hạn thành công
      await NotificationService.sendRenewalSuccessNotification(userId, currentPlan._id, newEndDate);
      
      return {
        success: true,
        plan_id: currentPlan._id,
        renewal_record_id: renewalRecord._id,
        new_end_date: newEndDate
      };
    } catch (error) {
      LogService.error('Error processing renewal', error);
      throw error;
    }
  }
  
  async setAutoRenewal(userId, planId, enabled = true) {
    try {
      const plan = await UserPlanRepository.findById(planId);
      
      if (!plan || plan.user_id.toString() !== userId.toString()) {
        throw new Error('Plan not found or does not belong to user');
      }
      
      await UserPlanRepository.update(planId, {
        renewal_status: enabled ? 'auto' : 'disabled',
        updated_at: new Date()
      });
      
      return {
        success: true,
        plan_id: planId,
        auto_renewal: enabled
      };
    } catch (error) {
      LogService.error('Error setting auto renewal', error);
      throw error;
    }
  }
  
  async processAutoRenewals() {
    try {
      // Tìm các gói sắp hết hạn trong 24 giờ tới với auto renewal bật
      const expiringPlans = await UserPlanRepository.findExpiringPlans(24, 'auto');
      
      for (const plan of expiringPlans) {
        try {
          // Tạo và xử lý đơn hàng gia hạn
          const order = await this.createRenewalOrder(plan.user_id, plan._id);
          const result = await this.processRenewal(plan.user_id, order.order_id);
          
          if (result.success) {
            LogService.info(`Auto renewal successful for plan ${plan._id}`);
          } else {
            // Gửi thông báo về lỗi gia hạn
            await NotificationService.sendRenewalFailedNotification(
              plan.user_id, 
              plan._id, 
              result.error || 'Unknown error'
            );
            LogService.error(`Auto renewal failed for plan ${plan._id}: ${result.message}`);
          }
        } catch (error) {
          LogService.error(`Error in auto renewal for plan ${plan._id}`, error);
          
          // Tạo bản ghi gia hạn thất bại
          await RenewalRecordRepository.create({
            user_id: plan.user_id,
            original_plan_id: plan._id,
            renewal_date: new Date(),
            auto_renewal: true,
            status: 'failed',
            error_message: error.message
          });
          
          // Gửi thông báo thất bại
          await NotificationService.sendRenewalFailedNotification(
            plan.user_id, 
            plan._id, 
            error.message
          );
        }
      }
      
      return {
        success: true,
        processed_count: expiringPlans.length
      };
    } catch (error) {
      LogService.error('Error processing auto renewals', error);
      throw error;
    }
  }
}

export default new RenewalService();
```

### 6. Billing System (`src/billing`)

Quản lý đơn hàng, gói dịch vụ và thanh toán:

- **Services**: Quản lý đơn hàng, gói dịch vụ
- **Providers**: Tích hợp cổng thanh toán (Stripe, PayPal...)
- **Plans**: Quản lý các loại gói dịch vụ và giá

### 7. Monitoring (`src/monitoring`)

Giám sát và theo dõi trạng thái hệ thống:

- **Services**: Quản lý việc giám sát
- **Checkers**: Kiểm tra tình trạng và hiệu suất proxy
- **Alerts**: Cảnh báo khi phát hiện vấn đề

### 8. Analytics (`src/analytics`)

Thu thập và phân tích dữ liệu sử dụng:

- **Collectors**: Thu thập dữ liệu sử dụng proxy
- **Processors**: Xử lý dữ liệu thô
- **Reporters**: Tạo báo cáo và thống kê

### 9. Database Layer (`src/database`)

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

### 10. Web Dashboard (`src/web`)

Giao diện người dùng, gồm:

- **Dashboard**: Tổng quan và quản lý proxy
- **Auth**: Đăng nhập, đăng ký
- **Wallet**: Quản lý giao dịch ví tiền
- **Renewal**: Quản lý gia hạn gói proxy
- **Billing**: Quản lý gói dịch vụ và thanh toán
- **Account**: Quản lý thông tin tài khoản

## Quy trình xử lý chính

### Quy trình nạp tiền và hoa hồng

1. Người dùng chọn phương thức nạp tiền (chuyển khoản, thẻ tín dụng, PayPal...)
2. Hệ thống tạo một giao dịch nạp tiền với trạng thái `pending`
3. Người dùng hoàn thành thanh toán qua cổng thanh toán hoặc upload bằng chứng chuyển khoản
4. Admin xác nhận (với chuyển khoản) hoặc hệ thống xác nhận tự động (với cổng thanh toán)
5. `WalletService.processDeposit` xử lý nạp tiền và tính toán hoa hồng theo mức nạp
6. Tiền gốc và tiền thưởng được cộng vào ví người dùng
7. Giao dịch được ghi nhận với đầy đủ thông tin về hoa hồng

### Cấp proxy cho người dùng

1. Người dùng mua gói proxy (tĩnh, xoay hoặc bandwidth)
2. Hệ thống xử lý đơn hàng qua `OrderController → BillingService`
3. `WalletService` kiểm tra và trừ tiền từ ví người dùng
4. Sau khi thanh toán thành công, `ProxyService` sẽ:
   - Với proxy tĩnh/xoay: Tìm và cấp các proxy còn trống
   - Với bandwidth: Cấp quyền truy cập vào pool proxy

### Thay thế proxy chết

1. Hệ thống phát hiện proxy chết qua `MonitoringService.checkProxyStatus()`
2. `ProxyReplacementService.processAutoReplacements()` định kỳ tự động thay thế proxy chết
3. Hoặc người dùng gọi API `POST /api/v1/proxies/{proxy_id}/replace`
4. `ProxyReplacementService.replaceProxy()` tìm proxy mới phù hợp và thực hiện thay thế
5. Cập nhật thông tin trong gói proxy của người dùng
6. Gửi thông báo cho người dùng về việc thay thế
7. Trả về thông tin proxy mới cho người dùng

### Xử lý yêu cầu proxy

1. Người dùng gửi yêu cầu HTTP qua proxy với thông tin xác thực
2. `AuthMiddleware` xác thực yêu cầu
3. `ProxyHandler` xác định proxy cần sử dụng
4. Chuyển tiếp yêu cầu qua proxy đã chọn
5. `UsageLogger` ghi lại thông tin sử dụng

### Xoay proxy

1. Người dùng gọi API `POST /proxies/rotate`
2. `AuthMiddleware` xác thực yêu cầu
3. `ProxyController` gọi `RotatingProxyService.rotateProxy()`
4. Hệ thống chọn proxy mới và cập nhật `current_proxy_id`
5. Trả về thông tin proxy mới cho người dùng

### Gia hạn proxy

1. Người dùng chọn gói proxy muốn gia hạn trong dashboard
2. `RenewalService.createRenewalOrder()` tạo đơn hàng gia hạn
3. `RenewalService.processRenewal()` xử lý thanh toán từ ví người dùng
4. Cập nhật thời hạn mới cho gói proxy
5. Tạo bản ghi gia hạn trong `RenewalRecords`

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