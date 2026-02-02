package identity

import (
	"fmt"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// BatchOperationType represents the type of batch operation
type BatchOperationType string

const (
	BatchOperationTypeSuspend    BatchOperationType = "suspend"
	BatchOperationTypeActivate   BatchOperationType = "activate"
	BatchOperationTypeChangePlan BatchOperationType = "change_plan"
	BatchOperationTypeDelete     BatchOperationType = "delete"
)

// BatchOperationStatus represents the status of a batch operation
type BatchOperationStatus string

const (
	BatchOperationStatusPending    BatchOperationStatus = "pending"
	BatchOperationStatusConfirmed  BatchOperationStatus = "confirmed"
	BatchOperationStatusProcessing BatchOperationStatus = "processing"
	BatchOperationStatusCompleted  BatchOperationStatus = "completed"
	BatchOperationStatusFailed     BatchOperationStatus = "failed"
	BatchOperationStatusCancelled  BatchOperationStatus = "cancelled"
)

// BatchOperationItemStatus represents the status of an individual item in a batch operation
type BatchOperationItemStatus string

const (
	BatchItemStatusPending BatchOperationItemStatus = "pending"
	BatchItemStatusSuccess BatchOperationItemStatus = "success"
	BatchItemStatusFailed  BatchOperationItemStatus = "failed"
	BatchItemStatusSkipped BatchOperationItemStatus = "skipped"
)

// MaxBatchSize is the maximum number of tenants allowed in a single batch operation
const MaxBatchSize = 100

// BatchOperation represents a batch operation on tenants
type BatchOperation struct {
	ID              uuid.UUID             `json:"id"`
	OperationType   BatchOperationType    `json:"operation_type"`
	Status          BatchOperationStatus  `json:"status"`
	TotalCount      int                   `json:"total_count"`
	ProcessedCount  int                   `json:"processed_count"`
	SuccessCount    int                   `json:"success_count"`
	FailedCount     int                   `json:"failed_count"`
	SkippedCount    int                   `json:"skipped_count"`
	Payload         BatchOperationPayload `json:"payload"`
	Items           []BatchOperationItem  `json:"items"`
	CreatedByUserID uuid.UUID             `json:"created_by_user_id"`
	ConfirmedAt     *time.Time            `json:"confirmed_at,omitempty"`
	CompletedAt     *time.Time            `json:"completed_at,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// BatchOperationPayload contains operation-specific data
type BatchOperationPayload struct {
	// For suspend operation
	SuspendReason         string     `json:"suspend_reason,omitempty"`
	ScheduledReactivateAt *time.Time `json:"scheduled_reactivate_at,omitempty"`

	// For change plan operation
	NewPlan      string `json:"new_plan,omitempty"`
	ChangeReason string `json:"change_reason,omitempty"`
}

// BatchOperationItem represents a single tenant in a batch operation
type BatchOperationItem struct {
	ID           uuid.UUID                `json:"id"`
	TenantID     uuid.UUID                `json:"tenant_id"`
	TenantCode   string                   `json:"tenant_code"`
	TenantName   string                   `json:"tenant_name"`
	Status       BatchOperationItemStatus `json:"status"`
	ErrorMessage string                   `json:"error_message,omitempty"`
	ProcessedAt  *time.Time               `json:"processed_at,omitempty"`
}

// BatchOperationPreview represents a preview of a batch operation before confirmation
type BatchOperationPreview struct {
	OperationType   BatchOperationType   `json:"operation_type"`
	TotalCount      int                  `json:"total_count"`
	AffectedTenants []BatchPreviewTenant `json:"affected_tenants"`
	Warnings        []string             `json:"warnings,omitempty"`
}

// BatchPreviewTenant represents a tenant in the batch preview
type BatchPreviewTenant struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	CurrentStatus string    `json:"current_status"`
	CurrentPlan   string    `json:"current_plan,omitempty"`
	CanProcess    bool      `json:"can_process"`
	SkipReason    string    `json:"skip_reason,omitempty"`
}

// NewBatchOperation creates a new batch operation
func NewBatchOperation(
	operationType BatchOperationType,
	tenantIDs []uuid.UUID,
	payload BatchOperationPayload,
	createdByUserID uuid.UUID,
) (*BatchOperation, error) {
	if len(tenantIDs) == 0 {
		return nil, shared.NewDomainError("EMPTY_BATCH", "At least one tenant must be specified")
	}
	if len(tenantIDs) > MaxBatchSize {
		return nil, shared.NewDomainError("BATCH_SIZE_EXCEEDED", fmt.Sprintf("Maximum batch size is %d tenants", MaxBatchSize))
	}

	now := time.Now()
	items := make([]BatchOperationItem, len(tenantIDs))
	for i, tenantID := range tenantIDs {
		items[i] = BatchOperationItem{
			ID:       uuid.New(),
			TenantID: tenantID,
			Status:   BatchItemStatusPending,
		}
	}

	return &BatchOperation{
		ID:              uuid.New(),
		OperationType:   operationType,
		Status:          BatchOperationStatusPending,
		TotalCount:      len(tenantIDs),
		ProcessedCount:  0,
		SuccessCount:    0,
		FailedCount:     0,
		SkippedCount:    0,
		Payload:         payload,
		Items:           items,
		CreatedByUserID: createdByUserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Confirm confirms the batch operation for execution
func (b *BatchOperation) Confirm() error {
	if b.Status != BatchOperationStatusPending {
		return shared.NewDomainError("INVALID_STATUS", "Batch operation must be pending to confirm")
	}
	now := time.Now()
	b.Status = BatchOperationStatusConfirmed
	b.ConfirmedAt = &now
	b.UpdatedAt = now
	return nil
}

// StartProcessing marks the batch operation as processing
func (b *BatchOperation) StartProcessing() error {
	if b.Status != BatchOperationStatusConfirmed {
		return shared.NewDomainError("INVALID_STATUS", "Batch operation must be confirmed to start processing")
	}
	b.Status = BatchOperationStatusProcessing
	b.UpdatedAt = time.Now()
	return nil
}

// MarkItemSuccess marks an item as successfully processed
func (b *BatchOperation) MarkItemSuccess(tenantID uuid.UUID) {
	for i := range b.Items {
		if b.Items[i].TenantID == tenantID {
			now := time.Now()
			b.Items[i].Status = BatchItemStatusSuccess
			b.Items[i].ProcessedAt = &now
			b.ProcessedCount++
			b.SuccessCount++
			b.UpdatedAt = now
			break
		}
	}
}

// MarkItemFailed marks an item as failed
func (b *BatchOperation) MarkItemFailed(tenantID uuid.UUID, errorMessage string) {
	for i := range b.Items {
		if b.Items[i].TenantID == tenantID {
			now := time.Now()
			b.Items[i].Status = BatchItemStatusFailed
			b.Items[i].ErrorMessage = errorMessage
			b.Items[i].ProcessedAt = &now
			b.ProcessedCount++
			b.FailedCount++
			b.UpdatedAt = now
			break
		}
	}
}

// MarkItemSkipped marks an item as skipped
func (b *BatchOperation) MarkItemSkipped(tenantID uuid.UUID, reason string) {
	for i := range b.Items {
		if b.Items[i].TenantID == tenantID {
			now := time.Now()
			b.Items[i].Status = BatchItemStatusSkipped
			b.Items[i].ErrorMessage = reason
			b.Items[i].ProcessedAt = &now
			b.ProcessedCount++
			b.SkippedCount++
			b.UpdatedAt = now
			break
		}
	}
}

// Complete marks the batch operation as completed
func (b *BatchOperation) Complete() {
	now := time.Now()
	if b.FailedCount > 0 && b.SuccessCount == 0 {
		b.Status = BatchOperationStatusFailed
	} else {
		b.Status = BatchOperationStatusCompleted
	}
	b.CompletedAt = &now
	b.UpdatedAt = now
}

// Cancel cancels the batch operation
func (b *BatchOperation) Cancel() error {
	if b.Status == BatchOperationStatusProcessing || b.Status == BatchOperationStatusCompleted {
		return shared.NewDomainError("CANNOT_CANCEL", "Cannot cancel a batch operation that is processing or completed")
	}
	b.Status = BatchOperationStatusCancelled
	b.UpdatedAt = time.Now()
	return nil
}

// GetProgress returns the progress percentage
func (b *BatchOperation) GetProgress() float64 {
	if b.TotalCount == 0 {
		return 100.0
	}
	return float64(b.ProcessedCount) / float64(b.TotalCount) * 100.0
}

// IsComplete returns true if the batch operation is complete
func (b *BatchOperation) IsComplete() bool {
	return b.Status == BatchOperationStatusCompleted || b.Status == BatchOperationStatusFailed
}

// SetTenantInfo sets tenant information for items (called after loading tenant data)
func (b *BatchOperation) SetTenantInfo(tenantID uuid.UUID, code, name string) {
	for i := range b.Items {
		if b.Items[i].TenantID == tenantID {
			b.Items[i].TenantCode = code
			b.Items[i].TenantName = name
			break
		}
	}
}
