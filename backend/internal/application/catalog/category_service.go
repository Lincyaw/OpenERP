package catalog

import (
	"context"
	"errors"

	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// CategoryService handles category-related business operations
type CategoryService struct {
	categoryRepo catalog.CategoryRepository
	productRepo  catalog.ProductRepository
}

// NewCategoryService creates a new CategoryService
func NewCategoryService(
	categoryRepo catalog.CategoryRepository,
	productRepo catalog.ProductRepository,
) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
	}
}

// Create creates a new category
func (s *CategoryService) Create(ctx context.Context, tenantID uuid.UUID, req CreateCategoryRequest) (*CategoryResponse, error) {
	// Check if code already exists
	exists, err := s.categoryRepo.ExistsByCode(ctx, tenantID, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, shared.NewDomainError("ALREADY_EXISTS", "Category with this code already exists")
	}

	var category *catalog.Category

	if req.ParentID != nil {
		// Create child category
		parent, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, *req.ParentID)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return nil, shared.NewDomainError("INVALID_PARENT", "Parent category not found")
			}
			return nil, err
		}

		category, err = catalog.NewChildCategory(tenantID, req.Code, req.Name, parent)
		if err != nil {
			return nil, err
		}
	} else {
		// Create root category
		category, err = catalog.NewCategory(tenantID, req.Code, req.Name)
		if err != nil {
			return nil, err
		}
	}

	// Set optional fields
	if req.Description != "" {
		if err := category.Update(req.Name, req.Description); err != nil {
			return nil, err
		}
	}

	if req.SortOrder != nil {
		category.SetSortOrder(*req.SortOrder)
	}

	// Set created_by if provided (from JWT context via handler)
	if req.CreatedBy != nil {
		category.SetCreatedBy(*req.CreatedBy)
	}

	// Save the category
	if err := s.categoryRepo.Save(ctx, category); err != nil {
		return nil, err
	}

	return ToCategoryResponse(category), nil
}

// GetByID retrieves a category by ID
func (s *CategoryService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*CategoryResponse, error) {
	category, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return ToCategoryResponse(category), nil
}

// List retrieves all categories for a tenant
func (s *CategoryService) List(ctx context.Context, tenantID uuid.UUID, filter CategoryListFilter) ([]CategoryListResponse, int64, error) {
	domainFilter := shared.Filter{
		Filters: make(map[string]interface{}),
	}

	if filter.Search != "" {
		domainFilter.Search = filter.Search
	}

	if filter.Status != "" {
		domainFilter.Filters["status"] = filter.Status
	}

	// Set pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		domainFilter.Page = filter.Page
		domainFilter.PageSize = filter.PageSize
	}

	// Set sorting
	if filter.SortBy != "" {
		domainFilter.OrderBy = filter.SortBy
		if filter.SortDesc {
			domainFilter.OrderDir = "desc"
		} else {
			domainFilter.OrderDir = "asc"
		}
	} else {
		domainFilter.OrderBy = "sort_order"
		domainFilter.OrderDir = "asc"
	}

	categories, err := s.categoryRepo.FindAllForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.categoryRepo.CountForTenant(ctx, tenantID, domainFilter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]CategoryListResponse, len(categories))
	for i, cat := range categories {
		responses[i] = ToCategoryListResponse(&cat)
	}

	return responses, total, nil
}

// GetTree retrieves categories as a tree structure
func (s *CategoryService) GetTree(ctx context.Context, tenantID uuid.UUID) ([]CategoryTreeNode, error) {
	// Get all categories for tenant
	categories, err := s.categoryRepo.FindAllForTenant(ctx, tenantID, shared.Filter{
		OrderBy:  "sort_order",
		OrderDir: "asc",
	})
	if err != nil {
		return nil, err
	}

	// Build tree structure
	return buildCategoryTree(categories), nil
}

// GetChildren retrieves direct children of a category
func (s *CategoryService) GetChildren(ctx context.Context, tenantID, parentID uuid.UUID) ([]CategoryListResponse, error) {
	children, err := s.categoryRepo.FindChildren(ctx, tenantID, parentID)
	if err != nil {
		return nil, err
	}

	responses := make([]CategoryListResponse, len(children))
	for i, cat := range children {
		responses[i] = ToCategoryListResponse(&cat)
	}

	return responses, nil
}

// GetRootCategories retrieves all root categories
func (s *CategoryService) GetRootCategories(ctx context.Context, tenantID uuid.UUID) ([]CategoryListResponse, error) {
	categories, err := s.categoryRepo.FindRootCategories(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	responses := make([]CategoryListResponse, len(categories))
	for i, cat := range categories {
		responses[i] = ToCategoryListResponse(&cat)
	}

	return responses, nil
}

// Update updates an existing category
func (s *CategoryService) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateCategoryRequest) (*CategoryResponse, error) {
	category, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Update name and description
	if req.Name != "" {
		description := req.Description
		if description == "" {
			description = category.Description
		}
		if err := category.Update(req.Name, description); err != nil {
			return nil, err
		}
	} else if req.Description != "" {
		if err := category.Update(category.Name, req.Description); err != nil {
			return nil, err
		}
	}

	// Update sort order
	if req.SortOrder != nil {
		category.SetSortOrder(*req.SortOrder)
	}

	// Save the category
	if err := s.categoryRepo.Save(ctx, category); err != nil {
		return nil, err
	}

	return ToCategoryResponse(category), nil
}

// Move moves a category to a new parent
func (s *CategoryService) Move(ctx context.Context, tenantID, id uuid.UUID, req MoveCategoryRequest) (*CategoryResponse, error) {
	category, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	var newPath string
	var newLevel int

	if req.ParentID != nil {
		// Moving to a new parent
		newParent, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, *req.ParentID)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return nil, shared.NewDomainError("INVALID_PARENT", "Parent category not found")
			}
			return nil, err
		}

		// Prevent circular reference
		if category.IsAncestorOf(newParent) {
			return nil, shared.NewDomainError("CIRCULAR_REFERENCE", "Cannot move category to its own descendant")
		}

		// Check depth limit
		if newParent.Level >= catalog.MaxCategoryDepth-1 {
			return nil, shared.NewDomainError("MAX_DEPTH_EXCEEDED", "Maximum category depth exceeded")
		}

		newPath = newParent.Path + "/" + category.ID.String()
		newLevel = newParent.Level + 1
	} else {
		// Moving to root
		newPath = category.ID.String()
		newLevel = 0
	}

	// Calculate level delta for descendants
	levelDelta := newLevel - category.Level

	// Update the category and its descendants
	if err := s.categoryRepo.UpdatePath(ctx, tenantID, id, newPath, levelDelta); err != nil {
		return nil, err
	}

	// Reload the category
	category, err = s.categoryRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	return ToCategoryResponse(category), nil
}

// Activate activates a category
func (s *CategoryService) Activate(ctx context.Context, tenantID, id uuid.UUID) (*CategoryResponse, error) {
	category, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := category.Activate(); err != nil {
		return nil, err
	}

	if err := s.categoryRepo.Save(ctx, category); err != nil {
		return nil, err
	}

	return ToCategoryResponse(category), nil
}

// Deactivate deactivates a category
func (s *CategoryService) Deactivate(ctx context.Context, tenantID, id uuid.UUID) (*CategoryResponse, error) {
	category, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := category.Deactivate(); err != nil {
		return nil, err
	}

	if err := s.categoryRepo.Save(ctx, category); err != nil {
		return nil, err
	}

	return ToCategoryResponse(category), nil
}

// Delete deletes a category
func (s *CategoryService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	// Check if category exists
	category, err := s.categoryRepo.FindByIDForTenant(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Check if category has children
	hasChildren, err := s.categoryRepo.HasChildren(ctx, tenantID, category.ID)
	if err != nil {
		return err
	}
	if hasChildren {
		return shared.NewDomainError("HAS_CHILDREN", "Cannot delete category with children")
	}

	// Check if category has products
	hasProducts, err := s.categoryRepo.HasProducts(ctx, tenantID, category.ID)
	if err != nil {
		return err
	}
	if hasProducts {
		return shared.NewDomainError("HAS_PRODUCTS", "Cannot delete category with associated products")
	}

	return s.categoryRepo.DeleteForTenant(ctx, tenantID, id)
}

// buildCategoryTree builds a tree structure from a flat list of categories
func buildCategoryTree(categories []catalog.Category) []CategoryTreeNode {
	// Create a map for quick lookup
	nodeMap := make(map[uuid.UUID]*CategoryTreeNode)
	var roots []CategoryTreeNode

	// First pass: create all nodes
	for _, cat := range categories {
		node := CategoryTreeNode{
			ID:          cat.ID,
			Code:        cat.Code,
			Name:        cat.Name,
			Description: cat.Description,
			ParentID:    cat.ParentID,
			Level:       cat.Level,
			SortOrder:   cat.SortOrder,
			Status:      string(cat.Status),
			Children:    []CategoryTreeNode{},
		}
		nodeMap[cat.ID] = &node
	}

	// Second pass: build relationships
	for _, cat := range categories {
		node := nodeMap[cat.ID]
		if cat.ParentID == nil {
			roots = append(roots, *node)
		} else {
			parent, exists := nodeMap[*cat.ParentID]
			if exists {
				parent.Children = append(parent.Children, *node)
			}
		}
	}

	// Final pass: update references (since we used copies)
	return updateTreeReferences(roots, nodeMap)
}

// updateTreeReferences recursively updates the tree with correct child references
func updateTreeReferences(nodes []CategoryTreeNode, nodeMap map[uuid.UUID]*CategoryTreeNode) []CategoryTreeNode {
	result := make([]CategoryTreeNode, len(nodes))
	for i, node := range nodes {
		result[i] = node
		if len(node.Children) > 0 {
			children := make([]CategoryTreeNode, 0)
			for _, child := range nodeMap[node.ID].Children {
				childNode := *nodeMap[child.ID]
				childNode.Children = getChildrenFromMap(child.ID, nodeMap)
				children = append(children, childNode)
			}
			result[i].Children = children
		}
	}
	return result
}

// getChildrenFromMap gets children for a node from the map
func getChildrenFromMap(parentID uuid.UUID, nodeMap map[uuid.UUID]*CategoryTreeNode) []CategoryTreeNode {
	parent, exists := nodeMap[parentID]
	if !exists {
		return []CategoryTreeNode{}
	}

	result := make([]CategoryTreeNode, len(parent.Children))
	for i, child := range parent.Children {
		childNode := *nodeMap[child.ID]
		childNode.Children = getChildrenFromMap(child.ID, nodeMap)
		result[i] = childNode
	}
	return result
}
