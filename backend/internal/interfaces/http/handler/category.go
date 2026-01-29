package handler

import (
	catalogapp "github.com/erp/backend/internal/application/catalog"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CategoryHandler handles category-related API endpoints
//
//	@Name	HandlerCreateWarehouseRequest
type CategoryHandler struct {
	BaseHandler
	categoryService *catalogapp.CategoryService
}

// NewCategoryHandler creates a new CategoryHandler
func NewCategoryHandler(categoryService *catalogapp.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// CreateCategoryRequest represents a request to create a new category
//
//	@Description	Request body for creating a new category
//	@Name			HandlerCreateWarehouseRequest
type CreateCategoryRequest struct {
	Code        string  `json:"code" binding:"required,min=1,max=50" example:"ELECTRONICS"`
	Name        string  `json:"name" binding:"required,min=1,max=100" example:"Electronics"`
	Description string  `json:"description" binding:"max=2000" example:"Electronic products and accessories"`
	ParentID    *string `json:"parent_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SortOrder   *int    `json:"sort_order" example:"0"`
}

// UpdateCategoryRequest represents a request to update a category
//
//	@Description	Request body for updating a category
//	@Name			HandlerUpdateCategoryRequest
type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100" example:"Updated Name"`
	Description string `json:"description" binding:"omitempty,max=2000" example:"Updated description"`
	SortOrder   *int   `json:"sort_order" example:"1"`
}

// MoveCategoryRequest represents a request to move a category
//
//	@Description	Request body for moving a category to a new parent
//	@Name			HandlerUpdateCategoryRequest
type MoveCategoryRequest struct {
	ParentID *string `json:"parent_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// CategoryResponse represents a category in the response
//
//	@Description	Category response object
//	@Name			HandlerCategoryResponse
type CategoryResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID    string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Code        string  `json:"code" example:"ELECTRONICS"`
	Name        string  `json:"name" example:"Electronics"`
	Description string  `json:"description" example:"Electronic products"`
	ParentID    *string `json:"parent_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Path        string  `json:"path" example:"550e8400-e29b-41d4-a716-446655440002/550e8400-e29b-41d4-a716-446655440000"`
	Level       int     `json:"level" example:"1"`
	SortOrder   int     `json:"sort_order" example:"0"`
	Status      string  `json:"status" example:"active"`
	CreatedAt   string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Version     int     `json:"version" example:"1"`
}

// CategoryListResponse represents a category list item
//
//	@Description	Category list item
//	@Name			HandlerCategoryListResponse
type CategoryListResponse struct {
	ID          string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code        string  `json:"code" example:"ELECTRONICS"`
	Name        string  `json:"name" example:"Electronics"`
	Description string  `json:"description" example:"Electronic products"`
	ParentID    *string `json:"parent_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Level       int     `json:"level" example:"1"`
	SortOrder   int     `json:"sort_order" example:"0"`
	Status      string  `json:"status" example:"active"`
	CreatedAt   string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// CategoryTreeNode represents a category node in tree structure
//
//	@Description	Category tree node with children
//	@Name			HandlerCategoryTreeNode
type CategoryTreeNode struct {
	ID          string             `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Code        string             `json:"code" example:"ELECTRONICS"`
	Name        string             `json:"name" example:"Electronics"`
	Description string             `json:"description" example:"Electronic products"`
	ParentID    *string            `json:"parent_id"`
	Level       int                `json:"level" example:"0"`
	SortOrder   int                `json:"sort_order" example:"0"`
	Status      string             `json:"status" example:"active"`
	Children    []CategoryTreeNode `json:"children"`
}

// Create godoc
//
//	@ID				createCategory
//	@Summary		Create a new category
//	@Description	Create a new product category. Can be a root category or a child of an existing category.
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			request		body		CreateCategoryRequest	true	"Category creation request"
//	@Success		201			{object}	APIResponse[CategoryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		409			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	// Convert to application DTO
	appReq := catalogapp.CreateCategoryRequest{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		appReq.CreatedBy = &userID
	}

	// Convert parent ID
	if req.ParentID != nil && *req.ParentID != "" {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			h.BadRequest(c, "Invalid parent ID format")
			return
		}
		appReq.ParentID = &parentID
	}

	if req.SortOrder != nil {
		appReq.SortOrder = req.SortOrder
	}

	category, err := h.categoryService.Create(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, category)
}

// GetByID godoc
//
//	@ID				getCategoryById
//	@Summary		Get category by ID
//	@Description	Retrieve a category by its ID
//	@Tags			categories
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Category ID"	format(uuid)
//	@Success		200			{object}	APIResponse[CategoryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/{id} [get]
func (h *CategoryHandler) GetByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid category ID format")
		return
	}

	category, err := h.categoryService.GetByID(c.Request.Context(), tenantID, categoryID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, category)
}

// List godoc
//
//	@ID				listCategories
//	@Summary		List categories
//	@Description	Retrieve a paginated list of categories
//	@Tags			categories
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			search		query		string	false	"Search keyword"
//	@Param			status		query		string	false	"Status filter (active, inactive)"	Enums(active, inactive)
//	@Param			page		query		int		false	"Page number"						default(1)
//	@Param			page_size	query		int		false	"Items per page"					default(20)
//	@Param			sort_by		query		string	false	"Sort by field"
//	@Param			sort_desc	query		bool	false	"Sort descending"
//	@Success		200			{object}	APIResponse[[]CategoryListResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter catalogapp.CategoryListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set default pagination
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}

	categories, total, err := h.categoryService.List(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, categories, total, filter.Page, filter.PageSize)
}

// GetTree godoc
//
//	@ID				getCategoryTree
//	@Summary		Get category tree
//	@Description	Retrieve all categories as a hierarchical tree structure
//	@Tags			categories
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[[]CategoryTreeNode]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/tree [get]
func (h *CategoryHandler) GetTree(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	tree, err := h.categoryService.GetTree(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, tree)
}

// GetChildren godoc
//
//	@ID				getCategoryChildren
//	@Summary		Get children of a category
//	@Description	Retrieve direct children of a specific category
//	@Tags			categories
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Parent Category ID"	format(uuid)
//	@Success		200			{object}	APIResponse[[]CategoryListResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/{id}/children [get]
func (h *CategoryHandler) GetChildren(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	parentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid parent category ID format")
		return
	}

	children, err := h.categoryService.GetChildren(c.Request.Context(), tenantID, parentID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, children)
}

// GetRoots godoc
//
//	@ID				getCategoryRoots
//	@Summary		Get root categories
//	@Description	Retrieve all root (top-level) categories
//	@Tags			categories
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Success		200			{object}	APIResponse[[]CategoryListResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/roots [get]
func (h *CategoryHandler) GetRoots(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	categories, err := h.categoryService.GetRootCategories(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, categories)
}

// Update godoc
//
//	@ID				updateCategory
//	@Summary		Update a category
//	@Description	Update an existing category's information
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string					false	"Tenant ID (optional for dev)"
//	@Param			id			path		string					true	"Category ID"	format(uuid)
//	@Param			request		body		UpdateCategoryRequest	true	"Category update request"
//	@Success		200			{object}	APIResponse[CategoryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/{id} [put]
func (h *CategoryHandler) Update(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid category ID format")
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := catalogapp.UpdateCategoryRequest{
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}

	category, err := h.categoryService.Update(c.Request.Context(), tenantID, categoryID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, category)
}

// Move godoc
//
//	@ID				moveCategory
//	@Summary		Move a category
//	@Description	Move a category to a new parent (or make it a root category)
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string				false	"Tenant ID (optional for dev)"
//	@Param			id			path		string				true	"Category ID"	format(uuid)
//	@Param			request		body		MoveCategoryRequest	true	"Move category request"
//	@Success		200			{object}	APIResponse[CategoryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/{id}/move [post]
func (h *CategoryHandler) Move(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid category ID format")
		return
	}

	var req MoveCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	appReq := catalogapp.MoveCategoryRequest{}
	if req.ParentID != nil && *req.ParentID != "" {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			h.BadRequest(c, "Invalid parent ID format")
			return
		}
		appReq.ParentID = &parentID
	}

	category, err := h.categoryService.Move(c.Request.Context(), tenantID, categoryID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, category)
}

// Activate godoc
//
//	@ID				activateCategory
//	@Summary		Activate a category
//	@Description	Activate an inactive category
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Category ID"	format(uuid)
//	@Success		200			{object}	APIResponse[CategoryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/{id}/activate [post]
func (h *CategoryHandler) Activate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid category ID format")
		return
	}

	category, err := h.categoryService.Activate(c.Request.Context(), tenantID, categoryID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, category)
}

// Deactivate godoc
//
//	@ID				deactivateCategory
//	@Summary		Deactivate a category
//	@Description	Deactivate an active category
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Param			X-Tenant-ID	header		string	false	"Tenant ID (optional for dev)"
//	@Param			id			path		string	true	"Category ID"	format(uuid)
//	@Success		200			{object}	APIResponse[CategoryResponse]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/{id}/deactivate [post]
func (h *CategoryHandler) Deactivate(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid category ID format")
		return
	}

	category, err := h.categoryService.Deactivate(c.Request.Context(), tenantID, categoryID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, category)
}

// Delete godoc
//
//	@ID				deleteCategory
//	@Summary		Delete a category
//	@Description	Delete a category. Category must have no children and no associated products.
//	@Tags			categories
//	@Produce		json
//	@Param			X-Tenant-ID	header	string	false	"Tenant ID (optional for dev)"
//	@Param			id			path	string	true	"Category ID"	format(uuid)
//	@Success		204			"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		409			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Security		BearerAuth
//	@Router			/catalog/categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid category ID format")
		return
	}

	err = h.categoryService.Delete(c.Request.Context(), tenantID, categoryID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.NoContent(c)
}

// Ensure dto import is used
var _ = dto.Response{}
