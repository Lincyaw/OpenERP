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

Use `semi-ui-skills` to write better frontend code with Semi Design, you can use this skill.


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

## 4. E2E Testing Specification

### 4.1 Test Environment Requirements

All integration (INT) and end-to-end tests **MUST** be executed in Docker environment with connection to real database:

| Component | Port | Description |
|------|------|------|
| Frontend | 3001 | Frontend application |
| Backend | 8081 | Backend API |
| PostgreSQL | 5433 | Real database |
| Redis | 6380 | Cache service |

See [4.4 Test Commands](#44-test-commands) for specific test commands.

### 4.2 E2E Testing Standards

1. **No Mocking**: Integration tests must connect to real database, API responses must NOT be mocked
2. **Data Isolation**: Each test case should have independent test data to avoid side effects
3. **Complete Flow**: Tests must cover the full workflow from UI operations to database changes
4. **Screenshots/Videos**: Failed test cases must automatically capture screenshots, key flows should be recorded
5. **Multi-Browser**: Must cover at least Chrome and Firefox
6. **Responsive**: Test both desktop and mobile viewports

### 4.3 Integration Test Acceptance Criteria

A completed integration (INT) task must satisfy:
- [ ] Docker environment (`docker-compose.test.yml`) starts successfully
- [ ] Seed data (`seed-data.sql`) loads completely
- [ ] Playwright E2E tests pass at 100% rate
- [ ] Tests cover all scenarios described in requirements
- [ ] HTML test report is generated
- [ ] No flaky tests (stable pass after 3 consecutive runs)

### 4.4 Test Commands

#### Convenience Scripts

```bash
# Start test environment (includes health checks, migration, seed, API checks)
./docker/quick-test.sh start

# Load seed data only
./docker/quick-test.sh seed

# View service status
./docker/quick-test.sh status

# View logs
./docker/quick-test.sh logs

# API smoke test
./docker/quick-test.sh api

# Stop test environment
./docker/quick-test.sh stop

# Stop and clean all data
./docker/quick-test.sh clean
```

#### Running E2E Tests

```bash
# 1. Start test environment
docker compose -f docker-compose.test.yml up -d

# 2. Load seed data
./docker/quick-test.sh seed

# 3. Run tests (execute from project root)
docker compose -f docker-compose.test.yml run --rm \
    --user "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e E2E_BASE_URL=http://frontend:80 \
    playwright npx playwright test --reporter=list

# Run specific test file
docker compose -f docker-compose.test.yml run --rm \
    --user "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e E2E_BASE_URL=http://frontend:80 \
    playwright npx playwright test tests/e2e/auth/auth.spec.ts --project=chromium --reporter=list

# 4. Stop test environment
docker compose -f docker-compose.test.yml down -v
```

**Notes**:
- By default uses 16 workers to run in parallel
- Add `-e CI=true` to switch to CI mode (2 workers, retry 2 times on failure)


