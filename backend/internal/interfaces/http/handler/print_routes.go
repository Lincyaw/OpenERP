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

// PrintingWSRoutes creates the route group for printing WebSocket endpoints
// Note: WebSocket endpoint uses its own authentication via query param or header
// because WebSocket clients may not be able to set Authorization headers
func PrintingWSRoutes(wsHandler *PrintingWSHandler, authMiddleware gin.HandlerFunc) *router.DomainGroup {
	group := router.NewDomainGroup("printing", "/printing")
	// Auth middleware is applied but will also check query param token in handler
	group.Use(authMiddleware)

	// WebSocket endpoint for real-time printing events
	group.GET("/ws", wsHandler.Connect)

	return group
}

// AutoPrintRuleRoutes creates the route group for auto print rule endpoints
// Note: These endpoints require admin role for modification operations
func AutoPrintRuleRoutes(handler *AutoPrintRuleHandler, authMiddleware gin.HandlerFunc) *router.DomainGroup {
	group := router.NewDomainGroup("printing-auto-rules", "/printing")
	group.Use(authMiddleware)

	// Auto print rule CRUD endpoints
	// POST /api/v1/printing/auto-rules - Create rule (admin only)
	group.POST("/auto-rules", handler.CreateRule)

	// GET /api/v1/printing/auto-rules - List rules
	group.GET("/auto-rules", handler.ListRules)

	// GET /api/v1/printing/auto-rules/lookup - Lookup rule by document type and trigger event
	group.GET("/auto-rules/lookup", handler.LookupRule)

	// GET /api/v1/printing/auto-rules/trigger-events - Get trigger events for document type
	group.GET("/auto-rules/trigger-events", handler.GetTriggerEvents)

	// GET /api/v1/printing/auto-rules/:id - Get rule by ID
	group.GET("/auto-rules/:id", handler.GetRule)

	// PUT /api/v1/printing/auto-rules/:id - Update rule (admin only)
	group.PUT("/auto-rules/:id", handler.UpdateRule)

	// DELETE /api/v1/printing/auto-rules/:id - Delete rule (admin only)
	group.DELETE("/auto-rules/:id", handler.DeleteRule)

	// POST /api/v1/printing/auto-rules/:id/enable - Enable rule (admin only)
	group.POST("/auto-rules/:id/enable", handler.EnableRule)

	// POST /api/v1/printing/auto-rules/:id/disable - Disable rule (admin only)
	group.POST("/auto-rules/:id/disable", handler.DisableRule)

	return group
}
