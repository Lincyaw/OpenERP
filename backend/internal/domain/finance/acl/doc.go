// Package acl provides the Anti-Corruption Layer (ACL) for the Finance bounded context.
//
// DDD-H04: Cross-context reference with ACL
//
// # Overview
//
// In Domain-Driven Design, an Anti-Corruption Layer protects a bounded context from
// being polluted by models and concepts from other contexts. This package provides
// the ACL components that isolate the Finance context from the Partner context
// (which owns the Customer aggregate).
//
// # Why ACL?
//
// The Finance context needs customer information for:
//   - Account Receivables: Track money owed by customers
//   - Receipt Vouchers: Record payments from customers
//   - Credit Memos: Handle customer returns/refunds
//
// Without ACL, the Finance domain would directly depend on the Partner domain's
// Customer aggregate, creating tight coupling and making changes in Partner
// propagate to Finance.
//
// # Components
//
// CustomerID: A value object that wraps uuid.UUID, providing type safety and
// semantic meaning within the Finance context.
//
// CustomerReference: A value object containing denormalized customer information
// (ID, name, code) needed by Finance. This is the Finance context's local view
// of a customer.
//
// CustomerQueryService: Interface for querying customer information. Implemented
// in the infrastructure layer, it queries Partner context and caches results.
//
// CustomerReferenceCache: Interface for caching customer references locally.
// Reduces queries to Partner context and enables eventual consistency.
//
// CustomerEventHandler: Interface for handling customer events from Partner context.
// Maintains cache and optionally updates denormalized data in Finance aggregates.
//
// # Event-Driven Updates
//
// The ACL subscribes to customer events from the Partner context:
//
//	CustomerCreated    -> Add to local cache
//	CustomerUpdated    -> Update local cache
//	CustomerDeleted    -> Invalidate cache entry
//	CustomerStatusChanged -> Update/invalidate based on status
//
// This event-driven approach ensures:
//  1. Loose coupling: Finance doesn't query Partner synchronously
//  2. Eventual consistency: Customer info is kept up-to-date
//  3. Resilience: Finance can operate with cached data if Partner is unavailable
//
// # Usage Example
//
//	// Creating a receivable with customer reference
//	customerRef, err := queryService.GetCustomerReference(ctx, tenantID, customerID)
//	if err != nil {
//	    return err
//	}
//	receivable, err := finance.NewAccountReceivableWithRef(
//	    tenantID,
//	    number,
//	    customerRef,  // Pass the reference, not raw ID
//	    sourceType,
//	    sourceID,
//	    sourceNumber,
//	    amount,
//	    dueDate,
//	)
//
// # Future Considerations
//
// The ACL can be extended to include:
//   - SupplierReference: For Account Payables (similar pattern)
//   - CustomerCreditInfo: Credit limit and status for credit checks
//   - CustomerPricingTier: For Finance-specific pricing calculations
package acl
