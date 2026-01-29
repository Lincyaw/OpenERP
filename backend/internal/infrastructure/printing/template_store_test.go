package printing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateStore(t *testing.T) {
	t.Run("creates store with embedded templates", func(t *testing.T) {
		store, err := NewTemplateStore(nil)
		require.NoError(t, err)
		require.NotNil(t, store)

		templates := store.GetAll()
		assert.NotEmpty(t, templates)
	})

	t.Run("creates store with empty config", func(t *testing.T) {
		store, err := NewTemplateStore(&TemplateStoreConfig{})
		require.NoError(t, err)
		require.NotNil(t, store)

		templates := store.GetAll()
		assert.NotEmpty(t, templates)
	})
}

func TestTemplateStore_GetByDocType(t *testing.T) {
	store, err := NewTemplateStore(nil)
	require.NoError(t, err)

	t.Run("returns templates for valid doc type", func(t *testing.T) {
		templates := store.GetByDocType(printing.DocTypeSalesDelivery)
		assert.NotEmpty(t, templates)

		for _, tmpl := range templates {
			assert.Equal(t, printing.DocTypeSalesDelivery, tmpl.DocType)
		}
	})

	t.Run("returns empty slice for invalid doc type", func(t *testing.T) {
		templates := store.GetByDocType(printing.DocType("INVALID"))
		assert.Empty(t, templates)
	})
}

func TestTemplateStore_GetDefault(t *testing.T) {
	store, err := NewTemplateStore(nil)
	require.NoError(t, err)

	t.Run("returns default template for doc type", func(t *testing.T) {
		tmpl := store.GetDefault(printing.DocTypeSalesDelivery)
		require.NotNil(t, tmpl)
		assert.True(t, tmpl.IsDefault)
		assert.Equal(t, printing.DocTypeSalesDelivery, tmpl.DocType)
	})

	t.Run("returns nil for doc type without templates", func(t *testing.T) {
		tmpl := store.GetDefault(printing.DocType("INVALID"))
		assert.Nil(t, tmpl)
	})
}

func TestTemplateStore_GetByID(t *testing.T) {
	store, err := NewTemplateStore(nil)
	require.NoError(t, err)

	t.Run("returns template by ID", func(t *testing.T) {
		// Get all templates and pick the first one
		templates := store.GetAll()
		require.NotEmpty(t, templates)

		first := templates[0]
		found := store.GetByID(first.ID)
		require.NotNil(t, found)
		assert.Equal(t, first.ID, found.ID)
		assert.Equal(t, first.Name, found.Name)
	})

	t.Run("returns nil for non-existent ID", func(t *testing.T) {
		found := store.GetByID("non-existent-id")
		assert.Nil(t, found)
	})
}

func TestTemplateStore_TemplateContent(t *testing.T) {
	store, err := NewTemplateStore(nil)
	require.NoError(t, err)

	t.Run("templates have loaded content", func(t *testing.T) {
		templates := store.GetAll()
		for _, tmpl := range templates {
			assert.NotEmpty(t, tmpl.Content, "Template %s should have content", tmpl.Name)
			assert.Contains(t, tmpl.Content, "<html", "Template %s should contain HTML", tmpl.Name)
		}
	})
}

func TestTemplateStore_StableIDs(t *testing.T) {
	t.Run("same doc type, paper size, and orientation produce same ID", func(t *testing.T) {
		id1 := generateTemplateID(printing.DocTypeSalesDelivery, printing.PaperSizeA4, printing.OrientationPortrait)
		id2 := generateTemplateID(printing.DocTypeSalesDelivery, printing.PaperSizeA4, printing.OrientationPortrait)
		assert.Equal(t, id1, id2)
	})

	t.Run("different paper sizes produce different IDs", func(t *testing.T) {
		id1 := generateTemplateID(printing.DocTypeSalesDelivery, printing.PaperSizeA4, printing.OrientationPortrait)
		id2 := generateTemplateID(printing.DocTypeSalesDelivery, printing.PaperSizeA5, printing.OrientationPortrait)
		assert.NotEqual(t, id1, id2)
	})

	t.Run("different orientations produce different IDs", func(t *testing.T) {
		id1 := generateTemplateID(printing.DocTypeStockTaking, printing.PaperSizeA4, printing.OrientationPortrait)
		id2 := generateTemplateID(printing.DocTypeStockTaking, printing.PaperSizeA4, printing.OrientationLandscape)
		assert.NotEqual(t, id1, id2)
	})
}

func TestTemplateStore_ToPrintTemplate(t *testing.T) {
	store, err := NewTemplateStore(nil)
	require.NoError(t, err)

	t.Run("converts static template to domain template", func(t *testing.T) {
		templates := store.GetAll()
		require.NotEmpty(t, templates)

		static := &templates[0]
		domain := static.ToPrintTemplate()

		require.NotNil(t, domain)
		assert.Equal(t, static.DocType, domain.DocumentType)
		assert.Equal(t, static.Name, domain.Name)
		assert.Equal(t, static.Content, domain.Content)
		assert.Equal(t, static.PaperSize, domain.PaperSize)
		assert.Equal(t, static.Orientation, domain.Orientation)
		assert.Equal(t, static.Margins, domain.Margins)
		assert.Equal(t, printing.TemplateStatusActive, domain.Status)
	})
}

func TestTemplateStore_ExternalDir(t *testing.T) {
	t.Run("loads from external dir when file exists", func(t *testing.T) {
		// Create temp dir with custom template
		tmpDir, err := os.MkdirTemp("", "templates")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a custom template file
		customContent := `<!DOCTYPE html><html><body>Custom Template</body></html>`
		err = os.WriteFile(filepath.Join(tmpDir, "sales_delivery_a4.html"), []byte(customContent), 0644)
		require.NoError(t, err)

		store, err := NewTemplateStore(&TemplateStoreConfig{
			ExternalDir: tmpDir,
		})
		require.NoError(t, err)

		// Find the sales delivery A4 template
		tmpl := store.GetDefault(printing.DocTypeSalesDelivery)
		require.NotNil(t, tmpl)

		// Should have loaded the custom content
		assert.Contains(t, tmpl.Content, "Custom Template")
	})

	t.Run("falls back to embedded when external file not found", func(t *testing.T) {
		// Create empty temp dir
		tmpDir, err := os.MkdirTemp("", "templates")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		store, err := NewTemplateStore(&TemplateStoreConfig{
			ExternalDir: tmpDir,
		})
		require.NoError(t, err)

		// Should still have templates from embedded
		templates := store.GetAll()
		assert.NotEmpty(t, templates)
	})
}

func TestTemplateStore_Reload(t *testing.T) {
	t.Run("reload refreshes templates", func(t *testing.T) {
		store, err := NewTemplateStore(nil)
		require.NoError(t, err)

		countBefore := len(store.GetAll())

		err = store.Reload()
		require.NoError(t, err)

		countAfter := len(store.GetAll())
		assert.Equal(t, countBefore, countAfter)
	})
}
