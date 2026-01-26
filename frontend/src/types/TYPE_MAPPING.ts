/**
 * Frontend-Backend Type Mapping Documentation
 *
 * This file documents how frontend TypeScript types align with backend Go types.
 * It serves as a reference for developers to understand type relationships.
 *
 * @module type-mapping
 * @version 1.0.0
 * @lastUpdated 2026-01-26
 */

/**
 * ## Overview
 *
 * The ERP system maintains type consistency between frontend TypeScript and
 * backend Go through:
 *
 * 1. **Auto-generated API types** - Generated via orval from OpenAPI spec
 * 2. **Domain value objects** - Manually maintained in `types/domain.ts`
 * 3. **Status enums** - Aligned with backend domain model status values
 *
 * ## API Type Generation
 *
 * ### Process
 * ```bash
 * # Step 1: Generate OpenAPI spec from Go annotations
 * cd backend && make docs
 *
 * # Step 2: Generate TypeScript types from OpenAPI spec
 * cd frontend && npx orval
 * ```
 *
 * ### Source Files
 * - Backend: `backend/docs/swagger.yaml` (source of truth)
 * - Frontend: `frontend/src/api/` (auto-generated, DO NOT edit)
 *
 * ### Configuration
 * - orval config: `frontend/orval.config.ts`
 * - Output mode: tags-split (organized by API tags)
 *
 * ## Type Mapping Reference
 *
 * ### Money Value Object
 *
 * | Backend (Go) | Frontend (TypeScript) |
 * |--------------|----------------------|
 * | `valueobject.Money` | `Money` interface |
 * | `amount: decimal.Decimal` | `amount: string` |
 * | `currency: Currency` | `currency: Currency` |
 *
 * The Money type is serialized as JSON with string amount for precision:
 * ```json
 * { "amount": "1234.56", "currency": "CNY" }
 * ```
 *
 * ### Currency Codes (ISO 4217)
 *
 * | Code | Description |
 * |------|-------------|
 * | CNY | Chinese Yuan (default) |
 * | USD | US Dollar |
 * | EUR | Euro |
 * | GBP | British Pound |
 * | JPY | Japanese Yen |
 * | HKD | Hong Kong Dollar |
 *
 * ### Status Enums Mapping
 *
 * #### Identity Module
 *
 * | Entity | Backend Type | Frontend Type | Values |
 * |--------|--------------|---------------|--------|
 * | Tenant | `identity.TenantStatus` | `TenantStatus` | active, inactive, suspended, trial |
 * | User | `identity.UserStatus` | `UserStatus` | pending, active, locked, deactivated |
 *
 * #### Catalog Module
 *
 * | Entity | Backend Type | Frontend Type | Values |
 * |--------|--------------|---------------|--------|
 * | Product | `catalog.ProductStatus` | `ProductStatus` | active, inactive, discontinued |
 * | Category | `catalog.CategoryStatus` | `CategoryStatus` | active, inactive |
 *
 * #### Partner Module
 *
 * | Entity | Backend Type | Frontend Type | Values |
 * |--------|--------------|---------------|--------|
 * | Customer | `partner.CustomerStatus` | `CustomerStatus` | active, inactive, suspended |
 * | Supplier | `partner.SupplierStatus` | `SupplierStatus` | active, inactive, blocked |
 * | Warehouse | `partner.WarehouseStatus` | `WarehouseStatus` | active, inactive |
 *
 * #### Trade Module
 *
 * | Entity | Backend Type | Frontend Type | Values |
 * |--------|--------------|---------------|--------|
 * | Sales Order | `trade.OrderStatus` | `SalesOrderStatus` | DRAFT, CONFIRMED, SHIPPED, COMPLETED, CANCELLED |
 * | Sales Return | `trade.ReturnStatus` | `SalesReturnStatus` | DRAFT, PENDING, APPROVED, RECEIVING, REJECTED, COMPLETED, CANCELLED |
 * | Purchase Order | `trade.PurchaseOrderStatus` | `PurchaseOrderStatus` | DRAFT, CONFIRMED, PARTIAL_RECEIVED, COMPLETED, CANCELLED |
 * | Purchase Return | `trade.PurchaseReturnStatus` | `PurchaseReturnStatus` | DRAFT, PENDING, APPROVED, REJECTED, SHIPPED, COMPLETED, CANCELLED |
 *
 * #### Finance Module
 *
 * | Entity | Backend Type | Frontend Type | Values |
 * |--------|--------------|---------------|--------|
 * | Receivable | `finance.ReceivableStatus` | `ReceivableStatus` | PENDING, PARTIAL, PAID, REVERSED, CANCELLED |
 * | Payable | `finance.PayableStatus` | `PayableStatus` | PENDING, PARTIAL, PAID, REVERSED, CANCELLED |
 * | Voucher | `finance.VoucherStatus` | `VoucherStatus` | DRAFT, CONFIRMED, ALLOCATED, CANCELLED |
 * | Expense | `finance.ExpenseStatus` | `ExpenseStatus` | DRAFT, PENDING, APPROVED, REJECTED, CANCELLED |
 * | Payment | `finance.PaymentStatus` | `PaymentStatus` | UNPAID, PAID |
 * | Income | `finance.IncomeStatus` | `IncomeStatus` | DRAFT, CONFIRMED, CANCELLED |
 * | Receipt | `finance.ReceiptStatus` | `ReceiptStatus` | PENDING, RECEIVED |
 * | Credit Memo | `finance.CreditMemoStatus` | `CreditMemoStatus` | PENDING, APPLIED, PARTIAL, VOIDED, REFUNDED |
 * | Debit Memo | `finance.DebitMemoStatus` | `DebitMemoStatus` | PENDING, APPLIED, PARTIAL, VOIDED, REFUNDED |
 * | Gateway Payment | `finance.GatewayPaymentStatus` | `GatewayPaymentStatus` | PENDING, PAID, FAILED |
 * | Refund | `finance.RefundStatus` | `RefundStatus` | PENDING, SUCCESS, FAILED |
 *
 * #### Inventory Module
 *
 * | Entity | Backend Type | Frontend Type | Values |
 * |--------|--------------|---------------|--------|
 * | Stock Taking | `inventory.StockTakingStatus` | `StockTakingStatus` | DRAFT, COUNTING, PENDING_APPROVAL, APPROVED, REJECTED, CANCELLED |
 *
 * ## Usage Examples
 *
 * ### Using Money Type
 *
 * ```typescript
 * import { Money, createMoney, formatMoney, DEFAULT_CURRENCY } from '@/types'
 *
 * // Create a Money value
 * const price: Money = createMoney(199.99, 'CNY')
 *
 * // Display formatted
 * const display = formatMoney(price) // "¥199.99"
 *
 * // Compare Money values
 * import { moneyEquals } from '@/types'
 * const isEqual = moneyEquals(price, otherPrice)
 * ```
 *
 * ### Using Status Enums
 *
 * ```typescript
 * import { SalesOrderStatus, ProductStatus } from '@/types'
 *
 * // Type-safe status comparison
 * if (order.status === SalesOrderStatus.CONFIRMED) {
 *   // Handle confirmed order
 * }
 *
 * // Use as filter values
 * const activeProducts = products.filter(
 *   p => p.status === ProductStatus.ACTIVE
 * )
 *
 * // Status options for dropdown
 * const statusOptions = Object.values(SalesOrderStatus).map(status => ({
 *   value: status,
 *   label: getStatusLabel(status),
 * }))
 * ```
 *
 * ### Using Auto-generated API Types
 *
 * ```typescript
 * import type {
 *   HandlerSalesOrderResponse,
 *   HandlerCreateSalesOrderRequest,
 * } from '@/api/models'
 *
 * // Function with typed parameters
 * async function createOrder(
 *   request: HandlerCreateSalesOrderRequest
 * ): Promise<HandlerSalesOrderResponse> {
 *   const response = await tradeSalesOrdersAPI.createSalesOrder(request)
 *   return response.data
 * }
 * ```
 *
 * ## Maintenance Guidelines
 *
 * ### When to Update
 *
 * 1. **Backend API changes** - Regenerate orval types after backend changes
 * 2. **New status values** - Update `domain.ts` when backend adds new status values
 * 3. **New value objects** - Add new interfaces in `domain.ts`
 *
 * ### Best Practices
 *
 * 1. **Never edit auto-generated files** - Files in `src/api/` are overwritten
 * 2. **Use domain types** - Import from `@/types` not inline strings
 * 3. **Document changes** - Update this file when type mapping changes
 * 4. **Run type checks** - Verify with `npm run type-check` after updates
 *
 * ## Troubleshooting
 *
 * ### Type Mismatch Errors
 *
 * If you see type errors after regenerating:
 *
 * 1. Check backend OpenAPI annotations are correct
 * 2. Verify `swagger.yaml` was regenerated
 * 3. Clear orval cache and regenerate
 * 4. Check for breaking API changes in backend
 *
 * ### Missing Types
 *
 * If expected types are missing:
 *
 * 1. Ensure handler has complete swag annotations
 * 2. Run `make docs` in backend
 * 3. Run `npx orval` in frontend
 * 4. Check orval output for generation errors
 */

export {}
