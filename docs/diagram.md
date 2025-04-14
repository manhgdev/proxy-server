# Sơ đồ mối quan hệ - Hệ thống Proxy Server

## Sơ đồ tổng quan MongoDB Schema

```
┌───────────────────┐         ┌───────────────────┐                  ┌───────────────────┐          ┌───────────────────┐
│      Users        │         │      Wallets      │                  │  ProductPackages  │          │      Proxies      │
├───────────────────┤         ├───────────────────┤                  ├───────────────────┤          ├───────────────────┤
│ _id               │◄────────┤ _id               │                  │ _id               │          │ _id               │
│ username          │         │ user_id           │                  │ name              │          │ ip                │
│ password_hash     │         │ balance           │                  │ description       │◄─────────┤ port              │
│ email             │         │ currency          │                  │ type              │          │ username          │
│ fullname          │         │ locked_amount     │                  │ category          │          │ protocol          │
│ phone             │         │ created_at        │                  │ protocol          │          │ protocol          │
│ created_at        │         │ updated_at        │                  │ is_rotating       │          │ type              │
│ updated_at        │         │ last_deposit_at   │                  │ is_bandwidth      │          │ category          │
│ active            │         │ last_usage_at     │                  │ duration_days     │          │ country           │
│ api_key           │         │ is_active         │                  │ price             │          │ city              │
│ access_token      │         └─────────┬─────────┘                  │ [...]             │          │ isp               │
│ wallet_id     ────┼───────────────────┘                           └────────┬──────────┘          │ status            │
└──────────┬────────┘                                                        │                     │ sold              │
           │                                                                 │                     │ assigned          │
           │                                                                 │                     │ [...]             │
           │                                                                 │                     └──┬────────────────┘
           │                                                                 │                        │
┌──────────▼────────┐         ┌───────────────────┐             ┌────────▼──────────┐                │
│      Orders       │◄────────┤ WalletTransactions│             │     UserPlans     │                │
├───────────────────┤         ├───────────────────┤             ├───────────────────┤                │
│ _id               │         │ _id               │             │ _id               │                │
│ user_id    ───────┼─────────┤►wallet_id         │             │ user_id           │                │
│ wallet_id         │         │ user_id           │             │ package_id ───────┼─────┐          │
│ order_number      │         │ type              │             │ order_id    ──────┼─────┘          │
│ total_amount      │         │ amount            │             │ plan_type         │                │
│ payment_source    │         │ balance_before    │             │ start_date        │                │
│ wallet_trans_id ──┼─────────┤►balance_after     │             │ end_date          │                │
│ status            │         │ currency          │             │ active            │                │
│ payment_method    │         │ status            │             │ api_key           │                │
│ payment_status    │         │ description       │             └─┬─────┬───────────┘                │
│ created_at        │         │ metadata          │               │     │                            │
│ updated_at        │         │ created_at        │               │     │                            │
│ items: [          │         │ updated_at        │               │     │                            │
│   {               │         └───────────────────┘               │     │                            │
│     package_id    │                                             │     │                            │
│     quantity      │                                             │     │                            │
│     price         │                                             │     │                            │
│     custom_config │                                             │     │                            │
│   }               │                                             │     │                            │
│ ]                 │                                             │     │                            │
└───────────────────┘                                             │     │                            │
                                                                  │     │                            │
                                                                  │     │                            │
                    ┌─────────────────────────────────────────────┘     └───────────────┐           │
                    │                                                                   │           │
        ┌───────────▼───────────┐                                          ┌────────────▼──────────▼──┐
        │  StaticProxyPlans     │                                          │   BandwidthPlans         │
        ├───────────────────────┤                                          ├─────────────────────────┤
        │ _id                   │                                          │ _id                     │
        │ user_plan_id          │                                          │ user_plan_id            │
        │ proxies: [ObjectId] ──┼──────────────────────────────────────────┤►gb_amount               │
        │ protocol              │                                          │ gb_remaining            │
        │ category              │                                          │ gb_used                 │
        │ is_rotating           │                                          │ price_per_gb            │
        │ rotation_interval     │                                          │ custom_settings         │
        │ rotation_url          │                                          │ allowed_pools           │
        │ custom_username       │                                          │ allowed_countries       │
        │ custom_password       │                                          │ current_proxy_id ───────┼──┐
        │ current_proxy_id ─────┼──────────────────────────────────────────┼──────────────────┐      │  │
        │ last_rotation         │                                          │ access_key        │      │  │
        └───────────────────────┘                                          └──────────┬────────┘      │  │
                                                                                      │               │  │
                                                                                      │               │  │
                                                                          ┌───────────▼───────────────┘  │
                                                                          │                              │
                                                                          │                              │
                    ┌─────────────────────────────────────────────────┐  │┌─────────────────────────────┘
                    │      ProxyUsageLogs                             │  ││
                    ├─────────────────────────────────────────────────┤  ││
                    │ _id                                             │  ││
                    │ user_id                                         │  ││
                    │ plan_id                                         │  ││
                    │ proxy_id                               ─────────┼──┼┼┘
                    │ timestamp                                       │  ││
                    │ request_url                                     │  ││
                    │ bytes_sent                                      │  ││
                    │ bytes_received                                  │  ││
                    │ total_bytes                                     │  ││
                    │ gb_used                                         │  ││
                    │ success                                         │  ││
                    │ [...]                                           │  ││
                    └─────────────────────────────────────────────────┘  ││
                                                                          ││
                    ┌─────────────────────────────────────────────────┐  ││
                    │      BandwidthTopups                            │  ││
                    ├─────────────────────────────────────────────────┤  ││
                    │ _id                                             │  ││
                    │ user_id                                         │  ││
                    │ plan_id                                         │  │┘
                    │ order_id                                        │  │
                    │ wallet_transaction_id                           │  │
                    │ gb_amount                                       │  │
                    │ price                                           │  │
                    │ previous_gb_remaining                           │  │
                    │ new_gb_total                                    │  │
                    │ created_at                                      │  │
                    └─────────────────────────────────────────────────┘  │
                                                                          │
                    ┌─────────────────────────────────────────────────┐  │
                    │         Alerts                                  │  │
                    ├─────────────────────────────────────────────────┤  │
                    │ _id                                             │  │
                    │ user_id                                         │  │
                    │ plan_id                                         │  ┘
                    │ type                                            │
                    │ message                                         │
                    │ triggered_at                                    │
                    │ is_read                                         │
                    └─────────────────────────────────────────────────┘
```

## Mối quan hệ chính

1. **User → Wallet**: Mỗi người dùng có một ví điện tử riêng
2. **Wallet → WalletTransactions**: Ví ghi nhận tất cả các giao dịch nạp tiền, rút tiền, thanh toán
3. **User → Orders**: Người dùng có thể tạo nhiều đơn hàng
4. **Wallet → Orders**: Các đơn hàng được thanh toán từ ví của người dùng
5. **Orders → WalletTransactions**: Mỗi đơn hàng tạo ra một giao dịch trừ tiền từ ví
6. **Orders → UserPlans**: Mỗi đơn hàng thành công tạo ra một hoặc nhiều gói dịch vụ 
7. **UserPlans → StaticProxyPlans/BandwidthPlans**: Mỗi gói dịch vụ có chi tiết thuộc một trong hai loại
8. **StaticProxyPlans → Proxies**: Gói proxy tĩnh/xoay chứa tham chiếu đến nhiều proxy cụ thể
9. **BandwidthPlans → ProxyPools**: Gói bandwidth sử dụng các pool proxy thay vì proxy cụ thể
10. **ProxyUsageLogs**: Ghi lại mọi hoạt động sử dụng proxy của người dùng
11. **BandwidthTopups → WalletTransactions**: Nạp thêm GB sẽ tạo ra giao dịch trừ tiền từ ví
12. **Alerts**: Thông báo cho người dùng về các sự kiện quan trọng (hết hạn, sắp hết bandwidth, số dư thấp...) 