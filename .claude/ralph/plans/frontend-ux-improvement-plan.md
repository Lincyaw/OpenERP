# 前端用户体验改造计划

> **版本**: 1.0.0
> **创建日期**: 2025-01-27
> **状态**: 待审批

---

## 目录

1. [执行摘要](#1-执行摘要)
2. [问题清单](#2-问题清单)
3. [改造计划](#3-改造计划)
4. [实施路线图](#4-实施路线图)
5. [验收标准](#5-验收标准)

---

## 1. 执行摘要

### 1.1 背景

本计划基于产品经理从用户视角对 ERP 系统前端的全面审查。系统需要服务不同角色（管理员、销售、仓库、财务）和不同年龄层（20-50岁以上）的用户。

### 1.2 主要发现

| 类别 | 优点 | 需改进 |
|------|------|--------|
| 设计系统 | 完善的 Token 体系 | - |
| 主题支持 | 三种主题已定义 | Elder 主题有 bug |
| 可访问性 | 基础设施完善 | 缺少 ARIA 增强 |
| 错误处理 | Axios 拦截器良好 | 用户提示不友好 |
| 移动端 | 响应式断点完整 | 导航体验差 |
| 角色定制 | 权限系统完善 | Dashboard 无差异化 |

### 1.3 改造优先级

```
P0 (严重)  → 立即修复，影响核心功能
P1 (高)    → 本迭代完成，影响用户体验
P2 (中)    → 下迭代完成，提升体验
P3 (低)    → 后续优化，锦上添花
```

---

## 2. 问题清单

### 2.1 P0 - 严重问题

#### [P0-001] Elder 主题存储逻辑错误

**文件**: `frontend/src/hooks/useTheme.ts:110-114`

**问题描述**:
```typescript
// 当前代码
const setThemeWithPersist = useCallback(
  (newTheme: Theme) => {
    setTheme(newTheme === 'elder' ? 'light' : newTheme)  // BUG!
    applyTheme(newTheme)
  },
  [setTheme]
)
```

当用户选择 Elder 主题时：
1. `applyTheme('elder')` 正确应用到 DOM
2. `setTheme('light')` 错误地存储为 light
3. 刷新页面后，加载的是 light 主题而非 elder

**影响用户**: 50岁以上用户无法可靠使用大字体/高对比度主题

**修复方案**:
```typescript
const setThemeWithPersist = useCallback(
  (newTheme: Theme) => {
    setTheme(newTheme)  // 正确存储原始值
    applyTheme(newTheme)
  },
  [setTheme]
)
```

**工作量**: 0.5h

---

### 2.2 P1 - 高优先级问题

#### [P1-001] 错误提示不友好

**当前状态**: 错误信息直接显示技术细节或英文

**问题示例**:
```
❌ "Network Error"
❌ "Request failed with status code 500"
❌ "TypeError: Cannot read property 'id' of undefined"
```

**影响用户**: 所有非技术用户

**改造方案**: 创建统一错误处理服务

**新增文件**: `frontend/src/services/error-handler.ts`

```typescript
// 错误类型枚举
export enum ErrorType {
  NETWORK = 'NETWORK',           // 网络错误
  AUTH = 'AUTH',                 // 认证错误
  PERMISSION = 'PERMISSION',     // 权限错误
  VALIDATION = 'VALIDATION',     // 验证错误
  NOT_FOUND = 'NOT_FOUND',       // 资源不存在
  CONFLICT = 'CONFLICT',         // 数据冲突
  RATE_LIMIT = 'RATE_LIMIT',     // 请求限制
  SERVER = 'SERVER',             // 服务器错误
  UNKNOWN = 'UNKNOWN',           // 未知错误
}

// 错误消息映射（支持 i18n）
export const errorMessages: Record<ErrorType, ErrorMessageConfig> = {
  [ErrorType.NETWORK]: {
    title: 'error.network.title',           // "网络连接失败"
    message: 'error.network.message',       // "请检查您的网络连接后重试"
    action: 'error.network.action',         // "重试"
    icon: 'IconWifi',
  },
  [ErrorType.AUTH]: {
    title: 'error.auth.title',              // "登录已过期"
    message: 'error.auth.message',          // "请重新登录以继续操作"
    action: 'error.auth.action',            // "重新登录"
    icon: 'IconLock',
  },
  [ErrorType.PERMISSION]: {
    title: 'error.permission.title',        // "权限不足"
    message: 'error.permission.message',    // "您没有执行此操作的权限，请联系管理员"
    action: null,
    icon: 'IconShield',
  },
  [ErrorType.VALIDATION]: {
    title: 'error.validation.title',        // "信息填写有误"
    message: 'error.validation.message',    // "请检查标红的字段并修正"
    action: null,
    icon: 'IconAlertCircle',
  },
  [ErrorType.NOT_FOUND]: {
    title: 'error.notFound.title',          // "内容不存在"
    message: 'error.notFound.message',      // "您访问的内容可能已被删除或移动"
    action: 'error.notFound.action',        // "返回首页"
    icon: 'IconSearch',
  },
  [ErrorType.CONFLICT]: {
    title: 'error.conflict.title',          // "数据冲突"
    message: 'error.conflict.message',      // "该数据已被其他用户修改，请刷新后重试"
    action: 'error.conflict.action',        // "刷新页面"
    icon: 'IconRefresh',
  },
  [ErrorType.RATE_LIMIT]: {
    title: 'error.rateLimit.title',         // "操作过于频繁"
    message: 'error.rateLimit.message',     // "请稍后再试"
    action: null,
    icon: 'IconClock',
  },
  [ErrorType.SERVER]: {
    title: 'error.server.title',            // "服务器繁忙"
    message: 'error.server.message',        // "系统正在维护中，请稍后重试"
    action: 'error.server.action',          // "重试"
    helpText: 'error.server.help',          // "如问题持续，请联系客服：400-xxx-xxxx"
    icon: 'IconServer',
  },
  [ErrorType.UNKNOWN]: {
    title: 'error.unknown.title',           // "操作失败"
    message: 'error.unknown.message',       // "发生了一个错误，请稍后重试"
    action: 'error.unknown.action',         // "重试"
    icon: 'IconAlertTriangle',
  },
}

// 错误类型检测
export function detectErrorType(error: unknown): ErrorType {
  if (axios.isAxiosError(error)) {
    if (!error.response) return ErrorType.NETWORK

    const status = error.response.status
    switch (status) {
      case 401: return ErrorType.AUTH
      case 403: return ErrorType.PERMISSION
      case 404: return ErrorType.NOT_FOUND
      case 409: return ErrorType.CONFLICT
      case 422: return ErrorType.VALIDATION
      case 429: return ErrorType.RATE_LIMIT
      case 500:
      case 502:
      case 503:
      case 504:
        return ErrorType.SERVER
      default:
        return ErrorType.UNKNOWN
    }
  }
  return ErrorType.UNKNOWN
}

// 统一错误处理函数
export function handleError(error: unknown, options?: ErrorHandlerOptions): void {
  const errorType = detectErrorType(error)
  const config = errorMessages[errorType]

  // 根据选项决定显示方式
  if (options?.silent) return

  if (options?.useModal) {
    showErrorModal(config, options)
  } else {
    showErrorToast(config, options)
  }

  // 可选：上报错误日志
  if (options?.reportError !== false) {
    reportErrorToServer(error, errorType)
  }
}
```

**修改文件**: `frontend/src/services/axios-instance.ts`

```typescript
// 在响应拦截器中使用统一错误处理
axiosInstance.interceptors.response.use(
  (response) => response,
  async (error) => {
    // ... 现有的 token 刷新逻辑 ...

    // 使用统一错误处理
    handleError(error, {
      silent: error.config?.silentError,  // 允许单个请求禁用提示
      context: error.config?.errorContext, // 自定义上下文
    })

    return Promise.reject(error)
  }
)
```

**工作量**: 8h

---

#### [P1-002] 表单验证错误提示不清晰

**当前状态**: 验证错误可能不够具体

**问题示例**:
```
❌ "该字段不能为空"        → 哪个字段？
❌ "格式不正确"            → 什么格式？
❌ "Invalid phone number"  → 英文提示
```

**改造方案**: 增强验证消息

**修改文件**: `frontend/src/locales/zh-CN/validation.json`

```json
{
  "required": "请填写{field}",
  "email": "请输入正确的邮箱地址，例如：name@example.com",
  "phone": "请输入11位手机号码，例如：13800138000",
  "idCard": "请输入18位身份证号码",
  "minLength": "{field}至少需要{min}个字符",
  "maxLength": "{field}不能超过{max}个字符",
  "min": "{field}不能小于{min}",
  "max": "{field}不能大于{max}",
  "pattern": {
    "alphanumeric": "{field}只能包含字母和数字",
    "chinese": "{field}只能包含中文字符",
    "sku": "SKU格式：字母开头，可包含字母、数字和连字符",
    "barcode": "条形码格式：8-14位数字"
  },
  "unique": "该{field}已存在，请使用其他值",
  "range": "{field}必须在{min}到{max}之间",
  "date": {
    "invalid": "请输入正确的日期",
    "future": "日期不能是将来的时间",
    "past": "日期不能是过去的时间",
    "before": "日期必须早于{date}",
    "after": "日期必须晚于{date}"
  },
  "file": {
    "size": "文件大小不能超过{size}MB",
    "type": "只支持以下格式：{types}",
    "required": "请选择要上传的文件"
  },
  "password": {
    "weak": "密码强度不足，请包含大小写字母、数字和特殊字符",
    "mismatch": "两次输入的密码不一致"
  },
  "number": {
    "invalid": "请输入有效的数字",
    "integer": "请输入整数",
    "positive": "请输入大于0的数字",
    "decimal": "最多保留{places}位小数"
  }
}
```

**新增组件**: `frontend/src/components/common/form/FormErrorSummary.tsx`

```typescript
/**
 * 表单错误汇总组件 - 在表单顶部显示所有错误
 * 适合老年用户和复杂表单
 */
interface FormErrorSummaryProps {
  errors: FieldErrors
  fieldLabels: Record<string, string>
  onFieldClick?: (fieldName: string) => void
}

export function FormErrorSummary({ errors, fieldLabels, onFieldClick }: FormErrorSummaryProps) {
  const errorList = Object.entries(errors)

  if (errorList.length === 0) return null

  return (
    <div
      className="form-error-summary"
      role="alert"
      aria-live="polite"
    >
      <div className="form-error-summary__header">
        <IconAlertCircle />
        <span>请修正以下 {errorList.length} 个问题：</span>
      </div>
      <ul className="form-error-summary__list">
        {errorList.map(([field, error]) => (
          <li key={field}>
            <button
              type="button"
              onClick={() => onFieldClick?.(field)}
              className="form-error-summary__link"
            >
              {fieldLabels[field] || field}: {error?.message}
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}
```

**工作量**: 6h

---

#### [P1-003] 移动端侧边栏导航体验差

**当前状态**:
- 无汉堡菜单按钮
- 无遮罩层
- 无滑动手势支持

**影响用户**: 外勤销售人员使用手机时

**改造方案**:

**修改文件**: `frontend/src/components/layout/Header.tsx`

```typescript
// 添加移动端菜单按钮
export function Header() {
  const { sidebarCollapsed, toggleSidebar, setSidebarVisible } = useAppStore()
  const [isMobile, setIsMobile] = useState(window.innerWidth <= 768)

  useEffect(() => {
    const handleResize = () => setIsMobile(window.innerWidth <= 768)
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  return (
    <header className="header">
      <div className="header__left">
        {/* 移动端汉堡菜单 */}
        {isMobile && (
          <Button
            icon={<IconMenu />}
            theme="borderless"
            onClick={() => setSidebarVisible(true)}
            aria-label="打开导航菜单"
            className="header__menu-button"
          />
        )}

        <Breadcrumb />
      </div>
      {/* ... 其他内容 ... */}
    </header>
  )
}
```

**修改文件**: `frontend/src/components/layout/Sidebar.tsx`

```typescript
// 添加移动端遮罩和手势
export function Sidebar() {
  const { sidebarVisible, setSidebarVisible } = useAppStore()
  const [isMobile, setIsMobile] = useState(window.innerWidth <= 768)

  // 滑动手势支持
  const touchStartX = useRef<number>(0)

  const handleTouchStart = (e: TouchEvent) => {
    touchStartX.current = e.touches[0].clientX
  }

  const handleTouchEnd = (e: TouchEvent) => {
    const deltaX = e.changedTouches[0].clientX - touchStartX.current
    if (deltaX < -50) {  // 左滑关闭
      setSidebarVisible(false)
    }
  }

  return (
    <>
      {/* 移动端遮罩 */}
      {isMobile && sidebarVisible && (
        <div
          className="sidebar-overlay"
          onClick={() => setSidebarVisible(false)}
          aria-hidden="true"
        />
      )}

      <nav
        className={cn('sidebar', {
          'sidebar--collapsed': sidebarCollapsed,
          'sidebar--mobile-visible': isMobile && sidebarVisible,
        })}
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
        role="navigation"
        aria-label="主导航"
      >
        {/* 移动端关闭按钮 */}
        {isMobile && (
          <button
            className="sidebar__close"
            onClick={() => setSidebarVisible(false)}
            aria-label="关闭导航菜单"
          >
            <IconClose />
          </button>
        )}

        {/* ... 现有内容 ... */}
      </nav>
    </>
  )
}
```

**修改文件**: `frontend/src/components/layout/Sidebar.css`

```css
/* 移动端遮罩 */
.sidebar-overlay {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: var(--z-index-modal-backdrop);
  animation: fadeIn var(--duration-normal) var(--easing-standard);
}

/* 移动端侧边栏 */
@media (max-width: 768px) {
  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 280px;
    transform: translateX(-100%);
    z-index: var(--z-index-modal);
    transition: transform var(--duration-normal) var(--easing-standard);
  }

  .sidebar--mobile-visible {
    transform: translateX(0);
  }

  .sidebar__close {
    position: absolute;
    top: var(--spacing-4);
    right: var(--spacing-4);
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    cursor: pointer;
  }
}

/* 汉堡菜单按钮 */
.header__menu-button {
  display: none;
}

@media (max-width: 768px) {
  .header__menu-button {
    display: flex;
    min-width: 44px;
    min-height: 44px;
  }
}
```

**工作量**: 6h

---

#### [P1-004] 缺少角色定制化仪表板

**当前状态**: 所有角色看到相同的 Dashboard

**影响用户**: 所有用户需要在众多信息中找到与自己相关的内容

**改造方案**:

**新增文件**: `frontend/src/config/dashboard-config.ts`

```typescript
// 角色对应的仪表板卡片配置
export const dashboardCardsByRole: Record<string, string[]> = {
  admin: [
    'systemHealth',      // 系统健康状态
    'userActivity',      // 用户活跃度
    'products',          // 产品总览
    'customers',         // 客户总览
    'salesOrders',       // 销售订单
    'lowStockAlert',     // 库存预警
    'receivables',       // 应收款
    'payables',          // 应付款
  ],
  sales: [
    'myOrders',          // 我的订单
    'mySalesTarget',     // 我的销售目标
    'myCustomers',       // 我的客户
    'lowStockAlert',     // 库存预警（可用库存）
    'receivables',       // 应收款（我的客户）
  ],
  warehouse: [
    'lowStockAlert',     // 库存预警
    'pendingInbound',    // 待入库
    'pendingOutbound',   // 待出库
    'stockTaking',       // 盘点任务
    'inventoryValue',    // 库存价值
  ],
  finance: [
    'cashFlow',          // 现金流
    'receivables',       // 应收款
    'payables',          // 应付款
    'profitMargin',      // 利润率
    'agingReport',       // 账龄分析
  ],
}

// 卡片配置定义
export const cardDefinitions: Record<string, CardDefinition> = {
  products: {
    key: 'products',
    title: 'dashboard.cards.products',
    icon: 'IconBox',
    color: 'blue',
    api: getCatalogProductsStatsCount,
    permissions: ['catalog:product:read'],
  },
  // ... 其他卡片定义
}
```

**修改文件**: `frontend/src/pages/Dashboard.tsx`

```typescript
export function Dashboard() {
  const { user } = useAuthStore()
  const userRole = user?.roles?.[0] || 'guest'

  // 根据角色获取可见卡片
  const visibleCardKeys = useMemo(() => {
    const roleCards = dashboardCardsByRole[userRole] || dashboardCardsByRole.admin
    // 再次过滤权限
    return roleCards.filter(cardKey => {
      const def = cardDefinitions[cardKey]
      if (!def.permissions) return true
      return user?.permissions?.some(p => def.permissions.includes(p))
    })
  }, [userRole, user?.permissions])

  // 可定制化：用户可以调整卡片顺序和显示/隐藏
  const [cardOrder, setCardOrder] = useLocalStorage<string[]>(
    `dashboard-cards-${user?.id}`,
    visibleCardKeys
  )

  return (
    <div className="dashboard">
      <DashboardCustomizer
        availableCards={visibleCardKeys}
        currentOrder={cardOrder}
        onChange={setCardOrder}
      />

      <div className="dashboard__cards">
        {cardOrder.map(cardKey => (
          <DashboardCard
            key={cardKey}
            definition={cardDefinitions[cardKey]}
          />
        ))}
      </div>
    </div>
  )
}
```

**工作量**: 16h

---

#### [P1-005] 缺少快捷操作入口

**当前状态**: 所有操作需要通过菜单导航

**改造方案**:

**新增组件**: `frontend/src/components/common/QuickActions.tsx`

```typescript
/**
 * 浮动操作按钮 (FAB) - 快捷操作入口
 * 根据当前页面和用户角色显示不同操作
 */
interface QuickAction {
  key: string
  label: string
  icon: React.ReactNode
  onClick: () => void
  permissions?: string[]
  hotkey?: string  // 如 'ctrl+n'
}

const globalActions: QuickAction[] = [
  {
    key: 'newSalesOrder',
    label: 'quickAction.newSalesOrder',
    icon: <IconPlus />,
    hotkey: 'ctrl+shift+o',
    permissions: ['trade:sales:create'],
    onClick: () => navigate('/trade/sales/new'),
  },
  {
    key: 'newProduct',
    label: 'quickAction.newProduct',
    icon: <IconBox />,
    hotkey: 'ctrl+shift+p',
    permissions: ['catalog:product:create'],
    onClick: () => navigate('/catalog/products/new'),
  },
  {
    key: 'newCustomer',
    label: 'quickAction.newCustomer',
    icon: <IconUser />,
    hotkey: 'ctrl+shift+c',
    permissions: ['partner:customer:create'],
    onClick: () => navigate('/partner/customers/new'),
  },
  {
    key: 'search',
    label: 'quickAction.globalSearch',
    icon: <IconSearch />,
    hotkey: 'ctrl+k',
    onClick: () => openGlobalSearch(),
  },
]

export function QuickActions() {
  const [expanded, setExpanded] = useState(false)
  const { hasPermission } = useAuthStore()

  const visibleActions = globalActions.filter(action => {
    if (!action.permissions) return true
    return action.permissions.some(p => hasPermission(p))
  })

  // 注册键盘快捷键
  useHotkeys(visibleActions)

  return (
    <div className="quick-actions">
      {expanded && (
        <div className="quick-actions__menu">
          {visibleActions.map(action => (
            <Tooltip key={action.key} content={`${t(action.label)} (${action.hotkey})`}>
              <Button
                icon={action.icon}
                onClick={action.onClick}
                theme="light"
              />
            </Tooltip>
          ))}
        </div>
      )}

      <Button
        icon={expanded ? <IconClose /> : <IconPlus />}
        className="quick-actions__trigger"
        onClick={() => setExpanded(!expanded)}
        size="large"
        theme="solid"
        aria-expanded={expanded}
        aria-label="快捷操作"
      />
    </div>
  )
}
```

**新增组件**: `frontend/src/components/common/GlobalSearch.tsx`

```typescript
/**
 * 全局搜索弹窗 - Ctrl+K 打开
 * 搜索产品、客户、订单等
 */
export function GlobalSearch() {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])

  // Ctrl+K 打开搜索
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault()
        setOpen(true)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  return (
    <Modal
      visible={open}
      onCancel={() => setOpen(false)}
      footer={null}
      className="global-search-modal"
      closable={false}
    >
      <Input
        prefix={<IconSearch />}
        placeholder="搜索产品、客户、订单..."
        value={query}
        onChange={setQuery}
        autoFocus
        size="large"
      />

      <div className="global-search__results">
        {results.map(result => (
          <SearchResultItem key={result.id} result={result} />
        ))}
      </div>

      <div className="global-search__tips">
        <span>↑↓ 选择</span>
        <span>↵ 打开</span>
        <span>esc 关闭</span>
      </div>
    </Modal>
  )
}
```

**工作量**: 12h

---

### 2.3 P2 - 中优先级问题

#### [P2-001] 仪表板信息密度过高

**改造方案**: 添加可折叠分组和视图切换

```typescript
// 简洁视图 vs 完整视图
type DashboardView = 'compact' | 'full'

// Elder 主题默认使用简洁视图
const defaultView = theme === 'elder' ? 'compact' : 'full'
```

**工作量**: 4h

---

#### [P2-002] 表格操作缺少图标

**修改文件**: `frontend/src/components/common/table/DataTable.tsx`

```typescript
// 为每个操作添加图标
const actionIcons: Record<string, React.ReactNode> = {
  view: <IconEye />,
  edit: <IconEdit />,
  delete: <IconDelete />,
  adjust: <IconRefresh />,
  setThreshold: <IconSetting />,
}

// 修改 actions 渲染
{actions.map(action => (
  <Dropdown.Item
    key={action.key}
    icon={action.icon || actionIcons[action.key]}
  >
    {action.label}
  </Dropdown.Item>
))}
```

**工作量**: 2h

---

#### [P2-003] 缺少面包屑导航

**当前状态**: Header 中已有 Breadcrumb 组件，需确认是否正确显示

**检查要点**:
- 确认所有嵌套路由都有正确的 breadcrumb 配置
- 确认面包屑在移动端的显示

**工作量**: 2h

---

#### [P2-004] 登录页面缺少辅助功能

**修改文件**: `frontend/src/pages/Login.tsx`

```typescript
// 添加显示/隐藏密码
const [showPassword, setShowPassword] = useState(false)
const [capsLockOn, setCapsLockOn] = useState(false)

<Input
  type={showPassword ? 'text' : 'password'}
  suffix={
    <Button
      icon={showPassword ? <IconEyeOff /> : <IconEye />}
      theme="borderless"
      onClick={() => setShowPassword(!showPassword)}
      aria-label={showPassword ? '隐藏密码' : '显示密码'}
    />
  }
  onKeyDown={(e) => setCapsLockOn(e.getModifierState('CapsLock'))}
/>

{capsLockOn && (
  <div className="login__caps-warning" role="alert">
    <IconAlertTriangle /> 大写锁定已开启
  </div>
)}

// 记住我选项
<Checkbox
  checked={rememberMe}
  onChange={setRememberMe}
>
  记住我（7天内免登录）
</Checkbox>
```

**工作量**: 3h

---

#### [P2-005] 缺少全局 ErrorBoundary

**新增文件**: `frontend/src/components/common/ErrorBoundary.tsx`

```typescript
interface ErrorBoundaryState {
  hasError: boolean
  error?: Error
  errorInfo?: ErrorInfo
}

export class ErrorBoundary extends Component<Props, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // 上报错误
    reportErrorToServer(error, errorInfo)
    this.setState({ errorInfo })
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="error-boundary">
          <div className="error-boundary__content">
            <IconAlertTriangle size="extra-large" />
            <h1>页面出现问题</h1>
            <p>抱歉，页面遇到了一些问题。请尝试刷新页面。</p>

            <div className="error-boundary__actions">
              <Button
                theme="solid"
                onClick={() => window.location.reload()}
              >
                刷新页面
              </Button>
              <Button
                theme="light"
                onClick={() => window.location.href = '/'}
              >
                返回首页
              </Button>
            </div>

            {/* 仅开发环境显示详细错误 */}
            {import.meta.env.DEV && (
              <details className="error-boundary__details">
                <summary>技术详情</summary>
                <pre>{this.state.error?.stack}</pre>
              </details>
            )}

            <p className="error-boundary__help">
              如问题持续，请联系技术支持
            </p>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
```

**工作量**: 4h

---

#### [P2-006] 缺少操作确认弹窗规范

**新增组件**: `frontend/src/components/common/ConfirmDialog.tsx`

```typescript
interface ConfirmDialogProps {
  type: 'info' | 'warning' | 'danger'
  title: string
  content: React.ReactNode
  confirmText?: string
  cancelText?: string
  onConfirm: () => void | Promise<void>
  onCancel?: () => void

  // 危险操作需要输入确认
  dangerConfirmText?: string  // 如 "删除"，用户需要输入才能确认
}

/**
 * 统一的确认弹窗组件
 *
 * 使用示例:
 * confirm({
 *   type: 'danger',
 *   title: '确认删除',
 *   content: '删除后数据将无法恢复',
 *   dangerConfirmText: '删除',
 *   onConfirm: () => deleteItem(id),
 * })
 */
```

**工作量**: 4h

---

### 2.4 P3 - 低优先级问题

#### [P3-001] 警告色对比度不足

**修改文件**: `frontend/src/styles/tokens/colors.css`

```css
/* 文字专用的警告色（更深） */
--color-warning-text: #ad6800;  /* 替代 #faad14 用于文字 */
```

**工作量**: 1h

---

#### [P3-002] Logo 缺少无障碍属性

**修改文件**: `frontend/src/components/layout/Sidebar.tsx`

```typescript
<div className="sidebar__logo" role="img" aria-label="ERP 系统">
  <div className="sidebar__logo-icon" aria-hidden="true">
    <IconGridView size="large" />
  </div>
  {!sidebarCollapsed && <span className="sidebar__logo-text">ERP System</span>}
</div>
```

**工作量**: 0.5h

---

#### [P3-003] 表单错误未通知屏幕阅读器

**修改文件**: `frontend/src/components/common/form/FormFieldWrapper.tsx`

```typescript
{hasError && (
  <span
    className="form-field-error"
    role="alert"
    aria-live="polite"
  >
    {error}
  </span>
)}
```

**工作量**: 1h

---

#### [P3-004] 缺少页面加载骨架屏

**新增组件**: `frontend/src/components/common/Skeleton.tsx`

```typescript
// 通用骨架屏组件
export const TableSkeleton = ({ rows = 5 }) => (
  <div className="skeleton-table">
    <div className="skeleton-table__header" />
    {Array(rows).fill(0).map((_, i) => (
      <div key={i} className="skeleton-table__row" />
    ))}
  </div>
)

export const CardSkeleton = () => (
  <div className="skeleton-card">
    <div className="skeleton-card__header" />
    <div className="skeleton-card__content" />
  </div>
)
```

**工作量**: 3h

---

## 3. 改造计划

### 3.1 第一阶段：基础修复（第1周）

| 任务 | 优先级 | 工作量 | 负责人 |
|------|--------|--------|--------|
| Elder 主题 bug 修复 | P0 | 0.5h | - |
| 统一错误处理服务 | P1 | 8h | - |
| 表单验证消息增强 | P1 | 6h | - |

**里程碑**: 错误提示用户友好，主题切换正常

---

### 3.2 第二阶段：移动端优化（第2周）

| 任务 | 优先级 | 工作量 | 负责人 |
|------|--------|--------|--------|
| 移动端导航改造 | P1 | 6h | - |
| 全局 ErrorBoundary | P2 | 4h | - |
| 登录页辅助功能 | P2 | 3h | - |

**里程碑**: 移动端可正常使用

---

### 3.3 第三阶段：角色定制化（第3周）

| 任务 | 优先级 | 工作量 | 负责人 |
|------|--------|--------|--------|
| 角色定制化仪表板 | P1 | 16h | - |
| 仪表板视图切换 | P2 | 4h | - |

**里程碑**: 不同角色看到定制化内容

---

### 3.4 第四阶段：效率提升（第4周）

| 任务 | 优先级 | 工作量 | 负责人 |
|------|--------|--------|--------|
| 快捷操作 FAB | P1 | 8h | - |
| 全局搜索 Ctrl+K | P1 | 4h | - |
| 表格操作图标 | P2 | 2h | - |
| 确认弹窗规范 | P2 | 4h | - |

**里程碑**: 操作效率提升

---

### 3.5 第五阶段：细节完善（第5周）

| 任务 | 优先级 | 工作量 | 负责人 |
|------|--------|--------|--------|
| 面包屑检查完善 | P2 | 2h | - |
| 骨架屏组件 | P3 | 3h | - |
| 警告色对比度 | P3 | 1h | - |
| Logo 无障碍 | P3 | 0.5h | - |
| 表单错误 ARIA | P3 | 1h | - |

**里程碑**: 无障碍合规，细节完善

---

## 4. 实施路线图

```
第1周          第2周          第3周          第4周          第5周
 │              │              │              │              │
 ├─ P0修复      ├─ 移动端      ├─ 角色定制    ├─ 快捷操作    ├─ 细节完善
 ├─ 错误处理    ├─ 错误边界    ├─ 仪表板      ├─ 全局搜索    ├─ 骨架屏
 └─ 表单验证    └─ 登录优化    └─ 视图切换    └─ 表格图标    └─ 无障碍
                                              └─ 确认弹窗

总工作量：约 72 小时（9人日）
```

---

## 5. 验收标准

### 5.1 功能验收

| 检查项 | 验收标准 |
|--------|----------|
| Elder 主题 | 切换后刷新页面仍保持 elder 主题 |
| 错误提示 | 所有错误显示中文友好提示，无技术术语 |
| 移动导航 | 768px 以下显示汉堡菜单，侧边栏可滑动关闭 |
| 角色仪表板 | 不同角色登录后看到不同的卡片组合 |
| 快捷操作 | Ctrl+K 打开全局搜索，FAB 可快速创建订单/产品 |

### 5.2 无障碍验收

| 检查项 | 验收标准 |
|--------|----------|
| 键盘导航 | Tab 键可遍历所有交互元素 |
| 焦点可见 | 所有可交互元素有明显的焦点状态 |
| 屏幕阅读器 | 错误消息可被 VoiceOver/NVDA 朗读 |
| 对比度 | 所有文字对比度 >= 4.5:1 (WCAG AA) |

### 5.3 性能验收

| 检查项 | 验收标准 |
|--------|----------|
| 首屏加载 | Dashboard 加载时间 < 2s |
| 交互响应 | 按钮点击反馈 < 100ms |
| 动画流畅 | 侧边栏动画 60fps |

### 5.4 用户测试

| 用户群 | 测试场景 |
|--------|----------|
| 销售人员 | 使用手机创建订单完整流程 |
| 仓库人员 | 使用平板处理库存预警 |
| 老年用户 | 使用 Elder 主题完成登录和查看报表 |
| 新用户 | 首次使用系统完成基本操作 |

---

## 附录

### A. 国际化键值清单

需要添加到 `frontend/src/locales/zh-CN/common.json`:

```json
{
  "error": {
    "network": {
      "title": "网络连接失败",
      "message": "请检查您的网络连接后重试",
      "action": "重试"
    },
    "auth": {
      "title": "登录已过期",
      "message": "请重新登录以继续操作",
      "action": "重新登录"
    },
    "permission": {
      "title": "权限不足",
      "message": "您没有执行此操作的权限，请联系管理员"
    },
    "validation": {
      "title": "信息填写有误",
      "message": "请检查标红的字段并修正"
    },
    "notFound": {
      "title": "内容不存在",
      "message": "您访问的内容可能已被删除或移动",
      "action": "返回首页"
    },
    "conflict": {
      "title": "数据冲突",
      "message": "该数据已被其他用户修改，请刷新后重试",
      "action": "刷新页面"
    },
    "rateLimit": {
      "title": "操作过于频繁",
      "message": "请稍后再试"
    },
    "server": {
      "title": "服务器繁忙",
      "message": "系统正在维护中，请稍后重试",
      "action": "重试",
      "help": "如问题持续，请联系客服"
    },
    "unknown": {
      "title": "操作失败",
      "message": "发生了一个错误，请稍后重试",
      "action": "重试"
    }
  },
  "quickAction": {
    "newSalesOrder": "新建销售订单",
    "newProduct": "新建产品",
    "newCustomer": "新建客户",
    "globalSearch": "全局搜索"
  },
  "login": {
    "capsLockWarning": "大写锁定已开启",
    "showPassword": "显示密码",
    "hidePassword": "隐藏密码",
    "rememberMe": "记住我（7天内免登录）"
  },
  "errorBoundary": {
    "title": "页面出现问题",
    "message": "抱歉，页面遇到了一些问题。请尝试刷新页面。",
    "refresh": "刷新页面",
    "goHome": "返回首页",
    "technicalDetails": "技术详情",
    "helpText": "如问题持续，请联系技术支持"
  }
}
```

### B. 相关文件索引

| 文件路径 | 说明 |
|----------|------|
| `frontend/src/hooks/useTheme.ts` | 主题管理 Hook |
| `frontend/src/services/axios-instance.ts` | Axios 拦截器 |
| `frontend/src/components/layout/Sidebar.tsx` | 侧边栏组件 |
| `frontend/src/components/layout/Header.tsx` | 顶部栏组件 |
| `frontend/src/pages/Dashboard.tsx` | 仪表板页面 |
| `frontend/src/pages/Login.tsx` | 登录页面 |
| `frontend/src/components/common/form/` | 表单组件目录 |
| `frontend/src/components/common/table/` | 表格组件目录 |
| `frontend/src/locales/` | 国际化资源目录 |
| `frontend/src/styles/tokens/` | 设计 Token 目录 |

---

**文档结束**
