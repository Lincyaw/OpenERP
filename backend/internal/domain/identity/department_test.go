package identity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDepartment(t *testing.T) {
	tenantID := uuid.New()

	t.Run("creates department with valid inputs", func(t *testing.T) {
		dept, err := NewDepartment(tenantID, "SALES", "Sales Department")
		require.NoError(t, err)

		assert.Equal(t, "SALES", dept.Code)
		assert.Equal(t, "Sales Department", dept.Name)
		assert.Equal(t, tenantID, dept.TenantID)
		assert.Equal(t, DepartmentStatusActive, dept.Status)
		assert.Equal(t, 0, dept.Level)
		assert.Nil(t, dept.ParentID)
		assert.NotNil(t, dept.Metadata)
	})

	t.Run("normalizes code to uppercase", func(t *testing.T) {
		dept, err := NewDepartment(tenantID, "sales", "Sales")
		require.NoError(t, err)
		assert.Equal(t, "SALES", dept.Code)
	})

	t.Run("fails with empty code", func(t *testing.T) {
		_, err := NewDepartment(tenantID, "", "Name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "code cannot be empty")
	})

	t.Run("fails with invalid code characters", func(t *testing.T) {
		_, err := NewDepartment(tenantID, "123CODE", "Name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must start with a letter")
	})

	t.Run("fails with empty name", func(t *testing.T) {
		_, err := NewDepartment(tenantID, "CODE", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})
}

func TestDepartment_SetParent(t *testing.T) {
	tenantID := uuid.New()

	t.Run("sets parent with valid parent", func(t *testing.T) {
		parentID := uuid.New()
		parentPath := "/" + parentID.String()

		dept, _ := NewDepartment(tenantID, "CHILD", "Child Dept")
		err := dept.SetParent(&parentID, parentPath, 0)

		require.NoError(t, err)
		assert.Equal(t, &parentID, dept.ParentID)
		assert.Equal(t, parentPath+"/"+dept.ID.String(), dept.Path)
		assert.Equal(t, 1, dept.Level)
	})

	t.Run("sets as root when parent is nil", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "ROOT", "Root Dept")
		err := dept.SetParent(nil, "", 0)

		require.NoError(t, err)
		assert.Nil(t, dept.ParentID)
		assert.Equal(t, "/"+dept.ID.String(), dept.Path)
		assert.Equal(t, 0, dept.Level)
	})

	t.Run("fails when setting self as parent", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")
		selfID := dept.ID

		err := dept.SetParent(&selfID, "/"+selfID.String(), 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be its own parent")
	})
}

func TestDepartment_IsAncestorOf(t *testing.T) {
	tenantID := uuid.New()

	t.Run("returns true for descendant path", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "PARENT", "Parent")
		_ = dept.SetParent(nil, "", 0)

		otherPath := dept.Path + "/" + uuid.New().String()
		assert.True(t, dept.IsAncestorOf(otherPath))
	})

	t.Run("returns false for non-descendant path", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")
		_ = dept.SetParent(nil, "", 0)

		otherPath := "/" + uuid.New().String()
		assert.False(t, dept.IsAncestorOf(otherPath))
	})

	t.Run("returns false for same path", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")
		_ = dept.SetParent(nil, "", 0)

		assert.False(t, dept.IsAncestorOf(dept.Path))
	})
}

func TestDepartment_IsDescendantOf(t *testing.T) {
	tenantID := uuid.New()
	parentID := uuid.New()
	parentPath := "/" + parentID.String()

	t.Run("returns true for ancestor path", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "CHILD", "Child")
		_ = dept.SetParent(&parentID, parentPath, 0)

		assert.True(t, dept.IsDescendantOf(parentPath))
	})

	t.Run("returns false for non-ancestor path", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")
		_ = dept.SetParent(nil, "", 0)

		otherPath := "/" + uuid.New().String()
		assert.False(t, dept.IsDescendantOf(otherPath))
	})
}

func TestDepartment_GetAncestorIDs(t *testing.T) {
	t.Run("returns nil for root department", func(t *testing.T) {
		tenantID := uuid.New()
		dept, _ := NewDepartment(tenantID, "ROOT", "Root")
		_ = dept.SetParent(nil, "", 0)

		ancestors := dept.GetAncestorIDs()
		assert.Nil(t, ancestors)
	})

	t.Run("returns parent IDs for nested department", func(t *testing.T) {
		tenantID := uuid.New()
		grandparentID := uuid.New()
		parentID := uuid.New()

		grandparentPath := "/" + grandparentID.String()
		parentPath := grandparentPath + "/" + parentID.String()

		dept, _ := NewDepartment(tenantID, "CHILD", "Child")
		_ = dept.SetParent(&parentID, parentPath, 1)

		ancestors := dept.GetAncestorIDs()
		require.Len(t, ancestors, 2)
		assert.Equal(t, grandparentID, ancestors[0])
		assert.Equal(t, parentID, ancestors[1])
	})
}

func TestDepartment_Status(t *testing.T) {
	tenantID := uuid.New()

	t.Run("activate department", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")
		_ = dept.Deactivate()

		err := dept.Activate()
		require.NoError(t, err)
		assert.True(t, dept.IsActive())
	})

	t.Run("deactivate department", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")

		err := dept.Deactivate()
		require.NoError(t, err)
		assert.False(t, dept.IsActive())
	})

	t.Run("fails to activate already active", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")

		err := dept.Activate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already active")
	})

	t.Run("fails to deactivate already inactive", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")
		_ = dept.Deactivate()

		err := dept.Deactivate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already inactive")
	})
}

func TestDepartment_SetManager(t *testing.T) {
	tenantID := uuid.New()
	managerID := uuid.New()

	t.Run("sets manager", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")

		dept.SetManager(&managerID)
		assert.Equal(t, &managerID, dept.ManagerID)
	})

	t.Run("clears manager", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")
		dept.SetManager(&managerID)

		dept.SetManager(nil)
		assert.Nil(t, dept.ManagerID)
	})
}

func TestDepartment_Metadata(t *testing.T) {
	tenantID := uuid.New()

	t.Run("sets and gets metadata", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")

		dept.SetMetadata("location", "Building A")
		value, exists := dept.GetMetadata("location")

		assert.True(t, exists)
		assert.Equal(t, "Building A", value)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Dept")

		_, exists := dept.GetMetadata("nonexistent")
		assert.False(t, exists)
	})
}

func TestDepartment_Update(t *testing.T) {
	tenantID := uuid.New()

	t.Run("updates name and description", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Old Name")

		err := dept.Update("New Name", "New Description")
		require.NoError(t, err)
		assert.Equal(t, "New Name", dept.Name)
		assert.Equal(t, "New Description", dept.Description)
	})

	t.Run("fails with empty name", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "DEPT", "Name")

		err := dept.Update("", "Description")
		require.Error(t, err)
	})
}

func TestDepartment_IsRoot(t *testing.T) {
	tenantID := uuid.New()

	t.Run("returns true for root department", func(t *testing.T) {
		dept, _ := NewDepartment(tenantID, "ROOT", "Root")
		assert.True(t, dept.IsRoot())
	})

	t.Run("returns false for child department", func(t *testing.T) {
		parentID := uuid.New()
		dept, _ := NewDepartment(tenantID, "CHILD", "Child")
		_ = dept.SetParent(&parentID, "/"+parentID.String(), 0)

		assert.False(t, dept.IsRoot())
	})
}
