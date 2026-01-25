package partner

import (
	"time"

	"github.com/erp/backend/internal/domain/partner"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// =============================================================================
// Customer DTOs
// =============================================================================

// CreateCustomerRequest represents a request to create a new customer
type CreateCustomerRequest struct {
	Code        string           `json:"code" binding:"required,min=1,max=50"`
	Name        string           `json:"name" binding:"required,min=1,max=200"`
	ShortName   string           `json:"short_name" binding:"max=100"`
	Type        string           `json:"type" binding:"required,oneof=individual organization"`
	ContactName string           `json:"contact_name" binding:"max=100"`
	Phone       string           `json:"phone" binding:"max=50"`
	Email       string           `json:"email" binding:"omitempty,email,max=200"`
	Address     string           `json:"address" binding:"max=500"`
	City        string           `json:"city" binding:"max=100"`
	Province    string           `json:"province" binding:"max=100"`
	PostalCode  string           `json:"postal_code" binding:"max=20"`
	Country     string           `json:"country" binding:"max=100"`
	TaxID       string           `json:"tax_id" binding:"max=50"`
	CreditLimit *decimal.Decimal `json:"credit_limit"`
	Notes       string           `json:"notes"`
	SortOrder   *int             `json:"sort_order"`
	Attributes  string           `json:"attributes"`
	CreatedBy   *uuid.UUID       `json:"-"` // Set from JWT context, not from request body
}

// UpdateCustomerRequest represents a request to update a customer
type UpdateCustomerRequest struct {
	Name        *string          `json:"name" binding:"omitempty,min=1,max=200"`
	ShortName   *string          `json:"short_name" binding:"omitempty,max=100"`
	ContactName *string          `json:"contact_name" binding:"omitempty,max=100"`
	Phone       *string          `json:"phone" binding:"omitempty,max=50"`
	Email       *string          `json:"email" binding:"omitempty,email,max=200"`
	Address     *string          `json:"address" binding:"omitempty,max=500"`
	City        *string          `json:"city" binding:"omitempty,max=100"`
	Province    *string          `json:"province" binding:"omitempty,max=100"`
	PostalCode  *string          `json:"postal_code" binding:"omitempty,max=20"`
	Country     *string          `json:"country" binding:"omitempty,max=100"`
	TaxID       *string          `json:"tax_id" binding:"omitempty,max=50"`
	CreditLimit *decimal.Decimal `json:"credit_limit"`
	Level       *string          `json:"level" binding:"omitempty,oneof=normal silver gold platinum vip"`
	Notes       *string          `json:"notes"`
	SortOrder   *int             `json:"sort_order"`
	Attributes  *string          `json:"attributes"`
}

// UpdateCustomerCodeRequest represents a request to update a customer's code
type UpdateCustomerCodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50"`
}

// CustomerResponse represents a customer in API responses
type CustomerResponse struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	ShortName   string          `json:"short_name"`
	Type        string          `json:"type"`
	Level       string          `json:"level"`
	Status      string          `json:"status"`
	ContactName string          `json:"contact_name"`
	Phone       string          `json:"phone"`
	Email       string          `json:"email"`
	Address     string          `json:"address"`
	City        string          `json:"city"`
	Province    string          `json:"province"`
	PostalCode  string          `json:"postal_code"`
	Country     string          `json:"country"`
	FullAddress string          `json:"full_address"`
	TaxID       string          `json:"tax_id"`
	CreditLimit decimal.Decimal `json:"credit_limit"`
	Balance     decimal.Decimal `json:"balance"`
	Notes       string          `json:"notes"`
	SortOrder   int             `json:"sort_order"`
	Attributes  string          `json:"attributes"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Version     int             `json:"version"`
}

// CustomerListResponse represents a list item for customers
type CustomerListResponse struct {
	ID          uuid.UUID       `json:"id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	ShortName   string          `json:"short_name"`
	Type        string          `json:"type"`
	Level       string          `json:"level"`
	Status      string          `json:"status"`
	ContactName string          `json:"contact_name"`
	Phone       string          `json:"phone"`
	Email       string          `json:"email"`
	City        string          `json:"city"`
	CreditLimit decimal.Decimal `json:"credit_limit"`
	Balance     decimal.Decimal `json:"balance"`
	CreatedAt   time.Time       `json:"created_at"`
}

// CustomerListFilter represents filter options for customer list
type CustomerListFilter struct {
	Search   string `form:"search"`
	Status   string `form:"status" binding:"omitempty,oneof=active inactive suspended"`
	Type     string `form:"type" binding:"omitempty,oneof=individual organization"`
	Level    string `form:"level" binding:"omitempty,oneof=normal silver gold platinum vip"`
	City     string `form:"city"`
	Province string `form:"province"`
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	OrderBy  string `form:"order_by"`
	OrderDir string `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// ToCustomerResponse converts a domain Customer to CustomerResponse
func ToCustomerResponse(c *partner.Customer) CustomerResponse {
	return CustomerResponse{
		ID:          c.ID,
		TenantID:    c.TenantID,
		Code:        c.Code,
		Name:        c.Name,
		ShortName:   c.ShortName,
		Type:        string(c.Type),
		Level:       string(c.Level),
		Status:      string(c.Status),
		ContactName: c.ContactName,
		Phone:       c.Phone,
		Email:       c.Email,
		Address:     c.Address,
		City:        c.City,
		Province:    c.Province,
		PostalCode:  c.PostalCode,
		Country:     c.Country,
		FullAddress: c.GetFullAddress(),
		TaxID:       c.TaxID,
		CreditLimit: c.CreditLimit,
		Balance:     c.Balance,
		Notes:       c.Notes,
		SortOrder:   c.SortOrder,
		Attributes:  c.Attributes,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Version:     c.Version,
	}
}

// ToCustomerListResponse converts a domain Customer to CustomerListResponse
func ToCustomerListResponse(c *partner.Customer) CustomerListResponse {
	return CustomerListResponse{
		ID:          c.ID,
		Code:        c.Code,
		Name:        c.Name,
		ShortName:   c.ShortName,
		Type:        string(c.Type),
		Level:       string(c.Level),
		Status:      string(c.Status),
		ContactName: c.ContactName,
		Phone:       c.Phone,
		Email:       c.Email,
		City:        c.City,
		CreditLimit: c.CreditLimit,
		Balance:     c.Balance,
		CreatedAt:   c.CreatedAt,
	}
}

// ToCustomerListResponses converts a slice of domain Customers to CustomerListResponses
func ToCustomerListResponses(customers []partner.Customer) []CustomerListResponse {
	responses := make([]CustomerListResponse, len(customers))
	for i, c := range customers {
		responses[i] = ToCustomerListResponse(&c)
	}
	return responses
}

// =============================================================================
// Supplier DTOs
// =============================================================================

// CreateSupplierRequest represents a request to create a new supplier
type CreateSupplierRequest struct {
	Code        string           `json:"code" binding:"required,min=1,max=50"`
	Name        string           `json:"name" binding:"required,min=1,max=200"`
	ShortName   string           `json:"short_name" binding:"max=100"`
	Type        string           `json:"type" binding:"required,oneof=manufacturer distributor retailer service"`
	ContactName string           `json:"contact_name" binding:"max=100"`
	Phone       string           `json:"phone" binding:"max=50"`
	Email       string           `json:"email" binding:"omitempty,email,max=200"`
	Address     string           `json:"address" binding:"max=500"`
	City        string           `json:"city" binding:"max=100"`
	Province    string           `json:"province" binding:"max=100"`
	PostalCode  string           `json:"postal_code" binding:"max=20"`
	Country     string           `json:"country" binding:"max=100"`
	TaxID       string           `json:"tax_id" binding:"max=50"`
	BankName    string           `json:"bank_name" binding:"max=200"`
	BankAccount string           `json:"bank_account" binding:"max=100"`
	CreditDays  *int             `json:"credit_days"`
	CreditLimit *decimal.Decimal `json:"credit_limit"`
	Rating      *int             `json:"rating" binding:"omitempty,min=0,max=5"`
	Notes       string           `json:"notes"`
	SortOrder   *int             `json:"sort_order"`
	Attributes  string           `json:"attributes"`
	CreatedBy   *uuid.UUID       `json:"-"` // Set from JWT context, not from request body
}

// UpdateSupplierRequest represents a request to update a supplier
type UpdateSupplierRequest struct {
	Name        *string          `json:"name" binding:"omitempty,min=1,max=200"`
	ShortName   *string          `json:"short_name" binding:"omitempty,max=100"`
	ContactName *string          `json:"contact_name" binding:"omitempty,max=100"`
	Phone       *string          `json:"phone" binding:"omitempty,max=50"`
	Email       *string          `json:"email" binding:"omitempty,email,max=200"`
	Address     *string          `json:"address" binding:"omitempty,max=500"`
	City        *string          `json:"city" binding:"omitempty,max=100"`
	Province    *string          `json:"province" binding:"omitempty,max=100"`
	PostalCode  *string          `json:"postal_code" binding:"omitempty,max=20"`
	Country     *string          `json:"country" binding:"omitempty,max=100"`
	TaxID       *string          `json:"tax_id" binding:"omitempty,max=50"`
	BankName    *string          `json:"bank_name" binding:"omitempty,max=200"`
	BankAccount *string          `json:"bank_account" binding:"omitempty,max=100"`
	CreditDays  *int             `json:"credit_days"`
	CreditLimit *decimal.Decimal `json:"credit_limit"`
	Rating      *int             `json:"rating" binding:"omitempty,min=0,max=5"`
	Notes       *string          `json:"notes"`
	SortOrder   *int             `json:"sort_order"`
	Attributes  *string          `json:"attributes"`
}

// UpdateSupplierCodeRequest represents a request to update a supplier's code
type UpdateSupplierCodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50"`
}

// SupplierResponse represents a supplier in API responses
type SupplierResponse struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	ShortName       string          `json:"short_name"`
	Type            string          `json:"type"`
	Status          string          `json:"status"`
	ContactName     string          `json:"contact_name"`
	Phone           string          `json:"phone"`
	Email           string          `json:"email"`
	Address         string          `json:"address"`
	City            string          `json:"city"`
	Province        string          `json:"province"`
	PostalCode      string          `json:"postal_code"`
	Country         string          `json:"country"`
	FullAddress     string          `json:"full_address"`
	TaxID           string          `json:"tax_id"`
	BankName        string          `json:"bank_name"`
	BankAccount     string          `json:"bank_account"`
	CreditDays      int             `json:"credit_days"`
	CreditLimit     decimal.Decimal `json:"credit_limit"`
	Balance         decimal.Decimal `json:"balance"`
	AvailableCredit decimal.Decimal `json:"available_credit"`
	Rating          int             `json:"rating"`
	Notes           string          `json:"notes"`
	SortOrder       int             `json:"sort_order"`
	Attributes      string          `json:"attributes"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Version         int             `json:"version"`
}

// SupplierListResponse represents a list item for suppliers
type SupplierListResponse struct {
	ID          uuid.UUID       `json:"id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	ShortName   string          `json:"short_name"`
	Type        string          `json:"type"`
	Status      string          `json:"status"`
	ContactName string          `json:"contact_name"`
	Phone       string          `json:"phone"`
	Email       string          `json:"email"`
	City        string          `json:"city"`
	CreditDays  int             `json:"credit_days"`
	CreditLimit decimal.Decimal `json:"credit_limit"`
	Balance     decimal.Decimal `json:"balance"`
	Rating      int             `json:"rating"`
	CreatedAt   time.Time       `json:"created_at"`
}

// SupplierListFilter represents filter options for supplier list
type SupplierListFilter struct {
	Search    string `form:"search"`
	Status    string `form:"status" binding:"omitempty,oneof=active inactive blocked"`
	Type      string `form:"type" binding:"omitempty,oneof=manufacturer distributor retailer service"`
	City      string `form:"city"`
	Province  string `form:"province"`
	MinRating *int   `form:"min_rating" binding:"omitempty,min=0,max=5"`
	MaxRating *int   `form:"max_rating" binding:"omitempty,min=0,max=5"`
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"page_size" binding:"min=1,max=100"`
	OrderBy   string `form:"order_by"`
	OrderDir  string `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// ToSupplierResponse converts a domain Supplier to SupplierResponse
func ToSupplierResponse(s *partner.Supplier) SupplierResponse {
	return SupplierResponse{
		ID:              s.ID,
		TenantID:        s.TenantID,
		Code:            s.Code,
		Name:            s.Name,
		ShortName:       s.ShortName,
		Type:            string(s.Type),
		Status:          string(s.Status),
		ContactName:     s.ContactName,
		Phone:           s.Phone,
		Email:           s.Email,
		Address:         s.Address,
		City:            s.City,
		Province:        s.Province,
		PostalCode:      s.PostalCode,
		Country:         s.Country,
		FullAddress:     s.GetFullAddress(),
		TaxID:           s.TaxID,
		BankName:        s.BankName,
		BankAccount:     s.BankAccount,
		CreditDays:      s.CreditDays,
		CreditLimit:     s.CreditLimit,
		Balance:         s.Balance,
		AvailableCredit: s.GetAvailableCredit(),
		Rating:          s.Rating,
		Notes:           s.Notes,
		SortOrder:       s.SortOrder,
		Attributes:      s.Attributes,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
		Version:         s.Version,
	}
}

// ToSupplierListResponse converts a domain Supplier to SupplierListResponse
func ToSupplierListResponse(s *partner.Supplier) SupplierListResponse {
	return SupplierListResponse{
		ID:          s.ID,
		Code:        s.Code,
		Name:        s.Name,
		ShortName:   s.ShortName,
		Type:        string(s.Type),
		Status:      string(s.Status),
		ContactName: s.ContactName,
		Phone:       s.Phone,
		Email:       s.Email,
		City:        s.City,
		CreditDays:  s.CreditDays,
		CreditLimit: s.CreditLimit,
		Balance:     s.Balance,
		Rating:      s.Rating,
		CreatedAt:   s.CreatedAt,
	}
}

// ToSupplierListResponses converts a slice of domain Suppliers to SupplierListResponses
func ToSupplierListResponses(suppliers []partner.Supplier) []SupplierListResponse {
	responses := make([]SupplierListResponse, len(suppliers))
	for i, s := range suppliers {
		responses[i] = ToSupplierListResponse(&s)
	}
	return responses
}

// =============================================================================
// Warehouse DTOs
// =============================================================================

// CreateWarehouseRequest represents a request to create a new warehouse
type CreateWarehouseRequest struct {
	Code        string     `json:"code" binding:"required,min=1,max=50"`
	Name        string     `json:"name" binding:"required,min=1,max=200"`
	ShortName   string     `json:"short_name" binding:"max=100"`
	Type        string     `json:"type" binding:"required,oneof=physical virtual consign transit"`
	ContactName string     `json:"contact_name" binding:"max=100"`
	Phone       string     `json:"phone" binding:"max=50"`
	Email       string     `json:"email" binding:"omitempty,email,max=200"`
	Address     string     `json:"address" binding:"max=500"`
	City        string     `json:"city" binding:"max=100"`
	Province    string     `json:"province" binding:"max=100"`
	PostalCode  string     `json:"postal_code" binding:"max=20"`
	Country     string     `json:"country" binding:"max=100"`
	IsDefault   *bool      `json:"is_default"`
	Capacity    *int       `json:"capacity"`
	Notes       string     `json:"notes"`
	SortOrder   *int       `json:"sort_order"`
	Attributes  string     `json:"attributes"`
	CreatedBy   *uuid.UUID `json:"-"` // Set from JWT context, not from request body
}

// UpdateWarehouseRequest represents a request to update a warehouse
type UpdateWarehouseRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=200"`
	ShortName   *string `json:"short_name" binding:"omitempty,max=100"`
	ContactName *string `json:"contact_name" binding:"omitempty,max=100"`
	Phone       *string `json:"phone" binding:"omitempty,max=50"`
	Email       *string `json:"email" binding:"omitempty,email,max=200"`
	Address     *string `json:"address" binding:"omitempty,max=500"`
	City        *string `json:"city" binding:"omitempty,max=100"`
	Province    *string `json:"province" binding:"omitempty,max=100"`
	PostalCode  *string `json:"postal_code" binding:"omitempty,max=20"`
	Country     *string `json:"country" binding:"omitempty,max=100"`
	IsDefault   *bool   `json:"is_default"`
	Capacity    *int    `json:"capacity"`
	Notes       *string `json:"notes"`
	SortOrder   *int    `json:"sort_order"`
	Attributes  *string `json:"attributes"`
}

// UpdateWarehouseCodeRequest represents a request to update a warehouse's code
type UpdateWarehouseCodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50"`
}

// WarehouseResponse represents a warehouse in API responses
type WarehouseResponse struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	ContactName string    `json:"contact_name"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Address     string    `json:"address"`
	City        string    `json:"city"`
	Province    string    `json:"province"`
	PostalCode  string    `json:"postal_code"`
	Country     string    `json:"country"`
	FullAddress string    `json:"full_address"`
	IsDefault   bool      `json:"is_default"`
	Capacity    int       `json:"capacity"`
	Notes       string    `json:"notes"`
	SortOrder   int       `json:"sort_order"`
	Attributes  string    `json:"attributes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
}

// WarehouseListResponse represents a list item for warehouses
type WarehouseListResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	ContactName string    `json:"contact_name"`
	Phone       string    `json:"phone"`
	City        string    `json:"city"`
	IsDefault   bool      `json:"is_default"`
	Capacity    int       `json:"capacity"`
	CreatedAt   time.Time `json:"created_at"`
}

// WarehouseListFilter represents filter options for warehouse list
type WarehouseListFilter struct {
	Search    string `form:"search"`
	Status    string `form:"status" binding:"omitempty,oneof=active inactive"`
	Type      string `form:"type" binding:"omitempty,oneof=physical virtual consign transit"`
	City      string `form:"city"`
	Province  string `form:"province"`
	IsDefault *bool  `form:"is_default"`
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"page_size" binding:"min=1,max=100"`
	OrderBy   string `form:"order_by"`
	OrderDir  string `form:"order_dir" binding:"omitempty,oneof=asc desc"`
}

// ToWarehouseResponse converts a domain Warehouse to WarehouseResponse
func ToWarehouseResponse(w *partner.Warehouse) WarehouseResponse {
	return WarehouseResponse{
		ID:          w.ID,
		TenantID:    w.TenantID,
		Code:        w.Code,
		Name:        w.Name,
		ShortName:   w.ShortName,
		Type:        string(w.Type),
		Status:      string(w.Status),
		ContactName: w.ContactName,
		Phone:       w.Phone,
		Email:       w.Email,
		Address:     w.Address,
		City:        w.City,
		Province:    w.Province,
		PostalCode:  w.PostalCode,
		Country:     w.Country,
		FullAddress: w.GetFullAddress(),
		IsDefault:   w.IsDefault,
		Capacity:    w.Capacity,
		Notes:       w.Notes,
		SortOrder:   w.SortOrder,
		Attributes:  w.Attributes,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
		Version:     w.Version,
	}
}

// ToWarehouseListResponse converts a domain Warehouse to WarehouseListResponse
func ToWarehouseListResponse(w *partner.Warehouse) WarehouseListResponse {
	return WarehouseListResponse{
		ID:          w.ID,
		Code:        w.Code,
		Name:        w.Name,
		ShortName:   w.ShortName,
		Type:        string(w.Type),
		Status:      string(w.Status),
		ContactName: w.ContactName,
		Phone:       w.Phone,
		City:        w.City,
		IsDefault:   w.IsDefault,
		Capacity:    w.Capacity,
		CreatedAt:   w.CreatedAt,
	}
}

// ToWarehouseListResponses converts a slice of domain Warehouses to WarehouseListResponses
func ToWarehouseListResponses(warehouses []partner.Warehouse) []WarehouseListResponse {
	responses := make([]WarehouseListResponse, len(warehouses))
	for i, w := range warehouses {
		responses[i] = ToWarehouseListResponse(&w)
	}
	return responses
}

// =============================================================================
// Balance Transaction DTOs
// =============================================================================

// RechargeBalanceRequest represents a request to recharge customer balance
type RechargeBalanceRequest struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Reference string  `json:"reference" binding:"max=100"`
	Remark    string  `json:"remark" binding:"max=500"`
}

// AdjustBalanceRequest represents a request to adjust customer balance
type AdjustBalanceRequest struct {
	Amount     float64 `json:"amount" binding:"required,gt=0"`
	IsIncrease bool    `json:"is_increase"`
	Reference  string  `json:"reference" binding:"max=100"`
	Remark     string  `json:"remark" binding:"required,min=1,max=500"`
}

// BalanceTransactionResponse represents a balance transaction in API responses
type BalanceTransactionResponse struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	CustomerID      uuid.UUID       `json:"customer_id"`
	TransactionType string          `json:"transaction_type"`
	Amount          decimal.Decimal `json:"amount"`
	BalanceBefore   decimal.Decimal `json:"balance_before"`
	BalanceAfter    decimal.Decimal `json:"balance_after"`
	SourceType      string          `json:"source_type"`
	SourceID        *string         `json:"source_id,omitempty"`
	Reference       string          `json:"reference"`
	Remark          string          `json:"remark"`
	OperatorID      *uuid.UUID      `json:"operator_id,omitempty"`
	TransactionDate time.Time       `json:"transaction_date"`
	CreatedAt       time.Time       `json:"created_at"`
}

// BalanceTransactionListFilter represents filter options for balance transaction list
type BalanceTransactionListFilter struct {
	CustomerID      *uuid.UUID `form:"-"`
	TransactionType string     `form:"transaction_type" binding:"omitempty,oneof=RECHARGE CONSUME REFUND ADJUSTMENT EXPIRE"`
	SourceType      string     `form:"source_type" binding:"omitempty,oneof=MANUAL SALES_ORDER SALES_RETURN RECEIPT_VOUCHER SYSTEM"`
	DateFrom        string     `form:"date_from"`
	DateTo          string     `form:"date_to"`
	Page            int        `form:"page" binding:"min=1"`
	PageSize        int        `form:"page_size" binding:"min=1,max=100"`
}

// BalanceSummaryResponse represents customer balance summary
type BalanceSummaryResponse struct {
	CustomerID     uuid.UUID       `json:"customer_id"`
	CurrentBalance decimal.Decimal `json:"current_balance"`
	TotalRecharge  decimal.Decimal `json:"total_recharge"`
	TotalConsume   decimal.Decimal `json:"total_consume"`
	TotalRefund    decimal.Decimal `json:"total_refund"`
}

// ToBalanceTransactionResponse converts a domain BalanceTransaction to BalanceTransactionResponse
func ToBalanceTransactionResponse(t *partner.BalanceTransaction) BalanceTransactionResponse {
	return BalanceTransactionResponse{
		ID:              t.ID,
		TenantID:        t.TenantID,
		CustomerID:      t.CustomerID,
		TransactionType: string(t.TransactionType),
		Amount:          t.Amount,
		BalanceBefore:   t.BalanceBefore,
		BalanceAfter:    t.BalanceAfter,
		SourceType:      string(t.SourceType),
		SourceID:        t.SourceID,
		Reference:       t.Reference,
		Remark:          t.Remark,
		OperatorID:      t.OperatorID,
		TransactionDate: t.TransactionDate,
		CreatedAt:       t.CreatedAt,
	}
}

// ToBalanceTransactionResponses converts a slice of domain BalanceTransactions to BalanceTransactionResponses
func ToBalanceTransactionResponses(transactions []*partner.BalanceTransaction) []BalanceTransactionResponse {
	responses := make([]BalanceTransactionResponse, len(transactions))
	for i, t := range transactions {
		responses[i] = ToBalanceTransactionResponse(t)
	}
	return responses
}
