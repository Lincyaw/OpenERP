package event

import (
	"github.com/erp/backend/internal/domain/finance"
	"github.com/erp/backend/internal/domain/inventory"
	"github.com/erp/backend/internal/domain/trade"
)

// RegisterAllEvents registers all domain event types with the serializer
// This is required for the OutboxProcessor to deserialize events from the outbox table
func RegisterAllEvents(serializer *EventSerializer) {
	// Trade domain - Sales Order events
	serializer.Register("SalesOrderCreated", &trade.SalesOrderCreatedEvent{})
	serializer.Register("SalesOrderConfirmed", &trade.SalesOrderConfirmedEvent{})
	serializer.Register("SalesOrderShipped", &trade.SalesOrderShippedEvent{})
	serializer.Register("SalesOrderCompleted", &trade.SalesOrderCompletedEvent{})
	serializer.Register("SalesOrderCancelled", &trade.SalesOrderCancelledEvent{})

	// Trade domain - Purchase Order events
	serializer.Register("PurchaseOrderCreated", &trade.PurchaseOrderCreatedEvent{})
	serializer.Register("PurchaseOrderConfirmed", &trade.PurchaseOrderConfirmedEvent{})
	serializer.Register("PurchaseOrderReceived", &trade.PurchaseOrderReceivedEvent{})
	serializer.Register("PurchaseOrderCompleted", &trade.PurchaseOrderCompletedEvent{})
	serializer.Register("PurchaseOrderCancelled", &trade.PurchaseOrderCancelledEvent{})

	// Trade domain - Sales Return events
	serializer.Register("SalesReturnCreated", &trade.SalesReturnCreatedEvent{})
	serializer.Register("SalesReturnSubmitted", &trade.SalesReturnSubmittedEvent{})
	serializer.Register("SalesReturnApproved", &trade.SalesReturnApprovedEvent{})
	serializer.Register("SalesReturnReceived", &trade.SalesReturnReceivedEvent{})
	serializer.Register("SalesReturnRejected", &trade.SalesReturnRejectedEvent{})
	serializer.Register("SalesReturnCompleted", &trade.SalesReturnCompletedEvent{})
	serializer.Register("SalesReturnCancelled", &trade.SalesReturnCancelledEvent{})

	// Trade domain - Purchase Return events
	serializer.Register("PurchaseReturnCreated", &trade.PurchaseReturnCreatedEvent{})
	serializer.Register("PurchaseReturnSubmitted", &trade.PurchaseReturnSubmittedEvent{})
	serializer.Register("PurchaseReturnApproved", &trade.PurchaseReturnApprovedEvent{})
	serializer.Register("PurchaseReturnRejected", &trade.PurchaseReturnRejectedEvent{})
	serializer.Register("PurchaseReturnShipped", &trade.PurchaseReturnShippedEvent{})
	serializer.Register("PurchaseReturnCompleted", &trade.PurchaseReturnCompletedEvent{})
	serializer.Register("PurchaseReturnCancelled", &trade.PurchaseReturnCancelledEvent{})

	// Inventory domain events
	serializer.Register("StockIncreased", &inventory.StockIncreasedEvent{})
	serializer.Register("StockDecreased", &inventory.StockDecreasedEvent{})
	serializer.Register("StockLocked", &inventory.StockLockedEvent{})
	serializer.Register("StockUnlocked", &inventory.StockUnlockedEvent{})
	serializer.Register("StockDeducted", &inventory.StockDeductedEvent{})
	serializer.Register("StockAdjusted", &inventory.StockAdjustedEvent{})
	serializer.Register("InventoryCostChanged", &inventory.InventoryCostChangedEvent{})
	serializer.Register("StockBelowThreshold", &inventory.StockBelowThresholdEvent{})

	// Inventory domain - Stock Taking events
	serializer.Register("StockTakingCreated", &inventory.StockTakingCreatedEvent{})
	serializer.Register("StockTakingStarted", &inventory.StockTakingStartedEvent{})
	serializer.Register("StockTakingSubmitted", &inventory.StockTakingSubmittedEvent{})
	serializer.Register("StockTakingApproved", &inventory.StockTakingApprovedEvent{})
	serializer.Register("StockTakingRejected", &inventory.StockTakingRejectedEvent{})
	serializer.Register("StockTakingCancelled", &inventory.StockTakingCancelledEvent{})

	// Finance domain - Account Receivable events
	serializer.Register("AccountReceivableCreated", &finance.AccountReceivableCreatedEvent{})
	serializer.Register("AccountReceivablePaid", &finance.AccountReceivablePaidEvent{})
	serializer.Register("AccountReceivablePartiallyPaid", &finance.AccountReceivablePartiallyPaidEvent{})
	serializer.Register("AccountReceivableReversed", &finance.AccountReceivableReversedEvent{})
	serializer.Register("AccountReceivableCancelled", &finance.AccountReceivableCancelledEvent{})

	// Finance domain - Account Payable events
	serializer.Register("AccountPayableCreated", &finance.AccountPayableCreatedEvent{})
	serializer.Register("AccountPayablePaid", &finance.AccountPayablePaidEvent{})
	serializer.Register("AccountPayablePartiallyPaid", &finance.AccountPayablePartiallyPaidEvent{})
	serializer.Register("AccountPayableReversed", &finance.AccountPayableReversedEvent{})
	serializer.Register("AccountPayableCancelled", &finance.AccountPayableCancelledEvent{})

	// Finance domain - Receipt Voucher events
	serializer.Register("ReceiptVoucherCreated", &finance.ReceiptVoucherCreatedEvent{})
	serializer.Register("ReceiptVoucherConfirmed", &finance.ReceiptVoucherConfirmedEvent{})
	serializer.Register("ReceiptVoucherAllocated", &finance.ReceiptVoucherAllocatedEvent{})
	serializer.Register("ReceiptVoucherCancelled", &finance.ReceiptVoucherCancelledEvent{})

	// Finance domain - Payment Voucher events
	serializer.Register("PaymentVoucherCreated", &finance.PaymentVoucherCreatedEvent{})
	serializer.Register("PaymentVoucherConfirmed", &finance.PaymentVoucherConfirmedEvent{})
	serializer.Register("PaymentVoucherAllocated", &finance.PaymentVoucherAllocatedEvent{})
	serializer.Register("PaymentVoucherCancelled", &finance.PaymentVoucherCancelledEvent{})

	// Finance domain - Credit Memo events
	serializer.Register("CreditMemoCreated", &finance.CreditMemoCreatedEvent{})
	serializer.Register("CreditMemoApplied", &finance.CreditMemoAppliedEvent{})
	serializer.Register("CreditMemoPartiallyApplied", &finance.CreditMemoPartiallyAppliedEvent{})
	serializer.Register("CreditMemoVoided", &finance.CreditMemoVoidedEvent{})
	serializer.Register("CreditMemoRefunded", &finance.CreditMemoRefundedEvent{})

	// Finance domain - Debit Memo events
	serializer.Register("DebitMemoCreated", &finance.DebitMemoCreatedEvent{})
	serializer.Register("DebitMemoApplied", &finance.DebitMemoAppliedEvent{})
	serializer.Register("DebitMemoPartiallyApplied", &finance.DebitMemoPartiallyAppliedEvent{})
	serializer.Register("DebitMemoVoided", &finance.DebitMemoVoidedEvent{})
	serializer.Register("DebitMemoRefundReceived", &finance.DebitMemoRefundReceivedEvent{})

	// Finance domain - Expense Record events
	serializer.Register("ExpenseRecordCreated", &finance.ExpenseRecordCreatedEvent{})
	serializer.Register("ExpenseRecordSubmitted", &finance.ExpenseRecordSubmittedEvent{})
	serializer.Register("ExpenseRecordApproved", &finance.ExpenseRecordApprovedEvent{})
	serializer.Register("ExpenseRecordRejected", &finance.ExpenseRecordRejectedEvent{})
	serializer.Register("ExpenseRecordCancelled", &finance.ExpenseRecordCancelledEvent{})
	serializer.Register("ExpenseRecordPaid", &finance.ExpenseRecordPaidEvent{})

	// Finance domain - Other Income Record events
	serializer.Register("OtherIncomeRecordCreated", &finance.OtherIncomeRecordCreatedEvent{})
	serializer.Register("OtherIncomeRecordConfirmed", &finance.OtherIncomeRecordConfirmedEvent{})
	serializer.Register("OtherIncomeRecordCancelled", &finance.OtherIncomeRecordCancelledEvent{})
	serializer.Register("OtherIncomeRecordReceived", &finance.OtherIncomeRecordReceivedEvent{})

	// Finance domain - Payment Gateway events
	serializer.Register("GatewayPaymentCompleted", &finance.GatewayPaymentCompletedEvent{})
	serializer.Register("GatewayRefundCompleted", &finance.GatewayRefundCompletedEvent{})
}
