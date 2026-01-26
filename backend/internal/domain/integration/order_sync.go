package integration

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Order Sync Types
// ---------------------------------------------------------------------------

// OrderSyncDirection represents the direction of order synchronization
type OrderSyncDirection string

const (
	// OrderSyncDirectionInbound indicates order is pulled from platform to ERP
	OrderSyncDirectionInbound OrderSyncDirection = "INBOUND"
	// OrderSyncDirectionOutbound indicates order/status is pushed from ERP to platform
	OrderSyncDirectionOutbound OrderSyncDirection = "OUTBOUND"
)

// IsValid returns true if the direction is valid
func (d OrderSyncDirection) IsValid() bool {
	switch d {
	case OrderSyncDirectionInbound, OrderSyncDirectionOutbound:
		return true
	default:
		return false
	}
}

// String returns the string representation of OrderSyncDirection
func (d OrderSyncDirection) String() string {
	return string(d)
}

// PlatformOrderSyncRecord represents a record of a platform order sync operation
type PlatformOrderSyncRecord struct {
	// ID is the unique identifier of the sync record
	ID uuid.UUID
	// TenantID is the tenant this record belongs to
	TenantID uuid.UUID
	// PlatformCode identifies the source platform
	PlatformCode PlatformCode
	// PlatformOrderID is the order ID on the platform
	PlatformOrderID string
	// LocalOrderID is our internal sales order ID (after conversion)
	LocalOrderID *uuid.UUID
	// LocalOrderNumber is our internal order number
	LocalOrderNumber string
	// Direction indicates if this was inbound (pull) or outbound (push)
	Direction OrderSyncDirection
	// Status is the sync status
	Status SyncStatus
	// PlatformStatus is the order status on the platform at sync time
	PlatformStatus PlatformOrderStatus
	// ErrorMessage contains any error during sync
	ErrorMessage string
	// SyncedAt is when the sync was attempted
	SyncedAt time.Time
	// CreatedAt is when this record was created
	CreatedAt time.Time
	// UpdatedAt is when this record was last updated
	UpdatedAt time.Time
}

// OrderConversionResult represents the result of converting a platform order
// to an internal sales order
type OrderConversionResult struct {
	// Success indicates if conversion was successful
	Success bool
	// LocalOrderID is the created sales order ID
	LocalOrderID uuid.UUID
	// LocalOrderNumber is the assigned order number
	LocalOrderNumber string
	// Warnings contains non-fatal issues encountered during conversion
	Warnings []string
	// ErrorMessage contains the error if Success is false
	ErrorMessage string
}

// OrderSyncConfig represents configuration for order synchronization
type OrderSyncConfig struct {
	// TenantID is the tenant this config belongs to
	TenantID uuid.UUID
	// PlatformCode is the platform this config applies to
	PlatformCode PlatformCode
	// IsEnabled indicates if sync is enabled
	IsEnabled bool
	// SyncIntervalMinutes is how often to sync (in minutes)
	SyncIntervalMinutes int
	// AutoCreateCustomer indicates if new customers should be auto-created
	AutoCreateCustomer bool
	// DefaultWarehouseID is the default warehouse for synced orders
	DefaultWarehouseID uuid.UUID
	// DefaultSalespersonID is the default salesperson for synced orders
	DefaultSalespersonID *uuid.UUID
	// AutoLockStock indicates if stock should be auto-locked on sync
	AutoLockStock bool
	// OrderPrefixFormat is the prefix format for converted order numbers
	// e.g., "TB{YYYYMMDD}" for Taobao orders
	OrderPrefixFormat string
	// StatusMappings maps platform status to internal workflow actions
	StatusMappings map[PlatformOrderStatus]string
}

// ---------------------------------------------------------------------------
// OrderSyncService Interface
// ---------------------------------------------------------------------------

// OrderSyncService defines the interface for synchronizing orders between
// external e-commerce platforms and the ERP system.
// This is an application service interface, not a port - it orchestrates
// the order sync workflow using the EcommercePlatform port.
type OrderSyncService interface {
	// ---------------------------------------------------------------------------
	// Inbound Sync (Platform -> ERP)
	// ---------------------------------------------------------------------------

	// PullOrders pulls orders from a platform and stores them for processing
	// This is typically called by a scheduler/cron job
	PullOrders(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode, startTime, endTime time.Time) (*SyncResult, error)

	// ConvertToSalesOrder converts a platform order to an internal sales order
	// This involves:
	//   1. Looking up product mappings
	//   2. Creating/finding the customer
	//   3. Converting addresses
	//   4. Creating the sales order
	//   5. Optionally locking stock
	ConvertToSalesOrder(ctx context.Context, tenantID uuid.UUID, platformOrder *PlatformOrder) (*OrderConversionResult, error)

	// GetPendingOrders returns platform orders that haven't been converted yet
	GetPendingOrders(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode) ([]PlatformOrder, error)

	// ---------------------------------------------------------------------------
	// Outbound Sync (ERP -> Platform)
	// ---------------------------------------------------------------------------

	// SyncShipmentToPlatform updates the platform with shipping information
	// when a sales order is shipped in the ERP
	SyncShipmentToPlatform(ctx context.Context, tenantID uuid.UUID, localOrderID uuid.UUID, shippingCompany, trackingNumber string) error

	// SyncOrderStatusToPlatform pushes order status updates to the platform
	SyncOrderStatusToPlatform(ctx context.Context, tenantID uuid.UUID, localOrderID uuid.UUID, status PlatformOrderStatus) error

	// ---------------------------------------------------------------------------
	// Sync Records
	// ---------------------------------------------------------------------------

	// GetSyncRecord retrieves a sync record by platform order ID
	GetSyncRecord(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode, platformOrderID string) (*PlatformOrderSyncRecord, error)

	// GetSyncRecordByLocalOrder retrieves a sync record by local order ID
	GetSyncRecordByLocalOrder(ctx context.Context, tenantID uuid.UUID, localOrderID uuid.UUID) (*PlatformOrderSyncRecord, error)

	// ListSyncRecords lists sync records with filtering
	ListSyncRecords(ctx context.Context, tenantID uuid.UUID, filter OrderSyncRecordFilter) ([]PlatformOrderSyncRecord, int64, error)

	// ---------------------------------------------------------------------------
	// Configuration
	// ---------------------------------------------------------------------------

	// GetSyncConfig retrieves sync configuration for a platform
	GetSyncConfig(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode) (*OrderSyncConfig, error)

	// UpdateSyncConfig updates sync configuration
	UpdateSyncConfig(ctx context.Context, config *OrderSyncConfig) error
}

// OrderSyncRecordFilter defines filter criteria for sync records
type OrderSyncRecordFilter struct {
	// PlatformCode filters by platform (optional)
	PlatformCode *PlatformCode
	// Direction filters by sync direction (optional)
	Direction *OrderSyncDirection
	// Status filters by sync status (optional)
	Status *SyncStatus
	// StartTime filters records from this time
	StartTime *time.Time
	// EndTime filters records until this time
	EndTime *time.Time
	// Page number (1-indexed)
	Page int
	// Page size
	PageSize int
}

// ---------------------------------------------------------------------------
// OrderSyncRepository Interface
// ---------------------------------------------------------------------------

// OrderSyncRecordRepository defines the interface for persisting order sync records
type OrderSyncRecordRepository interface {
	// Save creates or updates a sync record
	Save(ctx context.Context, record *PlatformOrderSyncRecord) error

	// FindByPlatformOrder finds a record by platform order ID
	FindByPlatformOrder(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode, platformOrderID string) (*PlatformOrderSyncRecord, error)

	// FindByLocalOrder finds a record by local order ID
	FindByLocalOrder(ctx context.Context, tenantID uuid.UUID, localOrderID uuid.UUID) (*PlatformOrderSyncRecord, error)

	// FindAll finds all records matching the filter
	FindAll(ctx context.Context, tenantID uuid.UUID, filter OrderSyncRecordFilter) ([]PlatformOrderSyncRecord, error)

	// Count counts records matching the filter
	Count(ctx context.Context, tenantID uuid.UUID, filter OrderSyncRecordFilter) (int64, error)

	// Delete deletes a sync record
	Delete(ctx context.Context, id uuid.UUID) error
}

// OrderSyncConfigRepository defines the interface for persisting sync configs
type OrderSyncConfigRepository interface {
	// Save creates or updates a sync config
	Save(ctx context.Context, config *OrderSyncConfig) error

	// FindByTenantAndPlatform finds a config by tenant and platform
	FindByTenantAndPlatform(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode) (*OrderSyncConfig, error)

	// FindAllForTenant finds all configs for a tenant
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID) ([]OrderSyncConfig, error)

	// Delete deletes a config
	Delete(ctx context.Context, tenantID uuid.UUID, platformCode PlatformCode) error
}
