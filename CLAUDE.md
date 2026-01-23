## 1. Project Overview

### 1.1 Project Introduction

An inventory management system based on DDD (Domain-Driven Design), adopting a modularized monolithic architecture with multi-tenancy support.


- Detailed design specification: `.claude/ralph/docs/spec.md`
- Project progress planning: `.claude/ralph/docs/task.md`

---

## 2. Technology Stack

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
