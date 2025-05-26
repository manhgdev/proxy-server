# Thiết kế MongoDB Schema cho Hệ thống Proxy Tích hợp

## Collections chính

### 1. Users
```javascript
{
  _id: ObjectId,
  username: String,
  password_hash: String,
  email: String,
  fullname: String,
  phone: String,
  created_at: Date,
  updated_at: Date,
  active: Boolean,
  api_key: String,         // Key bảo mật API
  access_token: String,    // Token truy cập
  wallet_id: ObjectId,     // Tham chiếu đến ví tiền
  billing_info: {
    company_name: String,
    tax_id: String,
    address: String,
    payment_methods: [
      {
        type: String,      // "credit_card", "paypal", etc.
        details: Object    // Card info, etc.
      }
    ]
  }
}
```

### 2. Wallets
```javascript
{
  _id: ObjectId,
  user_id: ObjectId,       // Tham chiếu đến người dùng
  balance: Number,         // Số dư hiện tại 
  currency: String,        // VND, USD, etc.
  locked_amount: Number,   // Số tiền đang tạm khóa
  created_at: Date,
  updated_at: Date,
  last_deposit_at: Date,
  last_usage_at: Date,
  is_active: Boolean
}
```

### 3. WalletTransactions
```javascript
{
  _id: ObjectId,
  wallet_id: ObjectId,
  user_id: ObjectId,
  type: String,            // "deposit", "withdrawal", "purchase", "refund", "renewal"
  amount: Number,          // Số tiền giao dịch
  balance_before: Number,  // Số dư trước giao dịch
  balance_after: Number,   // Số dư sau giao dịch
  currency: String,
  status: String,          // "pending", "completed", "failed", "cancelled"
  description: String,
  bonus_amount: Number,    // Tiền thưởng/hoa hồng nếu có
  bonus_percent: Number,   // Phần trăm thưởng/hoa hồng
  metadata: {
    payment_method: String,    // "bank_transfer", "credit_card", etc.
    order_id: ObjectId,        // Tham chiếu đến đơn hàng (nếu có)
    plan_id: ObjectId,         // Tham chiếu đến UserPlans nếu là renewal
    renewal_id: ObjectId,      // Tham chiếu đến RenewalRecords nếu là renewal
    transaction_id: String,    // ID giao dịch từ bên thứ ba
    payment_proof: String      // URL đến ảnh chứng từ
  },
  created_at: Date,
  updated_at: Date
}
```

### 4. BonusTiers
```javascript
{
  _id: ObjectId,
  name: String,              // Tên mức thưởng
  min_amount: Number,        // Số tiền nạp tối thiểu
  bonus_percent: Number,     // Phần trăm thưởng
  bonus_max: Number,         // Số tiền thưởng tối đa (nếu có)
  active: Boolean,
  currency: String,          // VND, USD, etc.
  start_date: Date,          // Ngày bắt đầu áp dụng
  end_date: Date,            // Ngày kết thúc (null nếu không giới hạn)
  description: String,
  created_at: Date,
  updated_at: Date
}
```

### 5. ProductPackages
```javascript
{
  _id: ObjectId,
  name: String,                // "Datacenter Static", "Residential Bandwidth", etc.
  description: String,
  type: String,                // "static", "rotating", "bandwidth"
  category: String,            // "residential", "datacenter"
  protocol: String,            // "http", "socks5", "mixed"
  is_rotating: Boolean,
  is_bandwidth: Boolean,       // True for bandwidth-based packages
  duration_days: Number,
  renewal_available: Boolean,  // Có thể gia hạn hay không
  renewal_price: Number,       // Giá gia hạn (thường sẽ thấp hơn giá mua mới)
  renewal_discount: Number,    // % giảm giá khi gia hạn
  price: Number,               // Base price or per IP price
  price_per_gb: Number,        // For bandwidth packages
  min_quantity: Number,        // Minimum number of IPs or GB
  max_quantity: Number,
  default_quantity: Number,
  allowed_countries: [String],
  allowed_isps: [String],
  features: [String],          // "sticky_ip", "country_routing", etc.
  price_tiers: [
    {
      min_quantity: Number,    // Min IPs or GB
      price: Number            // Price at this tier
    }
  ],
  custom_quantity_allowed: Boolean,
  active: Boolean,
  created_at: Date,
  updated_at: Date
}
```

### 6. Proxies
```javascript
{
  _id: ObjectId,
  ip: String,
  port: Number,
  username: String,
  password: String,
  protocol: String,           // "http", "socks5"
  type: String,               // "static", "rotating"
  category: String,           // "residential", "datacenter" 
  country: String,            // ISO country code
  city: String,
  region: String,
  isp: String,
  asn: Number,
  connection_type: String,    // "mobile", "broadband", "fiber"
  status: String,             // "active", "inactive", "error", "dead"
  last_checked: Date,
  last_status_change: Date,
  success_rate: Number,       // 0-100%
  avg_response_time: Number,  // ms
  is_blacklisted: Boolean,
  tags: [String],
  pool_id: ObjectId,          // Reference to a ProxyPool if applicable
  sold: Boolean,              // True if it has ever been sold
  assigned: Boolean,          // True if currently assigned to an active plan
  first_sold_at: Date,
  first_user_id: ObjectId,
  created_at: Date,
  updated_at: Date
}
```

### 7. ProxyPools
```javascript
{
  _id: ObjectId,
  name: String,                // "Vietnam Mobile", "US Residential"
  description: String,
  group: String,               // "vn-mobile", "us-residential"
  countries: [String],
  isps: [String],
  connection_types: [String],
  proxy_count: Number,
  active_proxy_count: Number,
  entry_point: String,         // Gateway address
  port_range: {
    start: Number,
    end: Number
  },
  username_format: String,
  password_format: String,
  is_bandwidth_pool: Boolean,  // True if this pool is used for bandwidth plans
  active: Boolean,
  created_at: Date,
  updated_at: Date
}
```

### 8. Orders
```javascript
{
  _id: ObjectId,
  user_id: ObjectId,
  wallet_id: ObjectId,         // Tham chiếu đến ví tiền để thanh toán
  order_number: String,
  total_amount: Number,
  payment_source: String,      // "wallet", "credit_card", "bank_transfer"
  wallet_transaction_id: ObjectId,  // Tham chiếu đến giao dịch ví (nếu thanh toán bằng ví)
  order_type: String,          // "new", "renewal", "topup"
  renewal_for: ObjectId,       // Reference to UserPlans being renewed
  items: [
    {
      package_id: ObjectId,     // Reference to a ProductPackage
      package_name: String,
      quantity: Number,         // Number of IPs or GB
      price: Number,            // Price per unit
      subtotal: Number,
      custom_config: {
        username: String,
        password: String,
        rotation_interval: Number,
        sticky_ip: Boolean,
        countries: [String],
        isps: [String]
      }
    }
  ],
  status: String,              // "pending", "completed", "cancelled"
  payment_method: String,
  payment_status: String,      // "pending", "paid", "failed"
  created_at: Date,
  updated_at: Date
}
```

### 9. UserPlans
```javascript
{
  _id: ObjectId,
  user_id: ObjectId,
  package_id: ObjectId,        // Reference to the ProductPackage
  order_id: ObjectId,
  plan_type: String,           // "static", "rotating", "bandwidth"
  start_date: Date,
  end_date: Date,
  active: Boolean,
  api_key: String,             // API key for this specific plan
  renewal_status: String,      // "not_set", "pending", "auto", "disabled"
  renewal_reminder_sent: Boolean,
  renewal_reminder_date: Date,
  renewal_price: Number,       // Stored price for renewal
  original_plan_id: ObjectId,  // If this is a renewed plan, reference to the original
  renewal_count: Number,       // Number of times this plan has been renewed
  created_at: Date,
  updated_at: Date
}
```

### 10. RenewalRecords
```javascript
{
  _id: ObjectId,
  user_id: ObjectId,
  original_plan_id: ObjectId,   // Plan being renewed
  new_plan_id: ObjectId,        // Newly created plan after renewal
  order_id: ObjectId,           // Order for the renewal
  wallet_transaction_id: ObjectId,
  renewal_date: Date,
  renewal_price: Number,
  renewal_duration_days: Number,
  auto_renewal: Boolean,        // Whether it was automatic
  status: String,               // "success", "failed"
  error_message: String,        // If failed
  created_at: Date
}
```

### 11. StaticProxyPlans (embedded document or separate collection)
```javascript
{
  _id: ObjectId,
  user_plan_id: ObjectId,      // Reference to UserPlans
  proxies: [ObjectId],         // IDs of proxies assigned to this plan
  protocol: String,            // "http", "socks5"
  category: String,            // "residential", "datacenter"
  is_rotating: Boolean,
  rotation_interval: Number,   // Seconds between rotations (if rotating)
  rotation_url: String,        // URL to trigger rotation
  custom_username: String,     // User's custom auth credentials
  custom_password: String,
  current_proxy_id: ObjectId,  // Current proxy for rotating proxies
  last_rotation: Date
}
```

### 12. ProxyReplacements
```javascript
{
  _id: ObjectId,
  user_id: ObjectId,
  plan_id: ObjectId,            // Reference to UserPlans
  static_plan_id: ObjectId,     // Reference to StaticProxyPlans
  original_proxy_id: ObjectId,  // Proxy bị thay thế
  original_proxy_ip: String,
  new_proxy_id: ObjectId,       // Proxy mới
  new_proxy_ip: String,
  reason: String,               // "dead", "poor_performance", "blacklisted", "user_request"
  requested_at: Date,
  processed_at: Date,
  status: String,              // "pending", "completed", "rejected"
  auto_replaced: Boolean,      // Tự động hay do người dùng yêu cầu
  api_request_id: String,      // ID của request API nếu do người dùng yêu cầu
  created_at: Date,
  updated_at: Date
}
```

### 13. BandwidthPlans (embedded document or separate collection)
```javascript
{
  _id: ObjectId,
  user_plan_id: ObjectId,      // Reference to UserPlans
  gb_amount: Number,           // Total GB purchased
  gb_remaining: Number,
  gb_used: Number,
  price_per_gb: Number,
  custom_settings: {
    sticky_ip: Boolean,
    session_duration: Number,  // minutes
    country_routing: [String],
    isp_preference: [String]
  },
  allowed_pools: [ObjectId],   // References to ProxyPools
  allowed_countries: [String],
  current_proxy_id: ObjectId,  // Current proxy being used
  access_key: String
}
```

### 14. ProxyUsageLogs
```javascript
{
  _id: ObjectId,
  user_id: ObjectId,
  plan_id: ObjectId,           // Reference to UserPlans
  proxy_id: ObjectId,
  pool_id: ObjectId,
  timestamp: Date,
  request_url: String,
  bytes_sent: Number,
  bytes_received: Number,
  total_bytes: Number,
  gb_used: Number,             // For bandwidth plans
  ip_used: String,
  country: String,
  isp: String,
  response_time: Number,
  status_code: Number,
  success: Boolean,
  user_agent: String,
  client_ip: String
}
```

### 15. BandwidthTopups
```javascript
{
  _id: ObjectId,
  user_id: ObjectId,
  plan_id: ObjectId,           // Reference to BandwidthPlans
  order_id: ObjectId,
  wallet_transaction_id: ObjectId,  // Tham chiếu đến giao dịch ví
  gb_amount: Number,
  price: Number,
  previous_gb_remaining: Number,
  new_gb_total: Number,
  created_at: Date
}
```

### 16. Alerts
```javascript
{
  _id: ObjectId,
  user_id: ObjectId,
  plan_id: ObjectId,           // Reference to UserPlans
  type: String,                // "expiry", "low_bandwidth", "low_balance", "renewal_reminder", "proxy_dead"
  message: String,
  data: Object,                // Additional alert data
  triggered_at: Date,
  is_read: Boolean,
  notification_sent: Boolean,
  notification_method: String  // "email", "sms", "dashboard"
}
```

### 17. Settings
```javascript
{
  _id: ObjectId,
  group: String,              // "renewal", "alerts", "system", "bonus"
  key: String,
  value: Mixed,               // String, Number, Boolean, or Object
  description: String,
  editable: Boolean,
  created_at: Date,
  updated_at: Date,
  created_by: ObjectId
}
```

## Indexes

```javascript
// Common indexes across collections
db.users.createIndex({ "username": 1 }, { unique: true });
db.users.createIndex({ "email": 1 }, { unique: true });
db.users.createIndex({ "api_key": 1 }, { unique: true });
db.users.createIndex({ "wallet_id": 1 }, { unique: true });

db.wallets.createIndex({ "user_id": 1 }, { unique: true });
db.wallets.createIndex({ "balance": 1 });

db.walletTransactions.createIndex({ "wallet_id": 1 });
db.walletTransactions.createIndex({ "user_id": 1 });
db.walletTransactions.createIndex({ "created_at": 1 });
db.walletTransactions.createIndex({ "type": 1 });
db.walletTransactions.createIndex({ "status": 1 });
db.walletTransactions.createIndex({ "wallet_id": 1, "created_at": 1 });
db.walletTransactions.createIndex({ "metadata.renewal_id": 1 });

db.bonusTiers.createIndex({ "min_amount": 1 });
db.bonusTiers.createIndex({ "active": 1 });
db.bonusTiers.createIndex({ "currency": 1 });
db.bonusTiers.createIndex({ "min_amount": 1, "active": 1, "currency": 1 });

db.productPackages.createIndex({ "name": 1 }, { unique: true });
db.productPackages.createIndex({ "type": 1 });
db.productPackages.createIndex({ "category": 1 });
db.productPackages.createIndex({ "is_bandwidth": 1 });
db.productPackages.createIndex({ "is_rotating": 1 });
db.productPackages.createIndex({ "active": 1 });
db.productPackages.createIndex({ "renewal_available": 1 });

db.proxies.createIndex({ "ip": 1, "port": 1 }, { unique: true });
db.proxies.createIndex({ "status": 1 });
db.proxies.createIndex({ "protocol": 1 });
db.proxies.createIndex({ "category": 1 });
db.proxies.createIndex({ "country": 1 });
db.proxies.createIndex({ "isp": 1 });
db.proxies.createIndex({ "sold": 1 });
db.proxies.createIndex({ "assigned": 1 });
db.proxies.createIndex({ "pool_id": 1 });
db.proxies.createIndex({ "type": 1, "category": 1, "country": 1, "sold": 1 });
db.proxies.createIndex({ "status": 1, "assigned": 1 });

db.proxyReplacements.createIndex({ "user_id": 1 });
db.proxyReplacements.createIndex({ "plan_id": 1 });
db.proxyReplacements.createIndex({ "original_proxy_id": 1 });
db.proxyReplacements.createIndex({ "new_proxy_id": 1 });
db.proxyReplacements.createIndex({ "status": 1 });
db.proxyReplacements.createIndex({ "requested_at": 1 });
db.proxyReplacements.createIndex({ "api_request_id": 1 });

db.orders.createIndex({ "user_id": 1 });
db.orders.createIndex({ "wallet_id": 1 });
db.orders.createIndex({ "order_number": 1 }, { unique: true });
db.orders.createIndex({ "created_at": 1 });
db.orders.createIndex({ "status": 1 });
db.orders.createIndex({ "wallet_transaction_id": 1 });
db.orders.createIndex({ "order_type": 1 });
db.orders.createIndex({ "renewal_for": 1 });

db.userPlans.createIndex({ "user_id": 1 });
db.userPlans.createIndex({ "api_key": 1 }, { unique: true });
db.userPlans.createIndex({ "end_date": 1 });
db.userPlans.createIndex({ "active": 1 });
db.userPlans.createIndex({ "plan_type": 1 });
db.userPlans.createIndex({ "user_id": 1, "active": 1 });
db.userPlans.createIndex({ "renewal_status": 1 });
db.userPlans.createIndex({ "end_date": 1, "renewal_status": 1, "active": 1 });
db.userPlans.createIndex({ "original_plan_id": 1 });

db.renewalRecords.createIndex({ "user_id": 1 });
db.renewalRecords.createIndex({ "original_plan_id": 1 });
db.renewalRecords.createIndex({ "new_plan_id": 1 });
db.renewalRecords.createIndex({ "order_id": 1 });
db.renewalRecords.createIndex({ "renewal_date": 1 });
db.renewalRecords.createIndex({ "status": 1 });
db.renewalRecords.createIndex({ "auto_renewal": 1 });

db.proxyUsageLogs.createIndex({ "timestamp": 1 });
db.proxyUsageLogs.createIndex({ "user_id": 1 });
db.proxyUsageLogs.createIndex({ "plan_id": 1 });
db.proxyUsageLogs.createIndex({ "proxy_id": 1 });
db.proxyUsageLogs.createIndex({ "timestamp": 1, "user_id": 1 });
// For bandwidth tracking
db.proxyUsageLogs.createIndex({ "timestamp": 1, "plan_id": 1 });
// Time series collection setting if MongoDB supports it
db.proxyUsageLogs.createTimeSeriesCollection({
    timeField: "timestamp",
    metaField: "plan_id",
    granularity: "minutes"
});

db.alerts.createIndex({ "user_id": 1 });
db.alerts.createIndex({ "plan_id": 1 });
db.alerts.createIndex({ "is_read": 1 });
db.alerts.createIndex({ "triggered_at": 1 });
db.alerts.createIndex({ "user_id": 1, "is_read": 1 });
db.alerts.createIndex({ "type": 1 });

db.settings.createIndex({ "group": 1, "key": 1 }, { unique: true });
db.settings.createIndex({ "group": 1 });
``` 