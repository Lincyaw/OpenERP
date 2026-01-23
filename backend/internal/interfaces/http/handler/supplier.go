package handler

import (
	partnerapp "github.com/erp/backend/internal/application/partner"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SupplierHandler handles supplier-related API endpoints
type SupplierHandler struct {
	BaseHandler
	supplierService *partnerapp.SupplierService
}

// NewSupplierHandler creates a new SupplierHandler
func NewSupplierHandler(supplierService *partnerapp.SupplierService) *SupplierHandler {
	return &SupplierHandler{
		supplierService: supplierService,
	}
}

// CreateSupplierRequest represents a request to create a new supplier
// @Description Request body for creating a new supplier
type CreateSupplierRequest struct {
	Code        string   `json:"code" binding:"required,min=1,max=50" example:"SUP-001"`
	Name        string   `json:"name" binding:"required,min=1,max=200" example:"Global Supplies Inc"`
	ShortName   string   `json:"short_name" binding:"max=100" example:"Global"`
	Type        string   `json:"type" binding:"required,oneof=manufacturer distributor retailer service" example:"distributor"`
	ContactName string   `json:"contact_name" binding:"max=100" example:"Mike Johnson"`
	Phone       string   `json:"phone" binding:"max=50" example:"13600136000"`
	Email       string   `json:"email" binding:"omitempty,email,max=200" example:"mike@globalsupplies.com"`
	Address     string   `json:"address" binding:"max=500" example:"789 Industrial Park"`
	City        string   `json:"city" binding:"max=100" example:"Guangzhou"`
	Province    string   `json:"province" binding:"max=100" example:"Guangdong"`
	PostalCode  string   `json:"postal_code" binding:"max=20" example:"510000"`
	Country     string   `json:"country" binding:"max=100" example:"China"`
	TaxID       string   `json:"tax_id" binding:"max=50" example:"91440000MA5FL8L972"`
	BankName    string   `json:"bank_name" binding:"max=200" example:"Industrial Bank"`
	BankAccount string   `json:"bank_account" binding:"max=100" example:"6222021234567890123"`
	CreditDays  *int     `json:"credit_days" example:"30"`
	CreditLimit *float64 `json:"credit_limit" example:"50000.00"`
	Rating      *int     `json:"rating" binding:"omitempty,min=0,max=5" example:"4"`
	Notes       string   `json:"notes" example:"Reliable supplier"`
	SortOrder   *int     `json:"sort_order" example:"0"`
	Attributes  string   `json:"attributes" example:"{}"`
}

// UpdateSupplierRequest represents a request to update a supplier
// @Description Request body for updating a supplier
type UpdateSupplierRequest struct {
	Name        *string  `json:"name" binding:"omitempty,min=1,max=200" example:"Global Supplies International"`
	ShortName   *string  `json:"short_name" binding:"omitempty,max=100" example:"GSI"`
	ContactName *string  `json:"contact_name" binding:"omitempty,max=100" example:"Sarah Chen"`
	Phone       *string  `json:"phone" binding:"omitempty,max=50" example:"13700137000"`
	Email       *string  `json:"email" binding:"omitempty,email,max=200" example:"sarah@globalsupplies.com"`
	Address     *string  `json:"address" binding:"omitempty,max=500" example:"Building A, Tech Park"`
	City        *string  `json:"city" binding:"omitempty,max=100" example:"Shenzhen"`
	Province    *string  `json:"province" binding:"omitempty,max=100" example:"Guangdong"`
	PostalCode  *string  `json:"postal_code" binding:"omitempty,max=20" example:"518000"`
	Country     *string  `json:"country" binding:"omitempty,max=100" example:"China"`
	TaxID       *string  `json:"tax_id" binding:"omitempty,max=50" example:"91440300MA5FL8L972"`
	BankName    *string  `json:"bank_name" binding:"omitempty,max=200" example:"China Merchants Bank"`
	BankAccount *string  `json:"bank_account" binding:"omitempty,max=100" example:"6226221234567890123"`
	CreditDays  *int     `json:"credit_days" example:"45"`
	CreditLimit *float64 `json:"credit_limit" example:"100000.00"`
	Rating      *int     `json:"rating" binding:"omitempty,min=0,max=5" example:"5"`
	Notes       *string  `json:"notes" example:"Premium supplier"`
	SortOrder   *int     `json:"sort_order" example:"1"`
	Attributes  *string  `json:"attributes" example:"{}"`
}

// UpdateSupplierCodeRequest represents a request to update a supplier's code
// @Description Request body for updating a supplier's code
type UpdateSupplierCodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50" example:"SUP-002"`
}

// SetRatingRequest represents a request to set supplier rating
// @Description Request body for setting supplier rating
type SetRatingRequest struct {
	Rating int `json:"rating" binding:"required,min=0,max=5" example:"5"`
}

// SetPaymentTermsRequest represents a request to set payment terms
// @Description Request body for setting supplier payment terms
type SetPaymentTermsRequest struct {
	CreditDays  int     `json:"credit_days" binding:"min=0" example:"30"`
	CreditLimit float64 `json:"credit_limit" binding:"min=0" example:"50000.00"`
}

// Create godoc
// @Summary      Create a new supplier
// @Description  Create a new supplier in the partner module
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreateSupplierRequest true "Supplier creation request"
// @Success      201 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      409 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers [post]
func (h *SupplierHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := partnerapp.CreateSupplierRequest{
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
		BankName:    req.BankName,
		BankAccount: req.BankAccount,
		Rating:      req.Rating,
		Notes:       req.Notes,
		Attributes:  req.Attributes,
	}

	if req.CreditDays != nil {
		appReq.CreditDays = req.CreditDays
	}
	if req.CreditLimit != nil {
		d := decimal.NewFromFloat(*req.CreditLimit)
		appReq.CreditLimit = &d
	}
	if req.SortOrder != nil {
		appReq.SortOrder = req.SortOrder
	}

	supplier, err := h.supplierService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, supplier)
}

// GetByID godoc
// @Summary      Get supplier by ID
// @Description  Retrieve a supplier by its ID
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id} [get]
func (h *SupplierHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	supplier, err := h.supplierService.GetByID(c.Request.Context(), tenantID, supplierID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// GetByCode godoc
// @Summary      Get supplier by code
// @Description  Retrieve a supplier by its code
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        code path string true "Supplier Code"
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/code/{code} [get]
func (h *SupplierHandler) GetByCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	code := c.Param("code")
	if code == "" {
		h.BadRequest(c, "Supplier code is required")
		return
	}

	supplier, err := h.supplierService.GetByCode(c.Request.Context(), tenantID, code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// List godoc
// @Summary      List suppliers
// @Description  Retrieve a paginated list of suppliers with optional filtering
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (name, code, phone, email)"
// @Param        status query string false "Supplier status" Enums(active, inactive, blocked)
// @Param        type query string false "Supplier type" Enums(manufacturer, distributor, retailer, service)
// @Param        city query string false "City"
// @Param        province query string false "Province"
// @Param        min_rating query int false "Minimum rating (0-5)"
// @Param        max_rating query int false "Maximum rating (0-5)"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(sort_order)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(asc)
// @Success      200 {object} dto.Response{data=[]SupplierListResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers [get]
func (h *SupplierHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter partnerapp.SupplierListFilter
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

	suppliers, total, err := h.supplierService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, suppliers, total, filter.Page, filter.PageSize)
}

// Update godoc
// @Summary      Update a supplier
// @Description  Update an existing supplier's details
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Param        request body UpdateSupplierRequest true "Supplier update request"
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      409 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id} [put]
func (h *SupplierHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	var req UpdateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := partnerapp.UpdateSupplierRequest{
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
		BankName:    req.BankName,
		BankAccount: req.BankAccount,
		CreditDays:  req.CreditDays,
		Rating:      req.Rating,
		Notes:       req.Notes,
		SortOrder:   req.SortOrder,
		Attributes:  req.Attributes,
	}

	if req.CreditLimit != nil {
		d := decimal.NewFromFloat(*req.CreditLimit)
		appReq.CreditLimit = &d
	}

	supplier, err := h.supplierService.Update(c.Request.Context(), tenantID, supplierID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// UpdateCode godoc
// @Summary      Update supplier code
// @Description  Update a supplier's code
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Param        request body UpdateSupplierCodeRequest true "New supplier code"
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      409 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id}/code [put]
func (h *SupplierHandler) UpdateCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	var req UpdateSupplierCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	supplier, err := h.supplierService.UpdateCode(c.Request.Context(), tenantID, supplierID, req.Code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// Delete godoc
// @Summary      Delete a supplier
// @Description  Delete a supplier by ID
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Success      204
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id} [delete]
func (h *SupplierHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	err = h.supplierService.Delete(c.Request.Context(), tenantID, supplierID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// Activate godoc
// @Summary      Activate a supplier
// @Description  Activate an inactive or blocked supplier
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id}/activate [post]
func (h *SupplierHandler) Activate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	supplier, err := h.supplierService.Activate(c.Request.Context(), tenantID, supplierID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// Deactivate godoc
// @Summary      Deactivate a supplier
// @Description  Deactivate an active supplier
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id}/deactivate [post]
func (h *SupplierHandler) Deactivate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	supplier, err := h.supplierService.Deactivate(c.Request.Context(), tenantID, supplierID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// Block godoc
// @Summary      Block a supplier
// @Description  Block a supplier from doing business
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id}/block [post]
func (h *SupplierHandler) Block(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	supplier, err := h.supplierService.Block(c.Request.Context(), tenantID, supplierID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// SetRating godoc
// @Summary      Set supplier rating
// @Description  Set a supplier's rating (0-5)
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Param        request body SetRatingRequest true "Supplier rating"
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id}/rating [put]
func (h *SupplierHandler) SetRating(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	var req SetRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	supplier, err := h.supplierService.SetRating(c.Request.Context(), tenantID, supplierID, req.Rating)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// SetPaymentTerms godoc
// @Summary      Set supplier payment terms
// @Description  Set a supplier's credit days and credit limit
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Supplier ID" format(uuid)
// @Param        request body SetPaymentTermsRequest true "Payment terms"
// @Success      200 {object} dto.Response{data=SupplierResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/{id}/payment-terms [put]
func (h *SupplierHandler) SetPaymentTerms(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	var req SetPaymentTermsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	creditLimit := decimal.NewFromFloat(req.CreditLimit)
	supplier, err := h.supplierService.SetPaymentTerms(c.Request.Context(), tenantID, supplierID, req.CreditDays, creditLimit)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, supplier)
}

// SupplierCountByStatusResponse represents the supplier count by status response
type SupplierCountByStatusResponse struct {
	Active   int64 `json:"active" example:"80"`
	Inactive int64 `json:"inactive" example:"15"`
	Blocked  int64 `json:"blocked" example:"5"`
	Total    int64 `json:"total" example:"100"`
}

// CountByStatus godoc
// @Summary      Get supplier counts by status
// @Description  Get the count of suppliers grouped by status
// @Tags         suppliers
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Success      200 {object} dto.Response{data=SupplierCountByStatusResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /partner/suppliers/stats/count [get]
func (h *SupplierHandler) CountByStatus(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	counts, err := h.supplierService.CountByStatus(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	response := SupplierCountByStatusResponse{
		Active:   counts["active"],
		Inactive: counts["inactive"],
		Blocked:  counts["blocked"],
		Total:    counts["total"],
	}

	h.Success(c, response)
}
