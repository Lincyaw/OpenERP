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

### 3.1 Backend

| Category | Technology |
|------|----------|
| Language | Go |
| Web Framework | Gin |
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

### 3.2 Frontend

| Category | Technology |
|------|----------|
| Framework | React |
| Language | TypeScript |
| UI Component Library | Semi Design @douyinfe/semi-ui |
| State Management | Zustand |
| Routing | React Router |
| HTTP Client | Axios |
| Forms | React Hook Form |
| Charts | ECharts / @visactor/vchart |
| Build Tool | Vite |
| Testing | Vitest + React Testing Library |
| E2E Testing | Playwright |

### 3.3 Semi Design Installation

```bash
npm install @douyinfe/semi-ui
npm install @douyinfe/semi-icons
```

Use `semi-ui-skills` to write better frontend code with Semi Design.

---

## 4. Docker Compose Configuration

### 4.1 Two Usage Modes

This project supports two development/deployment modes:

| Mode | Description | Use Case |
|------|-------------|----------|
| **Docker Mode** | All services run in Docker containers | Quick start, CI/CD, demos |
| **Local Dev Mode** | Database in Docker, frontend/backend run locally | Daily development, debugging |

### 4.2 Configuration Files

| File | Purpose |
|------|---------|
| `.env.example` | Configuration template (copy to `.env`) |
| `.env` | Your local configuration (gitignored) |
| `backend/config.toml` | Application default configuration |

### 4.3 Unified Port Configuration

| Service | Port |
|---------|------|
| Backend | 8080 |
| Frontend | 3000 |
| PostgreSQL | 5432 |
| Redis | 6379 |

### 4.4 Quick Start

**First-time setup:**
```bash
make setup
```

**Docker Mode (all services in containers):**
```bash
make docker-up      # Start all services
# Access: http://localhost:3000
# Login:  admin / admin123
make docker-down    # Stop all services
```

**Local Development Mode:**
```bash
make dev            # Start database (postgres + redis)
make dev-backend    # Terminal 1: Run backend locally
make dev-frontend   # Terminal 2: Run frontend locally
make dev-stop       # Stop database
```

### 4.5 Environment Variable Overrides

Override any `backend/config.toml` value using `ERP_` prefix:

```bash
ERP_DATABASE_PASSWORD=secret
ERP_JWT_SECRET=my-secret-key
ERP_LOG_LEVEL=debug
```

**Mapping:** `ERP_DATABASE_HOST` → `[database] host` in TOML

---

## 5. API Contract & Code Generation

### 5.1 Overview

This project uses **OpenAPI 3.0** specification as the single source of truth for API contracts. Frontend TypeScript SDK is auto-generated from the OpenAPI spec, reducing maintenance effort and ensuring type safety.

### 5.2 Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Spec Generation | swaggo/swag | Generate OpenAPI from Go annotations |
| Spec Location | `backend/docs/openapi.yaml` | Single source of truth |
| Client Generator | orval | Generate TypeScript axios client |
| Generated SDK | `frontend/src/api/` | Auto-generated, DO NOT edit manually |

### 5.3 Backend: Writing OpenAPI Annotations

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

### 5.4 Backend: Generating OpenAPI Spec

```bash
# Generate OpenAPI spec from annotations
make api-docs

# Output files:
# - backend/docs/swagger.yaml (OpenAPI spec)
# - backend/docs/swagger.json (OpenAPI spec)
# - backend/docs/docs.go (Go embed file)
```

### 5.5 Frontend: Generating TypeScript SDK

```bash
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

### 5.6 Workflow

1. **Backend developer** adds/modifies API handler with swag annotations
2. **Run** `make api-docs` to regenerate OpenAPI spec
3. **Run** `npm run api:generate` in frontend to regenerate SDK
4. **Frontend developer** uses typed SDK with full autocomplete

### 5.7 Rules

- **NEVER manually edit** files in `frontend/src/api/` - they are auto-generated
- **ALWAYS regenerate SDK** after backend API changes
- **ALWAYS include** all swag annotations for new handlers
- **CI should verify** OpenAPI spec is up-to-date with code
- **Review generated SDK** in PR to catch breaking changes

---

## 6. E2E Testing Specification

### 6.1 Test Environment

All E2E tests use unified ports:

| Component | Port |
|-----------|------|
| Frontend | 3000 |
| Backend | 8080 |
| PostgreSQL | 5432 |
| Redis | 6379 |

### 6.2 E2E Testing Standards

1. **No Mocking**: Integration tests must connect to real database, API responses must NOT be mocked
2. **Data Isolation**: Each test case should have independent test data to avoid side effects
3. **Complete Flow**: Tests must cover the full workflow from UI operations to database changes
4. **Screenshots/Videos**: Failed test cases must automatically capture screenshots, key flows should be recorded
5. **Multi-Browser**: Must cover at least Chrome and Firefox
6. **Responsive**: Test both desktop and mobile viewports

### 6.3 Integration Test Acceptance Criteria

A completed integration (INT) task must satisfy:
- [ ] Docker environment starts successfully
- [ ] Seed data (`seed-data.sql`) loads completely
- [ ] Playwright E2E tests pass at 100% rate
- [ ] Tests cover all scenarios described in requirements
- [ ] HTML test report is generated
- [ ] No flaky tests (stable pass after 3 consecutive runs)

### 6.4 Test Commands

```bash
# Run E2E tests (resets environment, runs all tests)
make e2e

# Run E2E tests with Playwright UI (requires local services)
make e2e-ui

# Run E2E tests in debug mode
make e2e-debug

# Run specific test file
make e2e ARGS="tests/e2e/auth/auth.spec.ts"

# Run only Chromium browser
make e2e ARGS="--project=chromium"

# Run tests against locally running services
make e2e-local
```

### 6.5 Database Operations

```bash
# Load seed data
make db-seed

# Reset database (clean + migrate + seed)
make db-reset

# Open psql shell
make db-psql
```

**Notes**:
- E2E tests automatically reset the database before running
- Default: 16 parallel workers
- CI mode (CI=true): 2 workers with 2 retries on failure
