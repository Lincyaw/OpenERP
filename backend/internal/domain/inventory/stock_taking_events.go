package inventory

import (
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregate type constant for StockTaking
const AggregateTypeStockTaking = "StockTaking"

// StockTaking event type constants
const (
	EventTypeStockTakingCreated   = "StockTakingCreated"
	EventTypeStockTakingStarted   = "StockTakingStarted"
	EventTypeStockTakingSubmitted = "StockTakingSubmitted"
	EventTypeStockTakingApproved  = "StockTakingApproved"
	EventTypeStockTakingRejected  = "StockTakingRejected"
	EventTypeStockTakingCancelled = "StockTakingCancelled"
)

// StockTakingCreatedEvent is raised when a stock taking is created
type StockTakingCreatedEvent struct {
	shared.BaseDomainEvent
	StockTakingID uuid.UUID `json:"stock_taking_id"`
	TakingNumber  string    `json:"taking_number"`
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	CreatedByID   uuid.UUID `json:"created_by_id"`
	CreatedByName string    `json:"created_by_name"`
}

// NewStockTakingCreatedEvent creates a new StockTakingCreatedEvent
func NewStockTakingCreatedEvent(st *StockTaking) *StockTakingCreatedEvent {
	return &StockTakingCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockTakingCreated, AggregateTypeStockTaking, st.ID, st.TenantID),
		StockTakingID:   st.ID,
		TakingNumber:    st.TakingNumber,
		WarehouseID:     st.WarehouseID,
		WarehouseName:   st.WarehouseName,
		CreatedByID:     st.CreatedByID,
		CreatedByName:   st.CreatedByName,
	}
}

// EventType returns the event type name
func (e *StockTakingCreatedEvent) EventType() string {
	return EventTypeStockTakingCreated
}

// StockTakingStartedEvent is raised when stock taking counting starts
type StockTakingStartedEvent struct {
	shared.BaseDomainEvent
	StockTakingID uuid.UUID `json:"stock_taking_id"`
	TakingNumber  string    `json:"taking_number"`
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	TotalItems    int       `json:"total_items"`
}

// NewStockTakingStartedEvent creates a new StockTakingStartedEvent
func NewStockTakingStartedEvent(st *StockTaking) *StockTakingStartedEvent {
	return &StockTakingStartedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockTakingStarted, AggregateTypeStockTaking, st.ID, st.TenantID),
		StockTakingID:   st.ID,
		TakingNumber:    st.TakingNumber,
		WarehouseID:     st.WarehouseID,
		TotalItems:      st.TotalItems,
	}
}

// EventType returns the event type name
func (e *StockTakingStartedEvent) EventType() string {
	return EventTypeStockTakingStarted
}

// StockTakingSubmittedEvent is raised when stock taking is submitted for approval
type StockTakingSubmittedEvent struct {
	shared.BaseDomainEvent
	StockTakingID   uuid.UUID       `json:"stock_taking_id"`
	TakingNumber    string          `json:"taking_number"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	TotalItems      int             `json:"total_items"`
	DifferenceItems int             `json:"difference_items"`
	TotalDifference decimal.Decimal `json:"total_difference"`
}

// NewStockTakingSubmittedEvent creates a new StockTakingSubmittedEvent
func NewStockTakingSubmittedEvent(st *StockTaking) *StockTakingSubmittedEvent {
	return &StockTakingSubmittedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockTakingSubmitted, AggregateTypeStockTaking, st.ID, st.TenantID),
		StockTakingID:   st.ID,
		TakingNumber:    st.TakingNumber,
		WarehouseID:     st.WarehouseID,
		TotalItems:      st.TotalItems,
		DifferenceItems: st.DifferenceItems,
		TotalDifference: st.TotalDifference,
	}
}

// EventType returns the event type name
func (e *StockTakingSubmittedEvent) EventType() string {
	return EventTypeStockTakingSubmitted
}

// StockTakingApprovedEvent is raised when stock taking is approved
// This event should trigger inventory adjustments
type StockTakingApprovedEvent struct {
	shared.BaseDomainEvent
	StockTakingID   uuid.UUID       `json:"stock_taking_id"`
	TakingNumber    string          `json:"taking_number"`
	WarehouseID     uuid.UUID       `json:"warehouse_id"`
	ApprovedByID    uuid.UUID       `json:"approved_by_id"`
	ApprovedByName  string          `json:"approved_by_name"`
	DifferenceItems int             `json:"difference_items"`
	TotalDifference decimal.Decimal `json:"total_difference"`
}

// NewStockTakingApprovedEvent creates a new StockTakingApprovedEvent
func NewStockTakingApprovedEvent(st *StockTaking) *StockTakingApprovedEvent {
	var approverID uuid.UUID
	if st.ApprovedByID != nil {
		approverID = *st.ApprovedByID
	}
	return &StockTakingApprovedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockTakingApproved, AggregateTypeStockTaking, st.ID, st.TenantID),
		StockTakingID:   st.ID,
		TakingNumber:    st.TakingNumber,
		WarehouseID:     st.WarehouseID,
		ApprovedByID:    approverID,
		ApprovedByName:  st.ApprovedByName,
		DifferenceItems: st.DifferenceItems,
		TotalDifference: st.TotalDifference,
	}
}

// EventType returns the event type name
func (e *StockTakingApprovedEvent) EventType() string {
	return EventTypeStockTakingApproved
}

// StockTakingRejectedEvent is raised when stock taking is rejected
type StockTakingRejectedEvent struct {
	shared.BaseDomainEvent
	StockTakingID  uuid.UUID `json:"stock_taking_id"`
	TakingNumber   string    `json:"taking_number"`
	WarehouseID    uuid.UUID `json:"warehouse_id"`
	RejectedByID   uuid.UUID `json:"rejected_by_id"`
	RejectedByName string    `json:"rejected_by_name"`
	Reason         string    `json:"reason"`
}

// NewStockTakingRejectedEvent creates a new StockTakingRejectedEvent
func NewStockTakingRejectedEvent(st *StockTaking) *StockTakingRejectedEvent {
	var rejectedByID uuid.UUID
	if st.ApprovedByID != nil {
		rejectedByID = *st.ApprovedByID
	}
	return &StockTakingRejectedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockTakingRejected, AggregateTypeStockTaking, st.ID, st.TenantID),
		StockTakingID:   st.ID,
		TakingNumber:    st.TakingNumber,
		WarehouseID:     st.WarehouseID,
		RejectedByID:    rejectedByID,
		RejectedByName:  st.ApprovedByName,
		Reason:          st.ApprovalNote,
	}
}

// EventType returns the event type name
func (e *StockTakingRejectedEvent) EventType() string {
	return EventTypeStockTakingRejected
}

// StockTakingCancelledEvent is raised when stock taking is cancelled
type StockTakingCancelledEvent struct {
	shared.BaseDomainEvent
	StockTakingID uuid.UUID `json:"stock_taking_id"`
	TakingNumber  string    `json:"taking_number"`
	WarehouseID   uuid.UUID `json:"warehouse_id"`
	Reason        string    `json:"reason"`
}

// NewStockTakingCancelledEvent creates a new StockTakingCancelledEvent
func NewStockTakingCancelledEvent(st *StockTaking) *StockTakingCancelledEvent {
	return &StockTakingCancelledEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeStockTakingCancelled, AggregateTypeStockTaking, st.ID, st.TenantID),
		StockTakingID:   st.ID,
		TakingNumber:    st.TakingNumber,
		WarehouseID:     st.WarehouseID,
		Reason:          st.Remark,
	}
}

// EventType returns the event type name
func (e *StockTakingCancelledEvent) EventType() string {
	return EventTypeStockTakingCancelled
}
