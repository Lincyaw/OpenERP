package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystemHandler(t *testing.T) {
	h := NewSystemHandler()
	assert.NotNil(t, h)
	assert.False(t, h.startTime.IsZero())
}

func TestSystemHandler_GetSystemInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewSystemHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/system/info", nil)

	h.GetSystemInfo(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "ERP Backend API", data["name"])
	assert.Equal(t, "1.0.0", data["version"])
	assert.NotEmpty(t, data["go_version"])
	assert.NotEmpty(t, data["uptime"])
}

func TestSystemHandler_Ping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewSystemHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/system/ping", nil)

	h.Ping(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "pong", data["message"])
	assert.NotEmpty(t, data["timestamp"])

	// Verify timestamp is valid RFC3339
	timestamp := data["timestamp"].(string)
	_, err = time.Parse(time.RFC3339, timestamp)
	assert.NoError(t, err)
}
