package csvimport

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityType(t *testing.T) {
	t.Run("ValidEntityTypes", func(t *testing.T) {
		types := ValidEntityTypes()
		assert.Contains(t, types, EntityProducts)
		assert.Contains(t, types, EntityCustomers)
		assert.Contains(t, types, EntitySuppliers)
		assert.Contains(t, types, EntityInventory)
		assert.Contains(t, types, EntityCategories)
	})

	t.Run("IsValidEntityType", func(t *testing.T) {
		assert.True(t, IsValidEntityType("products"))
		assert.True(t, IsValidEntityType("customers"))
		assert.True(t, IsValidEntityType("suppliers"))
		assert.True(t, IsValidEntityType("inventory"))
		assert.True(t, IsValidEntityType("categories"))
		assert.False(t, IsValidEntityType("invalid"))
		assert.False(t, IsValidEntityType(""))
	})
}

func TestImportSession(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("NewImportSession", func(t *testing.T) {
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		assert.NotEqual(t, uuid.Nil, session.ID)
		assert.Equal(t, tenantID, session.TenantID)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, EntityProducts, session.EntityType)
		assert.Equal(t, "test.csv", session.FileName)
		assert.Equal(t, int64(1024), session.FileSize)
		assert.Equal(t, StateCreated, session.State)
		assert.Nil(t, session.CompletedAt)
	})

	t.Run("UpdateState", func(t *testing.T) {
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		session.UpdateState(StateValidating)
		assert.Equal(t, StateValidating, session.State)
		assert.Nil(t, session.CompletedAt)

		session.UpdateState(StateCompleted)
		assert.Equal(t, StateCompleted, session.State)
		assert.NotNil(t, session.CompletedAt)
	})

	t.Run("SetValidationResult", func(t *testing.T) {
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)
		result := &ValidationResult{
			ValidationID: session.ID.String(),
			TotalRows:    100,
			ValidRows:    95,
			ErrorRows:    5,
			Errors:       []RowError{{Row: 10, Column: "code", Message: "required"}},
			Preview:      []map[string]any{{"code": "001", "name": "Widget"}},
		}

		session.SetValidationResult(result)

		assert.Equal(t, 100, session.TotalRows)
		assert.Equal(t, 95, session.ValidRows)
		assert.Equal(t, 5, session.ErrorRows)
		assert.Len(t, session.Errors, 1)
		assert.Len(t, session.Preview, 1)
	})

	t.Run("IsValid", func(t *testing.T) {
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)
		session.ErrorRows = 0
		assert.True(t, session.IsValid())

		session.ErrorRows = 5
		assert.False(t, session.IsValid())
	})
}

func TestImportContext(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

	t.Run("NewImportContext", func(t *testing.T) {
		ctx := context.Background()
		importCtx := NewImportContext(ctx, session)

		assert.NotNil(t, importCtx.Context())
		assert.Equal(t, session, importCtx.Session())
		assert.Nil(t, importCtx.Parser())
	})

	t.Run("Cancel", func(t *testing.T) {
		ctx := context.Background()
		importCtx := NewImportContext(ctx, session)

		importCtx.Cancel()

		assert.Equal(t, context.Canceled, importCtx.Context().Err())
		assert.Equal(t, StateCancelled, session.State)
	})

	t.Run("Valid rows management", func(t *testing.T) {
		ctx := context.Background()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)
		importCtx := NewImportContext(ctx, session)

		row1 := &Row{LineNumber: 2, Data: map[string]string{"code": "001"}}
		row2 := &Row{LineNumber: 3, Data: map[string]string{"code": "002"}}

		importCtx.AddValidRow(row1)
		importCtx.AddValidRow(row2)

		validRows := importCtx.ValidRows()
		assert.Len(t, validRows, 2)
	})

	t.Run("Error row tracking", func(t *testing.T) {
		ctx := context.Background()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)
		importCtx := NewImportContext(ctx, session)

		importCtx.MarkRowError(5)
		importCtx.MarkRowError(10)

		assert.True(t, importCtx.HasRowError(5))
		assert.True(t, importCtx.HasRowError(10))
		assert.False(t, importCtx.HasRowError(7))
		assert.Equal(t, 2, importCtx.ErrorCount())
	})

	t.Run("With validators", func(t *testing.T) {
		ctx := context.Background()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		rules := []FieldRule{Field("code").Required().Build()}
		fieldVal := NewFieldValidator(rules, 10)

		importCtx := NewImportContext(ctx, session, WithFieldValidator(fieldVal))

		assert.NotNil(t, importCtx)
	})
}

func TestImportProcessor(t *testing.T) {
	t.Run("NewImportProcessor with defaults", func(t *testing.T) {
		processor := NewImportProcessor()
		assert.NotNil(t, processor)
	})

	t.Run("NewImportProcessor with options", func(t *testing.T) {
		processor := NewImportProcessor(
			WithMaxFileSize(5*1024*1024),
			WithMaxRows(50000),
			WithMaxErrors(50),
			WithPreviewRows(10),
		)
		assert.NotNil(t, processor)
	})

	t.Run("Validate simple CSV", func(t *testing.T) {
		processor := NewImportProcessor()
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		csv := "code,name,unit\n001,Widget,pcs\n002,Gadget,pcs\n003,Gizmo,pcs"
		rules := []FieldRule{
			Field("code").Required().String().MaxLength(50).Build(),
			Field("name").Required().String().MaxLength(200).Build(),
			Field("unit").Required().String().MaxLength(20).Build(),
		}

		result, err := processor.Validate(context.Background(), session, strings.NewReader(csv), rules)

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalRows)
		assert.Equal(t, 3, result.ValidRows)
		assert.Equal(t, 0, result.ErrorRows)
		assert.True(t, result.IsValid())
	})

	t.Run("Validate CSV with errors", func(t *testing.T) {
		processor := NewImportProcessor()
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		csv := "code,name,unit\n001,Widget,pcs\n,Gadget,pcs\n003,,pcs"
		rules := []FieldRule{
			Field("code").Required().Build(),
			Field("name").Required().Build(),
			Field("unit").Required().Build(),
		}

		result, err := processor.Validate(context.Background(), session, strings.NewReader(csv), rules)

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalRows)
		assert.Equal(t, 1, result.ValidRows)
		assert.Equal(t, 2, result.ErrorRows)
		assert.False(t, result.IsValid())
		assert.GreaterOrEqual(t, len(result.Errors), 2)
	})

	t.Run("Validate generates preview", func(t *testing.T) {
		processor := NewImportProcessor(WithPreviewRows(3))
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		csv := "code,name\n001,A\n002,B\n003,C\n004,D\n005,E"
		rules := []FieldRule{
			Field("code").Build(),
			Field("name").Build(),
		}

		result, err := processor.Validate(context.Background(), session, strings.NewReader(csv), rules)

		require.NoError(t, err)
		assert.Len(t, result.Preview, 3)
		assert.Equal(t, "001", result.Preview[0]["code"])
		assert.Equal(t, "002", result.Preview[1]["code"])
		assert.Equal(t, "003", result.Preview[2]["code"])
	})

	t.Run("Validate with reference lookup", func(t *testing.T) {
		lookupFn := func(refType, value string) (bool, error) {
			return value == "CAT-001", nil
		}

		processor := NewImportProcessor(WithReferenceLookup(lookupFn))
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		csv := "code,category\n001,CAT-001\n002,CAT-999"
		rules := []FieldRule{
			Field("code").Required().Build(),
			Field("category").Reference("category").Build(),
		}

		result, err := processor.Validate(context.Background(), session, strings.NewReader(csv), rules)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ErrorRows)
	})

	t.Run("Validate with uniqueness lookup", func(t *testing.T) {
		lookupFn := func(entityType, field, value string) (bool, error) {
			return value == "EXISTING", nil
		}

		processor := NewImportProcessor(WithUniqueLookup(lookupFn))
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		csv := "code,name\nNEW,Widget\nEXISTING,Gadget"
		rules := []FieldRule{
			Field("code").Required().Unique().Build(),
			Field("name").Required().Build(),
		}

		result, err := processor.Validate(context.Background(), session, strings.NewReader(csv), rules)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ErrorRows)
	})

	t.Run("Validate context cancellation", func(t *testing.T) {
		processor := NewImportProcessor()
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		csv := "code,name\n001,Widget"
		rules := []FieldRule{
			Field("code").Build(),
		}

		_, err := processor.Validate(ctx, session, strings.NewReader(csv), rules)

		assert.Error(t, err)
		assert.Equal(t, StateCancelled, session.State)
	})

	t.Run("Session state updates", func(t *testing.T) {
		processor := NewImportProcessor()
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		csv := "code,name\n001,Widget"
		rules := []FieldRule{
			Field("code").Build(),
			Field("name").Build(),
		}

		_, err := processor.Validate(context.Background(), session, strings.NewReader(csv), rules)

		require.NoError(t, err)
		assert.Equal(t, StateValidated, session.State)
	})

	t.Run("Session state updates on error", func(t *testing.T) {
		processor := NewImportProcessor()
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		csv := "code,name\n,Widget" // Missing required field
		rules := []FieldRule{
			Field("code").Required().Build(),
			Field("name").Build(),
		}

		_, err := processor.Validate(context.Background(), session, strings.NewReader(csv), rules)

		require.NoError(t, err)
		assert.Equal(t, StateFailed, session.State)
	})
}

func TestInMemorySessionStore(t *testing.T) {
	t.Run("Save and Get", func(t *testing.T) {
		store := NewInMemorySessionStore(time.Hour)
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		err := store.Save(session)
		require.NoError(t, err)

		retrieved, err := store.Get(session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrieved.ID)
	})

	t.Run("Get non-existent session", func(t *testing.T) {
		store := NewInMemorySessionStore(time.Hour)

		session, err := store.Get(uuid.New())
		require.NoError(t, err)
		assert.Nil(t, session)
	})

	t.Run("Delete session", func(t *testing.T) {
		store := NewInMemorySessionStore(time.Hour)
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		store.Save(session)
		err := store.Delete(session.ID)
		require.NoError(t, err)

		retrieved, _ := store.Get(session.ID)
		assert.Nil(t, retrieved)
	})

	t.Run("GetByTenant", func(t *testing.T) {
		store := NewInMemorySessionStore(time.Hour)
		tenantID := uuid.New()
		userID := uuid.New()

		session1 := NewImportSession(tenantID, userID, EntityProducts, "test1.csv", 1024)
		session2 := NewImportSession(tenantID, userID, EntityProducts, "test2.csv", 1024)
		session3 := NewImportSession(uuid.New(), userID, EntityProducts, "test3.csv", 1024) // Different tenant

		store.Save(session1)
		store.Save(session2)
		store.Save(session3)

		sessions, err := store.GetByTenant(tenantID, 10)
		require.NoError(t, err)
		assert.Len(t, sessions, 2)
	})

	t.Run("TTL expiration", func(t *testing.T) {
		store := NewInMemorySessionStore(time.Millisecond * 10)
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		store.Save(session)

		// Wait for TTL to expire
		time.Sleep(time.Millisecond * 20)

		retrieved, err := store.Get(session.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("Cleanup removes expired", func(t *testing.T) {
		store := NewInMemorySessionStore(time.Millisecond * 10)
		tenantID := uuid.New()
		userID := uuid.New()
		session := NewImportSession(tenantID, userID, EntityProducts, "test.csv", 1024)

		store.Save(session)

		// Wait for TTL to expire
		time.Sleep(time.Millisecond * 20)

		store.Cleanup()

		// Direct check - should have been cleaned up
		store.mu.RLock()
		defer store.mu.RUnlock()
		assert.Empty(t, store.sessions)
	})
}
