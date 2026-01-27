# ADR-001: Product Status Methods Semantic Clarification

## Context

The Product aggregate has two methods (`Deactivate()` and `Disable()`) that achieve the same state transition (Active -> Inactive) but with different event emissions:

| Method | Status Transition | Events Emitted |
|--------|-------------------|----------------|
| `Deactivate()` | Active -> Inactive | `ProductStatusChangedEvent` only |
| `Disable()` | Active -> Inactive | `ProductStatusChangedEvent` + `ProductDisabledEvent` |

This creates semantic ambiguity and violates the Ubiquitous Language principle from DDD.

### Current State Analysis

**Spec.md Definition (lines 556-557, 594-598):**
```
+enable()
+disable()

领域事件:
- ProductCreated
- ProductPriceChanged
- ProductDisabled
```

The specification explicitly defines `enable()`/`disable()` methods with `ProductDisabled` as the domain event.

**Cross-Codebase Patterns:**
| Entity | Enable Method | Disable Method | Pattern |
|--------|--------------|----------------|---------|
| Product | `Activate()` | `Deactivate()` + `Disable()` | INCONSISTENT |
| Warehouse | `Enable()` | `Disable()` | Enable/Disable |
| Role | `Enable()` | `Disable()` | Enable/Disable |
| Customer | `Activate()` | `Deactivate()` | Activate/Deactivate |
| Supplier | `Activate()` | `Deactivate()` | Activate/Deactivate |

**Current Usage:**
- API Layer: Only exposes `Deactivate` endpoint (`/catalog/products/{id}/deactivate`)
- Application Service: Only uses `product.Deactivate()` - `Disable()` is never called from services
- `ProductDisabledEvent` is only emitted through `Disable()` which is not exposed

## Decision

**Recommendation: Keep current implementation with documentation clarification, defer refactoring**

After careful analysis, we recommend:

1. **Keep both methods** for backward compatibility
2. **Add clear documentation** explaining the semantic difference:
   - `Deactivate()`: Internal status change (admin toggles product visibility)
   - `Disable()`: Cross-context notification (triggers inventory/trade integration)
3. **Document as technical debt** for future alignment with spec
4. **No breaking API changes** at this time

### Rationale

1. **Backward Compatibility**: Changing from `Deactivate` to `Disable` in API would break existing integrations
2. **Minimal Impact**: The `Disable()` method with cross-context events is currently unused - this is a code smell but not a runtime issue
3. **Future Migration Path**: When spec-aligned refactoring is needed, can be done as part of a larger breaking change release

### When to Apply Full Refactoring

Apply the full refactoring (merging to `Enable()`/`Disable()` per spec) when:
- Major version release is planned
- Cross-context integration via events becomes a requirement
- API versioning strategy is established

## Consequences

### Positive
- No breaking changes to existing API consumers
- Minimal code churn
- Clear documentation for developers
- Preserves both event emission patterns for future use

### Negative
- Technical debt remains (Ubiquitous Language inconsistency)
- Two methods for same state transition is confusing
- Spec deviation documented but not resolved

### Alternatives Considered

1. **Merge to `enable()`/`disable()` per spec**
   - Pros: Full spec alignment, clear semantics
   - Cons: Breaking API change, requires client migration
   - **Rejected**: Too much churn for LOW severity issue

2. **Merge to `Activate()`/`Deactivate()`**
   - Pros: Consistent with Customer/Supplier pattern
   - Cons: Still deviates from spec
   - **Rejected**: Doesn't solve the underlying issue

3. **Add parameter `SetStatus(active bool, notifyOthers bool)`**
   - Pros: Single method, flexible
   - Cons: Leaks integration concerns into domain
   - **Rejected**: Violates DDD domain purity

## Trade-Offs

- **Pros**: Stability, no breaking changes, documented understanding
- **Cons**: Technical debt accumulation, spec deviation
- **Risk Level**: 🟢 LOW

## Status

Accepted

## Date

2026-01-27
