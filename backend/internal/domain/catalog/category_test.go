package catalog

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCategory(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates root category with valid inputs", func(t *testing.T) {
		category, err := NewCategory(tenantID, "ELECTRONICS", "Electronics")
		require.NoError(t, err)
		require.NotNil(t, category)

		assert.Equal(t, tenantID, category.TenantID)
		assert.Equal(t, "ELECTRONICS", category.Code)
		assert.Equal(t, "Electronics", category.Name)
		assert.Nil(t, category.ParentID)
		assert.Equal(t, 0, category.Level)
		assert.Equal(t, CategoryStatusActive, category.Status)
		assert.True(t, category.IsRoot())
		assert.NotEmpty(t, category.ID)
		assert.Equal(t, category.ID.String(), category.Path)
	})

	t.Run("converts code to uppercase", func(t *testing.T) {
		category, err := NewCategory(tenantID, "electronics", "Electronics")
		require.NoError(t, err)
		assert.Equal(t, "ELECTRONICS", category.Code)
	})

	t.Run("publishes CategoryCreated event", func(t *testing.T) {
		category, err := NewCategory(tenantID, "TEST", "Test")
		require.NoError(t, err)

		events := category.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeCategoryCreated, events[0].EventType())
	})

	t.Run("fails with empty code", func(t *testing.T) {
		_, err := NewCategory(tenantID, "", "Electronics")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code cannot be empty")
	})

	t.Run("fails with code too long", func(t *testing.T) {
		longCode := "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCDEFGHIJKLMNOP"
		_, err := NewCategory(tenantID, longCode, "Electronics")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed 50 characters")
	})

	t.Run("fails with invalid code characters", func(t *testing.T) {
		_, err := NewCategory(tenantID, "ELEC@TRONICS", "Electronics")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can only contain letters")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		_, err := NewCategory(tenantID, "ELECTRONICS", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("accepts code with underscore and hyphen", func(t *testing.T) {
		category, err := NewCategory(tenantID, "ELEC_TRONICS-001", "Electronics")
		require.NoError(t, err)
		assert.Equal(t, "ELEC_TRONICS-001", category.Code)
	})
}

func TestNewChildCategory(t *testing.T) {
	tenantID := uuid.New()
	parent, err := NewCategory(tenantID, "ELECTRONICS", "Electronics")
	require.NoError(t, err)

	t.Run("creates child category under parent", func(t *testing.T) {
		child, err := NewChildCategory(tenantID, "PHONES", "Phones", parent)
		require.NoError(t, err)
		require.NotNil(t, child)

		assert.Equal(t, tenantID, child.TenantID)
		assert.Equal(t, "PHONES", child.Code)
		assert.Equal(t, "Phones", child.Name)
		assert.NotNil(t, child.ParentID)
		assert.Equal(t, parent.ID, *child.ParentID)
		assert.Equal(t, 1, child.Level)
		assert.False(t, child.IsRoot())
		assert.Equal(t, parent.Path+"/"+child.ID.String(), child.Path)
	})

	t.Run("fails with nil parent", func(t *testing.T) {
		_, err := NewChildCategory(tenantID, "PHONES", "Phones", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Parent category is required")
	})

	t.Run("respects max depth", func(t *testing.T) {
		// Create a chain of categories up to max depth
		current := parent
		for i := 1; i < MaxCategoryDepth-1; i++ {
			next, err := NewChildCategory(tenantID, "LEVEL"+string(rune('A'+i)), "Level "+string(rune('A'+i)), current)
			require.NoError(t, err)
			current = next
		}

		// This should succeed (level = MaxCategoryDepth - 1)
		_, err := NewChildCategory(tenantID, "LAST", "Last", current)
		require.NoError(t, err)

		// Create one more to test the limit
		deepParent, _ := NewCategory(tenantID, "DEEP", "Deep")
		deepParent.Level = MaxCategoryDepth - 1

		_, err = NewChildCategory(tenantID, "TOODEEP", "Too Deep", deepParent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "depth cannot exceed")
	})
}

func TestCategoryUpdate(t *testing.T) {
	tenantID := uuid.New()
	category, _ := NewCategory(tenantID, "TEST", "Test")
	category.ClearDomainEvents() // Clear creation event

	t.Run("updates name and description", func(t *testing.T) {
		originalVersion := category.GetVersion()
		err := category.Update("Updated Name", "New description")
		require.NoError(t, err)

		assert.Equal(t, "Updated Name", category.Name)
		assert.Equal(t, "New description", category.Description)
		assert.Equal(t, originalVersion+1, category.GetVersion())
	})

	t.Run("publishes CategoryUpdated event", func(t *testing.T) {
		category.ClearDomainEvents()
		err := category.Update("Another Name", "Another description")
		require.NoError(t, err)

		events := category.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeCategoryUpdated, events[0].EventType())
	})

	t.Run("fails with empty name", func(t *testing.T) {
		err := category.Update("", "Description")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})
}

func TestCategoryUpdateCode(t *testing.T) {
	tenantID := uuid.New()
	category, _ := NewCategory(tenantID, "OLD_CODE", "Test")
	category.ClearDomainEvents()

	t.Run("updates code", func(t *testing.T) {
		err := category.UpdateCode("NEW_CODE")
		require.NoError(t, err)
		assert.Equal(t, "NEW_CODE", category.Code)
	})

	t.Run("converts code to uppercase", func(t *testing.T) {
		err := category.UpdateCode("lowercase")
		require.NoError(t, err)
		assert.Equal(t, "LOWERCASE", category.Code)
	})

	t.Run("fails with invalid code", func(t *testing.T) {
		err := category.UpdateCode("INVALID@CODE")
		require.Error(t, err)
	})
}

func TestCategoryStatus(t *testing.T) {
	tenantID := uuid.New()

	t.Run("activates inactive category", func(t *testing.T) {
		category, _ := NewCategory(tenantID, "TEST", "Test")
		category.Status = CategoryStatusInactive
		category.ClearDomainEvents()

		err := category.Activate()
		require.NoError(t, err)
		assert.Equal(t, CategoryStatusActive, category.Status)
		assert.True(t, category.IsActive())

		events := category.GetDomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeCategoryStatusChanged, events[0].EventType())
	})

	t.Run("fails to activate already active category", func(t *testing.T) {
		category, _ := NewCategory(tenantID, "TEST", "Test")
		// Category is active by default
		err := category.Activate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})

	t.Run("deactivates active category", func(t *testing.T) {
		category, _ := NewCategory(tenantID, "TEST", "Test")
		category.ClearDomainEvents()

		err := category.Deactivate()
		require.NoError(t, err)
		assert.Equal(t, CategoryStatusInactive, category.Status)
		assert.False(t, category.IsActive())

		events := category.GetDomainEvents()
		require.Len(t, events, 1)
	})

	t.Run("fails to deactivate already inactive category", func(t *testing.T) {
		category, _ := NewCategory(tenantID, "TEST", "Test")
		category.Status = CategoryStatusInactive

		err := category.Deactivate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already inactive")
	})
}

func TestCategorySortOrder(t *testing.T) {
	tenantID := uuid.New()
	category, _ := NewCategory(tenantID, "TEST", "Test")
	originalVersion := category.GetVersion()

	category.SetSortOrder(10)
	assert.Equal(t, 10, category.SortOrder)
	assert.Equal(t, originalVersion+1, category.GetVersion())
}

func TestCategoryTreeMethods(t *testing.T) {
	tenantID := uuid.New()

	t.Run("GetAncestorIDs returns ancestor IDs", func(t *testing.T) {
		parent, _ := NewCategory(tenantID, "PARENT", "Parent")
		child, _ := NewChildCategory(tenantID, "CHILD", "Child", parent)
		grandchild, _ := NewChildCategory(tenantID, "GRANDCHILD", "Grandchild", child)

		ancestors := grandchild.GetAncestorIDs()
		require.Len(t, ancestors, 2)
		assert.Equal(t, parent.ID, ancestors[0])
		assert.Equal(t, child.ID, ancestors[1])
	})

	t.Run("GetAncestorIDs returns nil for root", func(t *testing.T) {
		root, _ := NewCategory(tenantID, "ROOT", "Root")
		ancestors := root.GetAncestorIDs()
		assert.Nil(t, ancestors)
	})

	t.Run("IsAncestorOf returns true for ancestor", func(t *testing.T) {
		parent, _ := NewCategory(tenantID, "PARENT", "Parent")
		child, _ := NewChildCategory(tenantID, "CHILD", "Child", parent)
		grandchild, _ := NewChildCategory(tenantID, "GRANDCHILD", "Grandchild", child)

		assert.True(t, parent.IsAncestorOf(child))
		assert.True(t, parent.IsAncestorOf(grandchild))
		assert.True(t, child.IsAncestorOf(grandchild))
		assert.False(t, grandchild.IsAncestorOf(parent))
		assert.False(t, child.IsAncestorOf(parent))
	})

	t.Run("IsAncestorOf handles nil", func(t *testing.T) {
		parent, _ := NewCategory(tenantID, "PARENT", "Parent")
		assert.False(t, parent.IsAncestorOf(nil))
	})

	t.Run("IsDescendantOf returns true for descendant", func(t *testing.T) {
		parent, _ := NewCategory(tenantID, "PARENT", "Parent")
		child, _ := NewChildCategory(tenantID, "CHILD", "Child", parent)

		assert.True(t, child.IsDescendantOf(parent))
		assert.False(t, parent.IsDescendantOf(child))
	})
}

func TestCategoryEvents(t *testing.T) {
	tenantID := uuid.New()
	category, _ := NewCategory(tenantID, "TEST", "Test")

	t.Run("CategoryCreatedEvent has correct fields", func(t *testing.T) {
		events := category.GetDomainEvents()
		require.Len(t, events, 1)

		event, ok := events[0].(*CategoryCreatedEvent)
		require.True(t, ok)

		assert.Equal(t, category.ID, event.CategoryID)
		assert.Equal(t, category.Code, event.Code)
		assert.Equal(t, category.Name, event.Name)
		assert.Equal(t, category.ParentID, event.ParentID)
		assert.Equal(t, category.Level, event.Level)
		assert.Equal(t, tenantID, event.TenantID())
		assert.Equal(t, EventTypeCategoryCreated, event.EventType())
		assert.Equal(t, AggregateTypeCategory, event.AggregateType())
	})

	t.Run("CategoryDeletedEvent has correct fields", func(t *testing.T) {
		event := NewCategoryDeletedEvent(category)
		assert.Equal(t, category.ID, event.CategoryID)
		assert.Equal(t, category.Code, event.Code)
		assert.Equal(t, category.ParentID, event.ParentID)
		assert.Equal(t, EventTypeCategoryDeleted, event.EventType())
	})
}

func TestValidateCategoryCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid uppercase", "ELECTRONICS", false},
		{"valid lowercase", "electronics", false},
		{"valid with underscore", "ELEC_TRONICS", false},
		{"valid with hyphen", "ELEC-TRONICS", false},
		{"valid with numbers", "CATEGORY01", false},
		{"empty", "", true},
		{"too long", "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCDEFGHIJKLMNOP", true},
		{"special character @", "TEST@CODE", true},
		{"special character space", "TEST CODE", true},
		{"special character /", "TEST/CODE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCategoryCode(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCategoryName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "Electronics", false},
		{"valid with spaces", "Home Electronics", false},
		{"valid chinese", "电子产品", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 101)), true},
		{"max length", string(make([]byte, 100)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCategoryName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
