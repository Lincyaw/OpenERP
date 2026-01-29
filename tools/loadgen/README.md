# ERP Load Generator

A circuit-based API load testing tool for the ERP system. The load generator treats API endpoints as circuit components, automatically connecting them through semantic parameter pools to simulate realistic user traffic.

## Quick Start

```bash
# Build the load generator
make loadgen-build

# Run with default configuration (5m, 100 QPS)
make loadgen-run

# Validate configuration only
make loadgen-validate

# See execution plan without running
make loadgen-dry-run
```

## Installation

The load generator is built as a standalone Go binary:

```bash
# From project root
make loadgen-build

# Binary location
./tools/loadgen/bin/loadgen --version
```

## Usage

### Basic Commands

```bash
# Run with config file
loadgen -config configs/erp.yaml

# Override duration and QPS
loadgen -config configs/erp.yaml -duration 10m -qps 50

# Verbose output
loadgen -config configs/erp.yaml -v

# List all configured endpoints
loadgen -config configs/erp.yaml -list

# Validate configuration
loadgen -config configs/erp.yaml -validate

# Dry run (show execution plan)
loadgen -config configs/erp.yaml -dry-run
```

### CLI Parameters

| Parameter | Short | Description | Default |
|-----------|-------|-------------|---------|
| `-config` | `-c` | Path to YAML configuration file | Required |
| `-duration` | `-d` | Override test duration | From config |
| `-concurrency` | | Override max worker pool size | From config |
| `-qps` | | Override base QPS | From config |
| `-verbose` | `-v` | Enable verbose output | false |
| `-list` | `-l` | List all endpoints | false |
| `-validate` | | Validate config and exit | false |
| `-dry-run` | | Show execution plan | false |
| `-version` | | Show version info | false |
| `-output` | | Output format: console, json | console |
| `-output-file` | | JSON output file path | Auto-generated |
| `-prometheus` | | Prometheus metrics endpoint | Disabled |

### OpenAPI Parsing

The load generator can parse OpenAPI/Swagger specs:

```bash
# List endpoints from OpenAPI spec
loadgen -openapi backend/docs/swagger.yaml -list

# Verbose mode (show parameters)
loadgen -openapi backend/docs/swagger.yaml -list -v

# Run semantic type inference
loadgen -openapi backend/docs/swagger.yaml -infer -v
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make loadgen-build` | Build the load generator binary |
| `make loadgen-run` | Run with default ERP config (5m, 100 QPS) |
| `make loadgen-stress` | Run stress test (30m, 500 QPS ramp-up) |
| `make loadgen-scenario SCENARIO=name` | Run specific scenario |
| `make loadgen-dry-run` | Show execution plan without running |
| `make loadgen-list` | List all configured endpoints |
| `make loadgen-validate` | Validate configuration |
| `make loadgen-test` | Run unit tests |
| `make loadgen-clean` | Remove build artifacts |

### Running Scenarios

```bash
# Available scenarios
make loadgen-scenario SCENARIO=browse_catalog
make loadgen-scenario SCENARIO=create_sales_order
make loadgen-scenario SCENARIO=create_purchase_order
make loadgen-scenario SCENARIO=check_inventory
make loadgen-scenario SCENARIO=review_finances
make loadgen-scenario SCENARIO=view_reports
```

## Configuration

### Configuration File Structure

The configuration file (`configs/erp.yaml`) defines:

```yaml
name: "ERP Load Test"
version: "1.0"

# Target system
target:
  baseURL: "http://localhost:8080"
  apiVersion: "v1"
  timeout: 30s

# Authentication
auth:
  type: "bearer"
  login:
    endpoint: "/auth/login"
    method: "POST"
    username: "admin"
    password: "admin123"
    tokenPath: "$.data.token.access_token"

# Test duration
duration: 5m

# Traffic shaping
trafficShaper:
  type: "step"  # constant, step, sine, spike, custom
  baseQPS: 10
  step:
    steps:
      - qps: 10
        duration: 30s
        rampDuration: 10s
      - qps: 50
        duration: 60s
        rampDuration: 20s

# Rate limiting
rateLimiter:
  type: "token_bucket"
  qps: 100
  burstSize: 50

# Worker pool
workerPool:
  minSize: 5
  maxSize: 100
  initialSize: 10

# Backpressure handling
backpressure:
  strategy: "reduce"
  errorRateThreshold: 0.1
  latencyP99Threshold: 1s

# Warmup phase
warmup:
  iterations: 10
  fill:
    - "entity.warehouse.id"
    - "entity.category.id"
    - "entity.product.id"

# Endpoints
endpoints:
  - name: "catalog.products.list"
    path: "/catalog/products"
    method: "GET"
    weight: 20
    tags: ["catalog", "read"]
```

### Traffic Shaping Types

#### Constant
```yaml
trafficShaper:
  type: "constant"
  baseQPS: 100
```

#### Step (Ramp-up)
```yaml
trafficShaper:
  type: "step"
  baseQPS: 10
  step:
    steps:
      - qps: 10
        duration: 30s
      - qps: 50
        duration: 60s
      - qps: 100
        duration: 120s
```

#### Sine Wave
```yaml
trafficShaper:
  type: "sine"
  baseQPS: 100
  amplitude: 0.5    # +/- 50% variation
  period: 60s
```

#### Spike
```yaml
trafficShaper:
  type: "spike"
  baseQPS: 50
  spike:
    spikeQPS: 500
    spikeDuration: 30s
    spikeInterval: 300s
```

### Endpoint Configuration

```yaml
endpoints:
  - name: "catalog.products.create"
    description: "Create a new product"
    path: "/catalog/products"
    method: "POST"
    weight: 3
    tags: ["catalog", "write", "producer"]
    body: |
      {
        "name": "Product-{{.random.string:8}}",
        "code": "SKU-{{.sequence:8}}",
        "category_id": "{{.entity.category.id}}"
      }
    expectedStatus: 201
    consumes:
      - "entity.category.id"
    produces:
      - semanticType: "entity.product.id"
        jsonPath: "$.data.id"
```

### Semantic Types

Semantic types enable automatic parameter passing between endpoints:

| Category | Types |
|----------|-------|
| Entities | `entity.customer.id`, `entity.product.id`, `entity.warehouse.id`, etc. |
| References | `ref.product.code`, `ref.sales_order.number`, etc. |
| Tokens | `token.access`, `token.refresh` |

### Data Generators

Template variables for dynamic data:

| Generator | Example | Description |
|-----------|---------|-------------|
| `{{.random.string:N}}` | `{{.random.string:8}}` | Random alphanumeric string |
| `{{.sequence:N}}` | `{{.sequence:6}}` | Sequential number, zero-padded |
| `{{.random.float:MIN:MAX}}` | `{{.random.float:10:1000}}` | Random float in range |
| `{{.faker.name}}` | | Random person name |
| `{{.faker.email}}` | | Random email address |
| `{{.faker.phone}}` | | Random phone number |
| `{{.entity.TYPE}}` | `{{.entity.customer.id}}` | Value from parameter pool |

## Workflows

Workflows define complete business process sequences:

```yaml
workflows:
  sales_cycle:
    description: "Complete sales order lifecycle"
    weight: 10
    timeout: 60s
    steps:
      - name: create_order
        endpoint: "POST /trade/sales-orders"
        body: |
          {
            "customer_id": "{customer_id}",
            "warehouse_id": "{warehouse_id}"
          }
        extract:
          order_id: "$.data.id"

      - name: add_item
        endpoint: "POST /trade/sales-orders/{order_id}/items"
        body: |
          {
            "product_id": "{product_id}",
            "quantity": 5
          }

      - name: confirm_order
        endpoint: "POST /trade/sales-orders/{order_id}/confirm"
```

## Output & Metrics

### Console Output

Real-time metrics displayed during test execution:

```
[01:00] ████████████████████  3,512 req | 113.9 QPS | 98.5% ok | p95: 43ms | shape: ▲ peak
```

### JSON Reports

```bash
loadgen -config configs/erp.yaml -output json -output-file results/test-{{.Timestamp}}.json
```

### Prometheus Metrics

```bash
# Enable Prometheus endpoint
loadgen -config configs/erp.yaml -prometheus :9090

# Metrics available at http://localhost:9090/metrics
```

Available metrics:
- `loadgen_requests_total{endpoint, status}` - Request counter
- `loadgen_request_duration_seconds{endpoint}` - Latency histogram
- `loadgen_current_qps` - Current queries per second
- `loadgen_pool_size{semantic}` - Parameter pool size

## SLO Assertions

Define performance assertions:

```yaml
assertions:
  global:
    maxErrorRate: 0.01        # 1% max error rate
    maxP95Latency: 200ms
    minSuccessRate: 0.99
  exitOnFailure: true

  endpoints:
    "POST /trade/sales-orders":
      maxP99Latency: 500ms
      maxErrorRate: 0.001
```

Exit codes:
- `0` - All assertions passed
- `2` - Assertion failure

## Architecture

### Circuit Board Metaphor

```
API Endpoint = Chip
├── Input Pins  = Request parameters
└── Output Pins = Response fields

Parameter Pool = Wire Bus
└── Connects outputs to inputs by semantic type

Load Generator = Circuit Board
└── Orchestrates execution, collects metrics
```

### Key Components

| Component | Description |
|-----------|-------------|
| `config/` | Configuration parsing and validation |
| `parser/` | OpenAPI parsing and semantic inference |
| `circuit/` | Circuit board, pins, dependency graph |
| `pool/` | Parameter pool (sharded, ring buffer) |
| `loadctrl/` | Traffic shaping, rate limiting, backpressure |
| `selector/` | Weighted endpoint selection |
| `generator/` | Data generators (faker, pattern, random) |
| `executor/` | Request building and execution |
| `client/` | HTTP client with auth handling |
| `metrics/` | Collection, reporting, Prometheus export |
| `warmup/` | Warmup phase execution |
| `workflow/` | Business workflow execution |

## Troubleshooting

### Common Issues

**Build fails with missing dependencies:**
```bash
cd tools/loadgen && go mod tidy
make loadgen-build
```

**Authentication errors:**
- Check `auth.login` configuration
- Verify backend is running: `make dev-backend`
- Check credentials in config

**Parameter pool empty:**
- Increase warmup iterations
- Add producer endpoints to fill required types
- Check semantic type mappings

**High error rates:**
- Reduce QPS
- Check backpressure settings
- Verify target system health

### Debug Mode

```bash
# Verbose output shows detailed execution
loadgen -config configs/erp.yaml -v

# Dry run to inspect configuration
loadgen -config configs/erp.yaml -dry-run -v

# List endpoints with details
loadgen -config configs/erp.yaml -list
```

## Development

### Running Tests

```bash
make loadgen-test

# With coverage
cd tools/loadgen && go test -cover ./...
```

### Project Structure

```
tools/loadgen/
├── cmd/
│   └── main.go           # CLI entry point
├── internal/
│   ├── config/           # Configuration
│   ├── parser/           # OpenAPI parsing
│   ├── circuit/          # Circuit board
│   ├── pool/             # Parameter pool
│   ├── loadctrl/         # Load control
│   ├── selector/         # Endpoint selection
│   ├── generator/        # Data generation
│   ├── executor/         # Request execution
│   ├── client/           # HTTP client
│   ├── metrics/          # Metrics & reporting
│   ├── warmup/           # Warmup execution
│   └── workflow/         # Workflow execution
├── configs/
│   ├── erp.yaml          # Main ERP config
│   └── test.yaml         # Test config
├── bin/
│   └── loadgen           # Built binary
└── README.md             # This file
```

## Related Documentation

- [loadgen.md](./loadgen.md) - Detailed design specification
- [CLAUDE.md](../../CLAUDE.md) - Project conventions
- [spec.md](../../.claude/ralph/docs/spec.md) - ERP system specification
