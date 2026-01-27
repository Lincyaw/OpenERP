# Circuit-Based API Load Generator

## 1. 概述

基于 OpenAPI 规范的"电路连接"式负载生成器。核心理念是将 API 端点视为电路元件，通过语义化的参数池实现自动连接。

### 1.1 核心比喻

```
┌─────────────────────────────────────────────────────────────────┐
│  API Endpoint = 芯片 (Chip)                                     │
│  ├── 输入引脚 (Input Pins) = 请求参数                            │
│  └── 输出引脚 (Output Pins) = 响应字段                           │
│                                                                 │
│  Parameter Pool = 导线网络 (Wire Bus)                            │
│  ├── 按语义类型分组 (customer_id, product_id, order_id...)       │
│  └── 自动连接相同语义的引脚                                       │
│                                                                 │
│  Load Generator = 电路板 (Circuit Board)                         │
│  └── 管理连接、调度执行、收集指标                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 设计目标

- **零配置启动**: 只需 OpenAPI spec + 认证信息即可运行
- **语义自动推断**: 从字段名和上下文自动推断参数语义
- **自愈能力**: 缺少依赖参数时自动触发生产者 endpoint
- **真实负载**: 混合读写操作，模拟真实用户行为
- **精确控制**: 任意控制 QPS、并发数、流量波形
- **流量整形**: 支持波峰、波谷、突发流量模拟

---

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CIRCUIT BOARD METAPHOR                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐            │
│  │   Customer   │     │   Product    │     │  Warehouse   │            │
│  │   Creator    │     │   Creator    │     │   Creator    │            │
│  │              │     │              │     │              │            │
│  │ OUT: cust_id │     │ OUT: prod_id │     │ OUT: wh_id   │            │
│  └──────┬───────┘     └──────┬───────┘     └──────┬───────┘            │
│         │                    │                    │                    │
│         └────────────────────┴────────────────────┘                    │
│                              │                                          │
│                              ▼                                          │
│                   ┌─────────────────────┐                              │
│                   │   PARAMETER POOL    │◄─────── Wires (导线)         │
│                   │   ──────────────    │                              │
│                   │  customer_id: [...] │                              │
│                   │  product_id: [...]  │                              │
│                   │  warehouse_id: [...] │                              │
│                   │  order_id: [...]    │                              │
│                   └─────────┬───────────┘                              │
│                             │                                          │
│         ┌───────────────────┼───────────────────┐                      │
│         ▼                   ▼                   ▼                      │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐               │
│  │ Sales Order  │   │Purchase Order│   │   Return     │               │
│  │   Creator    │   │   Creator    │   │   Creator    │               │
│  │              │   │              │   │              │               │
│  │ IN: cust_id  │   │ IN: supp_id  │   │ IN: order_id │               │
│  │ IN: prod_id  │   │ IN: prod_id  │   │              │               │
│  │ IN: wh_id    │   │ IN: wh_id    │   │ OUT: ret_id  │               │
│  │              │   │              │   │              │               │
│  │ OUT: ord_id  │   │ OUT: po_id   │   │              │               │
│  │ OUT: itm_id  │   │ OUT: itm_id  │   │              │               │
│  └──────────────┘   └──────────────┘   └──────────────┘               │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 增强组件架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ENHANCED ARCHITECTURE                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────┐    ┌────────────────┐    ┌────────────────┐            │
│  │ Traffic Shaper │───►│ Load Controller │───►│  Rate Limiter  │            │
│  │ (wave/spike)   │    │  (orchestrator) │    │ (token bucket) │            │
│  └────────────────┘    └───────┬────────┘    └───────┬────────┘            │
│                                │                      │                      │
│                    ┌───────────▼──────────┐          │                      │
│                    │   Worker Pool        │◄─────────┘                      │
│                    │   (adaptive sizing)  │                                  │
│                    └───────────┬──────────┘                                  │
│                                │                                             │
│  ┌────────────────┐            ▼            ┌────────────────┐              │
│  │ Weighted       │    ┌──────────────┐     │ Parameter Pool │              │
│  │ Selector       │───►│  Scheduler   │◄────│ (sharded)      │              │
│  │ (time-aware)   │    └──────┬───────┘     └────────┬───────┘              │
│  └────────────────┘           │                      │                      │
│                               ▼                      │                      │
│                    ┌──────────────────┐              │                      │
│                    │ Request Builder  │◄─────────────┘                      │
│                    └────────┬─────────┘                                     │
│                             │                                               │
│                             ▼                                               │
│                    ┌──────────────────┐     ┌────────────────┐              │
│                    │    Executor      │────►│ Metrics + SLO  │              │
│                    │ (with backpressure)│   │ Validator      │              │
│                    └──────────────────┘     └────────┬───────┘              │
│                                                      │                      │
│                                         ┌────────────┴────────────┐         │
│                                         ▼                         ▼         │
│                                  ┌────────────┐           ┌────────────┐    │
│                                  │ Prometheus │           │ Console/   │    │
│                                  │ Exporter   │           │ JSON/HTML  │    │
│                                  └────────────┘           └────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.3 组件层次

```
┌───────────────────────────────────────────────────────────────────────┐
│                      YAML Configuration                                │
│  (OpenAPI path, auth, load profile, traffic shape, weights)           │
└───────────────────────────────────────────────────────────────────────┬┘
                                                                        │
┌───────────────────────────────────────────────────────────────────────▼┐
│                     OpenAPI Parser                                      │
│  - Parse endpoints → EndpointUnit                                       │
│  - Infer semantic types for pins                                        │
│  - Build producer-consumer graph                                        │
└───────────────────────────────────────────────────────────────────────┬┘
                                                                        │
┌───────────────────────────────────────────────────────────────────────▼┐
│                     Circuit Board Engine                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │  Endpoint   │  │  Parameter  │  │  Dependency │  │  Producer   │   │
│  │   Units     │  │    Pool     │  │    Graph    │  │   Chain     │   │
│  │             │  │  (Sharded)  │  │             │  │  (Guarded)  │   │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘   │
└───────────────────────────────────────────────────────────────────────┬┘
                                                                        │
┌───────────────────────────────────────────────────────────────────────▼┐
│                     Load Control Layer (NEW)                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │   Traffic   │  │    Load     │  │    Rate     │  │ Backpressure│   │
│  │   Shaper    │  │ Controller  │  │  Limiter    │  │  Handler    │   │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘   │
└───────────────────────────────────────────────────────────────────────┬┘
                                                                        │
┌───────────────────────────────────────────────────────────────────────▼┐
│                     Execution Engine                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │  Scheduler   │  │   Executor   │  │   Metrics    │                  │
│  │ (Weighted)   │  │  (Workers)   │  │ (Collector)  │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 核心数据结构

### 3.1 语义类型 (Semantic Type)

参数的语义分类，用于自动连接输入输出引脚。

```go
type SemanticType string

const (
    // 实体 ID
    SemanticCustomerID    SemanticType = "entity.customer.id"
    SemanticCustomerCode  SemanticType = "entity.customer.code"
    SemanticSupplierID    SemanticType = "entity.supplier.id"
    SemanticProductID     SemanticType = "entity.product.id"
    SemanticProductCode   SemanticType = "entity.product.code"
    SemanticWarehouseID   SemanticType = "entity.warehouse.id"
    SemanticCategoryID    SemanticType = "entity.category.id"
    SemanticUserID        SemanticType = "entity.user.id"

    // 订单 ID
    SemanticSalesOrderID        SemanticType = "order.sales.id"
    SemanticSalesOrderItemID    SemanticType = "order.sales.item_id"
    SemanticPurchaseOrderID     SemanticType = "order.purchase.id"
    SemanticPurchaseOrderItemID SemanticType = "order.purchase.item_id"
    SemanticSalesReturnID       SemanticType = "order.return.sales.id"
    SemanticPurchaseReturnID    SemanticType = "order.return.purchase.id"

    // 财务
    SemanticReceivableID SemanticType = "finance.receivable.id"
    SemanticPayableID    SemanticType = "finance.payable.id"
    SemanticReceiptID    SemanticType = "finance.receipt.id"
    SemanticPaymentID    SemanticType = "finance.payment.id"

    // 库存
    SemanticInventoryItemID   SemanticType = "inventory.item.id"
    SemanticStockTakingID     SemanticType = "inventory.stock_taking.id"

    // 通用
    SemanticUUID     SemanticType = "common.uuid"
    SemanticCode     SemanticType = "common.code"
    SemanticName     SemanticType = "common.name"
    SemanticEmail    SemanticType = "common.email"
    SemanticPhone    SemanticType = "common.phone"
    SemanticAmount   SemanticType = "common.amount"
    SemanticQuantity SemanticType = "common.quantity"
    SemanticDate     SemanticType = "common.date"
)
```

### 3.2 引脚 (Pin)

端点的输入/输出参数。

```go
type Pin struct {
    Name     string       // 参数名称
    Semantic SemanticType // 语义类型
    Required bool         // 是否必需
    Location PinLocation  // 位置: path/query/body/header
    JSONPath string       // body 中的路径，如 "$.items[*].product_id"
}

type PinLocation string

const (
    PinLocationPath   PinLocation = "path"
    PinLocationQuery  PinLocation = "query"
    PinLocationBody   PinLocation = "body"
    PinLocationHeader PinLocation = "header"
)
```

### 3.3 端点单元 (Endpoint Unit)

一个 API 端点，相当于电路中的芯片。

```go
type EndpointUnit struct {
    ID          string      // 唯一标识: "POST /partner/customers"
    Path        string      // API 路径
    Method      string      // HTTP 方法
    OperationID string      // OpenAPI operationId
    Tags        []string    // OpenAPI tags

    InputPins   []Pin       // 输入引脚 (请求参数)
    OutputPins  []Pin       // 输出引脚 (响应字段)

    // 依赖关系 (从 InputPins 推导)
    Dependencies []SemanticType // 需要的输入参数语义
    Produces     []SemanticType // 产生的输出参数语义

    // 执行统计
    Stats ExecutionStats
}

type ExecutionStats struct {
    ExecutionCount int64
    SuccessCount   int64
    FailureCount   int64
    TotalLatency   time.Duration
    MinLatency     time.Duration
    MaxLatency     time.Duration
}
```

### 3.4 参数池 (Parameter Pool)

存储所有可用参数值的导线网络。使用分片设计避免并发争用。

```go
type ParameterValue struct {
    Value     interface{}  // 实际值
    Semantic  SemanticType // 语义类型
    CreatedAt time.Time    // 创建时间
    ExpiresAt *time.Time   // 过期时间（可选）
    Source    ValueSource  // 来源信息
    Metadata  map[string]interface{} // 关联的额外数据
}

type ValueSource struct {
    Endpoint      string // 来源端点
    ResponseField string // 响应字段路径
}

type ParameterPool interface {
    // 添加值到池中
    Add(semantic SemanticType, value ParameterValue)

    // 获取值
    Get(semantic SemanticType) *ParameterValue
    GetRandom(semantic SemanticType) *ParameterValue
    GetAll(semantic SemanticType) []ParameterValue

    // 获取或创建（触发生产者）
    GetOrCreate(ctx context.Context, semantic SemanticType) (*ParameterValue, error)

    // 生命周期
    Cleanup() // 清理过期值
    Clear(semantic *SemanticType) // 清空

    // 统计
    Size(semantic SemanticType) int
    Stats() PoolStats
}

// 分片参数池实现（高并发场景）
type ShardedParameterPool struct {
    shards    []*shard
    shardMask uint32
    limits    PoolLimits
}

type shard struct {
    mu    sync.RWMutex
    pools map[SemanticType]*ringBuffer // 使用环形缓冲区，自动淘汰旧值
}

type PoolLimits struct {
    MaxValuesPerType int           // 每种语义类型最大值数量，默认 10000
    DefaultTTL       time.Duration // 默认过期时间，默认 30m
    EvictionPolicy   string        // 淘汰策略: "lru" | "fifo" | "random"
}

type PoolStats struct {
    TotalValues   int64
    ValuesByType  map[SemanticType]int
    EvictionCount int64
    HitRate       float64
}
```

### 3.5 电路板 (Circuit Board)

管理所有组件的主控制器。

```go
type CircuitBoard struct {
    Units         map[string]*EndpointUnit // 所有端点单元
    Pool          ParameterPool            // 参数池
    Graph         *DependencyGraph         // 依赖图
    Producers     map[SemanticType][]*EndpointUnit // 语义类型 -> 生产者
    Consumers     map[SemanticType][]*EndpointUnit // 语义类型 -> 消费者
    ProducerGuard *ProducerChainGuard      // 生产者链保护
}

type DependencyGraph struct {
    // 检测循环依赖
    DetectCycles() [][]string

    // 获取执行计划
    GetExecutionPlan(target *EndpointUnit) []*EndpointUnit

    // 拓扑排序
    TopologicalSort() []*EndpointUnit
}

// 生产者链保护，防止级联过载
type ProducerChainGuard struct {
    MaxDepth        int           // 最大递归深度，默认 3
    CooldownPeriod  time.Duration // 冷却期，默认 1s
    MinPoolSize     int           // 低于此值触发补充，默认 5
    RefillBatchSize int           // 批量补充数量，默认 10

    // 运行时状态
    currentDepth    int32         // 当前递归深度（原子操作）
    lastRefillTime  sync.Map      // 每种语义类型的上次补充时间
}
```

---

## 4. 负载控制层 (Load Control Layer)

### 4.1 速率限制器 (Rate Limiter)

控制请求发送速率，支持多种限流算法。

```go
type RateLimiterType string

const (
    RateLimiterTokenBucket   RateLimiterType = "token_bucket"
    RateLimiterLeakyBucket   RateLimiterType = "leaky_bucket"
    RateLimiterSlidingWindow RateLimiterType = "sliding_window"
)

type RateLimiter interface {
    // Acquire 阻塞直到获得请求槽位
    Acquire(ctx context.Context) error

    // TryAcquire 非阻塞尝试获取
    TryAcquire() bool

    // SetRate 动态调整速率
    SetRate(qps float64)

    // CurrentRate 获取当前速率
    CurrentRate() float64

    // Stats 获取统计信息
    Stats() RateLimiterStats
}

type RateLimiterStats struct {
    TotalAcquired   int64
    TotalRejected   int64
    CurrentQPS      float64
    AvgWaitTime     time.Duration
}

// 令牌桶实现（推荐）
type TokenBucketLimiter struct {
    limiter   *rate.Limiter // golang.org/x/time/rate
    burstSize int
    mu        sync.RWMutex
}

func NewTokenBucketLimiter(qps float64, burst int) *TokenBucketLimiter {
    return &TokenBucketLimiter{
        limiter:   rate.NewLimiter(rate.Limit(qps), burst),
        burstSize: burst,
    }
}
```

### 4.2 流量整形器 (Traffic Shaper)

生成各种流量波形，模拟真实场景。

```go
type TrafficPattern string

const (
    PatternConstant  TrafficPattern = "constant"   // 恒定流量
    PatternRamp      TrafficPattern = "ramp"       // 线性增长
    PatternSineWave  TrafficPattern = "sine_wave"  // 正弦波
    PatternSpike     TrafficPattern = "spike"      // 突发尖峰
    PatternStep      TrafficPattern = "step"       // 阶梯形
    PatternCustom    TrafficPattern = "custom"     // 自定义曲线
)

type TrafficShaper interface {
    // GetTargetQPS 返回指定时刻的目标 QPS
    GetTargetQPS(elapsed time.Duration) float64

    // GetPhase 返回当前流量阶段名称
    GetPhase(elapsed time.Duration) string

    // TotalDuration 返回完整周期时长
    TotalDuration() time.Duration
}

// 正弦波整形器
type SineWaveShaper struct {
    Period      time.Duration // 周期
    Amplitude   float64       // 振幅 (0-1)，相对于基线的波动比例
    BaselineQPS float64       // 基线 QPS
}

func (s *SineWaveShaper) GetTargetQPS(elapsed time.Duration) float64 {
    phase := float64(elapsed) / float64(s.Period) * 2 * math.Pi
    return s.BaselineQPS * (1 + s.Amplitude*math.Sin(phase))
}

// 突发尖峰整形器
type SpikeShaper struct {
    BaselineQPS      float64
    PeakQPS          float64
    SpikeDuration    time.Duration
    RecoveryDuration time.Duration
    Interval         time.Duration // 尖峰间隔
}

func (s *SpikeShaper) GetTargetQPS(elapsed time.Duration) float64 {
    cyclePos := elapsed % s.Interval
    if cyclePos < s.SpikeDuration {
        // 尖峰阶段
        return s.PeakQPS
    } else if cyclePos < s.SpikeDuration+s.RecoveryDuration {
        // 恢复阶段（线性下降）
        progress := float64(cyclePos-s.SpikeDuration) / float64(s.RecoveryDuration)
        return s.PeakQPS - (s.PeakQPS-s.BaselineQPS)*progress
    }
    return s.BaselineQPS
}

// 阶梯形整形器
type StepShaper struct {
    Steps []StepConfig
}

type StepConfig struct {
    QPS      float64
    Duration time.Duration
}

// 自定义曲线整形器
type CustomShaper struct {
    Points        []TrafficPoint
    Interpolation string // "linear" | "step"
}

type TrafficPoint struct {
    Time time.Duration
    QPS  float64
}
```

### 4.3 负载控制器 (Load Controller)

协调流量整形和速率限制的中央控制器。

```go
type LoadController struct {
    rateLimiter   RateLimiter
    trafficShaper TrafficShaper
    workerPool    *WorkerPool
    metrics       *MetricsCollector

    startTime     time.Time
    adjustTicker  *time.Ticker

    // 自适应控制
    adaptive      bool
    targetP95     time.Duration
}

func (lc *LoadController) Run(ctx context.Context) {
    lc.startTime = time.Now()
    lc.adjustTicker = time.NewTicker(100 * time.Millisecond)
    defer lc.adjustTicker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-lc.adjustTicker.C:
            elapsed := time.Since(lc.startTime)
            targetQPS := lc.trafficShaper.GetTargetQPS(elapsed)

            // 自适应调整：如果 P95 延迟超标，降低 QPS
            if lc.adaptive {
                currentP95 := lc.metrics.GetP95Latency()
                if currentP95 > lc.targetP95 {
                    targetQPS *= 0.9 // 降低 10%
                }
            }

            lc.rateLimiter.SetRate(targetQPS)
            lc.workerPool.AdjustSize(lc.calculateOptimalWorkers(targetQPS))
        }
    }
}

func (lc *LoadController) calculateOptimalWorkers(targetQPS float64) int {
    avgLatency := lc.metrics.GetAvgLatency()
    if avgLatency == 0 {
        avgLatency = 50 * time.Millisecond // 默认估计
    }
    // 工作者数 = QPS * 平均延迟(秒)
    optimal := int(targetQPS * avgLatency.Seconds() * 1.5) // 1.5 倍余量
    return max(lc.workerPool.MinSize, min(optimal, lc.workerPool.MaxSize))
}
```

### 4.4 背压处理器 (Backpressure Handler)

检测目标系统饱和并自动调整。

```go
type BackpressureHandler struct {
    // 触发条件
    ErrorRateThreshold   float64       // 错误率阈值，默认 0.1 (10%)
    LatencyP99Threshold  time.Duration // P99 延迟阈值，默认 1s
    QueueDepthThreshold  int           // 队列深度阈值

    // 响应策略
    Strategy             BackpressureStrategy
    RecoveryPeriod       time.Duration // 恢复检测周期

    // 状态
    currentState         BackpressureState
    stateChangedAt       time.Time
}

type BackpressureStrategy string

const (
    StrategyDropRequests BackpressureStrategy = "drop"      // 丢弃新请求
    StrategyReduceRate   BackpressureStrategy = "reduce"    // 降低速率
    StrategyPause        BackpressureStrategy = "pause"     // 暂停发送
    StrategyCircuitBreak BackpressureStrategy = "circuit"   // 熔断
)

type BackpressureState string

const (
    StateNormal    BackpressureState = "normal"
    StateWarning   BackpressureState = "warning"
    StateCritical  BackpressureState = "critical"
    StateRecovery  BackpressureState = "recovery"
)

func (bp *BackpressureHandler) Evaluate(metrics *MetricsSnapshot) BackpressureAction {
    errorRate := float64(metrics.FailureCount) / float64(metrics.TotalCount)

    if errorRate > bp.ErrorRateThreshold || metrics.P99Latency > bp.LatencyP99Threshold {
        bp.currentState = StateCritical
        return BackpressureAction{
            Action:     bp.Strategy,
            ReduceRate: 0.5, // 降低 50%
        }
    }

    if bp.currentState == StateCritical && time.Since(bp.stateChangedAt) > bp.RecoveryPeriod {
        bp.currentState = StateRecovery
        return BackpressureAction{
            Action:     StrategyReduceRate,
            ReduceRate: 0.8, // 恢复到 80%
        }
    }

    return BackpressureAction{Action: "none"}
}
```

---

## 5. 请求分布控制

### 5.1 增强权重选择器

```go
type WeightedSelector struct {
    // 全局读写比例
    GlobalReadWriteRatio float64 // 0.8 表示 80% 读，20% 写

    // 分类权重
    CategoryWeights map[string]CategoryWeight

    // 端点级覆盖
    EndpointOverrides map[string]EndpointWeight

    // 操作类型权重
    OperationWeights map[string]int // GET: 60, POST: 25, PUT: 10, DELETE: 5

    // 时间调度
    TimeSchedules []TimeSchedule
}

type CategoryWeight struct {
    Weight         int
    ReadWriteRatio *float64 // 可选，覆盖全局比例
}

type EndpointWeight struct {
    Weight    int
    Schedules []TimeBasedWeight
}

type TimeBasedWeight struct {
    TimeRange string // "09:00-12:00" 或 cron 表达式
    Weight    int
}

type TimeSchedule struct {
    Start    time.Time
    End      time.Time
    Modifier float64 // 权重乘数
}

func (ws *WeightedSelector) SelectEndpoint(
    currentTime time.Time,
    availableEndpoints []*EndpointUnit,
) *EndpointUnit {
    weights := make([]float64, len(availableEndpoints))

    for i, ep := range availableEndpoints {
        baseWeight := ws.getBaseWeight(ep)
        timeModifier := ws.getTimeModifier(ep, currentTime)
        operationModifier := ws.getOperationModifier(ep.Method)

        weights[i] = float64(baseWeight) * timeModifier * operationModifier
    }

    return ws.weightedRandomSelect(availableEndpoints, weights)
}
```

### 5.2 会话模拟

模拟真实用户会话行为。

```go
type SessionSimulator struct {
    Enabled            bool
    ConcurrentSessions int
    SessionDuration    DurationRange
    Behaviors          map[string]UserBehavior
}

type DurationRange struct {
    Min time.Duration
    Max time.Duration
}

type UserBehavior struct {
    Weight            int
    ThinkTime         DurationRange // 请求间思考时间
    ActionsPerSession IntRange      // 每会话操作数
    PreferredActions  []string      // 偏好的端点模式
}

type IntRange struct {
    Min int
    Max int
}

type Session struct {
    ID              string
    StartTime       time.Time
    Behavior        *UserBehavior
    Parameters      *SessionParameterPool // 会话级参数池
    ActionCount     int
    MaxActions      int
}

func (s *Session) NextThinkTime() time.Duration {
    return randomDuration(s.Behavior.ThinkTime.Min, s.Behavior.ThinkTime.Max)
}

func (s *Session) IsExpired() bool {
    return s.ActionCount >= s.MaxActions
}
```

---

## 6. 语义推断规则

### 6.1 默认推断规则

从字段名和上下文自动推断语义类型：

| 字段名模式 | 推断语义 | 置信度 |
|-----------|---------|-------|
| `customer_id` | `entity.customer.id` | 1.0 |
| `product_id` | `entity.product.id` | 1.0 |
| `supplier_id` | `entity.supplier.id` | 1.0 |
| `warehouse_id` | `entity.warehouse.id` | 1.0 |
| `category_id` | `entity.category.id` | 1.0 |
| `sales_order_id` | `order.sales.id` | 1.0 |
| `purchase_order_id` | `order.purchase.id` | 1.0 |
| `POST /customers` 响应的 `$.data.id` | `entity.customer.id` | 0.9 |
| `POST /sales-orders` 响应的 `$.data.id` | `order.sales.id` | 0.9 |
| `code` | `common.code` | 0.7 |
| `name` | `common.name` | 0.7 |

### 6.2 推断规则定义

```go
type InferenceRule struct {
    Match      MatchCondition
    Semantic   SemanticType
    Confidence float64
}

type MatchCondition struct {
    FieldName    *regexp.Regexp // 字段名匹配
    FieldPath    *regexp.Regexp // JSON 路径匹配
    EndpointPath *regexp.Regexp // 端点路径匹配
    ParentType   *regexp.Regexp // 父类型名匹配
    DataType     string         // OpenAPI 数据类型
    Format       string         // OpenAPI format
}
```

---

## 7. 生产者-消费者关系图

### 7.1 ERP 系统的主要关系

```
┌────────────────────────────────────────────────────────────────────────────┐
│                      PRODUCER-CONSUMER RELATIONSHIP MAP                    │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  PRODUCERS (生产者)                    CONSUMERS (消费者)                  │
│  ───────────────────                   ───────────────────                  │
│                                                                            │
│  POST /auth/login                                                          │
│    └─► access_token ─────────────────► [ALL SECURED ENDPOINTS]            │
│                                                                            │
│  POST /partner/customers               POST /trade/sales-orders            │
│    └─► customer_id ──────────────────► └── customer_id (required)         │
│                                        POST /finance/receipts              │
│                                          └── customer_id                   │
│                                                                            │
│  POST /partner/suppliers               POST /trade/purchase-orders         │
│    └─► supplier_id ──────────────────► └── supplier_id (required)         │
│                                        POST /finance/payments              │
│                                          └── supplier_id                   │
│                                                                            │
│  POST /partner/warehouses              POST /trade/sales-orders            │
│    └─► warehouse_id ─────────────────► └── warehouse_id                   │
│                                        POST /trade/purchase-orders         │
│                                          └── warehouse_id                  │
│                                        POST /inventory/stock-takings       │
│                                          └── warehouse_id (required)       │
│                                                                            │
│  POST /catalog/products                POST /trade/sales-orders            │
│    └─► product_id ───────────────────► └── items[].product_id (required)  │
│                                        POST /trade/purchase-orders         │
│                                          └── items[].product_id (required) │
│                                                                            │
│  POST /trade/sales-orders              POST /trade/sales-orders/{id}/confirm│
│    └─► order_id ─────────────────────► └── {id} (path)                    │
│    └─► items[].id ───────────────────► POST /trade/sales-returns          │
│                                          └── sales_order_item_id          │
│                                                                            │
│  POST /trade/purchase-orders           POST /trade/purchase-orders/{id}/confirm│
│    └─► order_id ─────────────────────► └── {id} (path)                    │
│    └─► items[].id ───────────────────► POST /trade/purchase-returns       │
│                                          └── purchase_order_item_id       │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. 执行流程

### 8.1 初始化阶段

```
1. 解析配置文件
2. 加载 OpenAPI spec
3. 解析所有端点 → EndpointUnit
4. 推断语义类型
5. 构建生产者-消费者关系图
6. 检测循环依赖
7. 初始化负载控制组件（速率限制器、流量整形器）
```

### 8.2 预热阶段

```
1. 执行认证，获取 token
2. 加载种子数据到参数池（如果配置）
3. 执行基础生产者，填充参数池:
   - POST /partner/customers → customer_id
   - POST /partner/suppliers → supplier_id
   - POST /catalog/products → product_id
   - POST /partner/warehouses → warehouse_id
4. 验证参数池达到最小阈值
```

### 8.3 负载生成阶段

```
循环执行:
  1. 负载控制器调整目标 QPS（根据流量整形器）
  2. 速率限制器控制请求发送节奏
  3. 调度器按权重选择端点
  4. 检查输入引脚依赖:
     - 池中有 → 随机取一个
     - 池中无 → 触发生产者端点（受深度限制）
  5. 构建请求
  6. 执行请求
  7. 解析响应，提取输出引脚值
  8. 将输出值加入参数池
  9. 记录指标
  10. 背压检测，必要时调整负载
```

### 8.4 流程图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ENHANCED EXECUTION FLOW                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌──────────────────────────────────────────────────────────────────────┐ │
│   │ LOAD CONTROLLER (每 100ms 执行)                                       │ │
│   │ 1. Get target QPS from Traffic Shaper                                 │ │
│   │ 2. Apply adaptive adjustments (if enabled)                            │ │
│   │ 3. Update Rate Limiter                                                │ │
│   │ 4. Adjust Worker Pool size                                            │ │
│   └────────────────────────────┬─────────────────────────────────────────┘ │
│                                │                                            │
│                                ▼                                            │
│   ┌──────────────────────────────────────────────────────────────────────┐ │
│   │ RATE LIMITER                                                          │ │
│   │ - Acquire slot (blocking or non-blocking)                             │ │
│   │ - Token bucket / Leaky bucket / Sliding window                        │ │
│   └────────────────────────────┬─────────────────────────────────────────┘ │
│                                │                                            │
│                                ▼                                            │
│   ┌──────────────────────────────────────────────────────────────────────┐ │
│   │ SCHEDULER                                                             │ │
│   │ 1. Select endpoint by enhanced weights (time-aware)                   │ │
│   │ 2. Check dependencies satisfied                                       │ │
│   │ 3. If not: trigger producer (with depth guard)                        │ │
│   └────────────────────────────┬─────────────────────────────────────────┘ │
│                                │                                            │
│                                ▼                                            │
│   ┌──────────────────────────────────────────────────────────────────────┐ │
│   │ REQUEST BUILDER                                                       │ │
│   │ For each input pin:                                                   │ │
│   │   - path param  → replace in URL                                      │ │
│   │   - query param → add to query string                                 │ │
│   │   - body param  → set in JSON body                                    │ │
│   └────────────────────────────┬─────────────────────────────────────────┘ │
│                                │                                            │
│                                ▼                                            │
│   ┌──────────────────────────────────────────────────────────────────────┐ │
│   │ EXECUTOR                                                              │ │
│   │ - Send HTTP request                                                   │ │
│   │ - Measure latency                                                     │ │
│   │ - Handle errors/retries                                               │ │
│   └────────────────────────────┬─────────────────────────────────────────┘ │
│                                │                                            │
│                                ▼                                            │
│   ┌──────────────────────────────────────────────────────────────────────┐ │
│   │ BACKPRESSURE HANDLER                                                  │ │
│   │ - Evaluate error rate and latency                                     │ │
│   │ - Trigger reduction/pause if thresholds exceeded                      │ │
│   └────────────────────────────┬─────────────────────────────────────────┘ │
│                                │                                            │
│                                ▼                                            │
│   ┌──────────────────────────────────────────────────────────────────────┐ │
│   │ RESPONSE PARSER                                                       │ │
│   │ For each output pin:                                                  │ │
│   │   - Extract value from response (JSONPath)                            │ │
│   │   - Add to parameter pool with semantic type                          │ │
│   └────────────────────────────┬─────────────────────────────────────────┘ │
│                                │                                            │
│                                ▼                                            │
│   ┌──────────────────────────────────────────────────────────────────────┐ │
│   │ METRICS COLLECTOR                                                     │ │
│   │ - Record latency, status, errors                                      │ │
│   │ - Update throughput counters                                          │ │
│   │ - Export to Prometheus (if enabled)                                   │ │
│   └──────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. 配置格式

### 9.1 最小配置

```yaml
# loadgen.yaml
openapi: "./backend/docs/swagger.yaml"
baseUrl: "http://localhost:8080/api/v1"

auth:
  endpoint: "/auth/login"
  credentials:
    username: "admin"
    password: "admin123"
  tokenPath: "$.data.access_token"

execution:
  duration: 60        # 秒
  concurrency: 10     # 并发数
```

### 9.2 完整配置

```yaml
# loadgen.yaml
openapi: "./backend/docs/swagger.yaml"
baseUrl: "http://localhost:8080/api/v1"

auth:
  endpoint: "/auth/login"
  method: "POST"
  body:
    username: "admin"
    password: "admin123"
  tokenPath: "$.data.access_token"
  headerName: "Authorization"
  headerPrefix: "Bearer "
  refreshEndpoint: "/auth/refresh"

# ============================================================
# 负载控制配置 (NEW)
# ============================================================
loadProfile:
  # 目标吞吐量
  targetQPS: 100          # 目标 QPS
  maxQPS: 500             # 硬上限

  # 并发控制
  minWorkers: 5           # 最小工作者数
  maxWorkers: 100         # 最大工作者数

  # 速率限制策略
  rateLimiter: "token_bucket"  # token_bucket | leaky_bucket | sliding_window
  burstSize: 20           # 允许短时突发

  # 自适应控制
  adaptive: true          # 根据延迟自动调整
  targetP95Latency: 100ms # 目标 P95 延迟

# ============================================================
# 流量整形配置 (NEW)
# ============================================================
trafficShape:
  # 预定义模式
  pattern: "sine_wave"    # constant | ramp | sine_wave | spike | step | custom

  # 正弦波参数
  sineWave:
    period: 60s           # 一个完整周期
    amplitude: 0.5        # ±50% 波动
    baselineQPS: 100      # 基线 QPS

  # 突发尖峰参数
  spike:
    baselineQPS: 50
    peakQPS: 500
    spikeDuration: 30s
    recoveryDuration: 60s
    interval: 300s        # 尖峰间隔

  # 阶梯形参数
  step:
    steps:
      - qps: 50
        duration: 60s
      - qps: 100
        duration: 120s
      - qps: 200
        duration: 60s
      - qps: 50
        duration: 60s

  # 自定义曲线
  custom:
    points:
      - time: 0s
        qps: 10
      - time: 30s
        qps: 100
      - time: 60s
        qps: 500          # 峰值
      - time: 90s
        qps: 100
      - time: 120s
        qps: 10
    interpolation: "linear"  # linear | step

# ============================================================
# 背压控制 (NEW)
# ============================================================
backpressure:
  errorRateThreshold: 0.1      # 10% 错误率触发
  latencyP99Threshold: 1s      # P99 延迟阈值
  strategy: "reduce"           # drop | reduce | pause | circuit
  recoveryPeriod: 30s          # 恢复检测周期

# ============================================================
# 请求分布控制 (Enhanced)
# ============================================================
distribution:
  # 全局读写比例
  readWriteRatio: 0.8         # 80% 读，20% 写

  # 分类级权重
  categories:
    partner:
      weight: 3
      readWriteRatio: 0.9     # 覆盖：90% 读
    trade:
      weight: 5
      readWriteRatio: 0.7     # 覆盖：70% 读
    inventory:
      weight: 2
    finance:
      weight: 1

  # 操作类型权重
  operations:
    GET: 60
    POST: 25
    PUT: 10
    DELETE: 5

# 端点级权重（最高优先级）
weights:
  "POST /trade/sales-orders":
    weight: 10
    # 时间段权重调整
    schedule:
      - time: "09:00-12:00"
        weight: 20            # 上午高峰
      - time: "12:00-14:00"
        weight: 5             # 午休低谷
      - time: "14:00-18:00"
        weight: 15            # 下午高峰
  "POST /trade/purchase-orders": 8
  "GET /partner/customers": 5
  "POST /partner/customers": 2

# 排除的端点
exclude:
  - "DELETE *"
  - "/identity/*"
  - "/system/*"

# ============================================================
# 会话模拟 (NEW)
# ============================================================
sessions:
  enabled: true
  concurrentSessions: 100
  sessionDuration:
    min: 60s
    max: 300s

  behaviors:
    browser:
      weight: 70
      thinkTime:
        min: 1s
        max: 5s
      actionsPerSession:
        min: 5
        max: 20

    apiClient:
      weight: 30
      thinkTime:
        min: 100ms
        max: 500ms
      actionsPerSession:
        min: 50
        max: 200

# ============================================================
# 参数池配置 (Enhanced)
# ============================================================
pool:
  maxValuesPerType: 10000     # 每种语义类型最大值数量
  defaultTTL: 30m             # 默认过期时间
  evictionPolicy: "lru"       # lru | fifo | random

  # 生产者链保护
  producerChain:
    maxDepth: 3               # 最大递归深度
    cooldownPeriod: 1s        # 冷却期
    minPoolSize: 5            # 低于此值触发补充
    refillBatchSize: 10       # 批量补充数量

# 覆盖自动推断的语义类型
semanticOverrides:
  - path: "$.data.id"
    context: "POST /trade/sales-orders"
    semantic: "order.sales.id"
  - path: "$.data.items[*].id"
    context: "POST /trade/sales-orders"
    semantic: "order.sales.item_id"

# 数据生成规则
dataGenerators:
  "common.code":
    type: "pattern"
    pattern: "{PREFIX}-{TIMESTAMP}-{RANDOM:4}"
  "common.name":
    type: "faker"
    faker: "company.name"
  "common.quantity":
    type: "random"
    min: 1
    max: 100

# 业务流程定义
workflows:
  sales_cycle:
    weight: 5
    steps:
      - endpoint: "POST /trade/sales-orders"
        extract:
          order_id: "$.data.id"
      - endpoint: "POST /trade/sales-orders/{order_id}/confirm"
      - endpoint: "POST /trade/sales-orders/{order_id}/ship"

execution:
  duration: 300
  concurrency: 20
  rampUp: 30

  warmup:
    iterations: 20
    fill:
      - "entity.customer.id"
      - "entity.supplier.id"
      - "entity.product.id"
      - "entity.warehouse.id"

# ============================================================
# SLO 断言 (NEW)
# ============================================================
assertions:
  global:
    maxErrorRate: 0.01        # 1%
    maxP95Latency: 200ms
    minSuccessRate: 0.99

  endpoints:
    "POST /trade/sales-orders":
      maxP99Latency: 500ms
      maxErrorRate: 0.001

    "GET /partner/customers":
      maxP95Latency: 50ms

exitOnFailure: true           # 断言失败时退出

# ============================================================
# 输出配置 (Enhanced)
# ============================================================
output:
  console:
    enabled: true
    interval: 10s             # 刷新间隔

  json:
    enabled: true
    file: "./results/loadgen-{{.Timestamp}}.json"

  html:
    enabled: true
    file: "./results/loadgen-report.html"

  prometheus:
    enabled: true
    port: 9090
    path: "/metrics"

# ============================================================
# 场景定义 (NEW)
# ============================================================
scenarios:
  normal_day:
    duration: 8h
    trafficShape:
      pattern: "sine_wave"
      sineWave:
        period: 4h
        amplitude: 0.3
        baselineQPS: 100
    description: "模拟正常工作日流量"

  flash_sale:
    duration: 30m
    trafficShape:
      pattern: "spike"
      spike:
        baselineQPS: 100
        peakQPS: 2000
        spikeDuration: 5m
        interval: 10m
    focusEndpoints:
      - "POST /trade/sales-orders"
      - "GET /catalog/products"
    description: "模拟闪购促销流量"

  stress_test:
    duration: 1h
    trafficShape:
      pattern: "ramp"
      ramp:
        startQPS: 10
        endQPS: 1000
    findBreakingPoint: true
    description: "压力测试，寻找系统极限"
```

---

## 10. 目录结构

```
tools/loadgen/
├── cmd/
│   └── main.go                    # CLI 入口
├── internal/
│   ├── config/
│   │   ├── config.go              # 配置结构
│   │   ├── loader.go              # 配置加载
│   │   └── validation.go          # 配置验证
│   ├── parser/
│   │   ├── openapi.go             # OpenAPI 解析器
│   │   ├── inference.go           # 语义推断
│   │   └── rules.go               # 推断规则
│   ├── circuit/
│   │   ├── board.go               # 电路板主控
│   │   ├── unit.go                # 端点单元
│   │   ├── pin.go                 # 引脚定义
│   │   ├── graph.go               # 依赖图
│   │   └── producer_guard.go      # 生产者链保护 (NEW)
│   ├── pool/
│   │   ├── pool.go                # 参数池接口
│   │   ├── sharded.go             # 分片实现 (NEW)
│   │   ├── ringbuffer.go          # 环形缓冲区 (NEW)
│   │   └── value.go               # 参数值结构
│   ├── loadctrl/                  # 负载控制层 (NEW)
│   │   ├── controller.go          # 负载控制器
│   │   ├── ratelimiter.go         # 速率限制器
│   │   ├── shaper.go              # 流量整形器接口
│   │   ├── shaper_sine.go         # 正弦波整形器
│   │   ├── shaper_spike.go        # 突发整形器
│   │   ├── shaper_step.go         # 阶梯整形器
│   │   ├── shaper_custom.go       # 自定义整形器
│   │   └── backpressure.go        # 背压处理器
│   ├── selector/                  # 请求选择器 (NEW)
│   │   ├── weighted.go            # 增强权重选择器
│   │   ├── session.go             # 会话模拟器
│   │   └── schedule.go            # 时间调度
│   ├── generator/
│   │   ├── generator.go           # 数据生成器接口
│   │   ├── faker.go               # Faker 生成器
│   │   ├── pattern.go             # 模式生成器
│   │   └── random.go              # 随机生成器
│   ├── executor/
│   │   ├── executor.go            # 执行器
│   │   ├── scheduler.go           # 调度器
│   │   ├── worker.go              # 工作协程
│   │   ├── workerpool.go          # 工作者池 (NEW)
│   │   └── request.go             # 请求构建
│   ├── client/
│   │   ├── client.go              # HTTP 客户端
│   │   ├── auth.go                # 认证处理
│   │   └── response.go            # 响应解析
│   ├── metrics/
│   │   ├── collector.go           # 指标收集
│   │   ├── reporter.go            # 报告生成
│   │   ├── console.go             # 控制台输出
│   │   ├── prometheus.go          # Prometheus 导出 (NEW)
│   │   └── assertions.go          # SLO 断言 (NEW)
│   └── scenario/                  # 场景管理 (NEW)
│       ├── scenario.go            # 场景定义
│       └── runner.go              # 场景运行器
├── configs/
│   ├── erp.yaml                   # ERP 系统配置示例
│   ├── stress.yaml                # 压力测试配置
│   └── scenarios/                 # 场景配置目录
│       ├── normal_day.yaml
│       ├── flash_sale.yaml
│       └── stress_test.yaml
├── go.mod
├── go.sum
└── loadgen.md                     # 本文档
```

---

## 11. 使用方式

### 11.1 命令行

```bash
# 构建
cd tools/loadgen
go build -o loadgen ./cmd/...

# 最小运行
./loadgen -config configs/erp.yaml

# 指定时长和并发
./loadgen -config configs/erp.yaml -duration 5m -concurrency 50

# 指定目标 QPS
./loadgen -config configs/erp.yaml -qps 200

# 使用流量整形
./loadgen -config configs/erp.yaml -shape sine_wave -baseline-qps 100 -amplitude 0.5

# 运行特定场景
./loadgen -config configs/erp.yaml -scenario flash_sale

# 仅列出端点
./loadgen -config configs/erp.yaml -list

# 预热模式（只填充参数池）
./loadgen -config configs/erp.yaml -warmup-only

# Dry-run 模式（验证语义推断）
./loadgen -config configs/erp.yaml -dry-run

# 指定输出格式
./loadgen -config configs/erp.yaml -output json > results.json

# 启用 Prometheus 指标
./loadgen -config configs/erp.yaml -prometheus :9090
```

### 11.2 Makefile 集成

```makefile
# 添加到项目 Makefile

.PHONY: loadgen-build loadgen-run loadgen-stress loadgen-scenario

loadgen-build:
	go build -o bin/loadgen ./tools/loadgen/cmd/...

loadgen-run:
	./bin/loadgen -config tools/loadgen/configs/erp.yaml \
	  -duration 5m -qps 100

loadgen-stress:
	./bin/loadgen -config tools/loadgen/configs/erp.yaml \
	  -duration 30m -qps 500 -shape ramp

loadgen-scenario:
	./bin/loadgen -config tools/loadgen/configs/erp.yaml \
	  -scenario $(SCENARIO)

# 示例：make loadgen-scenario SCENARIO=flash_sale
```

---

## 12. 输出示例

### 12.1 控制台输出

```
========================================
  Circuit-Based API Load Generator
========================================

Config: tools/loadgen/configs/erp.yaml
OpenAPI: ./backend/docs/swagger.yaml
Base URL: http://localhost:8080/api/v1

Parsed 47 endpoints, 156 input pins, 89 output pins
Detected 0 circular dependencies

Load Profile:
  Target QPS:     100
  Max QPS:        500
  Rate Limiter:   token_bucket (burst: 20)
  Traffic Shape:  sine_wave (period: 60s, amplitude: ±50%)

Warming up parameter pool...
  ✓ entity.customer.id: 20 values
  ✓ entity.supplier.id: 20 values
  ✓ entity.product.id: 20 values
  ✓ entity.warehouse.id: 10 values

Starting load generation...
  Duration: 5m0s
  Workers: 10-100 (adaptive)

[00:30] ████████████████████  1,234 req |  82.3 QPS | 98.2% ok | p95: 45ms | shape: ↗ rising
[01:00] ████████████████████  3,512 req | 113.9 QPS | 98.5% ok | p95: 43ms | shape: ▲ peak
[01:30] ████████████████████  5,289 req |  88.9 QPS | 98.3% ok | p95: 44ms | shape: ↘ falling
[02:00] ████████████████████  6,789 req |  75.0 QPS | 98.4% ok | p95: 42ms | shape: ▼ trough
...

========================================
  RESULTS
========================================

Duration:      5m0s
Total Reqs:    25,534
Success:       25,112 (98.3%)
Failed:        422 (1.7%)

Throughput:
  Target QPS:   100 (sine wave ±50%)
  Actual Avg:   85.1 req/s
  Peak:         148.2 req/s
  Trough:       51.3 req/s

Latency:
  Min:         8ms
  Avg:         24ms
  P50:         21ms
  P95:         45ms
  P99:         89ms
  Max:         234ms

Load Control Stats:
  Rate Limiter Rejections: 156
  Backpressure Events:     3
  Worker Pool Adjustments: 47

Top Endpoints by Volume:
  1. GET /partner/customers           4,345 (17.0%)
  2. POST /trade/sales-orders         3,890 (15.2%)
  3. GET /catalog/products            3,567 (14.0%)
  4. POST /trade/purchase-orders      2,234 (8.7%)
  ...

Errors:
  - 500 Internal Server Error: 256 (POST /trade/sales-orders)
  - 409 Conflict: 145 (POST /partner/customers)
  - 422 Unprocessable Entity: 21 (POST /inventory/stock/adjust)

Parameter Pool Stats:
  entity.customer.id:    434 values (20 seed + 414 created)
  entity.product.id:     389 values (20 seed + 369 created)
  order.sales.id:        3,890 values
  order.purchase.id:     2,234 values
  Pool Evictions:        1,234 (LRU)

SLO Assertions:
  ✓ Global error rate: 1.7% (max: 5%)
  ✓ Global P95 latency: 45ms (max: 200ms)
  ✓ POST /trade/sales-orders P99: 89ms (max: 500ms)
  ✗ POST /trade/sales-orders error rate: 6.6% (max: 1%)

Exit Code: 1 (SLO assertion failed)
```

### 12.2 Prometheus 指标

```
# HELP loadgen_requests_total Total number of requests
# TYPE loadgen_requests_total counter
loadgen_requests_total{endpoint="POST /trade/sales-orders",status="success"} 3634
loadgen_requests_total{endpoint="POST /trade/sales-orders",status="failure"} 256

# HELP loadgen_request_duration_seconds Request duration in seconds
# TYPE loadgen_request_duration_seconds histogram
loadgen_request_duration_seconds_bucket{endpoint="POST /trade/sales-orders",le="0.01"} 234
loadgen_request_duration_seconds_bucket{endpoint="POST /trade/sales-orders",le="0.05"} 2890
loadgen_request_duration_seconds_bucket{endpoint="POST /trade/sales-orders",le="0.1"} 3456
loadgen_request_duration_seconds_bucket{endpoint="POST /trade/sales-orders",le="+Inf"} 3890

# HELP loadgen_current_qps Current queries per second
# TYPE loadgen_current_qps gauge
loadgen_current_qps 98.5

# HELP loadgen_target_qps Target queries per second from traffic shaper
# TYPE loadgen_target_qps gauge
loadgen_target_qps 100

# HELP loadgen_pool_size Current parameter pool size by semantic type
# TYPE loadgen_pool_size gauge
loadgen_pool_size{semantic="entity.customer.id"} 434
loadgen_pool_size{semantic="order.sales.id"} 3890

# HELP loadgen_backpressure_state Current backpressure state (0=normal, 1=warning, 2=critical)
# TYPE loadgen_backpressure_state gauge
loadgen_backpressure_state 0
```

---

## 13. 设计决策记录

### 13.1 ADR-001: 基于 OpenAPI 自动推断

**背景**: 需要减少配置负担，同时保持灵活性。

**决策**: 默认从 OpenAPI spec 自动推断语义类型，支持手动覆盖。

**结果**: 零配置启动，复杂场景可手动调整。

### 13.2 ADR-002: 参数池随机选择策略

**背景**: 需要模拟真实用户行为。

**决策**: 默认使用随机选择策略，支持 FIFO/Round-Robin/Weighted。

**结果**: 更真实的负载分布。

### 13.3 ADR-003: 自动触发生产者

**背景**: 参数池可能为空或数据不足。

**决策**: 当参数池缺少必需值时，自动触发能产生该值的端点。

**结果**: 自愈能力，无需预先创建所有数据。

### 13.4 ADR-004: Go 实现

**背景**: 需要高性能和与后端技术栈一致。

**决策**: 使用 Go 实现。

**结果**: 高并发性能，团队熟悉。

### 13.5 ADR-005: 令牌桶速率限制 (NEW)

**背景**: 需要精确控制 QPS 同时允许短时突发。

**决策**: 使用 `golang.org/x/time/rate` 实现令牌桶算法。

**结果**:
- 精确的 QPS 控制
- 支持突发流量（burst）
- 低开销、高性能
- 可动态调整速率

### 13.6 ADR-006: 分片参数池 (NEW)

**背景**: 高并发下单一参数池会成为瓶颈。

**决策**: 使用分片设计，按语义类型哈希分配到不同分片。

**结果**:
- 减少锁争用
- 线性扩展能力
- 每个分片独立淘汰

### 13.7 ADR-007: 生产者链深度限制 (NEW)

**背景**: 自动触发生产者可能导致级联调用。

**决策**: 限制最大递归深度（默认 3），添加冷却期。

**结果**:
- 防止无限递归
- 避免系统过载
- 保持自愈能力的同时增加稳定性

### 13.8 ADR-008: 流量整形器 (NEW)

**背景**: 真实流量有波峰波谷，恒定负载不真实。

**决策**: 实现多种流量模式（正弦波、突发、阶梯、自定义）。

**结果**:
- 模拟真实业务场景
- 发现系统在不同负载下的表现
- 支持压力测试找到系统极限

---

## 14. 实现路线图

### Phase 1.1 (MVP 增强) - 生产就绪必需

| 任务 | 优先级 | 预估 |
|------|--------|------|
| 速率限制器组件 | P0 | 2天 |
| 分片参数池实现 | P0 | 1天 |
| 生产者链深度保护 | P0 | 1天 |
| 参数池大小限制和淘汰 | P1 | 1天 |

### Phase 1.2 (流量控制)

| 任务 | 优先级 | 预估 |
|------|--------|------|
| 流量整形器接口 | P0 | 2天 |
| 正弦波、突发、阶梯模式 | P1 | 2天 |
| 自定义曲线支持 | P2 | 1天 |
| 背压处理器 | P1 | 1天 |

### Phase 1 (MVP 基础)

| 任务 | 优先级 | 预估 |
|------|--------|------|
| OpenAPI 解析器 | P0 | 3天 |
| 语义推断引擎 | P0 | 2天 |
| 参数池实现 | P0 | 2天 |
| 基础执行器 | P0 | 2天 |
| 控制台输出 | P0 | 1天 |

### Phase 2 (增强分布控制)

| 任务 | 优先级 | 预估 |
|------|--------|------|
| 增强权重选择器 | P1 | 2天 |
| 时间段权重调度 | P2 | 1天 |
| 读写比例控制 | P1 | 1天 |
| 会话模拟器 | P2 | 2天 |
| Workflow 支持 | P1 | 2天 |
| 数据生成器扩展 | P1 | 1天 |
| HTML 报告 | P2 | 1天 |

### Phase 3 (生产特性)

| 任务 | 优先级 | 预估 |
|------|--------|------|
| Prometheus 指标导出 | P1 | 1天 |
| SLO 断言验证 | P1 | 1天 |
| 场景管理 | P2 | 2天 |
| 分布式执行 | P3 | 5天 |
| 录制回放 | P3 | 3天 |
| CI/CD 集成 | P2 | 1天 |

---

## 15. 风险与缓解

| 风险 | 严重度 | 缓解措施 |
|------|--------|---------|
| 参数池并发争用 | 高 | 分片参数池 `ShardedParameterPool` |
| 生产者级联过载 | 高 | `maxDepth: 3`，冷却期，批量补充 |
| 长时间测试内存增长 | 中 | `maxPoolSize`，LRU 淘汰，TTL 过期 |
| 语义推断错误 | 中 | `--dry-run` 验证，置信度阈值，手动覆盖 |
| OpenAPI 规范不兼容 | 低 | 支持 3.0/3.1，优雅降级，警告日志 |
| 目标系统过载 | 中 | 背压检测，自动降速，熔断机制 |

---

## 16. 参考资料

- [OpenAPI Specification](https://spec.openapis.org/oas/v3.0.3)
- [JSONPath Syntax](https://goessner.net/articles/JsonPath/)
- [gofakeit - Fake Data Generator](https://github.com/brianvoe/gofakeit)
- [golang.org/x/time/rate - Token Bucket](https://pkg.go.dev/golang.org/x/time/rate)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
