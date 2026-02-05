# Timestamp Precision to Seconds Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Display all frontend timestamps with second precision (YYYY-MM-DD HH:mm:ss) instead of date-only or minute precision.

**Architecture:** Update the centralized `useFormatters()` hook to default to second precision, migrate all legacy date formatting patterns to use the hook, and ensure backend models/API properly support second-level timestamps.

**Tech Stack:** React, TypeScript, Go, PostgreSQL, Intl.DateTimeFormat

---

## Task 1: Update Central Formatter Hook to Show Seconds by Default

**Files:**
- Modify: `frontend/src/hooks/useFormatters.ts:1-100`

**Step 1: Read current implementation**

Run: `cat frontend/src/hooks/useFormatters.ts`
Expected: See current `formatDateTime` function with `showSeconds: false` default

**Step 2: Update formatDateTime to show seconds by default**

In `frontend/src/hooks/useFormatters.ts`, change the `formatDateTime` function:

```typescript
const formatDateTime = useCallback(
  (
    date: string | Date,
    options: {
      showSeconds?: boolean
      timeZone?: string
    } = {}
  ): string => {
    const { showSeconds = true, timeZone } = options // Changed from false to true

    try {
      const dateObj = typeof date === 'string' ? new Date(date) : date

      if (isNaN(dateObj.getTime())) {
        return '-'
      }

      const formatter = new Intl.DateTimeFormat(locale, {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: showSeconds ? '2-digit' : undefined,
        timeZone,
      })

      return formatter.format(dateObj)
    } catch (error) {
      console.error('Date formatting error:', error)
      return '-'
    }
  },
  [locale]
)
```

**Step 3: Update formatTime to show seconds by default**

In the same file, update `formatTime`:

```typescript
const formatTime = useCallback(
  (
    date: string | Date,
    options: {
      showSeconds?: boolean
      timeZone?: string
    } = {}
  ): string => {
    const { showSeconds = true, timeZone } = options // Changed from false to true

    try {
      const dateObj = typeof date === 'string' ? new Date(date) : date

      if (isNaN(dateObj.getTime())) {
        return '-'
      }

      const formatter = new Intl.DateTimeFormat(locale, {
        hour: '2-digit',
        minute: '2-digit',
        second: showSeconds ? '2-digit' : undefined,
        timeZone,
      })

      return formatter.format(dateObj)
    } catch (error) {
      console.error('Time formatting error:', error)
      return '-'
    }
  },
  [locale]
)
```

**Step 4: Verify changes compile**

Run: `cd frontend && npm run type-check`
Expected: No TypeScript errors

**Step 5: Commit**

```bash
git add frontend/src/hooks/useFormatters.ts
git commit -m "feat(frontend): default timestamp display to second precision"
```

---

## Task 2: Update Export Utility Default Format

**Files:**
- Modify: `frontend/src/utils/export.ts:1-50`

**Step 1: Read current implementation**

Run: `cat frontend/src/utils/export.ts | grep -A 20 formatDateForExport`
Expected: See default format as `'YYYY-MM-DD HH:mm'`

**Step 2: Change default format to include seconds**

In `frontend/src/utils/export.ts`, update the `formatDateForExport` function:

```typescript
export function formatDateForExport(
  value: any,
  format: 'YYYY-MM-DD' | 'YYYY-MM-DD HH:mm' | 'YYYY-MM-DD HH:mm:ss' = 'YYYY-MM-DD HH:mm:ss' // Changed default
): string {
  if (!value) return ''

  const date = new Date(value)
  if (isNaN(date.getTime())) return ''

  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')

  switch (format) {
    case 'YYYY-MM-DD':
      return `${year}-${month}-${day}`
    case 'YYYY-MM-DD HH:mm':
      return `${year}-${month}-${day} ${hours}:${minutes}`
    case 'YYYY-MM-DD HH:mm:ss':
      return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
    default:
      return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  }
}
```

**Step 3: Verify changes compile**

Run: `cd frontend && npm run type-check`
Expected: No TypeScript errors

**Step 4: Commit**

```bash
git add frontend/src/utils/export.ts
git commit -m "feat(frontend): default export timestamp format to seconds"
```

---

## Task 3: Migrate Finance Pages Custom formatDateTime Functions

**Files:**
- Modify: `frontend/src/pages/finance/ReceivableDetail.tsx:1-300`
- Modify: `frontend/src/pages/finance/PayableDetail.tsx:1-300`
- Modify: `frontend/src/pages/finance/CashFlow.tsx:1-400`

**Step 1: Update ReceivableDetail.tsx**

Remove the custom `formatDateTime` function and use the hook:

```typescript
// Remove this custom function:
// function formatDateTime(dateStr?: string): string {
//   if (!dateStr) return '-'
//   const date = new Date(dateStr)
//   return date.toLocaleString('zh-CN', {
//     year: 'numeric',
//     month: '2-digit',
//     day: '2-digit',
//     hour: '2-digit',
//     minute: '2-digit',
//   })
// }

// Add at top of component:
const { formatDateTime } = useFormatters()

// Usage remains the same:
// formatDateTime(receivable.created_at)
```

**Step 2: Update PayableDetail.tsx**

Apply the same changes as ReceivableDetail.tsx:

```typescript
// Remove custom formatDateTime function
// Add: const { formatDateTime } = useFormatters()
```

**Step 3: Update CashFlow.tsx**

Apply the same changes:

```typescript
// Remove custom formatDateTime function
// Add: const { formatDateTime } = useFormatters()
```

**Step 4: Verify changes compile**

Run: `cd frontend && npm run type-check`
Expected: No TypeScript errors

**Step 5: Commit**

```bash
git add frontend/src/pages/finance/ReceivableDetail.tsx \
        frontend/src/pages/finance/PayableDetail.tsx \
        frontend/src/pages/finance/CashFlow.tsx
git commit -m "refactor(frontend): migrate finance pages to use centralized formatDateTime"
```

---

## Task 4: Migrate Legacy toLocaleString() Calls (Batch 1: Trade Pages)

**Files:**
- Modify: `frontend/src/pages/trade/SalesOrders.tsx`
- Modify: `frontend/src/pages/trade/SalesOrderDetail.tsx`
- Modify: `frontend/src/pages/trade/PurchaseOrders.tsx`
- Modify: `frontend/src/pages/trade/PurchaseOrderDetail.tsx`

**Step 1: Find all toLocaleString calls in trade pages**

Run: `grep -n "toLocaleString\|toLocaleDateString" frontend/src/pages/trade/*.tsx`
Expected: List of files with line numbers

**Step 2: Update SalesOrders.tsx**

Replace all date formatting:

```typescript
// Before:
render: (date: string) => new Date(date).toLocaleDateString()

// After:
const { formatDate, formatDateTime } = useFormatters()
// For date-only fields:
render: (date: string) => formatDate(date, 'medium')
// For timestamp fields (created_at, updated_at, confirmed_at, etc.):
render: (date: string) => formatDateTime(date)
```

**Step 3: Update SalesOrderDetail.tsx**

Apply the same pattern:

```typescript
const { formatDate, formatDateTime } = useFormatters()

// Replace all:
// new Date(order.created_at).toLocaleString()
// with:
// formatDateTime(order.created_at)
```

**Step 4: Update PurchaseOrders.tsx and PurchaseOrderDetail.tsx**

Apply the same pattern to both files.

**Step 5: Verify changes compile**

Run: `cd frontend && npm run type-check`
Expected: No TypeScript errors

**Step 6: Commit**

```bash
git add frontend/src/pages/trade/*.tsx
git commit -m "refactor(frontend): migrate trade pages to centralized date formatters"
```

---

## Task 5: Migrate Legacy toLocaleString() Calls (Batch 2: Inventory Pages)

**Files:**
- Modify: `frontend/src/pages/inventory/StockList.tsx`
- Modify: `frontend/src/pages/inventory/StockDetail.tsx`
- Modify: `frontend/src/pages/inventory/StockMovements.tsx`
- Modify: `frontend/src/pages/inventory/InventoryAdjustments.tsx`

**Step 1: Find all toLocaleString calls in inventory pages**

Run: `grep -n "toLocaleString\|toLocaleDateString" frontend/src/pages/inventory/*.tsx`
Expected: List of files with line numbers

**Step 2: Update all inventory pages**

Apply the same pattern as Task 4:

```typescript
const { formatDate, formatDateTime } = useFormatters()

// Replace date-only fields:
render: (date: string) => formatDate(date, 'medium')

// Replace timestamp fields:
render: (date: string) => formatDateTime(date)
```

**Step 3: Verify changes compile**

Run: `cd frontend && npm run type-check`
Expected: No TypeScript errors

**Step 4: Commit**

```bash
git add frontend/src/pages/inventory/*.tsx
git commit -m "refactor(frontend): migrate inventory pages to centralized date formatters"
```

---

## Task 6: Migrate Legacy toLocaleString() Calls (Batch 3: Remaining Pages)

**Files:**
- Modify: `frontend/src/pages/catalog/*.tsx`
- Modify: `frontend/src/pages/partner/*.tsx`
- Modify: `frontend/src/pages/finance/*.tsx` (any remaining)
- Modify: `frontend/src/pages/identity/*.tsx`

**Step 1: Find all remaining toLocaleString calls**

Run: `grep -rn "toLocaleString\|toLocaleDateString" frontend/src/pages/ --include="*.tsx"`
Expected: List of remaining files

**Step 2: Update all remaining pages**

Apply the same pattern:

```typescript
const { formatDate, formatDateTime } = useFormatters()

// Replace appropriately based on field type
```

**Step 3: Verify changes compile**

Run: `cd frontend && npm run type-check`
Expected: No TypeScript errors

**Step 4: Commit**

```bash
git add frontend/src/pages/
git commit -m "refactor(frontend): complete migration to centralized date formatters"
```

---

## Task 7: Update Backend OpenAPI Annotations for Timestamp Fields

**Files:**
- Modify: `backend/internal/interfaces/http/handler/sales_order.go`
- Modify: `backend/internal/interfaces/http/handler/purchase_order.go`
- Modify: `backend/internal/interfaces/http/handler/stock.go`
- Modify: All other handler files with timestamp fields

**Step 1: Add format annotation to SalesOrderResponse**

In `backend/internal/interfaces/http/handler/sales_order.go`:

```go
type SalesOrderResponse struct {
    // ... other fields

    // @Description Order confirmation timestamp
    // @Example "2026-02-02T14:30:45Z"
    ConfirmedAt *time.Time `json:"confirmed_at,omitempty" format:"date-time"`

    // @Description Order shipment timestamp
    // @Example "2026-02-02T15:45:30Z"
    ShippedAt   *time.Time `json:"shipped_at,omitempty" format:"date-time"`

    // @Description Order completion timestamp
    // @Example "2026-02-02T16:20:15Z"
    CompletedAt *time.Time `json:"completed_at,omitempty" format:"date-time"`

    // @Description Order cancellation timestamp
    // @Example "2026-02-02T14:50:00Z"
    CancelledAt *time.Time `json:"cancelled_at,omitempty" format:"date-time"`

    // @Description Record creation timestamp
    // @Example "2026-02-02T14:00:00Z"
    CreatedAt   time.Time  `json:"created_at" format:"date-time"`

    // @Description Record last update timestamp
    // @Example "2026-02-02T14:30:45Z"
    UpdatedAt   time.Time  `json:"updated_at" format:"date-time"`
}
```

**Step 2: Add format annotation to SalesOrderListResponse**

In the same file:

```go
type SalesOrderListResponse struct {
    // ... other fields

    ConfirmedAt *time.Time `json:"confirmed_at,omitempty" format:"date-time"`
    ShippedAt   *time.Time `json:"shipped_at,omitempty" format:"date-time"`
    CreatedAt   time.Time  `json:"created_at" format:"date-time"`
    UpdatedAt   time.Time  `json:"updated_at" format:"date-time"`
}
```

**Step 3: Verify code compiles**

Run: `cd backend && go build ./cmd/server`
Expected: Successful compilation

**Step 4: Commit**

```bash
git add backend/internal/interfaces/http/handler/sales_order.go
git commit -m "docs(backend): add OpenAPI format annotations for sales order timestamps"
```

---

## Task 8: Update Remaining Backend Handler Timestamp Annotations

**Files:**
- Modify: `backend/internal/interfaces/http/handler/purchase_order.go`
- Modify: `backend/internal/interfaces/http/handler/stock.go`
- Modify: `backend/internal/interfaces/http/handler/receivable.go`
- Modify: `backend/internal/interfaces/http/handler/payable.go`
- Modify: All other handler files

**Step 1: Find all handler files with time.Time fields**

Run: `grep -rn "time.Time.*json:" backend/internal/interfaces/http/handler/ --include="*.go"`
Expected: List of all files with timestamp fields

**Step 2: Add format:"date-time" to all time.Time fields**

For each file, add `format:"date-time"` to the struct tag:

```go
CreatedAt time.Time `json:"created_at" format:"date-time"`
UpdatedAt time.Time `json:"updated_at" format:"date-time"`
```

**Step 3: Verify code compiles**

Run: `cd backend && go build ./cmd/server`
Expected: Successful compilation

**Step 4: Commit**

```bash
git add backend/internal/interfaces/http/handler/
git commit -m "docs(backend): add OpenAPI format annotations for all timestamp fields"
```

---

## Task 9: Regenerate OpenAPI Specification

**Files:**
- Modify: `backend/docs/swagger.yaml` (auto-generated)

**Step 1: Regenerate Swagger docs**

Run: `cd backend && make api-docs`
Expected: Success message, swagger.yaml updated

**Step 2: Verify timestamp fields have format annotation**

Run: `grep -A 2 "created_at:" backend/docs/swagger.yaml | head -20`
Expected: See `format: date-time` on timestamp fields

**Step 3: Commit**

```bash
git add backend/docs/swagger.yaml
git commit -m "docs(backend): regenerate OpenAPI spec with timestamp format annotations"
```

---

## Task 10: Regenerate Frontend API Client

**Files:**
- Modify: `frontend/src/api/` (auto-generated)

**Step 1: Regenerate TypeScript API client**

Run: `cd frontend && npm run api:generate`
Expected: Success message, API models updated

**Step 2: Verify timestamp types**

Run: `grep "created_at" frontend/src/api/models/handlerSalesOrderResponse.ts`
Expected: Still `string` type (OpenAPI Generator doesn't change this)

**Step 3: Commit**

```bash
git add frontend/src/api/
git commit -m "chore(frontend): regenerate API client with updated OpenAPI spec"
```

---

## Task 11: Write Unit Tests for useFormatters Hook

**Files:**
- Create: `frontend/src/hooks/__tests__/useFormatters.test.ts`

**Step 1: Write failing test for formatDateTime with seconds**

```typescript
import { renderHook } from '@testing-library/react'
import { useFormatters } from '../useFormatters'

describe('useFormatters', () => {
  describe('formatDateTime', () => {
    it('should display seconds by default', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatDateTime('2026-02-02T14:30:45Z')

      // Should include seconds
      expect(formatted).toMatch(/14:30:45/)
    })

    it('should hide seconds when showSeconds is false', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatDateTime('2026-02-02T14:30:45Z', {
        showSeconds: false,
      })

      // Should not include seconds
      expect(formatted).toMatch(/14:30/)
      expect(formatted).not.toMatch(/14:30:45/)
    })

    it('should handle invalid dates gracefully', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatDateTime('invalid-date')

      expect(formatted).toBe('-')
    })
  })

  describe('formatTime', () => {
    it('should display seconds by default', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatTime('2026-02-02T14:30:45Z')

      expect(formatted).toMatch(/14:30:45/)
    })

    it('should hide seconds when showSeconds is false', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatTime('2026-02-02T14:30:45Z', {
        showSeconds: false,
      })

      expect(formatted).toMatch(/14:30/)
      expect(formatted).not.toMatch(/14:30:45/)
    })
  })

  describe('formatDate', () => {
    it('should format date without time', () => {
      const { result } = renderHook(() => useFormatters())
      const formatted = result.current.formatDate('2026-02-02T14:30:45Z', 'medium')

      // Should not include time
      expect(formatted).not.toMatch(/14:30/)
    })
  })
})
```

**Step 2: Run test to verify it passes**

Run: `cd frontend && npm run test -- useFormatters.test.ts`
Expected: All tests pass

**Step 3: Commit**

```bash
git add frontend/src/hooks/__tests__/useFormatters.test.ts
git commit -m "test(frontend): add unit tests for useFormatters hook"
```

---

## Task 12: Write Unit Tests for Export Utility

**Files:**
- Create: `frontend/src/utils/__tests__/export.test.ts`

**Step 1: Write test for formatDateForExport**

```typescript
import { formatDateForExport } from '../export'

describe('formatDateForExport', () => {
  const testDate = '2026-02-02T14:30:45.123Z'

  it('should format with seconds by default', () => {
    const result = formatDateForExport(testDate)
    expect(result).toBe('2026-02-02 14:30:45')
  })

  it('should format date only when specified', () => {
    const result = formatDateForExport(testDate, 'YYYY-MM-DD')
    expect(result).toBe('2026-02-02')
  })

  it('should format with minutes when specified', () => {
    const result = formatDateForExport(testDate, 'YYYY-MM-DD HH:mm')
    expect(result).toBe('2026-02-02 14:30')
  })

  it('should format with seconds when specified', () => {
    const result = formatDateForExport(testDate, 'YYYY-MM-DD HH:mm:ss')
    expect(result).toBe('2026-02-02 14:30:45')
  })

  it('should handle null/undefined values', () => {
    expect(formatDateForExport(null)).toBe('')
    expect(formatDateForExport(undefined)).toBe('')
  })

  it('should handle invalid dates', () => {
    expect(formatDateForExport('invalid')).toBe('')
  })
})
```

**Step 2: Run test to verify it passes**

Run: `cd frontend && npm run test -- export.test.ts`
Expected: All tests pass

**Step 3: Commit**

```bash
git add frontend/src/utils/__tests__/export.test.ts
git commit -m "test(frontend): add unit tests for export utility"
```

---

## Task 13: Manual Testing - Verify Timestamp Display

**Step 1: Start development environment**

Run: `make dev && make dev-backend && make dev-frontend`
Expected: All services running

**Step 2: Test Sales Orders page**

1. Navigate to http://localhost:3000/trade/sales-orders
2. Verify `created_at` column shows format: `YYYY-MM-DD HH:mm:ss`
3. Click on an order to view details
4. Verify all timestamp fields show seconds

**Step 3: Test Inventory pages**

1. Navigate to http://localhost:3000/inventory/stock
2. Verify `updated_at` shows seconds
3. Check Stock Movements page
4. Verify movement timestamps show seconds

**Step 4: Test Finance pages**

1. Navigate to http://localhost:3000/finance/receivables
2. Verify `created_at`, `due_date` show seconds
3. Click on a receivable detail
4. Verify all timestamps show seconds

**Step 5: Test Export functionality**

1. Go to any list page (e.g., Sales Orders)
2. Click "Export" button
3. Open exported CSV/Excel file
4. Verify timestamp columns show format: `YYYY-MM-DD HH:mm:ss`

**Step 6: Document test results**

Create a test report:

```markdown
# Manual Test Results - Timestamp Precision

## Test Date: 2026-02-02

### Sales Orders
- [x] List page shows seconds
- [x] Detail page shows seconds
- [x] Export includes seconds

### Inventory
- [x] Stock list shows seconds
- [x] Stock movements show seconds
- [x] Adjustments show seconds

### Finance
- [x] Receivables show seconds
- [x] Payables show seconds
- [x] Cash flow shows seconds

### Issues Found
- None

### Screenshots
- (Attach screenshots if needed)
```

---

## Task 14: Update Documentation

**Files:**
- Modify: `frontend/README.md`

**Step 1: Add section about timestamp formatting**

In `frontend/README.md`, add a section:

```markdown
## Date and Time Formatting

### Centralized Formatters

All date/time formatting should use the `useFormatters()` hook:

```typescript
import { useFormatters } from '@/hooks/useFormatters'

function MyComponent() {
  const { formatDate, formatTime, formatDateTime } = useFormatters()

  return (
    <div>
      {/* Date only: 2026-02-02 */}
      <span>{formatDate(record.created_at, 'medium')}</span>

      {/* Time only: 14:30:45 */}
      <span>{formatTime(record.created_at)}</span>

      {/* Date and time: 2026-02-02 14:30:45 */}
      <span>{formatDateTime(record.created_at)}</span>

      {/* Hide seconds if needed */}
      <span>{formatDateTime(record.created_at, { showSeconds: false })}</span>
    </div>
  )
}
```

### Default Precision

- **Timestamps** (`created_at`, `updated_at`, etc.): Display with **second precision** by default
- **Dates** (`due_date`, `birth_date`, etc.): Display date only (no time)
- **Export**: CSV/Excel exports include seconds by default

### Locale Support

The formatters automatically use the user's locale setting:
- `zh-CN`: Chinese format (2026年2月2日 14:30:45)
- `en-US`: US format (2/2/2026, 2:30:45 PM)

### Legacy Patterns (Deprecated)

❌ **Don't use:**
```typescript
new Date(date).toLocaleDateString()
new Date(date).toLocaleString()
```

✅ **Use instead:**
```typescript
const { formatDate, formatDateTime } = useFormatters()
formatDate(date, 'medium')
formatDateTime(date)
```
```

**Step 2: Commit**

```bash
git add frontend/README.md
git commit -m "docs(frontend): add timestamp formatting guidelines"
```

---

## Task 15: Run Full Test Suite

**Step 1: Run frontend tests**

Run: `cd frontend && npm run test:run`
Expected: All tests pass

**Step 2: Run frontend type check**

Run: `cd frontend && npm run type-check`
Expected: No TypeScript errors

**Step 3: Run frontend lint**

Run: `cd frontend && npm run lint`
Expected: No lint errors

**Step 4: Run backend tests**

Run: `cd backend && go test ./...`
Expected: All tests pass

**Step 5: Build frontend**

Run: `cd frontend && npm run build`
Expected: Successful build

**Step 6: Build backend**

Run: `cd backend && go build ./cmd/server`
Expected: Successful build

**Step 7: Document test results**

If all tests pass, create summary:

```markdown
# Test Results Summary

## Frontend
- ✅ Unit tests: PASS
- ✅ Type check: PASS
- ✅ Lint: PASS
- ✅ Build: PASS

## Backend
- ✅ Unit tests: PASS
- ✅ Build: PASS

## Manual Testing
- ✅ All timestamp displays show seconds
- ✅ Export functionality includes seconds
- ✅ No visual regressions
```

---

## Task 16: Final Commit and Summary

**Step 1: Review all changes**

Run: `git log --oneline --graph --all -20`
Expected: See all commits from this implementation

**Step 2: Create implementation summary**

Create a summary of changes:

```markdown
# Implementation Summary: Timestamp Precision to Seconds

## Changes Made

### Frontend
1. Updated `useFormatters()` hook to show seconds by default
2. Updated export utility to include seconds by default
3. Migrated all pages from legacy date formatting to centralized hook
4. Added comprehensive unit tests

### Backend
1. Added OpenAPI `format:"date-time"` annotations to all timestamp fields
2. Regenerated Swagger documentation

### Documentation
1. Added timestamp formatting guidelines to frontend README
2. Documented best practices and deprecated patterns

## Files Modified
- Frontend: 50+ component files
- Backend: 15+ handler files
- Tests: 2 new test files
- Docs: 2 documentation files

## Testing
- ✅ All unit tests pass
- ✅ Manual testing completed
- ✅ No regressions found

## Migration Notes
- All timestamps now display with second precision by default
- Legacy `toLocaleString()` calls have been removed
- Export functionality includes seconds
- Backward compatible: can still hide seconds with `showSeconds: false`
```

**Step 3: Final verification**

Run: `git status`
Expected: Clean working directory (all changes committed)

---

## Completion Checklist

- [ ] Task 1: Updated useFormatters hook defaults
- [ ] Task 2: Updated export utility default
- [ ] Task 3: Migrated finance pages
- [ ] Task 4: Migrated trade pages
- [ ] Task 5: Migrated inventory pages
- [ ] Task 6: Migrated remaining pages
- [ ] Task 7: Updated sales order handler annotations
- [ ] Task 8: Updated all handler annotations
- [ ] Task 9: Regenerated OpenAPI spec
- [ ] Task 10: Regenerated frontend API client
- [ ] Task 11: Added useFormatters tests
- [ ] Task 12: Added export utility tests
- [ ] Task 13: Completed manual testing
- [ ] Task 14: Updated documentation
- [ ] Task 15: All tests passing
- [ ] Task 16: Final summary created

---

## Notes

- **No backend model changes needed**: PostgreSQL already stores microsecond precision, Go `time.Time` already supports it
- **No database migrations needed**: Schema already uses `TIMESTAMP WITH TIME ZONE`
- **Backward compatible**: Components can still opt out with `showSeconds: false`
- **Locale-aware**: Formatting respects user's locale setting
- **DRY principle**: All formatting centralized in one hook
- **Test coverage**: Unit tests ensure behavior is correct

## Estimated Impact

- **Files modified**: ~70 files
- **Lines changed**: ~200-300 lines
- **Risk level**: Low (mostly display changes, no business logic)
- **Breaking changes**: None (backward compatible)
