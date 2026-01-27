# Observability Alerting Rules

> **Status**: Design Document (Future Implementation)
>
> This document defines the alerting rules for the ERP system based on OpenTelemetry metrics.
> Current implementation exports metrics via file exporter; these rules are designed for future
> Prometheus AlertManager integration.

## Overview

This alerting configuration provides automated monitoring and notification for critical system
health indicators. Alerts are categorized by severity and configured with appropriate thresholds
based on ERP business requirements.

## Alert Severity Levels

| Level | Description | Response Time | Example |
|-------|-------------|---------------|---------|
| **critical** | Service degradation affecting users | Immediate (< 5 min) | High error rate, DB pool exhaustion |
| **warning** | Potential issue requiring attention | Within 1 hour | High latency, low stock levels |
| **info** | Informational, no immediate action | Next business day | Traffic spikes, approaching thresholds |

---

## Predefined Alert Rules

### 1. High Error Rate Alert

**Purpose**: Detect when the API is returning too many errors, indicating potential service issues.

```yaml
# prometheus-alerts.yaml
groups:
  - name: erp.http.alerts
    rules:
      - alert: HighErrorRate
        expr: |
          (
            sum(rate(http_server_request_total{http_status_code=~"5.."}[5m]))
            /
            sum(rate(http_server_request_total[5m]))
          ) > 0.01
        for: 2m
        labels:
          severity: critical
          team: backend
          service: erp
        annotations:
          summary: "High HTTP 5xx error rate detected"
          description: |
            Error rate is {{ $value | humanizePercentage }} over the last 5 minutes.
            Threshold: 1%
          runbook_url: "https://docs.example.com/runbooks/high-error-rate"
          dashboard_url: "https://grafana.example.com/d/erp-http/overview"
```

**Metric Details**:
- **Metric**: `http_server_request_total`
- **Labels**: `http.method`, `http.route`, `http.status_code`, `tenant_id`
- **Condition**: 5xx error rate > 1% over 5 minutes
- **Alert After**: 2 minutes sustained

**Recommended Actions**:
1. Check recent deployments
2. Review error logs for stack traces
3. Check downstream service health
4. Verify database connectivity

---

### 2. High Latency Alert

**Purpose**: Detect when API response times exceed acceptable thresholds.

```yaml
      - alert: HighLatency
        expr: |
          histogram_quantile(0.99,
            sum(rate(http_server_request_duration_seconds_bucket[5m])) by (le, http_route)
          ) > 2
        for: 5m
        labels:
          severity: warning
          team: backend
          service: erp
        annotations:
          summary: "High API latency detected"
          description: |
            P99 latency for {{ $labels.http_route }} is {{ $value | humanizeDuration }}.
            Threshold: 2 seconds
          runbook_url: "https://docs.example.com/runbooks/high-latency"
          dashboard_url: "https://grafana.example.com/d/erp-http/latency"
```

**Metric Details**:
- **Metric**: `http_server_request_duration_seconds` (Histogram)
- **Bucket Boundaries**: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
- **Condition**: P99 latency > 2 seconds
- **Alert After**: 5 minutes sustained

**Recommended Actions**:
1. Identify slow routes from labels
2. Check database query performance
3. Review N+1 query patterns
4. Check external service latencies

---

### 3. Database Connection Pool Exhaustion Alert

**Purpose**: Detect when database connection pool is near exhaustion, which causes request queuing.

```yaml
  - name: erp.database.alerts
    rules:
      - alert: DBPoolExhaustion
        expr: |
          (
            db_pool_connections{db_pool_state="in_use"}
            /
            db_pool_connections_max
          ) > 0.9
        for: 1m
        labels:
          severity: critical
          team: backend
          service: erp
        annotations:
          summary: "Database connection pool near exhaustion"
          description: |
            Pool utilization is {{ $value | humanizePercentage }}.
            In use: {{ with query "db_pool_connections{db_pool_state='in_use'}" }}{{ . | first | value }}{{ end }}
            Max: {{ with query "db_pool_connections_max" }}{{ . | first | value }}{{ end }}
          runbook_url: "https://docs.example.com/runbooks/db-pool-exhaustion"
          dashboard_url: "https://grafana.example.com/d/erp-db/pool"
```

**Metric Details**:
- **Metrics**:
  - `db_pool_connections` (Gauge, with `db.pool.state` label: `idle`, `in_use`, `open`)
  - `db_pool_connections_max` (Gauge)
- **Condition**: in_use / max > 90%
- **Alert After**: 1 minute sustained

**Recommended Actions**:
1. Check for long-running transactions
2. Identify connection leaks
3. Consider increasing pool size (temporary)
4. Review query optimization opportunities

---

### 4. Low Inventory Stock Alert

**Purpose**: Alert operations team when inventory levels are critically low.

```yaml
  - name: erp.business.alerts
    rules:
      - alert: LowInventoryStock
        expr: erp_inventory_low_stock_count > 10
        for: 5m
        labels:
          severity: warning
          team: operations
          service: erp
        annotations:
          summary: "Multiple products below minimum stock threshold"
          description: |
            {{ $value }} products are below minimum stock levels for tenant {{ $labels.tenant_id }}.
          runbook_url: "https://docs.example.com/runbooks/low-inventory"
          dashboard_url: "https://grafana.example.com/d/erp-inventory/stock"
```

**Metric Details**:
- **Metric**: `erp_inventory_low_stock_count` (Gauge)
- **Labels**: `tenant_id`
- **Condition**: More than 10 products below minimum threshold
- **Alert After**: 5 minutes sustained

**Recommended Actions**:
1. Review low-stock products in inventory dashboard
2. Create purchase orders for replenishment
3. Check for unusual sales patterns
4. Verify minimum stock thresholds are appropriate

---

## Additional Recommended Alerts

### 5. Slow Database Queries Alert

```yaml
      - alert: SlowDatabaseQueries
        expr: |
          rate(db_slow_query_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
          team: backend
          service: erp
        annotations:
          summary: "Elevated slow query rate"
          description: |
            Slow queries (>200ms) are occurring at {{ $value }} per second.
            Table: {{ $labels.db_table }}
          runbook_url: "https://docs.example.com/runbooks/slow-queries"
```

**Metric**: `db_slow_query_total` (Counter, threshold: 200ms by default)

---

### 6. Payment Failure Rate Alert

```yaml
      - alert: HighPaymentFailureRate
        expr: |
          (
            sum(rate(erp_payment_total{payment_status="failed"}[10m]))
            /
            sum(rate(erp_payment_total[10m]))
          ) > 0.05
        for: 5m
        labels:
          severity: critical
          team: finance
          service: erp
        annotations:
          summary: "High payment failure rate"
          description: |
            Payment failure rate is {{ $value | humanizePercentage }}.
            Check payment gateway status.
          runbook_url: "https://docs.example.com/runbooks/payment-failures"
```

**Metric**: `erp_payment_total` (Counter with `payment_status` label: `success`, `failed`)

---

### 7. Order Volume Anomaly Alert

```yaml
      - alert: OrderVolumeAnomaly
        expr: |
          abs(
            sum(rate(erp_order_created_total[1h]))
            -
            sum(rate(erp_order_created_total[1h] offset 1d))
          )
          /
          sum(rate(erp_order_created_total[1h] offset 1d)) > 0.5
        for: 30m
        labels:
          severity: info
          team: operations
          service: erp
        annotations:
          summary: "Unusual order volume detected"
          description: |
            Order volume has changed by more than 50% compared to the same time yesterday.
            Current rate: {{ with query "sum(rate(erp_order_created_total[1h]))" }}{{ . | first | value | humanize }}{{ end }}/hour
```

**Metric**: `erp_order_created_total` (Counter with `tenant_id`, `order_type` labels)

---

## Metrics Reference

### HTTP Metrics (from OBS-METRICS-002)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `http_server_request_total` | Counter | `http.method`, `http.route`, `http.status_code`, `tenant_id` | Total HTTP requests |
| `http_server_request_duration_seconds` | Histogram | `http.method`, `http.route` | Request latency distribution |
| `http_server_request_size_bytes` | Histogram | `http.method`, `http.route` | Request body size |
| `http_server_response_size_bytes` | Histogram | `http.method`, `http.route` | Response body size |
| `http_server_active_requests` | Gauge | `http.method`, `http.route` | Currently processing requests |

### Database Metrics (from OBS-METRICS-003)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `db_pool_connections` | Gauge | `db.pool.state` (idle/in_use/open) | Connection pool state |
| `db_pool_connections_max` | Gauge | - | Maximum pool connections |
| `db_query_total` | Counter | `db.operation` | Total queries by type |
| `db_query_duration_seconds` | Histogram | `db.operation` | Query latency distribution |
| `db_slow_query_total` | Counter | `db.table` | Slow queries (>200ms) |

### Business Metrics (from OBS-METRICS-004)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `erp_order_created_total` | Counter | `tenant_id`, `order_type` | Orders created |
| `erp_order_amount_total` | Counter | `tenant_id`, `order_type` | Total order value (fen) |
| `erp_payment_total` | Counter | `tenant_id`, `payment_method`, `payment_status` | Payment transactions |
| `erp_inventory_locked_quantity` | Gauge | `tenant_id`, `warehouse_id` | Locked inventory |
| `erp_inventory_low_stock_count` | Gauge | `tenant_id` | Products below min stock |

---

## AlertManager Configuration

### Notification Channels

```yaml
# alertmanager.yaml
global:
  resolve_timeout: 5m
  slack_api_url: 'https://hooks.slack.com/services/xxx'

route:
  receiver: 'default-receiver'
  group_by: ['alertname', 'severity', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    # Critical alerts - immediate notification
    - match:
        severity: critical
      receiver: 'critical-receiver'
      group_wait: 10s
      repeat_interval: 1h

    # Warning alerts - business hours
    - match:
        severity: warning
      receiver: 'warning-receiver'
      repeat_interval: 4h

    # Team-specific routing
    - match:
        team: finance
      receiver: 'finance-receiver'

    - match:
        team: operations
      receiver: 'operations-receiver'

receivers:
  - name: 'default-receiver'
    slack_configs:
      - channel: '#erp-alerts'
        send_resolved: true
        title: '{{ .Status | toUpper }}: {{ .CommonLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

  - name: 'critical-receiver'
    slack_configs:
      - channel: '#erp-critical'
        send_resolved: true
    pagerduty_configs:
      - service_key: '<pagerduty-service-key>'
        severity: critical

  - name: 'warning-receiver'
    slack_configs:
      - channel: '#erp-alerts'
        send_resolved: true

  - name: 'finance-receiver'
    email_configs:
      - to: 'finance-team@example.com'
        send_resolved: true

  - name: 'operations-receiver'
    slack_configs:
      - channel: '#erp-operations'
        send_resolved: true
```

### Inhibition Rules

```yaml
inhibit_rules:
  # Don't alert on warning if critical is already firing
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'service']

  # Suppress all alerts if service is in maintenance
  - source_match:
      alertname: 'ServiceMaintenance'
    target_match:
      service: 'erp'
    equal: ['service']
```

---

## Implementation Roadmap

### Phase 1: Infrastructure Setup (Future)
1. Deploy Prometheus server
2. Configure scrape targets for OpenTelemetry Collector
3. Deploy AlertManager
4. Configure basic notification channels (Slack)

### Phase 2: Core Alerts
1. Implement HTTP error rate alert
2. Implement latency alert
3. Implement DB pool exhaustion alert
4. Test alert firing and resolution

### Phase 3: Business Alerts
1. Implement inventory low stock alert
2. Implement payment failure alert
3. Configure team-specific routing
4. Set up PagerDuty integration for critical alerts

### Phase 4: Advanced Monitoring
1. Implement anomaly detection alerts
2. Configure silence/maintenance windows
3. Create Grafana dashboards with alert annotations
4. Document runbooks for each alert

---

## Design Decisions

### Trade-offs

| Decision | Pros | Cons | Risk Level |
|----------|------|------|------------|
| Use rate() over 5m windows | Smooths noise, reduces false positives | Slower to detect issues | 🟢 LOW |
| Critical alerts require 1-2min sustained | Avoids transient spikes | Slight delay in notification | 🟢 LOW |
| Team-based routing | Clear ownership | More configuration | 🟢 LOW |
| P99 for latency (not P95) | Catches tail latency | May be noisy for some routes | 🟡 MEDIUM |

### Alternatives Considered

1. **Push vs Pull metrics**
   - Chose pull (Prometheus) for simplicity and ecosystem support
   - Push (via OpenTelemetry Collector) provides flexibility for future changes

2. **Alert thresholds**
   - Started conservative (1% error rate, 2s latency)
   - Can be tuned based on observed baseline

3. **Multi-tenant alerting**
   - Current design uses `tenant_id` labels but alerts globally
   - Future: per-tenant SLA-based alerting for enterprise customers

---

## References

- [Prometheus Alerting Rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
- [AlertManager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [ERP Metrics Implementation](../../../backend/internal/infrastructure/telemetry/)

---

*Last Updated: 2026-01-27*
*Task: OBS-ALERT-001*
