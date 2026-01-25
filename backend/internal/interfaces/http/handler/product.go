package handler

import (
	catalogapp "github.com/erp/backend/internal/application/catalog"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProductHandler handles product-related API endpoints
type ProductHandler struct {
	BaseHandler
	productService *catalogapp.ProductService
}

// NewProductHandler creates a new ProductHandler
func NewProductHandler(productService *catalogapp.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// CreateProductRequest represents a request to create a new product
// @Description Request body for creating a new product
type CreateProductRequest struct {
	Code          string   `json:"code" binding:"required,min=1,max=50" example:"SKU-001"`
	Name          string   `json:"name" binding:"required,min=1,max=200" example:"Sample Product"`
	Description   string   `json:"description" binding:"max=2000" example:"This is a sample product description"`
	Barcode       string   `json:"barcode" binding:"max=50" example:"6901234567890"`
	CategoryID    *string  `json:"category_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Unit          string   `json:"unit" binding:"required,min=1,max=20" example:"pcs"`
	PurchasePrice *float64 `json:"purchase_price" example:"50.00"`
	SellingPrice  *float64 `json:"selling_price" example:"100.00"`
	MinStock      *float64 `json:"min_stock" example:"10"`
	SortOrder     *int     `json:"sort_order" example:"0"`
	Attributes    string   `json:"attributes" example:"{}"`
}

// UpdateProductRequest represents a request to update a product
// @Description Request body for updating a product
type UpdateProductRequest struct {
	Name          *string  `json:"name" binding:"omitempty,min=1,max=200" example:"Updated Product Name"`
	Description   *string  `json:"description" binding:"omitempty,max=2000" example:"Updated description"`
	Barcode       *string  `json:"barcode" binding:"omitempty,max=50" example:"6901234567891"`
	CategoryID    *string  `json:"category_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PurchasePrice *float64 `json:"purchase_price" example:"55.00"`
	SellingPrice  *float64 `json:"selling_price" example:"110.00"`
	MinStock      *float64 `json:"min_stock" example:"15"`
	SortOrder     *int     `json:"sort_order" example:"1"`
	Attributes    *string  `json:"attributes" example:"{}"`
}

// UpdateProductCodeRequest represents a request to update a product's code
// @Description Request body for updating a product's code
type UpdateProductCodeRequest struct {
	Code string `json:"code" binding:"required,min=1,max=50" example:"SKU-002"`
}

// Create godoc
// @Summary      Create a new product
// @Description  Create a new product in the catalog
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreateProductRequest true "Product creation request"
// @Success      201 {object} dto.Response{data=ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      409 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	// Convert to application DTO
	appReq := catalogapp.CreateProductRequest{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Barcode:     req.Barcode,
		Unit:        req.Unit,
		Attributes:  req.Attributes,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		appReq.CreatedBy = &userID
	}

	// Convert category ID
	if req.CategoryID != nil && *req.CategoryID != "" {
		catID, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			h.BadRequest(c, "Invalid category ID format")
			return
		}
		appReq.CategoryID = &catID
	}

	// Convert prices
	if req.PurchasePrice != nil {
		appReq.PurchasePrice = toDecimalPtr(*req.PurchasePrice)
	}
	if req.SellingPrice != nil {
		appReq.SellingPrice = toDecimalPtr(*req.SellingPrice)
	}
	if req.MinStock != nil {
		appReq.MinStock = toDecimalPtr(*req.MinStock)
	}
	if req.SortOrder != nil {
		appReq.SortOrder = req.SortOrder
	}

	product, err := h.productService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, product)
}

// GetByID godoc
// @Summary      Get product by ID
// @Description  Retrieve a product by its ID
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Product ID" format(uuid)
// @Success      200 {object} dto.Response{data=ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{id} [get]
func (h *ProductHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	product, err := h.productService.GetByID(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, product)
}

// GetByCode godoc
// @Summary      Get product by code
// @Description  Retrieve a product by its code
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        code path string true "Product Code"
// @Success      200 {object} dto.Response{data=ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/code/{code} [get]
func (h *ProductHandler) GetByCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	code := c.Param("code")
	if code == "" {
		h.BadRequest(c, "Product code is required")
		return
	}

	product, err := h.productService.GetByCode(c.Request.Context(), tenantID, code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, product)
}

// List godoc
// @Summary      List products
// @Description  Retrieve a paginated list of products with optional filtering
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (name, code, barcode)"
// @Param        status query string false "Product status" Enums(active, inactive, discontinued)
// @Param        category_id query string false "Category ID" format(uuid)
// @Param        unit query string false "Unit of measure"
// @Param        min_price query number false "Minimum selling price"
// @Param        max_price query number false "Maximum selling price"
// @Param        has_barcode query boolean false "Filter by barcode presence"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(sort_order)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(asc)
// @Success      200 {object} dto.Response{data=[]ProductListResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products [get]
func (h *ProductHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter catalogapp.ProductListFilter
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

	products, total, err := h.productService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, products, total, filter.Page, filter.PageSize)
}

// Update godoc
// @Summary      Update a product
// @Description  Update an existing product's details
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Product ID" format(uuid)
// @Param        request body UpdateProductRequest true "Product update request"
// @Success      200 {object} dto.Response{data=ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      409 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Convert to application DTO
	appReq := catalogapp.UpdateProductRequest{
		Name:        req.Name,
		Description: req.Description,
		Barcode:     req.Barcode,
		SortOrder:   req.SortOrder,
		Attributes:  req.Attributes,
	}

	// Convert category ID
	if req.CategoryID != nil {
		if *req.CategoryID == "" {
			// Clear category
			emptyUUID := uuid.Nil
			appReq.CategoryID = &emptyUUID
		} else {
			catID, err := uuid.Parse(*req.CategoryID)
			if err != nil {
				h.BadRequest(c, "Invalid category ID format")
				return
			}
			appReq.CategoryID = &catID
		}
	}

	// Convert prices
	if req.PurchasePrice != nil {
		appReq.PurchasePrice = toDecimalPtr(*req.PurchasePrice)
	}
	if req.SellingPrice != nil {
		appReq.SellingPrice = toDecimalPtr(*req.SellingPrice)
	}
	if req.MinStock != nil {
		appReq.MinStock = toDecimalPtr(*req.MinStock)
	}

	product, err := h.productService.Update(c.Request.Context(), tenantID, productID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, product)
}

// UpdateCode godoc
// @Summary      Update product code
// @Description  Update a product's code (SKU)
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Product ID" format(uuid)
// @Param        request body UpdateProductCodeRequest true "New product code"
// @Success      200 {object} dto.Response{data=ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      409 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{id}/code [put]
func (h *ProductHandler) UpdateCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	var req UpdateProductCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	product, err := h.productService.UpdateCode(c.Request.Context(), tenantID, productID, req.Code)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, product)
}

// Delete godoc
// @Summary      Delete a product
// @Description  Delete a product by ID
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Product ID" format(uuid)
// @Success      204
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	err = h.productService.Delete(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// Activate godoc
// @Summary      Activate a product
// @Description  Activate an inactive product
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Product ID" format(uuid)
// @Success      200 {object} dto.Response{data=ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{id}/activate [post]
func (h *ProductHandler) Activate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	product, err := h.productService.Activate(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, product)
}

// Deactivate godoc
// @Summary      Deactivate a product
// @Description  Deactivate an active product
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Product ID" format(uuid)
// @Success      200 {object} dto.Response{data=ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{id}/deactivate [post]
func (h *ProductHandler) Deactivate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	product, err := h.productService.Deactivate(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, product)
}

// Discontinue godoc
// @Summary      Discontinue a product
// @Description  Discontinue a product (cannot be reactivated)
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Product ID" format(uuid)
// @Success      200 {object} dto.Response{data=ProductResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/{id}/discontinue [post]
func (h *ProductHandler) Discontinue(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid product ID format")
		return
	}

	product, err := h.productService.Discontinue(c.Request.Context(), tenantID, productID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, product)
}

// CountByStatusResponse represents the product count by status response
type CountByStatusResponse struct {
	Active       int64 `json:"active" example:"50"`
	Inactive     int64 `json:"inactive" example:"10"`
	Discontinued int64 `json:"discontinued" example:"5"`
	Total        int64 `json:"total" example:"65"`
}

// CountByStatus godoc
// @Summary      Get product counts by status
// @Description  Get the count of products grouped by status
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Success      200 {object} dto.Response{data=CountByStatusResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/products/stats/count [get]
func (h *ProductHandler) CountByStatus(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	counts, err := h.productService.CountByStatus(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	response := CountByStatusResponse{
		Active:       counts["active"],
		Inactive:     counts["inactive"],
		Discontinued: counts["discontinued"],
		Total:        counts["total"],
	}

	h.Success(c, response)
}

// GetByCategory godoc
// @Summary      Get products by category
// @Description  Retrieve products belonging to a specific category
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        category_id path string true "Category ID" format(uuid)
// @Param        search query string false "Search term"
// @Param        status query string false "Product status" Enums(active, inactive, discontinued)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Param        order_by query string false "Order by field" default(sort_order)
// @Param        order_dir query string false "Order direction" Enums(asc, desc) default(asc)
// @Success      200 {object} dto.Response{data=[]ProductListResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /catalog/categories/{category_id}/products [get]
func (h *ProductHandler) GetByCategory(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	categoryID, err := uuid.Parse(c.Param("category_id"))
	if err != nil {
		h.BadRequest(c, "Invalid category ID format")
		return
	}

	var filter catalogapp.ProductListFilter
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

	products, total, err := h.productService.GetByCategory(c.Request.Context(), tenantID, categoryID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, products, total, filter.Page, filter.PageSize)
}

// Helper function to suppress unused import warning
var _ = dto.Response{}
