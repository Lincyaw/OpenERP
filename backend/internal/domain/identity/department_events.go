package identity

import (
	"github.com/erp/backend/internal/domain/shared"
)

// Aggregate type constant for Department
const AggregateTypeDepartment = "Department"

// Department domain event types
const (
	EventTypeDepartmentCreated        = "DepartmentCreated"
	EventTypeDepartmentUpdated        = "DepartmentUpdated"
	EventTypeDepartmentManagerChanged = "DepartmentManagerChanged"
	EventTypeDepartmentDeleted        = "DepartmentDeleted"
)

// DepartmentCreatedEvent is raised when a new department is created
type DepartmentCreatedEvent struct {
	shared.BaseDomainEvent
	Code string `json:"code"`
	Name string `json:"name"`
}

// NewDepartmentCreatedEvent creates a new DepartmentCreatedEvent
func NewDepartmentCreatedEvent(dept *Department) *DepartmentCreatedEvent {
	return &DepartmentCreatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeDepartmentCreated, AggregateTypeDepartment, dept.ID, dept.TenantID),
		Code:            dept.Code,
		Name:            dept.Name,
	}
}

// DepartmentUpdatedEvent is raised when a department is updated
type DepartmentUpdatedEvent struct {
	shared.BaseDomainEvent
	Name        string `json:"name"`
	Description string `json:"description"`
}

// NewDepartmentUpdatedEvent creates a new DepartmentUpdatedEvent
func NewDepartmentUpdatedEvent(dept *Department) *DepartmentUpdatedEvent {
	return &DepartmentUpdatedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeDepartmentUpdated, AggregateTypeDepartment, dept.ID, dept.TenantID),
		Name:            dept.Name,
		Description:     dept.Description,
	}
}

// DepartmentManagerChangedEvent is raised when a department's manager changes
type DepartmentManagerChangedEvent struct {
	shared.BaseDomainEvent
	ManagerID string `json:"manager_id"`
}

// NewDepartmentManagerChangedEvent creates a new DepartmentManagerChangedEvent
func NewDepartmentManagerChangedEvent(dept *Department) *DepartmentManagerChangedEvent {
	managerID := ""
	if dept.ManagerID != nil {
		managerID = dept.ManagerID.String()
	}

	return &DepartmentManagerChangedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeDepartmentManagerChanged, AggregateTypeDepartment, dept.ID, dept.TenantID),
		ManagerID:       managerID,
	}
}

// DepartmentDeletedEvent is raised when a department is deleted
type DepartmentDeletedEvent struct {
	shared.BaseDomainEvent
	Code string `json:"code"`
}

// NewDepartmentDeletedEvent creates a new DepartmentDeletedEvent
func NewDepartmentDeletedEvent(dept *Department) *DepartmentDeletedEvent {
	return &DepartmentDeletedEvent{
		BaseDomainEvent: shared.NewBaseDomainEvent(EventTypeDepartmentDeleted, AggregateTypeDepartment, dept.ID, dept.TenantID),
		Code:            dept.Code,
	}
}
