package handler

import (
	"net/http"
	"runtime"
	"time"

	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
)

// SystemHandler handles system-related API endpoints
// @name HandlerSystemInfoResponse
type SystemHandler struct {
	BaseHandler
	startTime time.Time
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{
		startTime: time.Now(),
	}
}

// SystemInfoResponse represents the system information response
// @name HandlerSystemInfoResponse
type SystemInfoResponse struct {
	Name      string `json:"name" example:"ERP Backend API"`
	Version   string `json:"version" example:"1.0.0"`
	GoVersion string `json:"go_version" example:"go1.25.5"`
	Uptime    string `json:"uptime" example:"1h30m45s"`
}

// GetSystemInfo godoc
// @ID           getSystemSystemInfo
// @Summary      Get system information
// @Description  Returns basic system information including version and uptime
// @Tags         system
// @Produce      json
// @Success      200 {object} APIResponse[SystemInfoResponse]
// @Failure      500 {object} ErrorResponse
// @Router       /system/info [get]
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	info := SystemInfoResponse{
		Name:      "ERP Backend API",
		Version:   "1.0.0",
		GoVersion: runtime.Version(),
		Uptime:    time.Since(h.startTime).Round(time.Second).String(),
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(info))
}

// PingResponse represents the ping response
// @name HandlerPingResponse
type PingResponse struct {
	Message   string `json:"message" example:"pong"`
	Timestamp string `json:"timestamp" example:"2026-01-23T12:00:00Z"`
}

// Ping godoc
// @ID           pingSystem
// @Summary      Ping the API
// @Description  Simple ping endpoint to check if the API is responsive
// @Tags         system
// @Produce      json
// @Success      200 {object} APIResponse[PingResponse]
// @Router       /system/ping [get]
func (h *SystemHandler) Ping(c *gin.Context) {
	response := PingResponse{
		Message:   "pong",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(response))
}
