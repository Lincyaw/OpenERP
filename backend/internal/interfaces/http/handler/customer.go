package handler

import (
	partnerapp "github.com/erp/backend/internal/application/partner"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CustomerHandler handles customer-related API endpoints
type CustomerHandler struct {
	BaseHandler
	customerService *partnerapp.CustomerService
}

// NewCustomerHandler creates a new CustomerHandler
func NewCustomerHandler(customerService *partnerapp.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
	}
}

// CreateCustomerRequest represents a request to create a new customer
// @Description Request body for creating a new customer
type CreateCustomerRequest struct {
	Code        string   `json:"code" binding:"required,min=1,max=50" example:"CUST-001"`
	Name        string   `json:"name" binding:"required,min=1,max=200" example:"Acme Corp"`
	ShortName   string   `json:"short_name" binding:"max=100" example:"Acme"`
	Type        string   `json:"type" binding:"required,oneof=individual organization" example:"organization"`
	ContactName string   `json:"contact_name" binding:"max=100" example:"John Doe"`
	Phone       string   `json:"phone" binding:"max=50" example:"13800138000"`
	Email       string   `json:"email" binding:"omitempty,email,max=200" example:"contact@acme.com"`
	Address     string   `json:"address" binding:"max=500" example:"123 Main St"`
	City        string   `json:"city" binding:"max=100" example:"Shanghai"`
	Province    string   `json:"province" binding:"max=100" example:"Shanghai"`
	PostalCode  string   `json:"postal_code" binding:"max=20" example:"200000"`
	Country     string   `json:"country" binding:"max=100" example:"China"`
	TaxID       string   `json:"tax_id" binding:"max=50" example:"91310000MA1FL8L972"`
	CreditLimit *float64 `json:"credit_limit" example:"10000.00"`
	Notes       string   `json:"notes" example:"VIP customer"`
	SortOrder   *int     `json:"sort_order" example:"0"`
	Attributes  string   `json:"attributes" example:"{}"`
}

// UpdateCustomerRequest represents a request to update a customer
// @Description Request body for updating a customer
type UpdateCustomerRequest struct {
	Name        *string  `json:"name" binding:"omitempty,min=1,max=200" example:"Acme Corporation"`
	ShortName   *string  `json:"short_name" binding:"omitempty,max=100" example:"Acme Co"`
	ContactName *string  `json:"contact_name" binding:"omitempty,max=100" example:"Jane Doe"`
	Phone       *string  `json:"phone" binding:"omitempty,max=50" example:"13900139000"`
	Email       *string  `json:"email" binding:"omitempty,email,max=200" example:"info@acme.com"`
	Address     *string  `json:"address" binding:"omitempty,max=500" example:"456 Oak Ave"`
	City        *string  `json:"city" binding:"omitempty,max=100" example:"Beijing"`
	Province    *string  `json:"province" binding:"omitempty,max=100" example:"Beijing"`
	PostalCode  *string  `json:"postal_code" binding:"omitempty,max=20" example:"100000"`
	Country     *string  `json:"country" binding:"omitempty,max=100" example:"China"`
	TaxID       *string  `json:"tax_id" binding:"omitempty,max=50" example:"91310000MA1FL8L972"`
	CreditLimit *float64 `json:"credit_limit" example:"20000.00"`
	Level       *string  `json:"level" binding:"omitempty,oneof=normal silver gold platinum vip" example:"gold"`
	Notes       *string  `json:"notes" example:"Updated notes"`
	SortOrder   *int     `json:"sort_order" example:"1"`
	Attributes  *string  `json:"attributes" example:"{}"`
}

// UpdateCustomerCodeRequest represents a request to update a customer's code
// @Description Request body for updating a customer's code
type UpdateCustomerCodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50" example:"CUST-002"`
}

// BalanceOperationRequest represents a request for balance operations
// @Description Request body for balance operations (add/deduct/refund)
type BalanceOperationRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0" example:"1000.00"`
}

// SetLevelRequest represents a request to set customer level
// @Description Request body for setting customer level
type SetLevelRequest struct {
	Level string `json:"level" binding:"required,oneof=normal silver gold platinum vip" example:"gold"`
}

// Create godoc
// @ID           createCustomer
// @Summary      Create a new customer
// @Description  Create a new customer in the partner module
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreateCustomerRequest true "Customer creation request"
// @Success      201 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers [post]
func (h *CustomerHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	// Convert to application DTO
	appReq := partnerapp.CreateCustomerRequest{
		Code:        req.Code,
		Name:        req.Name,
		ShortName:   req.ShortName,
		Type:        req.Type,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Email:       req.Email,
		Address:     req.Address,
		City:        req.City,
		Province:    req.Province,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
		TaxID:       req.TaxID,
		Notes:       req.Notes,
		Attributes:  req.Attributes,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		appReq.CreatedBy = &userID
	}

	if req.CreditLimit != nil {
		d := decimal.NewFromFloat(*req.CreditLimit)
		appReq.CreditLimit = &d
	}
	if req.SortOrder != nil {
		appReq.SortOrder = req.SortOrder
	}

	customer, err := h.customerService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, customer)
}

// GetByID godoc
// @ID           getCustomerById
// @Summary      Get customer by ID
// @Description  Retrieve a customer by its ID
// @Tags         customers
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id} [get]
func (h *CustomerHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	customer, err := h.customerService.GetByID(c.Request.Context(), tenantID, customerID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// GetByCode godoc
// @ID           getCustomerByCode
// @Summary      Get customer by code
// @Description  Retrieve a customer by its code
// @Tags         customers
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        code path string true "Customer Code"
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/code/{code} [get]
func (h *CustomerHandler) GetByCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	code := c.Param("code")
	if code == "" {
		h.BadRequest(c, "Customer code is required")
		return
	}

	customer, err := h.customerService.GetByCode(c.Request.Context(), tenantID, code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// List godoc
// @ID           listCustomers
// @Summary      List customers
// @Description  Retrieve a paginated list of customers with optional filtering
// @Tags         customers
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (name, code, phone, email)"
// @Param        status query string false "Customer status" Enums(active, inactive, suspended)
// @Param        type query string false "Customer type" Enums(individual, organization)
// @Param        level query string false "Customer level" Enums(normal, silver, gold, platinum, vip)
// @Param        city query string false "City"
// @Param        province query string false "Province"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(sort_order)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(asc)
// @Success      200 {object} APIResponse[[]CustomerListResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers [get]
func (h *CustomerHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter partnerapp.CustomerListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	customers, total, err := h.customerService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, customers, total, filter.Page, filter.PageSize)
}

// Update godoc
// @ID           updateCustomer
// @Summary      Update a customer
// @Description  Update an existing customer's details
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Param        request body UpdateCustomerRequest true "Customer update request"
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id} [put]
func (h *CustomerHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := partnerapp.UpdateCustomerRequest{
		Name:        req.Name,
		ShortName:   req.ShortName,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Email:       req.Email,
		Address:     req.Address,
		City:        req.City,
		Province:    req.Province,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
		TaxID:       req.TaxID,
		Level:       req.Level,
		Notes:       req.Notes,
		SortOrder:   req.SortOrder,
		Attributes:  req.Attributes,
	}

	if req.CreditLimit != nil {
		d := decimal.NewFromFloat(*req.CreditLimit)
		appReq.CreditLimit = &d
	}

	customer, err := h.customerService.Update(c.Request.Context(), tenantID, customerID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// UpdateCode godoc
// @ID           updateCustomerCode
// @Summary      Update customer code
// @Description  Update a customer's code
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Param        request body UpdateCustomerCodeRequest true "New customer code"
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/code [put]
func (h *CustomerHandler) UpdateCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	var req UpdateCustomerCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	customer, err := h.customerService.UpdateCode(c.Request.Context(), tenantID, customerID, req.Code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// Delete godoc
// @ID           deleteCustomer
// @Summary      Delete a customer
// @Description  Delete a customer by ID
// @Tags         customers
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Success      204
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id} [delete]
func (h *CustomerHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	err = h.customerService.Delete(c.Request.Context(), tenantID, customerID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// Activate godoc
// @ID           activateCustomer
// @Summary      Activate a customer
// @Description  Activate an inactive or suspended customer
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/activate [post]
func (h *CustomerHandler) Activate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	customer, err := h.customerService.Activate(c.Request.Context(), tenantID, customerID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// Deactivate godoc
// @ID           deactivateCustomer
// @Summary      Deactivate a customer
// @Description  Deactivate an active customer
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/deactivate [post]
func (h *CustomerHandler) Deactivate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	customer, err := h.customerService.Deactivate(c.Request.Context(), tenantID, customerID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// Suspend godoc
// @ID           suspendCustomer
// @Summary      Suspend a customer
// @Description  Suspend an active customer
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/suspend [post]
func (h *CustomerHandler) Suspend(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	customer, err := h.customerService.Suspend(c.Request.Context(), tenantID, customerID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// AddBalance godoc
// @ID           addCustomerBalance
// @Summary      Add balance to customer
// @Description  Add to a customer's prepaid balance
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Param        request body BalanceOperationRequest true "Balance amount to add"
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/balance/add [post]
func (h *CustomerHandler) AddBalance(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	var req BalanceOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	amount := decimal.NewFromFloat(req.Amount)
	customer, err := h.customerService.AddBalance(c.Request.Context(), tenantID, customerID, amount)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// DeductBalance godoc
// @ID           deductCustomerBalance
// @Summary      Deduct balance from customer
// @Description  Deduct from a customer's prepaid balance
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Param        request body BalanceOperationRequest true "Balance amount to deduct"
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/balance/deduct [post]
func (h *CustomerHandler) DeductBalance(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	var req BalanceOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	amount := decimal.NewFromFloat(req.Amount)
	customer, err := h.customerService.DeductBalance(c.Request.Context(), tenantID, customerID, amount)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// SetLevel godoc
// @ID           setCustomerLevel
// @Summary      Set customer level
// @Description  Set a customer's tier level
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Param        request body SetLevelRequest true "Customer level"
// @Success      200 {object} APIResponse[CustomerResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/level [put]
func (h *CustomerHandler) SetLevel(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	var req SetLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	customer, err := h.customerService.SetLevel(c.Request.Context(), tenantID, customerID, req.Level)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, customer)
}

// CustomerCountByStatusResponse represents the customer count by status response
type CustomerCountByStatusResponse struct {
	Active    int64 `json:"active" example:"100"`
	Inactive  int64 `json:"inactive" example:"20"`
	Suspended int64 `json:"suspended" example:"5"`
	Total     int64 `json:"total" example:"125"`
}

// CountByStatus godoc
// @ID           countCustomerByStatus
// @Summary      Get customer counts by status
// @Description  Get the count of customers grouped by status
// @Tags         customers
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Success      200 {object} APIResponse[CustomerCountByStatusResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/stats/count [get]
func (h *CustomerHandler) CountByStatus(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	counts, err := h.customerService.CountByStatus(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	response := CustomerCountByStatusResponse{
		Active:    counts["active"],
		Inactive:  counts["inactive"],
		Suspended: counts["suspended"],
		Total:     counts["total"],
	}

	h.Success(c, response)
}
