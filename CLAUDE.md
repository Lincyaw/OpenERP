## 1. Project Overview

### 1.1 Project Introduction

An inventory management system based on DDD (Domain-Driven Design), adopting a modularized monolithic architecture with multi-tenancy support.


- Detailed design specification: `spec.md`
- Project progress planning: `progress.md`

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
