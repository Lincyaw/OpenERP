package handler

import (
	catalogapp "github.com/erp/backend/internal/application/catalog"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ProductUnitHandler handles product unit-related API endpoints
type ProductUnitHandler struct {
	BaseHandler
	productUnitService *catalogapp.ProductUnitService
}

// NewProductUnitHandler creates a new ProductUnitHandler
func NewProductUnitHandler(productUnitService *catalogapp.ProductUnitService) *ProductUnitHandler {
	return &ProductUnitHandler{
		productUnitService: productUnitService,
	}
}

// CreateProductUnitRequest represents a request to create a product unit
// @Description Request body for creating a product unit
type CreateProductUnitRequest struct {
	UnitCode              string   `json:"unit_code" binding:"required,min=1,max=20" example:"BOX"`
	UnitName              string   `json:"unit_name" binding:"required,min=1,max=50" example:"箱"`
	ConversionRate        float64  `json:"conversion_rate" binding:"required,gt=0" example:"24"`
	DefaultPurchasePrice  *float64 `json:"default_purchase_price" example:"1200.00"`
	DefaultSellingPrice   *float64 `json:"default_selling_price" example:"2400.00"`
	IsDefaultPurchaseUnit bool     `json:"is_default_purchase_unit" example:"false"`
	IsDefaultSalesUnit    bool     `json:"is_default_sales_unit" example:"false"`
	SortOrder             *int     `json:"sort_order" example:"0"`
}

// UpdateProductUnitRequest represents a request to update a product unit
// @Description Request body for updating a product unit
type UpdateProductUnitRequest struct {
	UnitName              *string  `json:"unit_name" binding:"omitempty,min=1,max=50" example:"箱"`
	ConversionRate        *float64 `json:"conversion_rate" binding:"omitempty,gt=0" example:"24"`
	DefaultPurchasePrice  *float64 `json:"default_purchase_price" example:"1200.00"`
	DefaultSellingPrice   *float64 `json:"default_selling_price" example:"2400.00"`
	IsDefaultPurchaseUnit *bool    `json:"is_default_purchase_unit" example:"false"`
	IsDefaultSalesUnit    *bool    `json:"is_default_sales_unit" example:"false"`
	SortOrder             *int     `json:"sort_order" example:"0"`
}

// ConvertUnitRequest represents a request to convert quantity between units
// @Description Request body for unit conversion
type ConvertUnitRequest struct {
	Quantity     float64 `json:"quantity" binding:"required" example:"2"`
	FromUnitCode string  `json:"from_unit_code" binding:"required" example:"BOX"`
	ToUnitCode   string  `json:"to_unit_code" binding:"required" example:"pcs"`
}

// ProductUnitResponse represents a product unit in API responses
// @Description Product unit response
type ProductUnitResponse struct {
	ID                    string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductID             string  `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	UnitCode              string  `json:"unit_code" example:"BOX"`
	UnitName              string  `json:"unit_name" example:"箱"`
	ConversionRate        float64 `json:"conversion_rate" example:"24"`
	DefaultPurchasePrice  float64 `json:"default_purchase_price" example:"1200.00"`
	DefaultSellingPrice   float64 `json:"default_selling_price" example:"2400.00"`
	IsDefaultPurchaseUnit bool    `json:"is_default_purchase_unit" example:"false"`
	IsDefaultSalesUnit    bool    `json:"is_default_sales_unit" example:"false"`
	SortOrder             int     `json:"sort_order" example:"0"`
	CreatedAt             string  `json:"created_at" example:"2026-01-24T12:00:00Z"`
	UpdatedAt             string  `json:"updated_at" example:"2026-01-24T12:00:00Z"`
}

// ConvertUnitResponse represents the result of a unit conversion
// @Description Unit conversion result
type ConvertUnitResponse struct {
	FromQuantity float64 `json:"from_quantity" example:"2"`
	FromUnitCode string  `json:"from_unit_code" example:"BOX"`
	ToQuantity   float64 `json:"to_quantity" example:"48"`
	ToUnitCode   string  `json:"to_unit_code" example:"pcs"`
}

// Create godoc
// @Summary      Create a product unit
// @Description  Create a new alternate unit for a product
// @Tags         product-units
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        productId path string true "Product ID" format(uuid)
// @Param        request body CreateProductUnitRequest true "Product unit creation request"
// @Success      201 {object} dto.Response{data=ProductUnitResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      409 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{productId}/units [post]
func (h *ProductUnitHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	var req CreateProductUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := catalogapp.CreateProductUnitRequest{
		UnitCode:              req.UnitCode,
		UnitName:              req.UnitName,
		ConversionRate:        decimal.NewFromFloat(req.ConversionRate),
		IsDefaultPurchaseUnit: req.IsDefaultPurchaseUnit,
		IsDefaultSalesUnit:    req.IsDefaultSalesUnit,
		SortOrder:             req.SortOrder,
	}

	if req.DefaultPurchasePrice != nil {
		price := decimal.NewFromFloat(*req.DefaultPurchasePrice)
		appReq.DefaultPurchasePrice = &price
	}
	if req.DefaultSellingPrice != nil {
		price := decimal.NewFromFloat(*req.DefaultSellingPrice)
		appReq.DefaultSellingPrice = &price
	}

	unit, err := h.productUnitService.Create(c.Request.Context(), tenantID, productID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, unit)
}

// GetByID godoc
// @Summary      Get product unit by ID
// @Description  Retrieve a product unit by its ID
// @Tags         product-units
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        productId path string true "Product ID" format(uuid)
// @Param        id path string true "Unit ID" format(uuid)
// @Success      200 {object} dto.Response{data=ProductUnitResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{productId}/units/{id} [get]
func (h *ProductUnitHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid unit ID format")
		return
	}

	unit, err := h.productUnitService.GetByID(c.Request.Context(), tenantID, id)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, unit)
}

// List godoc
// @Summary      List product units
// @Description  List all alternate units for a product
// @Tags         product-units
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        productId path string true "Product ID" format(uuid)
// @Success      200 {object} dto.Response{data=[]ProductUnitResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{productId}/units [get]
func (h *ProductUnitHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	units, err := h.productUnitService.ListByProduct(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, units)
}

// Update godoc
// @Summary      Update product unit
// @Description  Update an existing product unit
// @Tags         product-units
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        productId path string true "Product ID" format(uuid)
// @Param        id path string true "Unit ID" format(uuid)
// @Param        request body UpdateProductUnitRequest true "Product unit update request"
// @Success      200 {object} dto.Response{data=ProductUnitResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{productId}/units/{id} [put]
func (h *ProductUnitHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid unit ID format")
		return
	}

	var req UpdateProductUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := catalogapp.UpdateProductUnitRequest{
		UnitName:              req.UnitName,
		IsDefaultPurchaseUnit: req.IsDefaultPurchaseUnit,
		IsDefaultSalesUnit:    req.IsDefaultSalesUnit,
		SortOrder:             req.SortOrder,
	}

	if req.ConversionRate != nil {
		rate := decimal.NewFromFloat(*req.ConversionRate)
		appReq.ConversionRate = &rate
	}
	if req.DefaultPurchasePrice != nil {
		price := decimal.NewFromFloat(*req.DefaultPurchasePrice)
		appReq.DefaultPurchasePrice = &price
	}
	if req.DefaultSellingPrice != nil {
		price := decimal.NewFromFloat(*req.DefaultSellingPrice)
		appReq.DefaultSellingPrice = &price
	}

	unit, err := h.productUnitService.Update(c.Request.Context(), tenantID, id, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, unit)
}

// Delete godoc
// @Summary      Delete product unit
// @Description  Delete a product unit
// @Tags         product-units
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        productId path string true "Product ID" format(uuid)
// @Param        id path string true "Unit ID" format(uuid)
// @Success      200 {object} dto.Response
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{productId}/units/{id} [delete]
func (h *ProductUnitHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid unit ID format")
		return
	}

	if err := h.productUnitService.Delete(c.Request.Context(), tenantID, id); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, dto.MessageResponse{Message: "Product unit deleted successfully"})
}

// Convert godoc
// @Summary      Convert quantity between units
// @Description  Convert quantity from one unit to another for a product
// @Tags         product-units
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        productId path string true "Product ID" format(uuid)
// @Param        request body ConvertUnitRequest true "Unit conversion request"
// @Success      200 {object} dto.Response{data=ConvertUnitResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{productId}/units/convert [post]
func (h *ProductUnitHandler) Convert(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	var req ConvertUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := catalogapp.ConvertUnitRequest{
		Quantity:     decimal.NewFromFloat(req.Quantity),
		FromUnitCode: req.FromUnitCode,
		ToUnitCode:   req.ToUnitCode,
	}

	result, err := h.productUnitService.ConvertQuantity(c.Request.Context(), tenantID, productID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, result)
}

// GetDefaultPurchaseUnit godoc
// @Summary      Get default purchase unit
// @Description  Get the default purchase unit for a product
// @Tags         product-units
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        productId path string true "Product ID" format(uuid)
// @Success      200 {object} dto.Response{data=ProductUnitResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{productId}/units/default-purchase [get]
func (h *ProductUnitHandler) GetDefaultPurchaseUnit(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	unit, err := h.productUnitService.GetDefaultPurchaseUnit(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, unit)
}

// GetDefaultSalesUnit godoc
// @Summary      Get default sales unit
// @Description  Get the default sales unit for a product
// @Tags         product-units
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        productId path string true "Product ID" format(uuid)
// @Success      200 {object} dto.Response{data=ProductUnitResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{productId}/units/default-sales [get]
func (h *ProductUnitHandler) GetDefaultSalesUnit(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	unit, err := h.productUnitService.GetDefaultSalesUnit(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, unit)
}
