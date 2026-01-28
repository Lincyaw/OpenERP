package models

import (
	"time"

	"github.com/erp/backend/internal/domain/printing"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
)

// PrintTemplateModel is the GORM model for print_templates table
type PrintTemplateModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	TenantID     uuid.UUID `gorm:"type:uuid;not null;index"`
	DocumentType string    `gorm:"column:document_type;type:varchar(50);not null"`
	Name         string    `gorm:"type:varchar(100);not null"`
	Description  string    `gorm:"type:text"`
	Content      string    `gorm:"type:text;not null"`
	PaperSize    string    `gorm:"column:paper_size;type:varchar(20);not null;default:'A4'"`
	Orientation  string    `gorm:"type:varchar(20);not null;default:'PORTRAIT'"`
	MarginTop    int       `gorm:"column:margin_top;not null;default:10"`
	MarginRight  int       `gorm:"column:margin_right;not null;default:10"`
	MarginBottom int       `gorm:"column:margin_bottom;not null;default:10"`
	MarginLeft   int       `gorm:"column:margin_left;not null;default:10"`
	IsDefault    bool      `gorm:"column:is_default;not null;default:false"`
	Status       string    `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
	Version      int       `gorm:"not null;default:1"`
}

// TableName returns the table name for PrintTemplateModel
func (PrintTemplateModel) TableName() string {
	return "print_templates"
}

// ToDomain converts PrintTemplateModel to domain PrintTemplate
func (m *PrintTemplateModel) ToDomain() *printing.PrintTemplate {
	return &printing.PrintTemplate{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        m.ID,
					CreatedAt: m.CreatedAt,
					UpdatedAt: m.UpdatedAt,
				},
				Version: m.Version,
			},
			TenantID: m.TenantID,
		},
		DocumentType: printing.DocType(m.DocumentType),
		Name:         m.Name,
		Description:  m.Description,
		Content:      m.Content,
		PaperSize:    printing.PaperSize(m.PaperSize),
		Orientation:  printing.Orientation(m.Orientation),
		Margins: printing.Margins{
			Top:    m.MarginTop,
			Right:  m.MarginRight,
			Bottom: m.MarginBottom,
			Left:   m.MarginLeft,
		},
		IsDefault: m.IsDefault,
		Status:    printing.TemplateStatus(m.Status),
	}
}

// PrintTemplateModelFromDomain creates a PrintTemplateModel from domain PrintTemplate
func PrintTemplateModelFromDomain(t *printing.PrintTemplate) *PrintTemplateModel {
	return &PrintTemplateModel{
		ID:           t.ID,
		TenantID:     t.TenantID,
		DocumentType: string(t.DocumentType),
		Name:         t.Name,
		Description:  t.Description,
		Content:      t.Content,
		PaperSize:    string(t.PaperSize),
		Orientation:  string(t.Orientation),
		MarginTop:    t.Margins.Top,
		MarginRight:  t.Margins.Right,
		MarginBottom: t.Margins.Bottom,
		MarginLeft:   t.Margins.Left,
		IsDefault:    t.IsDefault,
		Status:       string(t.Status),
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		Version:      t.Version,
	}
}

// PrintJobModel is the GORM model for print_jobs table
type PrintJobModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key"`
	TenantID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	TemplateID     uuid.UUID  `gorm:"column:template_id;type:uuid;not null"`
	DocumentType   string     `gorm:"column:document_type;type:varchar(50);not null"`
	DocumentID     uuid.UUID  `gorm:"column:document_id;type:uuid;not null"`
	DocumentNumber string     `gorm:"column:document_number;type:varchar(100);not null"`
	Status         string     `gorm:"type:varchar(20);not null;default:'PENDING'"`
	Copies         int        `gorm:"not null;default:1"`
	PdfURL         string     `gorm:"column:pdf_url;type:text"`
	ErrorMessage   string     `gorm:"column:error_message;type:text"`
	PrintedAt      *time.Time `gorm:"column:printed_at"`
	PrintedBy      *uuid.UUID `gorm:"column:printed_by;type:uuid"`
	CreatedAt      time.Time  `gorm:"not null"`
	UpdatedAt      time.Time  `gorm:"not null"`
	Version        int        `gorm:"not null;default:1"`
}

// TableName returns the table name for PrintJobModel
func (PrintJobModel) TableName() string {
	return "print_jobs"
}

// ToDomain converts PrintJobModel to domain PrintJob
func (m *PrintJobModel) ToDomain() *printing.PrintJob {
	return &printing.PrintJob{
		TenantAggregateRoot: shared.TenantAggregateRoot{
			BaseAggregateRoot: shared.BaseAggregateRoot{
				BaseEntity: shared.BaseEntity{
					ID:        m.ID,
					CreatedAt: m.CreatedAt,
					UpdatedAt: m.UpdatedAt,
				},
				Version: m.Version,
			},
			TenantID: m.TenantID,
		},
		TemplateID:     m.TemplateID,
		DocumentType:   printing.DocType(m.DocumentType),
		DocumentID:     m.DocumentID,
		DocumentNumber: m.DocumentNumber,
		Status:         printing.JobStatus(m.Status),
		Copies:         m.Copies,
		PdfURL:         m.PdfURL,
		ErrorMessage:   m.ErrorMessage,
		PrintedAt:      m.PrintedAt,
		PrintedBy:      m.PrintedBy,
	}
}

// PrintJobModelFromDomain creates a PrintJobModel from domain PrintJob
func PrintJobModelFromDomain(j *printing.PrintJob) *PrintJobModel {
	return &PrintJobModel{
		ID:             j.ID,
		TenantID:       j.TenantID,
		TemplateID:     j.TemplateID,
		DocumentType:   string(j.DocumentType),
		DocumentID:     j.DocumentID,
		DocumentNumber: j.DocumentNumber,
		Status:         string(j.Status),
		Copies:         j.Copies,
		PdfURL:         j.PdfURL,
		ErrorMessage:   j.ErrorMessage,
		PrintedAt:      j.PrintedAt,
		PrintedBy:      j.PrintedBy,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
		Version:        j.Version,
	}
}
