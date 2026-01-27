package handler

import (
	"github.com/erp/backend/internal/interfaces/http/router"
	"github.com/gin-gonic/gin"
)

// PrintRoutes creates the route group for print-related endpoints
func PrintRoutes(handler *PrintHandler, authMiddleware gin.HandlerFunc) *router.DomainGroup {
	group := router.NewDomainGroup("print", "/print")
	group.Use(authMiddleware)

	// Template management
	group.POST("/templates", handler.CreateTemplate)
	group.GET("/templates", handler.ListTemplates)
	group.GET("/templates/:id", handler.GetTemplate)
	group.PUT("/templates/:id", handler.UpdateTemplate)
	group.DELETE("/templates/:id", handler.DeleteTemplate)
	group.POST("/templates/:id/set-default", handler.SetDefaultTemplate)
	group.POST("/templates/:id/activate", handler.ActivateTemplate)
	group.POST("/templates/:id/deactivate", handler.DeactivateTemplate)
	group.GET("/templates/:id/content", handler.GetTemplateContent)
	group.PUT("/templates/:id/content", handler.UpdateTemplateContent)
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
