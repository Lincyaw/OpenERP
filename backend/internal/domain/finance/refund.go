package finance

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RefundRecordStatus represents the status of a refund record
type RefundRecordStatus string

const (
	// RefundRecordStatusPending indicates refund is being processed by gateway
	RefundRecordStatusPending RefundRecordStatus = "PENDING"
	// RefundRecordStatusProcessing indicates gateway is processing refund
	RefundRecordStatusProcessing RefundRecordStatus = "PROCESSING"
	// RefundRecordStatusSuccess indicates refund was successful
	RefundRecordStatusSuccess RefundRecordStatus = "SUCCESS"
	// RefundRecordStatusFailed indicates refund failed
	RefundRecordStatusFailed RefundRecordStatus = "FAILED"
	// RefundRecordStatusClosed indicates refund was closed/cancelled
	RefundRecordStatusClosed RefundRecordStatus = "CLOSED"
)

// IsValid checks if the status is a valid RefundRecordStatus
func (s RefundRecordStatus) IsValid() bool {
	switch s {
	case RefundRecordStatusPending, RefundRecordStatusProcessing,
		RefundRecordStatusSuccess, RefundRecordStatusFailed, RefundRecordStatusClosed:
		return true
	}
	return false
}

// String returns the string representation of RefundRecordStatus
func (s RefundRecordStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the refund is in a terminal state
func (s RefundRecordStatus) IsTerminal() bool {
	return s == RefundRecordStatusSuccess || s == RefundRecordStatusFailed || s == RefundRecordStatusClosed
}

// RefundSourceType represents the source type of a refund
type RefundSourceType string

const (
	// RefundSourceTypeSalesReturn indicates refund from sales return
	RefundSourceTypeSalesReturn RefundSourceType = "SALES_RETURN"
	// RefundSourceTypeCreditMemo indicates refund from credit memo
	RefundSourceTypeCreditMemo RefundSourceType = "CREDIT_MEMO"
	// RefundSourceTypeOrderCancel indicates refund from order cancellation
	RefundSourceTypeOrderCancel RefundSourceType = "ORDER_CANCEL"
	// RefundSourceTypeManual indicates manual refund
	RefundSourceTypeManual RefundSourceType = "MANUAL"
)

// IsValid checks if the source type is valid
func (t RefundSourceType) IsValid() bool {
	switch t {
	case RefundSourceTypeSalesReturn, RefundSourceTypeCreditMemo,
		RefundSourceTypeOrderCancel, RefundSourceTypeManual:
		return true
	}
	return false
}

// String returns the string representation of RefundSourceType
func (t RefundSourceType) String() string {
	return string(t)
}

// RefundRecord represents a refund record aggregate root
// It tracks refunds made through payment gateways (WeChat/Alipay)
// and links them to original payments and source documents (sales returns, credit memos)
type RefundRecord struct {
	shared.TenantAggregateRoot

	// Refund identification
	RefundNumber string `json:"refund_number"` // Internal refund reference number

	// Original payment reference
	OriginalPaymentID   *uuid.UUID `json:"original_payment_id"`   // Reference to original ReceiptVoucher
	OriginalOrderID     uuid.UUID  `json:"original_order_id"`     // Original sales order ID
	OriginalOrderNumber string     `json:"original_order_number"` // Original sales order number

	// Source document reference (what triggered this refund)
	SourceType   RefundSourceType `json:"source_type"`   // Type of source document
	SourceID     uuid.UUID        `json:"source_id"`     // ID of source document
	SourceNumber string           `json:"source_number"` // Number of source document

	// Customer information
	CustomerID   uuid.UUID `json:"customer_id"`
	CustomerName string    `json:"customer_name"`

	// Refund amounts
	RefundAmount       decimal.Decimal `json:"refund_amount"`        // Amount to be refunded
	ActualRefundAmount decimal.Decimal `json:"actual_refund_amount"` // Actual amount refunded (may differ)
	Currency           string          `json:"currency"`

	// Gateway information
	GatewayType          PaymentGatewayType `json:"gateway_type"`           // WeChat/Alipay
	GatewayRefundID      string             `json:"gateway_refund_id"`      // Refund ID from gateway
	GatewayOrderID       string             `json:"gateway_order_id"`       // Original order ID at gateway
	GatewayTransactionID string             `json:"gateway_transaction_id"` // Original transaction ID

	// Status tracking
	Status      RefundRecordStatus `json:"status"`
	Reason      string             `json:"reason"`       // Reason for refund
	Remark      string             `json:"remark"`       // Additional notes
	FailReason  string             `json:"fail_reason"`  // Reason if refund failed
	RawResponse string             `json:"raw_response"` // Gateway raw response (JSON)

	// Timestamps
	RequestedAt *time.Time `json:"requested_at"` // When refund was requested
	CompletedAt *time.Time `json:"completed_at"` // When refund was completed
	FailedAt    *time.Time `json:"failed_at"`    // When refund failed
}

// NewRefundRecord creates a new refund record
func NewRefundRecord(
	tenantID uuid.UUID,
	refundNumber string,
	originalOrderID uuid.UUID,
	originalOrderNumber string,
	sourceType RefundSourceType,
	sourceID uuid.UUID,
	sourceNumber string,
	customerID uuid.UUID,
	customerName string,
	refundAmount decimal.Decimal,
	gatewayType PaymentGatewayType,
	reason string,
) (*RefundRecord, error) {
	// Validate inputs
	if refundNumber == "" {
		return nil, shared.NewDomainError("INVALID_REFUND_NUMBER", "Refund number cannot be empty")
	}
	if len(refundNumber) > 50 {
		return nil, shared.NewDomainError("INVALID_REFUND_NUMBER", "Refund number cannot exceed 50 characters")
	}
	if originalOrderID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_ORIGINAL_ORDER", "Original order ID cannot be empty")
	}
	if !sourceType.IsValid() {
		return nil, shared.NewDomainError("INVALID_SOURCE_TYPE", "Invalid source type")
	}
	if sourceID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_SOURCE", "Source ID cannot be empty")
	}
	if customerID == uuid.Nil {
		return nil, shared.NewDomainError("INVALID_CUSTOMER", "Customer ID cannot be empty")
	}
	if customerName == "" {
		return nil, shared.NewDomainError("INVALID_CUSTOMER_NAME", "Customer name cannot be empty")
	}
	if refundAmount.LessThanOrEqual(decimal.Zero) {
		return nil, shared.NewDomainError("INVALID_AMOUNT", "Refund amount must be positive")
	}
	if !gatewayType.IsValid() {
		return nil, shared.NewDomainError("INVALID_GATEWAY_TYPE", "Invalid payment gateway type")
	}

	now := time.Now()
	rr := &RefundRecord{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		RefundNumber:        refundNumber,
		OriginalOrderID:     originalOrderID,
		OriginalOrderNumber: originalOrderNumber,
		SourceType:          sourceType,
		SourceID:            sourceID,
		SourceNumber:        sourceNumber,
		CustomerID:          customerID,
		CustomerName:        customerName,
		RefundAmount:        refundAmount,
		ActualRefundAmount:  decimal.Zero,
		Currency:            "CNY",
		GatewayType:         gatewayType,
		Status:              RefundRecordStatusPending,
		Reason:              reason,
		RequestedAt:         &now,
	}

	rr.AddDomainEvent(NewRefundRecordCreatedEvent(rr))

	return rr, nil
}

// NewRefundRecordFromCallback creates a refund record from a gateway callback
// This is used when we receive a refund callback but don't have an existing record
func NewRefundRecordFromCallback(
	tenantID uuid.UUID,
	refundNumber string,
	callback *RefundCallback,
) (*RefundRecord, error) {
	if refundNumber == "" {
		return nil, shared.NewDomainError("INVALID_REFUND_NUMBER", "Refund number cannot be empty")
	}
	if callback == nil {
		return nil, shared.NewDomainError("INVALID_CALLBACK", "Callback cannot be nil")
	}

	now := time.Now()
	rr := &RefundRecord{
		TenantAggregateRoot:  shared.NewTenantAggregateRoot(tenantID),
		RefundNumber:         refundNumber,
		RefundAmount:         callback.RefundAmount,
		ActualRefundAmount:   callback.RefundAmount,
		Currency:             "CNY",
		GatewayType:          callback.GatewayType,
		GatewayRefundID:      callback.GatewayRefundID,
		GatewayOrderID:       callback.GatewayOrderID,
		GatewayTransactionID: callback.GatewayTransactionID,
		Status:               RefundRecordStatusSuccess,
		RawResponse:          callback.RawPayload,
		RequestedAt:          &now,
		CompletedAt:          callback.RefundedAt,
	}

	if rr.CompletedAt == nil {
		rr.CompletedAt = &now
	}

	return rr, nil
}

// SetOriginalPayment links this refund to the original payment
func (r *RefundRecord) SetOriginalPayment(paymentID uuid.UUID) error {
	if paymentID == uuid.Nil {
		return shared.NewDomainError("INVALID_PAYMENT", "Payment ID cannot be empty")
	}

	r.OriginalPaymentID = &paymentID
	r.UpdatedAt = time.Now()

	return nil
}

// SetGatewayInfo sets the gateway response information
func (r *RefundRecord) SetGatewayInfo(gatewayRefundID, gatewayOrderID, gatewayTransactionID string) {
	r.GatewayRefundID = gatewayRefundID
	r.GatewayOrderID = gatewayOrderID
	r.GatewayTransactionID = gatewayTransactionID
	r.UpdatedAt = time.Now()
}

// MarkProcessing marks the refund as being processed by the gateway
func (r *RefundRecord) MarkProcessing(gatewayRefundID string) error {
	if r.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot mark refund as processing in %s status", r.Status))
	}

	r.Status = RefundRecordStatusProcessing
	r.GatewayRefundID = gatewayRefundID
	r.UpdatedAt = time.Now()

	return nil
}

// Complete marks the refund as successfully completed
func (r *RefundRecord) Complete(actualAmount decimal.Decimal, rawResponse string) error {
	if r.Status == RefundRecordStatusSuccess {
		return nil // Already completed, idempotent
	}
	if r.Status == RefundRecordStatusFailed || r.Status == RefundRecordStatusClosed {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot complete refund in %s status", r.Status))
	}

	now := time.Now()
	r.Status = RefundRecordStatusSuccess
	r.ActualRefundAmount = actualAmount
	r.RawResponse = rawResponse
	r.CompletedAt = &now
	r.UpdatedAt = now

	r.AddDomainEvent(NewRefundRecordCompletedEvent(r))

	return nil
}

// CompleteFromCallback updates the refund record from a successful callback
func (r *RefundRecord) CompleteFromCallback(callback *RefundCallback) error {
	if callback == nil {
		return shared.NewDomainError("INVALID_CALLBACK", "Callback cannot be nil")
	}

	// Update gateway info
	r.GatewayRefundID = callback.GatewayRefundID
	r.GatewayOrderID = callback.GatewayOrderID
	r.GatewayTransactionID = callback.GatewayTransactionID
	r.RawResponse = callback.RawPayload

	// Complete the refund
	return r.Complete(callback.RefundAmount, callback.RawPayload)
}

// Fail marks the refund as failed
func (r *RefundRecord) Fail(reason, rawResponse string) error {
	if r.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot fail refund in %s status", r.Status))
	}

	now := time.Now()
	r.Status = RefundRecordStatusFailed
	r.FailReason = reason
	r.RawResponse = rawResponse
	r.FailedAt = &now
	r.UpdatedAt = now

	r.AddDomainEvent(NewRefundRecordFailedEvent(r))

	return nil
}

// Close closes/cancels a pending refund
func (r *RefundRecord) Close(reason string) error {
	if r.Status.IsTerminal() {
		return shared.NewDomainError("INVALID_STATE", fmt.Sprintf("Cannot close refund in %s status", r.Status))
	}

	r.Status = RefundRecordStatusClosed
	r.FailReason = reason
	r.UpdatedAt = time.Now()

	return nil
}

// SetRemark sets the remark
func (r *RefundRecord) SetRemark(remark string) {
	r.Remark = remark
	r.UpdatedAt = time.Now()
}

// Helper methods

// IsPending returns true if refund is pending
func (r *RefundRecord) IsPending() bool {
	return r.Status == RefundRecordStatusPending
}

// IsProcessing returns true if refund is being processed
func (r *RefundRecord) IsProcessing() bool {
	return r.Status == RefundRecordStatusProcessing
}

// IsSuccess returns true if refund was successful
func (r *RefundRecord) IsSuccess() bool {
	return r.Status == RefundRecordStatusSuccess
}

// IsFailed returns true if refund failed
func (r *RefundRecord) IsFailed() bool {
	return r.Status == RefundRecordStatusFailed
}

// IsClosed returns true if refund was closed
func (r *RefundRecord) IsClosed() bool {
	return r.Status == RefundRecordStatusClosed
}

// CanRetry returns true if the refund can be retried
func (r *RefundRecord) CanRetry() bool {
	return r.Status == RefundRecordStatusFailed
}
