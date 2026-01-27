package printing

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrintTemplate(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name         string
		tenantID     uuid.UUID
		docType      DocType
		templateName string
		content      string
		paperSize    PaperSize
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid A4 template",
			tenantID:     tenantID,
			docType:      DocTypeSalesOrder,
			templateName: "Sales Order A4",
			content:      "<html><body>Test</body></html>",
			paperSize:    PaperSizeA4,
			expectError:  false,
		},
		{
			name:         "valid receipt template",
			tenantID:     tenantID,
			docType:      DocTypeSalesReceipt,
			templateName: "Receipt 58mm",
			content:      "<div>Receipt</div>",
			paperSize:    PaperSizeReceipt58MM,
			expectError:  false,
		},
		{
			name:         "invalid doc type",
			tenantID:     tenantID,
			docType:      DocType("INVALID"),
			templateName: "Test",
			content:      "<html>Test</html>",
			paperSize:    PaperSizeA4,
			expectError:  true,
			errorMsg:     "Invalid document type",
		},
		{
			name:         "empty name",
			tenantID:     tenantID,
			docType:      DocTypeSalesOrder,
			templateName: "",
			content:      "<html>Test</html>",
			paperSize:    PaperSizeA4,
			expectError:  true,
			errorMsg:     "name cannot be empty",
		},
		{
			name:         "whitespace only name",
			tenantID:     tenantID,
			docType:      DocTypeSalesOrder,
			templateName: "   ",
			content:      "<html>Test</html>",
			paperSize:    PaperSizeA4,
			expectError:  true,
			errorMsg:     "name cannot be empty",
		},
		{
			name:         "name too long",
			tenantID:     tenantID,
			docType:      DocTypeSalesOrder,
			templateName: strings.Repeat("a", 101),
			content:      "<html>Test</html>",
			paperSize:    PaperSizeA4,
			expectError:  true,
			errorMsg:     "cannot exceed 100 characters",
		},
		{
			name:         "empty content",
			tenantID:     tenantID,
			docType:      DocTypeSalesOrder,
			templateName: "Test",
			content:      "",
			paperSize:    PaperSizeA4,
			expectError:  true,
			errorMsg:     "content cannot be empty",
		},
		{
			name:         "whitespace only content",
			tenantID:     tenantID,
			docType:      DocTypeSalesOrder,
			templateName: "Test",
			content:      "   ",
			paperSize:    PaperSizeA4,
			expectError:  true,
			errorMsg:     "content cannot be empty",
		},
		{
			name:         "invalid paper size",
			tenantID:     tenantID,
			docType:      DocTypeSalesOrder,
			templateName: "Test",
			content:      "<html>Test</html>",
			paperSize:    PaperSize("INVALID"),
			expectError:  true,
			errorMsg:     "Invalid paper size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewPrintTemplate(tt.tenantID, tt.docType, tt.templateName, tt.content, tt.paperSize)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, template)

				assert.Equal(t, tt.tenantID, template.TenantID)
				assert.Equal(t, tt.docType, template.DocumentType)
				assert.Equal(t, strings.TrimSpace(tt.templateName), template.Name)
				assert.Equal(t, tt.content, template.Content)
				assert.Equal(t, tt.paperSize, template.PaperSize)
				assert.Equal(t, OrientationPortrait, template.Orientation)
				assert.Equal(t, TemplateStatusActive, template.Status)
				assert.False(t, template.IsDefault)
				assert.NotEmpty(t, template.ID)
				assert.NotZero(t, template.CreatedAt)
				assert.NotZero(t, template.UpdatedAt)

				// Check that receipt paper uses receipt margins
				if tt.paperSize.IsReceipt() {
					assert.Equal(t, ReceiptMargins(), template.Margins)
				} else {
					assert.Equal(t, DefaultMargins(), template.Margins)
				}

				// Check that an event was created
				events := template.GetDomainEvents()
				require.Len(t, events, 1)
				assert.Equal(t, EventTypePrintTemplateCreated, events[0].EventType())
			}
		})
	}
}

func TestPrintTemplate_Update(t *testing.T) {
	template := createTestTemplate(t)
	originalVersion := template.Version

	err := template.Update("Updated Name", "New Description")
	require.NoError(t, err)

	assert.Equal(t, "Updated Name", template.Name)
	assert.Equal(t, "New Description", template.Description)
	assert.Greater(t, template.Version, originalVersion)

	// Check that an update event was created
	events := template.GetDomainEvents()
	assert.Len(t, events, 2) // Created + Updated
}

func TestPrintTemplate_Update_EmptyName(t *testing.T) {
	template := createTestTemplate(t)

	err := template.Update("", "Description")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestPrintTemplate_UpdateContent(t *testing.T) {
	template := createTestTemplate(t)
	originalVersion := template.Version

	newContent := "<html><body>New Content</body></html>"
	err := template.UpdateContent(newContent)
	require.NoError(t, err)

	assert.Equal(t, newContent, template.Content)
	assert.Greater(t, template.Version, originalVersion)
}

func TestPrintTemplate_UpdateContent_Empty(t *testing.T) {
	template := createTestTemplate(t)

	err := template.UpdateContent("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content cannot be empty")
}

func TestPrintTemplate_SetPaperSize(t *testing.T) {
	template := createTestTemplate(t)

	err := template.SetPaperSize(PaperSizeA5)
	require.NoError(t, err)
	assert.Equal(t, PaperSizeA5, template.PaperSize)

	// Setting to receipt paper should auto-adjust margins
	err = template.SetPaperSize(PaperSizeReceipt80MM)
	require.NoError(t, err)
	assert.Equal(t, PaperSizeReceipt80MM, template.PaperSize)
	assert.Equal(t, ReceiptMargins(), template.Margins)
}

func TestPrintTemplate_SetPaperSize_Invalid(t *testing.T) {
	template := createTestTemplate(t)

	err := template.SetPaperSize(PaperSize("INVALID"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid paper size")
}

func TestPrintTemplate_SetOrientation(t *testing.T) {
	template := createTestTemplate(t)

	err := template.SetOrientation(OrientationLandscape)
	require.NoError(t, err)
	assert.Equal(t, OrientationLandscape, template.Orientation)
}

func TestPrintTemplate_SetOrientation_Invalid(t *testing.T) {
	template := createTestTemplate(t)

	err := template.SetOrientation(Orientation("INVALID"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid orientation")
}

func TestPrintTemplate_SetMargins(t *testing.T) {
	template := createTestTemplate(t)
	margins := Margins{Top: 20, Right: 15, Bottom: 25, Left: 10}

	err := template.SetMargins(margins)
	require.NoError(t, err)
	assert.Equal(t, margins, template.Margins)
}

func TestPrintTemplate_SetAsDefault(t *testing.T) {
	template := createTestTemplate(t)

	err := template.SetAsDefault()
	require.NoError(t, err)
	assert.True(t, template.IsDefault)

	// Check that an event was created
	events := template.GetDomainEvents()
	var hasDefaultEvent bool
	for _, e := range events {
		if e.EventType() == EventTypePrintTemplateSetAsDefault {
			hasDefaultEvent = true
			break
		}
	}
	assert.True(t, hasDefaultEvent)

	// Setting as default again should be a no-op
	template.ClearDomainEvents()
	err = template.SetAsDefault()
	require.NoError(t, err)
	assert.Len(t, template.GetDomainEvents(), 0)
}

func TestPrintTemplate_SetAsDefault_Inactive(t *testing.T) {
	template := createTestTemplate(t)
	template.Status = TemplateStatusInactive

	err := template.SetAsDefault()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inactive")
}

func TestPrintTemplate_UnsetDefault(t *testing.T) {
	template := createTestTemplate(t)
	_ = template.SetAsDefault()

	template.UnsetDefault()
	assert.False(t, template.IsDefault)
}

func TestPrintTemplate_Activate(t *testing.T) {
	template := createTestTemplate(t)
	template.Status = TemplateStatusInactive
	template.ClearDomainEvents()

	err := template.Activate()
	require.NoError(t, err)
	assert.Equal(t, TemplateStatusActive, template.Status)

	// Check that an event was created
	events := template.GetDomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypePrintTemplateStatusChanged, events[0].EventType())
}

func TestPrintTemplate_Activate_AlreadyActive(t *testing.T) {
	template := createTestTemplate(t)

	err := template.Activate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already active")
}

func TestPrintTemplate_Deactivate(t *testing.T) {
	template := createTestTemplate(t)
	template.ClearDomainEvents()

	err := template.Deactivate()
	require.NoError(t, err)
	assert.Equal(t, TemplateStatusInactive, template.Status)

	// Check that an event was created
	events := template.GetDomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypePrintTemplateStatusChanged, events[0].EventType())
}

func TestPrintTemplate_Deactivate_AlreadyInactive(t *testing.T) {
	template := createTestTemplate(t)
	template.Status = TemplateStatusInactive

	err := template.Deactivate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already inactive")
}

func TestPrintTemplate_Deactivate_Default(t *testing.T) {
	template := createTestTemplate(t)
	_ = template.SetAsDefault()

	err := template.Deactivate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deactivate")
	assert.Contains(t, err.Error(), "default")
}

func TestPrintTemplate_IsActive(t *testing.T) {
	template := createTestTemplate(t)
	assert.True(t, template.IsActive())

	template.Status = TemplateStatusInactive
	assert.False(t, template.IsActive())
}

func TestPrintTemplate_CanBeUsed(t *testing.T) {
	template := createTestTemplate(t)
	assert.True(t, template.CanBeUsed())

	// Inactive template cannot be used
	template.Status = TemplateStatusInactive
	assert.False(t, template.CanBeUsed())

	// Active but empty content cannot be used
	template.Status = TemplateStatusActive
	template.Content = ""
	assert.False(t, template.CanBeUsed())
}

// Helper function to create a test template
func createTestTemplate(t *testing.T) *PrintTemplate {
	t.Helper()
	template, err := NewPrintTemplate(
		uuid.New(),
		DocTypeSalesOrder,
		"Test Template",
		"<html><body>Test</body></html>",
		PaperSizeA4,
	)
	require.NoError(t, err)
	return template
}
