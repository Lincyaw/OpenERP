## 1. Project Overview

### 1.1 Project Introduction

An inventory management system based on DDD (Domain-Driven Design), adopting a modularized monolithic architecture with multi-tenancy support.


- Detailed design specification: `.claude/ralph/docs/spec.md`
- Project progress planning: `.claude/ralph/docs/task.md`
- **Frontend Design System**: `frontend/README.md` - **MUST READ before developing frontend**

---

## 2. Frontend Development Guidelines

> **CRITICAL**: Before implementing ANY frontend code, you MUST read and follow the design system documentation in `frontend/README.md`.

### Key Requirements

1. **Use Design Tokens**: Always use CSS variables (e.g., `--spacing-4`, `--color-primary`) instead of hardcoded values
2. **Responsive Design**: Use mobile-first approach with breakpoints (375px, 768px, 1024px, 1440px)
3. **Accessibility**: Follow WCAG 2.1 AA guidelines (contrast, focus, keyboard navigation)
4. **Theme Support**: Components must work in light, dark, and elder-friendly themes
5. **Layout Components**: Use `Container`, `Grid`, `Flex`, `Stack`, `Row` from `@/components/common`

---

## 3. Technology Stack

All technology stacks use the latest stable versions

### 2.1 Backend

| Category | Technology |
|------|----------|----------|
| Language | Go |
| Web Framework | Gin | v
| ORM | GORM | 
| Database | PostgreSQL | 
| Cache | Redis |
| Message Queue | Redis Stream |
| Validation | go-playground/validator |
| JWT | golang-jwt/jwt |
| Logging | zap | 
| Configuration | viper |
| Migration | golang-migrate |
| Testing | testify | 

### 2.2 Frontend

| Category | Technology | 
|------|----------|
| Framework | React | 
| Language | TypeScript |
| UI Component Library | Semi Design  @douyinfe/semi-ui |
| State Management | Zustand |
| Routing | React Router | 
| HTTP Client | Axios |
| Forms | React Hook Form |
| Charts | ECharts / @visactor/vchart |
| Build Tool | Vite | 
| Testing | Vitest + React Testing Library | 
| E2E Testing | Playwright | 

### 2.3 Semi Design Installation

```bash
npm install @douyinfe/semi-ui

npm install @douyinfe/semi-icons
```

---

## 3. API Contract & Code Generation

### 3.1 Overview

This project uses **OpenAPI 3.0** specification as the single source of truth for API contracts. Frontend TypeScript SDK is auto-generated from the OpenAPI spec, reducing maintenance effort and ensuring type safety.

### 3.2 Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Spec Generation | swaggo/swag | Generate OpenAPI from Go annotations |
| Spec Location | `backend/docs/openapi.yaml` | Single source of truth |
| Client Generator | orval | Generate TypeScript axios client |
| Generated SDK | `frontend/src/api/` | Auto-generated, DO NOT edit manually |

### 3.3 Backend: Writing OpenAPI Annotations

Every HTTP handler MUST include swag annotations:

```go
// CreateProduct godoc
// @Summary      Create a new product
// @Description  Create a new product in the catalog
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateProductRequest true "Product creation request"
// @Success      201 {object} dto.Response{data=dto.ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /products [post]
func (h *ProductHandler) Create(c *gin.Context) {
    // implementation
}
```

**Required annotations:**
- `@Summary` - Brief description (shown in UI)
- `@Description` - Detailed description
- `@Tags` - API grouping (products, customers, orders, etc.)
- `@Accept` / `@Produce` - Content types (usually json)
- `@Param` - Request parameters (path, query, body)
- `@Success` / `@Failure` - Response schemas with status codes
- `@Security` - Authentication requirement
- `@Router` - HTTP method and path

### 3.4 Backend: Generating OpenAPI Spec

```bash
# Install swag CLI (one-time)
go install github.com/swaggo/swag/cmd/swag@latest

# Generate OpenAPI spec from annotations
cd backend
swag init -g cmd/server/main.go -o docs --outputTypes yaml,json

# Output files:
# - docs/swagger.yaml (OpenAPI spec)
# - docs/swagger.json (OpenAPI spec)
# - docs/docs.go (Go embed file)
```

Add to `backend/Makefile`:
```makefile
.PHONY: docs
docs:
	swag init -g cmd/server/main.go -o docs --outputTypes yaml,json
```

### 3.5 Frontend: Generating TypeScript SDK

```bash
# Install orval (one-time)
npm install -D orval

# Generate SDK from OpenAPI spec
cd frontend
npx orval

# Output: src/api/ directory with typed axios client
```

**orval.config.ts** (frontend root):
```typescript
import { defineConfig } from 'orval'

export default defineConfig({
  erp: {
    input: '../backend/docs/swagger.yaml',
    output: {
      mode: 'tags-split',
      target: './src/api',
      schemas: './src/api/models',
      client: 'axios',
      override: {
        mutator: {
          path: './src/services/axios-instance.ts',
          name: 'axiosInstance',
        },
      },
    },
  },
})
```

Add to `frontend/package.json`:
```json
{
  "scripts": {
    "api:generate": "orval",
    "api:watch": "orval --watch"
  }
}
```

### 3.6 Workflow

1. **Backend developer** adds/modifies API handler with swag annotations
2. **Run** `make docs` in backend to regenerate OpenAPI spec
3. **Run** `npm run api:generate` in frontend to regenerate SDK
4. **Frontend developer** uses typed SDK with full autocomplete

### 3.7 Rules

- **NEVER manually edit** files in `frontend/src/api/` - they are auto-generated
- **ALWAYS regenerate SDK** after backend API changes
- **ALWAYS include** all swag annotations for new handlers
- **CI should verify** OpenAPI spec is up-to-date with code
- **Review generated SDK** in PR to catch breaking changes

---

## 4. E2E 测试规范

### 4.1 测试环境要求

所有联调（INT）和端到端测试**必须**在 Docker 环境下执行，连接真实数据库：

| 组件 | 端口 | 说明 |
|------|------|------|
| Frontend | 3001 | 前端应用 |
| Backend | 8081 | 后端 API |
| PostgreSQL | 5433 | 真实数据库 |
| Redis | 6380 | 缓存服务 |

```bash
# 启动测试环境
./docker/quick-test.sh start

# 加载测试数据
./docker/quick-test.sh seed

# 运行 E2E 测试
cd frontend && npm run e2e

# 清理环境
./docker/quick-test.sh clean
```

### 4.2 E2E 测试标准

1. **禁止 Mock**: 联调测试必须连接真实数据库，不允许 Mock API 响应
2. **数据隔离**: 每个测试用例应有独立的测试数据，避免相互影响
3. **完整流程**: 测试必须覆盖从 UI 操作到数据库变更的完整链路
4. **截图/视频**: 失败用例必须自动截图，关键流程录制视频
5. **多浏览器**: 至少覆盖 Chrome 和 Firefox
6. **响应式**: 测试桌面和移动端视口

### 4.3 联调测试验收标准

一个联调（INT）任务完成必须满足：
- [ ] Docker 环境 (`docker-compose.test.yml`) 启动成功
- [ ] Seed 数据 (`seed-data.sql`) 加载完成
- [ ] Playwright E2E 测试通过率 100%
- [ ] 测试覆盖所有 requirements 中描述的场景
- [ ] HTML 测试报告生成
- [ ] 无 Flaky 测试（连续运行 3 次稳定通过）

### 4.4 测试命令

```bash
# 单次运行所有 E2E 测试
npm run e2e

# 带浏览器界面运行
npm run e2e:headed

# 调试模式
npm run e2e:debug

# 生成并打开 HTML 报告
npm run e2e:report

# 指定浏览器
npm run e2e -- --project=chromium
npm run e2e -- --project=firefox

# CI 环境运行
npm run e2e:ci
```

### 4.5 测试凭证

测试环境使用 `seed-data.sql` 预置用户：

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin | test123 | 系统管理员 |
| sales | test123 | 销售经理 |
| warehouse | test123 | 仓库管理员 |
| finance | test123 | 财务经理 |

### 4.6 测试目录结构

```
frontend/tests/e2e/
├── auth/           # 认证相关测试
├── products/       # 商品模块测试
├── partners/       # 伙伴模块测试
├── inventory/      # 库存模块测试
├── transactions/   # 交易模块测试
├── finance/        # 财务模块测试
├── reports/        # 报表模块测试
├── settings/       # 设置模块测试
├── pages/          # Page Object 类
├── fixtures/       # 测试 fixtures
└── utils/          # 测试工具函数
```
