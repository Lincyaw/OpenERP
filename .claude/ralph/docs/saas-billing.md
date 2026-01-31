# SaaS 计费系统文档

本文档提供 SaaS 计费系统的完整配置指南，包括功能计费、使用量计费、Stripe 集成和 Webhook 配置。

## 目录

1. [系统概述](#系统概述)
2. [订阅套餐配置](#订阅套餐配置)
3. [功能计费配置](#功能计费配置)
4. [使用量计费配置](#使用量计费配置)
5. [Stripe 集成配置](#stripe-集成配置)
6. [Webhook 配置](#webhook-配置)
7. [API 参考](#api-参考)
8. [故障排除](#故障排除)

---

## 系统概述

### 计费模式

本系统采用**订阅制 + 用量计费**混合模式：

- **订阅制**：按月/年收取固定费用，解锁对应套餐功能
- **用量计费**：对超出配额的使用量进行计费（可选）

### 架构组件

```
┌─────────────────────────────────────────────────────────────────┐
│                         计费系统架构                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   租户管理   │    │  套餐功能   │    │  使用量计量  │         │
│  │   Tenant    │◄──►│ PlanFeature │◄──►│  UsageMeter │         │
│  └──────┬──────┘    └─────────────┘    └──────┬──────┘         │
│         │                                      │                │
│         │    ┌─────────────────────────┐      │                │
│         └───►│      配额服务           │◄─────┘                │
│              │    QuotaService        │                        │
│              └───────────┬───────────┘                         │
│                          │                                     │
│              ┌───────────▼───────────┐                         │
│              │     Stripe 适配器      │                         │
│              │   StripeAdapter       │                         │
│              └───────────┬───────────┘                         │
│                          │                                     │
└──────────────────────────┼─────────────────────────────────────┘
                           │
                  ┌────────▼────────┐
                  │   Stripe API    │
                  └─────────────────┘
```

---

## 订阅套餐配置

### 套餐定义

系统预置四个订阅套餐，定义在 `backend/internal/domain/identity/tenant.go`：

```go
// TenantPlan 代表租户的订阅套餐
type TenantPlan string

const (
    TenantPlanFree       TenantPlan = "free"       // 免费版
    TenantPlanBasic      TenantPlan = "basic"      // 基础版
    TenantPlanPro        TenantPlan = "pro"        // 专业版
    TenantPlanEnterprise TenantPlan = "enterprise" // 企业版
)
```

### 套餐配额

| 套餐 | 最大用户 | 最大仓库 | 最大商品 | 月费 |
|------|----------|----------|----------|------|
| `free` | 5 | 3 | 1,000 | ¥0 |
| `basic` | 10 | 5 | 5,000 | ¥199 |
| `pro` | 50 | 20 | 50,000 | ¥599 |
| `enterprise` | 9999 | 9999 | 999999 | 定制 |

### 配置套餐配额

套餐配额通过 `TenantConfig` 结构体配置：

```go
// 在 tenant.go 的 updateConfigForPlan 方法中配置
func (t *Tenant) updateConfigForPlan(plan TenantPlan) {
    switch plan {
    case TenantPlanFree:
        t.Config.MaxUsers = 5
        t.Config.MaxWarehouses = 3
        t.Config.MaxProducts = 1000
    case TenantPlanBasic:
        t.Config.MaxUsers = 10
        t.Config.MaxWarehouses = 5
        t.Config.MaxProducts = 5000
    // ... 其他套餐
    }
}
```

---

## 功能计费配置

### 功能门控机制

功能通过 `PlanFeature` 实体控制，定义在 `backend/internal/domain/identity/plan_feature.go`。

### 功能键列表

```go
// 核心功能
FeatureMultiWarehouse    = "multi_warehouse"     // 多仓库管理
FeatureBatchManagement   = "batch_management"    // 批次管理
FeatureSerialTracking    = "serial_tracking"     // 序列号追踪
FeatureMultiCurrency     = "multi_currency"      // 多币种支持
FeatureAdvancedReporting = "advanced_reporting"  // 高级报表
FeatureAPIAccess         = "api_access"          // API 访问
FeatureCustomFields      = "custom_fields"       // 自定义字段
FeatureAuditLog          = "audit_log"           // 审计日志
FeatureDataExport        = "data_export"         // 数据导出
FeatureDataImport        = "data_import"         // 数据导入

// 交易功能
FeatureSalesOrders       = "sales_orders"        // 销售订单
FeaturePurchaseOrders    = "purchase_orders"     // 采购订单
FeatureQuotations        = "quotations"          // 报价单
FeaturePriceManagement   = "price_management"    // 价格管理
FeatureDiscountRules     = "discount_rules"      // 折扣规则
FeatureCreditManagement  = "credit_management"   // 信用管理

// 财务功能
FeatureReceivables       = "receivables"         // 应收账款
FeaturePayables          = "payables"            // 应付账款
FeatureReconciliation    = "reconciliation"      // 对账
FeatureExpenseTracking   = "expense_tracking"    // 费用跟踪
FeatureFinancialReports  = "financial_reports"   // 财务报表

// 高级功能
FeatureWorkflowApproval  = "workflow_approval"   // 工作流审批
FeatureNotifications     = "notifications"       // 通知服务
FeatureIntegrations      = "integrations"        // 第三方集成
FeatureWhiteLabeling     = "white_labeling"      // 白标定制
FeaturePrioritySupport   = "priority_support"    // 优先支持
FeatureDedicatedSupport  = "dedicated_support"   // 专属支持
FeatureSLA               = "sla"                 // SLA 保障
```

### 功能矩阵

| 功能 | 免费版 | 基础版 | 专业版 | 企业版 |
|------|:------:|:------:|:------:|:------:|
| `multi_warehouse` | ✗ | ✓ | ✓ | ✓ |
| `batch_management` | ✗ | ✓ | ✓ | ✓ |
| `serial_tracking` | ✗ | ✗ | ✓ | ✓ |
| `multi_currency` | ✗ | ✗ | ✓ | ✓ |
| `advanced_reporting` | ✗ | ✗ | ✓ | ✓ |
| `api_access` | ✗ | ✗ | ✓ | ✓ |
| `custom_fields` | ✗ | ✗ | ✓ | ✓ |
| `audit_log` | ✗ | ✓ | ✓ | ✓ |
| `data_export` | ✓ | ✓ | ✓ | ✓ |
| `data_import` | 100行 | 1000行 | 10000行 | 无限 |
| `workflow_approval` | ✗ | ✗ | ✓ | ✓ |
| `notifications` | ✗ | ✓ | ✓ | ✓ |
| `white_labeling` | ✗ | ✗ | ✗ | ✓ |
| `priority_support` | ✗ | ✗ | ✓ | ✓ |
| `dedicated_support` | ✗ | ✗ | ✗ | ✓ |

### 功能检查代码示例

```go
// 检查租户是否有某功能
import "github.com/erp/backend/internal/domain/identity"

// 使用默认套餐功能检查
hasFeature := identity.PlanHasFeature(tenant.Plan, identity.FeatureMultiWarehouse)
if !hasFeature {
    return errors.New("该功能需要升级套餐")
}

// 获取功能限制
limit := identity.GetPlanFeatureLimit(tenant.Plan, identity.FeatureDataImport)
if limit != nil && importRows > *limit {
    return errors.New("导入行数超过套餐限制")
}
```

---

## 使用量计费配置

### 使用量类型

定义在 `backend/internal/domain/billing/usage_type.go`：

```go
// 计量类型
const (
    UsageTypeAPICalls          = "API_CALLS"          // API 调用次数
    UsageTypeStorageBytes      = "STORAGE_BYTES"      // 存储空间（字节）
    UsageTypeActiveUsers       = "ACTIVE_USERS"       // 活跃用户数
    UsageTypeOrdersCreated     = "ORDERS_CREATED"     // 创建订单数
    UsageTypeProductsSKU       = "PRODUCTS_SKU"       // 商品/SKU 数量
    UsageTypeWarehouses        = "WAREHOUSES"         // 仓库数量
    UsageTypeCustomers         = "CUSTOMERS"          // 客户数量
    UsageTypeSuppliers         = "SUPPLIERS"          // 供应商数量
    UsageTypeReportsGenerated  = "REPORTS_GENERATED"  // 报表生成次数
    UsageTypeDataExports       = "DATA_EXPORTS"       // 数据导出次数
    UsageTypeDataImportRows    = "DATA_IMPORT_ROWS"   // 数据导入行数
    UsageTypeIntegrationCalls  = "INTEGRATION_CALLS"  // 集成调用次数
    UsageTypeNotificationsSent = "NOTIFICATIONS_SENT" // 通知发送次数
    UsageTypeAttachmentBytes   = "ATTACHMENT_BYTES"   // 附件存储（字节）
)
```

### 使用量分类

| 分类 | 计量方式 | 重置周期 | 使用量类型 |
|------|----------|----------|------------|
| **可数资源** | 当前计数 | 无 | `ACTIVE_USERS`, `PRODUCTS_SKU`, `WAREHOUSES`, `CUSTOMERS`, `SUPPLIERS` |
| **累计指标** | 周期累加 | 月度 | `API_CALLS`, `ORDERS_CREATED`, `REPORTS_GENERATED`, `DATA_EXPORTS`, `DATA_IMPORT_ROWS`, `INTEGRATION_CALLS`, `NOTIFICATIONS_SENT` |
| **存储计量** | 当前占用 | 无 | `STORAGE_BYTES`, `ATTACHMENT_BYTES` |

### 配额配置

创建配额使用 `UsageQuota` 聚合根：

```go
import "github.com/erp/backend/internal/domain/billing"

// 创建套餐级配额
quota, err := billing.NewUsageQuota(
    "basic",                       // 套餐 ID
    billing.UsageTypeOrdersCreated, // 使用量类型
    1000,                          // 限制值
    billing.ResetPeriodMonthly,    // 重置周期
)
if err != nil {
    return err
}

// 设置软限制（80% 时警告）
quota.WithSoftLimit(800)

// 设置超限策略
quota.WithOveragePolicy(billing.OveragePolicyWarn)

// 创建租户级配额覆盖
tenantQuota, err := billing.NewTenantUsageQuota(
    tenantID,
    "basic",
    billing.UsageTypeOrdersCreated,
    2000, // 特殊租户可以创建更多订单
    billing.ResetPeriodMonthly,
)
```

### 超限策略

| 策略 | 常量 | 行为 | 适用场景 |
|------|------|------|----------|
| 阻止 | `BLOCK` | 拒绝操作 | 硬性资源限制 |
| 警告 | `WARN` | 允许但发警告 | 软性限制 |
| 计费 | `CHARGE` | 允许并计超额费 | 按量付费资源 |
| 限流 | `THROTTLE` | 降级服务质量 | 性能敏感资源 |

### 配额检查代码示例

```go
import billingapp "github.com/erp/backend/internal/application/billing"

// 创建配额服务
quotaService := billingapp.NewQuotaService(
    quotaRepo,
    usageRepo,
    meterRepo,
    tenantRepo,
    eventPublisher,
    logger,
    billingapp.DefaultQuotaServiceConfig(),
)

// 检查订单配额
err := quotaService.CheckOrderQuota(ctx, tenantID)
if err != nil {
    if quotaErr, ok := err.(*billingapp.QuotaExceededError); ok {
        // 配额超限
        return fiber.NewError(fiber.StatusTooManyRequests, quotaErr.Message)
    }
    return err
}

// 检查特定使用量配额
result, err := quotaService.CheckQuota(ctx, billingapp.QuotaCheckInput{
    TenantID:  tenantID,
    UsageType: billing.UsageTypeAPIcalls,
    Amount:    1,
})
if !result.Allowed {
    log.Warn("API 调用配额超限", "tenant_id", tenantID)
}
```

---

## Stripe 集成配置

### 环境配置

在 `config.toml` 或环境变量中配置 Stripe：

```toml
[stripe]
secret_key = "sk_test_xxx"              # Stripe Secret Key
publishable_key = "pk_test_xxx"         # Stripe Publishable Key
webhook_secret = "whsec_xxx"            # Webhook Signing Secret
is_test_mode = true                     # 是否测试模式
default_currency = "cny"                # 默认货币

# 套餐价格 ID 映射
[stripe.price_ids]
free = ""                               # 免费版无价格
basic = "price_1xxx"                    # 基础版月费价格 ID
pro = "price_2xxx"                      # 专业版月费价格 ID
enterprise = "price_3xxx"               # 企业版月费价格 ID
```

或使用环境变量：

```bash
export ERP_STRIPE_SECRET_KEY="sk_test_xxx"
export ERP_STRIPE_PUBLISHABLE_KEY="pk_test_xxx"
export ERP_STRIPE_WEBHOOK_SECRET="whsec_xxx"
export ERP_STRIPE_IS_TEST_MODE="true"
export ERP_STRIPE_DEFAULT_CURRENCY="cny"
```

### Stripe Dashboard 配置

#### 1. 创建产品和价格

在 Stripe Dashboard 中创建产品：

1. 登录 [Stripe Dashboard](https://dashboard.stripe.com)
2. 进入 **Products** > **Add product**
3. 为每个套餐创建产品和价格：

| 产品名称 | 价格 | 计费周期 | 价格 ID |
|----------|------|----------|---------|
| ERP Basic Plan | ¥199 | 月付 | `price_basic_monthly` |
| ERP Pro Plan | ¥599 | 月付 | `price_pro_monthly` |
| ERP Enterprise | 定制 | 月付 | `price_ent_monthly` |

#### 2. 配置价格元数据

在每个价格的 Metadata 中添加：

```json
{
  "plan_id": "basic"  // 或 "pro", "enterprise"
}
```

### 代码集成

#### 初始化 Stripe 客户端

```go
import "github.com/erp/backend/internal/infrastructure/billing"

config := &billing.StripeConfig{
    SecretKey:       os.Getenv("ERP_STRIPE_SECRET_KEY"),
    PublishableKey:  os.Getenv("ERP_STRIPE_PUBLISHABLE_KEY"),
    WebhookSecret:   os.Getenv("ERP_STRIPE_WEBHOOK_SECRET"),
    IsTestMode:      true,
    DefaultCurrency: "cny",
    PriceIDs: map[string]string{
        "basic":      "price_1xxx",
        "pro":        "price_2xxx",
        "enterprise": "price_3xxx",
    },
}

// 验证配置
if err := config.Validate(); err != nil {
    log.Fatal("Stripe 配置无效", err)
}

// 初始化客户端
config.InitStripeClient()
```

#### 创建订阅 Checkout Session

```go
import (
    "github.com/stripe/stripe-go/v81"
    "github.com/stripe/stripe-go/v81/checkout/session"
)

// 创建 Checkout Session
params := &stripe.CheckoutSessionParams{
    Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
    LineItems: []*stripe.CheckoutSessionLineItemParams{
        {
            Price:    stripe.String("price_basic_monthly"),
            Quantity: stripe.Int64(1),
        },
    },
    SuccessURL: stripe.String("https://app.example.com/billing/success?session_id={CHECKOUT_SESSION_ID}"),
    CancelURL:  stripe.String("https://app.example.com/billing/cancel"),
    CustomerEmail: stripe.String(tenant.ContactEmail),
    Metadata: map[string]string{
        "tenant_id": tenant.ID.String(),
        "plan_id":   "basic",
    },
    SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
        Metadata: map[string]string{
            "tenant_id": tenant.ID.String(),
            "plan_id":   "basic",
        },
    },
}

sess, err := session.New(params)
if err != nil {
    return err
}

// 返回 session URL 给前端跳转
return c.JSON(fiber.Map{"url": sess.URL})
```

---

## Webhook 配置

### Stripe Dashboard 配置

1. 登录 [Stripe Dashboard](https://dashboard.stripe.com)
2. 进入 **Developers** > **Webhooks** > **Add endpoint**
3. 配置端点：
   - **Endpoint URL**: `https://api.example.com/webhooks/stripe`
   - **Events to send**:
     - `customer.subscription.created`
     - `customer.subscription.updated`
     - `customer.subscription.deleted`
     - `invoice.paid`
     - `invoice.payment_failed`
4. 保存 **Signing secret** 到配置文件

### 本地开发测试

使用 Stripe CLI 进行本地 Webhook 测试：

```bash
# 安装 Stripe CLI
brew install stripe/stripe-cli/stripe

# 登录
stripe login

# 转发 Webhook 到本地
stripe listen --forward-to localhost:8080/webhooks/stripe

# 触发测试事件
stripe trigger customer.subscription.created
stripe trigger invoice.paid
stripe trigger invoice.payment_failed
```

### Webhook 处理器配置

Webhook 处理器在 `backend/internal/interfaces/http/handler/stripe_webhook_handler.go`：

```go
import (
    billingapp "github.com/erp/backend/internal/application/billing"
    "github.com/erp/backend/internal/infrastructure/billing"
)

// 创建 Webhook 服务
webhookService := billingapp.NewStripeWebhookService(billingapp.StripeWebhookServiceConfig{
    Config:     stripeConfig,
    TenantRepo: tenantRepo,
    EventBus:   eventBus,
    Logger:     logger,
})

// 创建 Handler
webhookHandler := handler.NewStripeWebhookHandler(webhookService)

// 注册路由（无需认证）
router.POST("/webhooks/stripe", webhookHandler.HandleStripeWebhook)
```

### Webhook 事件处理逻辑

| 事件 | 处理逻辑 |
|------|----------|
| `customer.subscription.created` | 1. 通过 Customer ID 查找租户<br>2. 更新租户 Stripe Subscription ID<br>3. 根据 metadata 中的 plan_id 设置套餐<br>4. 设置订阅过期时间<br>5. 激活租户 |
| `customer.subscription.updated` | 1. 通过 Subscription ID 查找租户<br>2. 同步套餐变更<br>3. 更新过期时间<br>4. 处理状态变更（active/past_due/canceled） |
| `customer.subscription.deleted` | 1. 清除租户 Subscription ID<br>2. 降级为免费套餐<br>3. 清除过期时间 |
| `invoice.paid` | 1. 通过 Customer ID 查找租户<br>2. 激活租户（如已暂停）<br>3. 延长订阅过期时间 |
| `invoice.payment_failed` | 1. 通过 Customer ID 查找租户<br>2. 暂停租户服务 |

### Webhook 安全配置

1. **签名验证**：所有 Webhook 请求都验证 `Stripe-Signature` header
2. **幂等处理**：Webhook 处理应具备幂等性，防止重复处理
3. **payload 大小限制**：限制为 64KB（防止 DoS）
4. **返回正确状态码**：
   - `200`: 处理成功
   - `401`: 签名验证失败
   - `413`: Payload 过大
   - `500`: 内部错误（Stripe 会重试）

---

## API 参考

### 使用量 API

#### 获取当前使用量

```http
GET /api/v1/tenants/current/usage
Authorization: Bearer <token>
```

**响应**：

```json
{
  "success": true,
  "data": {
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_name": "Acme Corp",
    "plan": "basic",
    "metrics": [
      {
        "name": "users",
        "display_name": "Users",
        "current": 5,
        "limit": 10,
        "percentage": 50.0,
        "unit": "count"
      },
      {
        "name": "warehouses",
        "display_name": "Warehouses",
        "current": 2,
        "limit": 5,
        "percentage": 40.0,
        "unit": "count"
      }
    ],
    "last_updated": "2024-01-15T10:30:00Z"
  }
}
```

#### 获取使用量历史

```http
GET /api/v1/tenants/current/usage/history?period=daily&start_date=2024-01-01&end_date=2024-01-31
Authorization: Bearer <token>
```

#### 获取配额信息

```http
GET /api/v1/tenants/current/quotas
Authorization: Bearer <token>
```

### 套餐功能 API

#### 获取当前租户功能

```http
GET /api/v1/tenants/current/features
Authorization: Bearer <token>
```

**响应**：

```json
{
  "success": true,
  "data": {
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "plan": "basic",
    "features": [
      {
        "feature_key": "multi_warehouse",
        "enabled": true,
        "limit": null,
        "description": "Multiple warehouse management"
      },
      {
        "feature_key": "data_import",
        "enabled": true,
        "limit": 1000,
        "description": "Import data from CSV"
      }
    ]
  }
}
```

#### 获取套餐列表（管理员）

```http
GET /api/v1/admin/plans
Authorization: Bearer <admin_token>
```

#### 获取套餐功能详情（管理员）

```http
GET /api/v1/admin/plans/{plan}/features
Authorization: Bearer <admin_token>
```

### Webhook API

#### Stripe Webhook 端点

```http
POST /webhooks/stripe
Stripe-Signature: <signature>
Content-Type: application/json

{
  "id": "evt_xxx",
  "type": "customer.subscription.created",
  "data": {
    "object": {
      "id": "sub_xxx",
      "customer": "cus_xxx",
      "status": "active",
      ...
    }
  }
}
```

**响应**：

```json
{
  "received": true,
  "event_id": "evt_xxx",
  "event_type": "customer.subscription.created",
  "message": "Webhook processed successfully"
}
```

---

## 故障排除

### 常见问题

#### 1. Webhook 签名验证失败

**症状**：Webhook 返回 401 错误

**解决方案**：
1. 确认 `webhook_secret` 配置正确
2. 使用 Stripe Dashboard 中的最新 signing secret
3. 检查是否使用了正确的环境（test/live）

```bash
# 验证 webhook secret
stripe listen --print-secret
```

#### 2. 租户升级后功能未生效

**症状**：套餐变更但功能检查仍返回旧套餐

**解决方案**：
1. 检查 Webhook 是否正确处理
2. 确认数据库中租户 `plan` 字段已更新
3. 清除功能缓存（如有）

```sql
-- 检查租户套餐
SELECT id, code, plan, stripe_subscription_id
FROM tenants
WHERE id = '<tenant_id>';
```

#### 3. 配额检查失败

**症状**：配额检查返回错误或不准确

**解决方案**：
1. 检查 `usage_quotas` 表中是否有该套餐的配额定义
2. 确认使用量记录 `usage_records` 正确写入
3. 检查计量周期边界计算

```sql
-- 检查配额定义
SELECT * FROM usage_quotas WHERE plan_id = 'basic';

-- 检查使用量记录
SELECT usage_type, SUM(quantity)
FROM usage_records
WHERE tenant_id = '<tenant_id>'
  AND period_start >= '2024-01-01'
GROUP BY usage_type;
```

#### 4. Stripe 支付后租户未激活

**症状**：支付成功但租户状态未更新

**解决方案**：
1. 检查 Webhook 是否配置正确
2. 查看应用日志中的 Webhook 处理记录
3. 确认租户 `stripe_customer_id` 已关联

```bash
# 检查 Webhook 日志
grep "Stripe webhook" /var/log/erp/app.log | tail -50

# 手动重发 Webhook
stripe events resend evt_xxx
```

### 日志级别配置

调试计费问题时，可提高日志级别：

```toml
[log]
level = "debug"  # 调试时使用 debug
```

或环境变量：

```bash
export ERP_LOG_LEVEL=debug
```

### 测试环境

使用 Stripe 测试卡进行端到端测试：

| 卡号 | 场景 |
|------|------|
| `4242424242424242` | 成功支付 |
| `4000000000000002` | 卡被拒绝 |
| `4000000000009995` | 余额不足 |
| `4000000000000341` | 附加认证 |

---

## 附录

### 相关文件

| 文件 | 说明 |
|------|------|
| `backend/internal/domain/billing/` | 计费领域模型 |
| `backend/internal/domain/identity/tenant.go` | 租户聚合根 |
| `backend/internal/domain/identity/plan_feature.go` | 套餐功能 |
| `backend/internal/application/billing/` | 计费应用服务 |
| `backend/internal/infrastructure/billing/` | Stripe 基础设施 |
| `backend/internal/interfaces/http/handler/usage_handler.go` | 使用量 API |
| `backend/internal/interfaces/http/handler/plan_feature_handler.go` | 功能 API |
| `backend/internal/interfaces/http/handler/stripe_webhook_handler.go` | Webhook 处理 |

### 数据库表

| 表 | 说明 |
|----|------|
| `tenants` | 租户信息（含 Stripe ID） |
| `usage_records` | 使用量记录 |
| `usage_quotas` | 配额定义 |
| `plan_features` | 套餐功能映射 |

### 参考文档

- [Stripe Billing Documentation](https://stripe.com/docs/billing)
- [Stripe Webhooks Guide](https://stripe.com/docs/webhooks)
- [Stripe CLI Reference](https://stripe.com/docs/stripe-cli)
