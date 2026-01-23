package handler

// CustomerResponse represents a customer in API responses
// @Description Customer details returned by the API
type CustomerResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID    string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Code        string  `json:"code" example:"CUST-001"`
	Name        string  `json:"name" example:"Acme Corp"`
	ShortName   string  `json:"short_name" example:"Acme"`
	Type        string  `json:"type" example:"organization" enums:"individual,organization"`
	Level       string  `json:"level" example:"normal" enums:"normal,silver,gold,platinum,vip"`
	Status      string  `json:"status" example:"active" enums:"active,inactive,suspended"`
	ContactName string  `json:"contact_name" example:"John Doe"`
	Phone       string  `json:"phone" example:"13800138000"`
	Email       string  `json:"email" example:"contact@acme.com"`
	Address     string  `json:"address" example:"123 Main St"`
	City        string  `json:"city" example:"Shanghai"`
	Province    string  `json:"province" example:"Shanghai"`
	PostalCode  string  `json:"postal_code" example:"200000"`
	Country     string  `json:"country" example:"China"`
	FullAddress string  `json:"full_address" example:"123 Main St, Shanghai, Shanghai 200000, China"`
	TaxID       string  `json:"tax_id" example:"91310000MA1FL8L972"`
	CreditLimit float64 `json:"credit_limit" example:"10000.00"`
	Balance     float64 `json:"balance" example:"5000.00"`
	Notes       string  `json:"notes" example:"VIP customer"`
	SortOrder   int     `json:"sort_order" example:"0"`
	Attributes  string  `json:"attributes" example:"{}"`
	CreatedAt   string  `json:"created_at" example:"2026-01-24T12:00:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2026-01-24T12:00:00Z"`
	Version     int     `json:"version" example:"1"`
}

// CustomerListResponse represents a customer list item
// @Description Customer list item with basic information
type CustomerListResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code        string `json:"code" example:"CUST-001"`
	Name        string `json:"name" example:"Acme Corp"`
	ShortName   string `json:"short_name" example:"Acme"`
	Type        string `json:"type" example:"organization" enums:"individual,organization"`
	Level       string `json:"level" example:"normal" enums:"normal,silver,gold,platinum,vip"`
	Status      string `json:"status" example:"active" enums:"active,inactive,suspended"`
	Phone       string `json:"phone" example:"13800138000"`
	Email       string `json:"email" example:"contact@acme.com"`
	City        string `json:"city" example:"Shanghai"`
	Province    string `json:"province" example:"Shanghai"`
	SortOrder   int    `json:"sort_order" example:"0"`
	CreatedAt   string `json:"created_at" example:"2026-01-24T12:00:00Z"`
}

// CustomerCountResponse represents customer count statistics
// @Description Customer counts by status
type CustomerCountResponse struct {
	Active    int64 `json:"active" example:"100"`
	Inactive  int64 `json:"inactive" example:"20"`
	Suspended int64 `json:"suspended" example:"5"`
	Total     int64 `json:"total" example:"125"`
}
