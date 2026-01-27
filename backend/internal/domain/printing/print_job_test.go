package printing

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrintJob(t *testing.T) {
	tenantID := uuid.New()
	templateID := uuid.New()
	documentID := uuid.New()
	printedBy := uuid.New()

	tests := []struct {
		name           string
		tenantID       uuid.UUID
		templateID     uuid.UUID
		docType        DocType
		documentID     uuid.UUID
		documentNumber string
		printedBy      uuid.UUID
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "valid print job",
			tenantID:       tenantID,
			templateID:     templateID,
			docType:        DocTypeSalesOrder,
			documentID:     documentID,
			documentNumber: "SO-2024-001",
			printedBy:      printedBy,
			expectError:    false,
		},
		{
			name:           "nil template ID",
			tenantID:       tenantID,
			templateID:     uuid.Nil,
			docType:        DocTypeSalesOrder,
			documentID:     documentID,
			documentNumber: "SO-2024-001",
			printedBy:      printedBy,
			expectError:    true,
			errorMsg:       "Template ID cannot be empty",
		},
		{
			name:           "invalid doc type",
			tenantID:       tenantID,
			templateID:     templateID,
			docType:        DocType("INVALID"),
			documentID:     documentID,
			documentNumber: "SO-2024-001",
			printedBy:      printedBy,
			expectError:    true,
			errorMsg:       "Invalid document type",
		},
		{
			name:           "nil document ID",
			tenantID:       tenantID,
			templateID:     templateID,
			docType:        DocTypeSalesOrder,
			documentID:     uuid.Nil,
			documentNumber: "SO-2024-001",
			printedBy:      printedBy,
			expectError:    true,
			errorMsg:       "Document ID cannot be empty",
		},
		{
			name:           "empty document number",
			tenantID:       tenantID,
			templateID:     templateID,
			docType:        DocTypeSalesOrder,
			documentID:     documentID,
			documentNumber: "",
			printedBy:      printedBy,
			expectError:    true,
			errorMsg:       "Document number cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := NewPrintJob(tt.tenantID, tt.templateID, tt.docType, tt.documentID, tt.documentNumber, tt.printedBy)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, job)

				assert.Equal(t, tt.tenantID, job.TenantID)
				assert.Equal(t, tt.templateID, job.TemplateID)
				assert.Equal(t, tt.docType, job.DocumentType)
				assert.Equal(t, tt.documentID, job.DocumentID)
				assert.Equal(t, tt.documentNumber, job.DocumentNumber)
				assert.Equal(t, JobStatusPending, job.Status)
				assert.Equal(t, 1, job.Copies)
				assert.NotNil(t, job.PrintedBy)
				assert.Equal(t, tt.printedBy, *job.PrintedBy)
				assert.Empty(t, job.PrinterName)
				assert.Empty(t, job.PdfURL)
				assert.Empty(t, job.ErrorMessage)
				assert.Nil(t, job.PrintedAt)
				assert.NotEmpty(t, job.ID)
				assert.NotZero(t, job.CreatedAt)

				// Check that an event was created
				events := job.GetDomainEvents()
				require.Len(t, events, 1)
				assert.Equal(t, EventTypePrintJobCreated, events[0].EventType())
			}
		})
	}
}

func TestPrintJob_SetCopies(t *testing.T) {
	job := createTestPrintJob(t)

	tests := []struct {
		name        string
		copies      int
		expectError bool
	}{
		{"valid 1 copy", 1, false},
		{"valid 10 copies", 10, false},
		{"valid 100 copies", 100, false},
		{"invalid 0 copies", 0, true},
		{"invalid negative copies", -1, true},
		{"invalid too many copies", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := job.SetCopies(tt.copies)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.copies, job.Copies)
			}
		})
	}
}

func TestPrintJob_SetPrinterName(t *testing.T) {
	job := createTestPrintJob(t)

	job.SetPrinterName("Brother HL-L2350DW")
	assert.Equal(t, "Brother HL-L2350DW", job.PrinterName)

	job.SetPrinterName("")
	assert.Empty(t, job.PrinterName)
}

func TestPrintJob_StartRendering(t *testing.T) {
	job := createTestPrintJob(t)
	job.ClearDomainEvents()

	err := job.StartRendering()
	require.NoError(t, err)
	assert.Equal(t, JobStatusRendering, job.Status)

	// Check that an event was created
	events := job.GetDomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypePrintJobStatusChanged, events[0].EventType())
}

func TestPrintJob_StartRendering_InvalidState(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
	}{
		{"from RENDERING", JobStatusRendering},
		{"from COMPLETED", JobStatusCompleted},
		{"from FAILED", JobStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := createTestPrintJob(t)
			job.Status = tt.status

			err := job.StartRendering()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Cannot start rendering")
		})
	}
}

func TestPrintJob_Complete(t *testing.T) {
	job := createTestPrintJob(t)
	_ = job.StartRendering()
	job.ClearDomainEvents()

	pdfURL := "/data/prints/tenant/2024/01/job-123.pdf"
	err := job.Complete(pdfURL)
	require.NoError(t, err)

	assert.Equal(t, JobStatusCompleted, job.Status)
	assert.Equal(t, pdfURL, job.PdfURL)
	assert.NotNil(t, job.PrintedAt)

	// Check that events were created
	events := job.GetDomainEvents()
	require.Len(t, events, 2) // StatusChanged + Completed

	eventTypes := make([]string, len(events))
	for i, e := range events {
		eventTypes[i] = e.EventType()
	}
	assert.Contains(t, eventTypes, EventTypePrintJobStatusChanged)
	assert.Contains(t, eventTypes, EventTypePrintJobCompleted)
}

func TestPrintJob_Complete_EmptyPdfURL(t *testing.T) {
	job := createTestPrintJob(t)
	_ = job.StartRendering()

	err := job.Complete("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PDF URL cannot be empty")
}

func TestPrintJob_Complete_InvalidState(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
	}{
		{"from PENDING", JobStatusPending},
		{"from COMPLETED", JobStatusCompleted},
		{"from FAILED", JobStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := createTestPrintJob(t)
			job.Status = tt.status

			err := job.Complete("/path/to/pdf")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Cannot complete")
		})
	}
}

func TestPrintJob_Fail(t *testing.T) {
	job := createTestPrintJob(t)
	_ = job.StartRendering()
	job.ClearDomainEvents()

	errorMsg := "Failed to render PDF: out of memory"
	err := job.Fail(errorMsg)
	require.NoError(t, err)

	assert.Equal(t, JobStatusFailed, job.Status)
	assert.Equal(t, errorMsg, job.ErrorMessage)

	// Check that events were created
	events := job.GetDomainEvents()
	require.Len(t, events, 2) // StatusChanged + Failed

	eventTypes := make([]string, len(events))
	for i, e := range events {
		eventTypes[i] = e.EventType()
	}
	assert.Contains(t, eventTypes, EventTypePrintJobStatusChanged)
	assert.Contains(t, eventTypes, EventTypePrintJobFailed)
}

func TestPrintJob_Fail_FromPending(t *testing.T) {
	job := createTestPrintJob(t)
	job.ClearDomainEvents()

	err := job.Fail("Template not found")
	require.NoError(t, err)
	assert.Equal(t, JobStatusFailed, job.Status)
}

func TestPrintJob_Fail_AlreadyTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
	}{
		{"from COMPLETED", JobStatusCompleted},
		{"from FAILED", JobStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := createTestPrintJob(t)
			job.Status = tt.status

			err := job.Fail("Error")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Cannot fail")
		})
	}
}

func TestPrintJob_StatusChecks(t *testing.T) {
	job := createTestPrintJob(t)

	// PENDING
	assert.True(t, job.IsPending())
	assert.False(t, job.IsRendering())
	assert.False(t, job.IsCompleted())
	assert.False(t, job.IsFailed())
	assert.False(t, job.IsTerminal())

	// RENDERING
	_ = job.StartRendering()
	assert.False(t, job.IsPending())
	assert.True(t, job.IsRendering())
	assert.False(t, job.IsCompleted())
	assert.False(t, job.IsFailed())
	assert.False(t, job.IsTerminal())

	// COMPLETED
	_ = job.Complete("/path/to/pdf")
	assert.False(t, job.IsPending())
	assert.False(t, job.IsRendering())
	assert.True(t, job.IsCompleted())
	assert.False(t, job.IsFailed())
	assert.True(t, job.IsTerminal())

	// Test FAILED separately
	job2 := createTestPrintJob(t)
	_ = job2.Fail("Error")
	assert.False(t, job2.IsPending())
	assert.False(t, job2.IsRendering())
	assert.False(t, job2.IsCompleted())
	assert.True(t, job2.IsFailed())
	assert.True(t, job2.IsTerminal())
}

func TestPrintJob_HasPDF(t *testing.T) {
	job := createTestPrintJob(t)
	assert.False(t, job.HasPDF())

	_ = job.StartRendering()
	_ = job.Complete("/path/to/pdf")
	assert.True(t, job.HasPDF())
}

// Helper function to create a test print job
func createTestPrintJob(t *testing.T) *PrintJob {
	t.Helper()
	job, err := NewPrintJob(
		uuid.New(),
		uuid.New(),
		DocTypeSalesOrder,
		uuid.New(),
		"SO-2024-001",
		uuid.New(),
	)
	require.NoError(t, err)
	return job
}
