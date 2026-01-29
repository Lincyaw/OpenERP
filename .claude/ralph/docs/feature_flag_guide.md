# Feature Flag 使用指南

> 本文档为开发者提供 Feature Flag 系统的完整使用指南，包括创建、使用和管理 Feature Flag 的最佳实践。

## 目录

1. [快速开始](#1-快速开始)
2. [后端使用](#2-后端使用)
3. [前端使用](#3-前端使用)
4. [最佳实践](#4-最佳实践)
5. [故障排查](#5-故障排查)

---

## 1. 快速开始

### 1.1 创建新 Flag

#### 通过 API 创建

```bash
# 创建一个布尔类型的 Feature Flag
curl -X POST http://localhost:8080/api/v1/feature-flags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "key": "enable_new_checkout",
    "name": "New Checkout Flow",
    "description": "Enables the new checkout experience",
    "type": "boolean",
    "default_value": {
      "enabled": false
    },
    "tags": ["checkout", "frontend"]
  }'
```

#### Flag 类型说明

| 类型 | 用途 | 示例 |
|------|------|------|
| `boolean` | 简单开关 | 功能开关、Kill Switch |
| `percentage` | 百分比灰度发布 | 渐进式发布 |
| `variant` | A/B 测试 | 多版本 UI 测试 |
| `user_segment` | 用户分群 | Beta 用户、VIP 功能 |

#### Flag Key 命名规范

```
# 格式: <action>_<feature>_<scope>

# 好的命名
enable_new_checkout           # 启用新结账流程
show_beta_dashboard           # 显示 Beta 面板
use_v2_pricing_algorithm      # 使用 V2 定价算法

# 避免的命名
flag1                         # 无意义
temp_test                     # 不够具体
NEW_FEATURE                   # 使用大写
```

### 1.2 启用 Flag

```bash
# 启用 Flag
curl -X POST http://localhost:8080/api/v1/feature-flags/enable_new_checkout/enable \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{}'

# 禁用 Flag
curl -X POST http://localhost:8080/api/v1/feature-flags/enable_new_checkout/disable \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 1.3 评估 Flag

```bash
# 单个 Flag 评估
curl -X POST http://localhost:8080/api/v1/feature-flags/enable_new_checkout/evaluate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "context": {
      "user_id": "user-uuid",
      "tenant_id": "tenant-uuid"
    }
  }'

# 批量评估
curl -X POST http://localhost:8080/api/v1/feature-flags/evaluate-batch \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "keys": ["enable_new_checkout", "show_beta_dashboard"],
    "context": {}
  }'
```

---

## 2. 后端使用

### 2.1 中间件注入获取

Feature Flag 中间件会在请求处理前预加载指定的 Flag 并注入到 Gin Context 中。

#### 配置中间件

```go
import (
    "github.com/erp/backend/internal/interfaces/http/middleware"
)

// 在路由设置中配置中间件
func SetupRoutes(r *gin.Engine, evaluator middleware.FeatureFlagEvaluator) {
    // 定义需要预加载的 Flag Keys
    preloadKeys := []string{
        "enable_new_checkout",
        "show_beta_dashboard",
        "use_v2_api",
    }

    // 使用中间件
    r.Use(middleware.FeatureFlagMiddleware(evaluator, preloadKeys))
}
```

#### 在 Handler 中使用

```go
func (h *MyHandler) HandleRequest(c *gin.Context) {
    // 方式 1: 检查布尔值
    if middleware.GetFeatureFlag(c, "enable_new_checkout") {
        // 新功能逻辑
        h.newCheckoutFlow(c)
        return
    }
    // 旧功能逻辑
    h.legacyCheckoutFlow(c)
}

func (h *MyHandler) HandleVariant(c *gin.Context) {
    // 方式 2: 获取 Variant 值 (用于 A/B 测试)
    variant := middleware.GetFeatureVariant(c, "checkout_variant")

    switch variant {
    case "A":
        h.checkoutVariantA(c)
    case "B":
        h.checkoutVariantB(c)
    default:
        h.checkoutDefault(c)
    }
}

func (h *MyHandler) HandleMultipleFlags(c *gin.Context) {
    // 方式 3: 获取所有预加载的 Flags
    flags := middleware.GetAllFlags(c)

    if flags["enable_analytics"].Enabled {
        // 记录分析数据
    }
}
```

#### 条件路由

```go
// 使用 WithFeatureFlag 包装 Handler，只有 Flag 启用时才执行
r.GET("/new-feature",
    middleware.WithFeatureFlag("enable_new_feature", h.NewFeatureHandler),
)
```

### 2.2 直接调用评估服务

当需要在中间件之外评估 Flag（如后台任务、事件处理器）时，可直接使用 EvaluationService。

```go
import (
    featureflagapp "github.com/erp/backend/internal/application/featureflag"
    "github.com/erp/backend/internal/application/featureflag/dto"
)

type MyService struct {
    evaluationService *featureflagapp.EvaluationService
}

func (s *MyService) ProcessOrder(ctx context.Context, order *Order) error {
    // 构建评估上下文
    evalCtx := dto.EvaluationContextDTO{
        UserID:   order.UserID.String(),
        TenantID: order.TenantID.String(),
    }

    // 方式 1: 检查是否启用
    enabled, err := s.evaluationService.IsEnabled(ctx, "enable_new_order_processing", evalCtx)
    if err != nil {
        return err
    }

    if enabled {
        return s.newOrderProcessing(ctx, order)
    }
    return s.legacyOrderProcessing(ctx, order)
}

func (s *MyService) GetPricingVariant(ctx context.Context, tenantID string) (string, error) {
    evalCtx := dto.EvaluationContextDTO{
        TenantID: tenantID,
    }

    // 方式 2: 获取 Variant
    variant, err := s.evaluationService.GetVariant(ctx, "pricing_algorithm", evalCtx)
    if err != nil {
        return "default", err
    }

    return variant, nil
}

func (s *MyService) GetAllClientFlags(ctx context.Context, userID, tenantID string) (map[string]bool, error) {
    evalCtx := dto.EvaluationContextDTO{
        UserID:   userID,
        TenantID: tenantID,
    }

    // 方式 3: 获取所有 Flag 的评估结果
    response, err := s.evaluationService.GetClientConfig(ctx, evalCtx)
    if err != nil {
        return nil, err
    }

    result := make(map[string]bool)
    for key, flag := range response.Flags {
        result[key] = flag.Enabled
    }
    return result, nil
}
```

### 2.3 条件执行代码块

#### 简单条件执行

```go
func (s *MyService) ExecuteWithFlag(ctx context.Context, evalCtx dto.EvaluationContextDTO) error {
    enabled, _ := s.evaluationService.IsEnabled(ctx, "enable_new_algorithm", evalCtx)

    if enabled {
        // 新算法
        return s.executeNewAlgorithm(ctx)
    }
    // 旧算法
    return s.executeLegacyAlgorithm(ctx)
}
```

#### 使用策略模式

```go
type ProcessingStrategy interface {
    Process(ctx context.Context, data *Data) error
}

type NewProcessingStrategy struct{}
type LegacyProcessingStrategy struct{}

func (s *MyService) GetProcessingStrategy(ctx context.Context, evalCtx dto.EvaluationContextDTO) ProcessingStrategy {
    enabled, _ := s.evaluationService.IsEnabled(ctx, "use_new_processing", evalCtx)

    if enabled {
        return &NewProcessingStrategy{}
    }
    return &LegacyProcessingStrategy{}
}
```

---

## 3. 前端使用

### 3.1 useFeatureFlag Hook

#### 基础用法

```tsx
import { useFeatureFlag } from '@/hooks/useFeatureFlag'

function NewFeatureButton() {
  const isEnabled = useFeatureFlag('enable_new_checkout')

  if (!isEnabled) {
    return null
  }

  return <Button>Try New Checkout</Button>
}
```

#### 带默认值

```tsx
function OptionalFeature() {
  // 如果 Flag 不存在，默认返回 true
  const isEnabled = useFeatureFlag('experimental_feature', true)

  return isEnabled ? <NewVersion /> : <OldVersion />
}
```

### 3.2 useFeatureVariant Hook

用于 A/B 测试场景。

```tsx
import { useFeatureVariant } from '@/hooks/useFeatureFlag'

function ABTestButton() {
  const variant = useFeatureVariant('button_color_test')

  const buttonColor = variant === 'blue' ? 'primary' : 'secondary'

  return <Button color={buttonColor}>Click Me</Button>
}

// 多变体场景
function PricingPage() {
  const variant = useFeatureVariant('pricing_layout')

  switch (variant) {
    case 'grid':
      return <PricingGrid />
    case 'list':
      return <PricingList />
    case 'cards':
      return <PricingCards />
    default:
      return <PricingDefault />
  }
}
```

### 3.3 useFeatureFlags Hook

批量获取多个 Flag。

```tsx
import { useFeatureFlags } from '@/hooks/useFeatureFlag'

function Dashboard() {
  const flags = useFeatureFlags(['analytics', 'notifications', 'dark_mode'])

  return (
    <div>
      {flags.analytics && <AnalyticsWidget />}
      {flags.notifications && <NotificationBell />}
      {flags.dark_mode && <DarkModeToggle />}
    </div>
  )
}
```

### 3.4 Feature 组件

声明式的条件渲染组件。

#### 基础用法

```tsx
import { Feature } from '@/components/common/Feature'

function App() {
  return (
    <Feature flag="enable_new_checkout">
      <NewCheckout />
    </Feature>
  )
}
```

#### 带 Fallback

```tsx
<Feature flag="enable_new_checkout" fallback={<OldCheckout />}>
  <NewCheckout />
</Feature>
```

#### 带 Loading 状态

```tsx
<Feature
  flag="enable_new_checkout"
  fallback={<OldCheckout />}
  loading={<LoadingSpinner />}
>
  <NewCheckout />
</Feature>
```

#### Variant 渲染（A/B 测试）

```tsx
<Feature flag="checkout_variant">
  {(variant) => {
    switch(variant) {
      case 'A': return <CheckoutA />
      case 'B': return <CheckoutB />
      default: return <CheckoutDefault />
    }
  }}
</Feature>
```

### 3.5 A/B 测试完整示例

```tsx
import { Feature, useFeatureVariant } from '@/hooks/useFeatureFlag'

// 方式 1: 使用 Feature 组件
function CheckoutPage() {
  return (
    <Feature flag="checkout_ab_test">
      {(variant) => {
        // 记录曝光
        useEffect(() => {
          analytics.track('checkout_variant_exposed', { variant })
        }, [variant])

        switch (variant) {
          case 'new_flow':
            return <NewCheckoutFlow />
          case 'simplified':
            return <SimplifiedCheckoutFlow />
          default:
            return <StandardCheckoutFlow />
        }
      }}
    </Feature>
  )
}

// 方式 2: 使用 Hook
function CheckoutButton() {
  const variant = useFeatureVariant('checkout_button_test')

  const handleClick = () => {
    // 记录转化
    analytics.track('checkout_button_clicked', { variant })
    // ...处理逻辑
  }

  if (variant === 'large') {
    return <Button size="large" onClick={handleClick}>Checkout</Button>
  }

  return <Button onClick={handleClick}>Checkout</Button>
}
```

### 3.6 Feature Flag 状态管理

```tsx
import {
  useFeatureFlagReady,
  useFeatureFlagLoading,
  useFeatureFlagError
} from '@/hooks/useFeatureFlag'

function App() {
  const isReady = useFeatureFlagReady()
  const isLoading = useFeatureFlagLoading()
  const error = useFeatureFlagError()

  // 等待 Flag 加载完成
  if (!isReady && isLoading) {
    return <GlobalLoadingSpinner />
  }

  // 显示错误（但仍使用缓存的 Flag）
  if (error) {
    console.warn('Feature flags error:', error)
  }

  return <MainApp />
}
```

---

## 4. 最佳实践

### 4.1 Flag 命名规范

```
# 格式
<verb>_<feature>_<optional_scope>

# 动词选择
enable_   : 功能开关
show_     : UI 显示控制
use_      : 使用特定实现
allow_    : 权限控制

# 示例
enable_new_checkout        # 启用新结账流程
show_beta_banner           # 显示 Beta 横幅
use_v2_pricing             # 使用 V2 定价
allow_bulk_operations      # 允许批量操作

# 避免
- 使用大写字母
- 使用数字开头
- 使用特殊字符（除了下划线、连字符、点号）
- 超过 100 个字符
```

### 4.2 Flag 生命周期管理

```
1. 创建 (DISABLED)
   ↓
2. 配置规则/测试
   ↓
3. 启用 (ENABLED) - 开始灰度
   ↓
4. 监控/调整
   ↓
5. 全量发布 (100%)
   ↓
6. 清理代码 - 移除 Flag 检查
   ↓
7. 归档 (ARCHIVED)
```

#### 建议的 Flag 存活时间

| Flag 类型 | 建议存活时间 | 说明 |
|-----------|-------------|------|
| 功能开关 | 1-4 周 | 功能稳定后应尽快清理 |
| A/B 测试 | 2-6 周 | 收集足够数据后清理 |
| Kill Switch | 永久 | 用于紧急情况，长期保留 |
| 权限控制 | 按需 | 作为功能一部分，随功能生命周期 |

### 4.3 清理过期 Flag

#### 识别可清理的 Flag

```bash
# 查找超过 30 天未修改的 Flag
curl -X GET "http://localhost:8080/api/v1/feature-flags?page_size=100" \
  -H "Authorization: Bearer <token>" | \
  jq '.data.flags[] | select(.updated_at < (now - 2592000 | todate))'
```

#### 清理流程

```
1. 识别候选 Flag
   - 已全量发布的功能 Flag
   - 已结束的 A/B 测试
   - 未使用超过 30 天的 Flag

2. 验证使用情况
   - 搜索代码库中的引用
   - 检查最近的评估日志

3. 清理代码
   - 移除前端 Flag 检查
   - 移除后端 Flag 检查
   - 更新测试

4. 归档 Flag
   curl -X DELETE "http://localhost:8080/api/v1/feature-flags/{key}" \
     -H "Authorization: Bearer <token>"
```

### 4.4 避免 Flag 嵌套

```tsx
// ❌ 不好的做法：嵌套 Flag
function Component() {
  const flagA = useFeatureFlag('feature_a')
  const flagB = useFeatureFlag('feature_b')
  const flagC = useFeatureFlag('feature_c')

  if (flagA) {
    if (flagB) {
      if (flagC) {
        return <VersionABC />
      }
      return <VersionAB />
    }
    return <VersionA />
  }
  return <Default />
}

// ✅ 好的做法：使用单一 Flag 或 Variant
function Component() {
  const variant = useFeatureVariant('component_version')

  switch (variant) {
    case 'v3': return <VersionV3 />
    case 'v2': return <VersionV2 />
    case 'v1': return <VersionV1 />
    default: return <Default />
  }
}

// ✅ 好的做法：使用组合 Flag
function Component() {
  const flags = useFeatureFlags(['feature_a', 'feature_b'])

  // 简单条件判断
  if (flags.feature_a && flags.feature_b) {
    return <CombinedFeature />
  }
  if (flags.feature_a) {
    return <FeatureA />
  }
  return <Default />
}
```

### 4.5 测试建议

#### 前端测试

```tsx
import { renderHook } from '@testing-library/react'
import { useFeatureFlagStore } from '@/store'
import { useFeatureFlag } from '@/hooks/useFeatureFlag'

describe('Feature Flag Tests', () => {
  beforeEach(() => {
    // 重置 store
    useFeatureFlagStore.setState({
      flags: {},
      isReady: true,
      isLoading: false,
      error: null,
    })
  })

  it('should return true when flag is enabled', () => {
    // 设置测试 Flag
    useFeatureFlagStore.setState({
      flags: {
        test_feature: { enabled: true, variant: null },
      },
    })

    const { result } = renderHook(() => useFeatureFlag('test_feature'))
    expect(result.current).toBe(true)
  })

  it('should return default value when flag does not exist', () => {
    const { result } = renderHook(() => useFeatureFlag('non_existent', true))
    expect(result.current).toBe(true)
  })
})
```

#### 后端测试

```go
func TestFeatureFlag_Integration(t *testing.T) {
    // 设置测试环境
    ctx := context.Background()

    // 创建测试 Flag
    flag, err := featureflag.NewBooleanFlag(
        "test_feature",
        "Test Feature",
        false,
        nil,
    )
    require.NoError(t, err)

    // 启用 Flag
    err = flag.Enable(nil)
    require.NoError(t, err)

    // 评估
    evaluator := featureflag.NewPureEvaluator()
    evalCtx := featureflag.NewEvaluationContext().WithUser("user-123")

    result := evaluator.Evaluate(flag, evalCtx, nil, nil)

    assert.True(t, result.IsEnabled())
}
```

---

## 5. 故障排查

### 5.1 Flag 不生效检查清单

```
□ Flag 是否存在？
  - 检查 Flag Key 拼写
  - 调用 GET /api/v1/feature-flags/{key} 确认

□ Flag 是否启用？
  - 检查 status 是否为 "enabled"
  - 检查是否被 disabled 或 archived

□ 用户是否在灰度范围内？
  - 检查百分比配置
  - 检查 Targeting Rules 条件

□ 是否有 Override？
  - 用户级 Override 优先级最高
  - 租户级 Override 次之
  - 检查 GET /api/v1/feature-flags/{key}/overrides

□ 前端缓存是否过期？
  - 清除 sessionStorage
  - 手动刷新：useFeatureFlagStore.getState().refresh()

□ 后端缓存是否过期？
  - 检查 Redis 缓存
  - 调用 evaluationService.InvalidateFlag(ctx, key)
```

### 5.2 缓存问题排查

#### 前端缓存

```tsx
// 检查当前缓存状态
const { flags, lastUpdated, isReady } = useFeatureFlagStore.getState()
console.log('Cached flags:', flags)
console.log('Last updated:', lastUpdated)
console.log('Is ready:', isReady)

// 强制刷新
const refresh = useFeatureFlagStore.getState().refresh
await refresh()

// 清除缓存并重新初始化
sessionStorage.removeItem('erp-feature-flags')
const initialize = useFeatureFlagStore.getState().initialize
await initialize()
```

#### 后端缓存

```go
// 检查缓存统计
stats := evaluationService.GetCacheStats(ctx)
if stats != nil {
    log.Printf("Cache hits: %d, misses: %d, hit rate: %.2f%%",
        stats.Hits, stats.Misses, stats.HitRate*100)
}

// 使缓存失效
err := evaluationService.InvalidateFlag(ctx, "flag_key")
if err != nil {
    log.Printf("Failed to invalidate cache: %v", err)
}

// 预热缓存
err = evaluationService.WarmupCache(ctx)
```

### 5.3 日志查看

#### 后端日志

```bash
# 查看 Feature Flag 相关日志
grep "feature_flag\|FeatureFlag" /var/log/erp/app.log

# 查看特定 Flag 的评估日志
grep "enable_new_checkout" /var/log/erp/app.log
```

#### 前端调试

```tsx
// 开启 Zustand DevTools
// 在 Chrome DevTools 中查看 FeatureFlagStore 状态变化

// 添加调试日志
const isEnabled = useFeatureFlag('my_flag')
console.log('[FeatureFlag] my_flag =', isEnabled)
```

### 5.4 常见错误及解决方案

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| `FLAG_NOT_FOUND` | Flag 不存在 | 检查 Key 拼写，确认 Flag 已创建 |
| `EVALUATION_ERROR` | 评估失败 | 检查后端日志，确认服务正常 |
| `ALREADY_ENABLED` | 重复启用 | Flag 已启用，无需再次操作 |
| `CANNOT_UPDATE` | 更新已归档的 Flag | 创建新 Flag 或取消归档 |
| 前端 Flag 值不更新 | 缓存过期 | 刷新页面或调用 refresh() |
| 评估结果不一致 | 缓存不同步 | 使缓存失效并重新评估 |

### 5.5 紧急情况处理

#### Kill Switch（紧急禁用功能）

```bash
# 立即禁用有问题的功能
curl -X POST http://localhost:8080/api/v1/feature-flags/problematic_feature/disable \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{}'

# 验证禁用生效
curl -X POST http://localhost:8080/api/v1/feature-flags/problematic_feature/evaluate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"context": {}}'
```

#### 清除所有缓存

```bash
# 后端：清除 Redis 缓存
redis-cli DEL "feature_flag:*"

# 前端：指导用户清除浏览器缓存
# 或在代码中强制刷新
sessionStorage.clear()
window.location.reload()
```

---

## 附录

### API 端点参考

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/feature-flags` | 列出所有 Flag |
| POST | `/api/v1/feature-flags` | 创建 Flag |
| GET | `/api/v1/feature-flags/:key` | 获取 Flag 详情 |
| PUT | `/api/v1/feature-flags/:key` | 更新 Flag |
| DELETE | `/api/v1/feature-flags/:key` | 归档 Flag |
| POST | `/api/v1/feature-flags/:key/enable` | 启用 Flag |
| POST | `/api/v1/feature-flags/:key/disable` | 禁用 Flag |
| POST | `/api/v1/feature-flags/:key/evaluate` | 评估 Flag |
| POST | `/api/v1/feature-flags/evaluate-batch` | 批量评估 |
| POST | `/api/v1/feature-flags/client-config` | 获取客户端配置 |
| GET | `/api/v1/feature-flags/:key/overrides` | 列出 Override |
| POST | `/api/v1/feature-flags/:key/overrides` | 创建 Override |
| DELETE | `/api/v1/feature-flags/:key/overrides/:id` | 删除 Override |
| GET | `/api/v1/feature-flags/:key/audit-logs` | 查看审计日志 |
| GET | `/api/v1/feature-flags/stream` | SSE 实时更新 |

### 相关文档

- [Feature Flag 系统设计文档](./feature_flag.md) - 详细的架构设计
- [API 文档](../../backend/docs/swagger.yaml) - OpenAPI 规范
- [测试用例](../../backend/tests/integration/feature_flag_test.go) - 集成测试示例
