package handler

// SupplierResponse represents a supplier in API responses
// @Description Supplier details returned by the API
type SupplierResponse struct {
	ID              string   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID        string   `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Code            string   `json:"code" example:"SUP-001"`
	Name            string   `json:"name" example:"Supplier Co"`
	ShortName       string   `json:"short_name" example:"Supplier"`
	Status          string   `json:"status" example:"active" enums:"active,inactive,blocked"`
	ContactName     string   `json:"contact_name" example:"John Doe"`
	Phone           string   `json:"phone" example:"13800138000"`
	Email           string   `json:"email" example:"contact@supplier.com"`
	Address         string   `json:"address" example:"456 Supply St"`
	City            string   `json:"city" example:"Shenzhen"`
	Province        string   `json:"province" example:"Guangdong"`
	PostalCode      string   `json:"postal_code" example:"518000"`
	Country         string   `json:"country" example:"China"`
	FullAddress     string   `json:"full_address" example:"456 Supply St, Shenzhen, Guangdong 518000, China"`
	TaxID           string   `json:"tax_id" example:"91440300MA5F99999X"`
	BankName        string   `json:"bank_name" example:"Bank of China"`
	BankAccount     string   `json:"bank_account" example:"6217001234567890123"`
	BankAccountName string   `json:"bank_account_name" example:"Supplier Co Ltd"`
	PaymentTermDays int      `json:"payment_term_days" example:"30"`
	CreditLimit     float64  `json:"credit_limit" example:"50000.00"`
	Rating          *float64 `json:"rating" example:"4.5"`
	Notes           string   `json:"notes" example:"Reliable supplier"`
	SortOrder       int      `json:"sort_order" example:"0"`
	Attributes      string   `json:"attributes" example:"{}"`
	CreatedAt       string   `json:"created_at" example:"2026-01-24T12:00:00Z"`
	UpdatedAt       string   `json:"updated_at" example:"2026-01-24T12:00:00Z"`
	Version         int      `json:"version" example:"1"`
}

// SupplierListResponse represents a supplier list item
// @Description Supplier list item with basic information
type SupplierListResponse struct {
	ID              string   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code            string   `json:"code" example:"SUP-001"`
	Name            string   `json:"name" example:"Supplier Co"`
	ShortName       string   `json:"short_name" example:"Supplier"`
	Status          string   `json:"status" example:"active" enums:"active,inactive,blocked"`
	Phone           string   `json:"phone" example:"13800138000"`
	Email           string   `json:"email" example:"contact@supplier.com"`
	City            string   `json:"city" example:"Shenzhen"`
	Province        string   `json:"province" example:"Guangdong"`
	PaymentTermDays int      `json:"payment_term_days" example:"30"`
	Rating          *float64 `json:"rating" example:"4.5"`
	SortOrder       int      `json:"sort_order" example:"0"`
	CreatedAt       string   `json:"created_at" example:"2026-01-24T12:00:00Z"`
}

// SupplierCountResponse represents supplier count statistics
// @Description Supplier counts by status
type SupplierCountResponse struct {
	Active   int64 `json:"active" example:"50"`
	Inactive int64 `json:"inactive" example:"10"`
	Blocked  int64 `json:"blocked" example:"2"`
	Total    int64 `json:"total" example:"62"`
}
