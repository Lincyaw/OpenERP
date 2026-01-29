package handler

import (
	"github.com/erp/backend/internal/interfaces/http/router"
	"github.com/gin-gonic/gin"
)

// PrintRoutes creates the route group for print-related endpoints
func PrintRoutes(handler *PrintHandler, authMiddleware gin.HandlerFunc) *router.DomainGroup {
	group := router.NewDomainGroup("print", "/print")
	group.Use(authMiddleware)

	// Template queries (read-only from static templates)
	group.GET("/templates/by-doc-type/:doc_type", handler.GetTemplatesByDocType)

	// Preview and PDF generation
	group.POST("/preview", handler.PreviewDocument)
	group.POST("/generate", handler.GeneratePDF)

	// Print jobs
	group.GET("/jobs", handler.ListJobs)
	group.GET("/jobs/:id", handler.GetJob)
	group.GET("/jobs/:id/download", handler.DownloadPDF)
	group.GET("/jobs/by-document/:doc_type/:document_id", handler.GetJobsByDocument)

	// Reference data
	group.GET("/document-types", handler.GetDocumentTypes)
	group.GET("/paper-sizes", handler.GetPaperSizes)

	return group
}
