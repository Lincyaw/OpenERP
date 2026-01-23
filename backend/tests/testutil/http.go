// Package testutil provides common test utilities for the ERP backend.
package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HTTPTestCase represents a test case for HTTP handler testing.
type HTTPTestCase struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	Headers        map[string]string
	ExpectedStatus int
	ExpectedBody   map[string]interface{}
	Setup          func(t *testing.T, tc *TestContext)
	Validate       func(t *testing.T, tc *TestContext)
}

// RunHTTPTestCases runs a slice of HTTP test cases against a handler.
func RunHTTPTestCases(t *testing.T, handler gin.HandlerFunc, cases []HTTPTestCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			RunHTTPTestCase(t, handler, tc)
		})
	}
}

// RunHTTPTestCase runs a single HTTP test case.
func RunHTTPTestCase(t *testing.T, handler gin.HandlerFunc, tc HTTPTestCase) {
	t.Helper()

	// Create request body
	var body io.Reader
	if tc.Body != nil {
		jsonBody, err := json.Marshal(tc.Body)
		require.NoError(t, err, "Failed to marshal request body")
		body = bytes.NewReader(jsonBody)
	}

	// Create request
	method := tc.Method
	if method == "" {
		method = http.MethodGet
	}
	path := tc.Path
	if path == "" {
		path = "/"
	}
	req := httptest.NewRequest(method, path, body)

	// Set headers
	if tc.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range tc.Headers {
		req.Header.Set(k, v)
	}

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Setup hook
	testCtx := &TestContext{Context: c, Recorder: w}
	if tc.Setup != nil {
		tc.Setup(t, testCtx)
	}

	// Execute handler
	handler(c)

	// Validate status code
	if tc.ExpectedStatus != 0 {
		assert.Equal(t, tc.ExpectedStatus, w.Code, "Unexpected status code")
	}

	// Validate body if expected
	if tc.ExpectedBody != nil {
		var actualBody map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &actualBody)
		require.NoError(t, err, "Failed to unmarshal response body")

		for key, expectedValue := range tc.ExpectedBody {
			assert.Equal(t, expectedValue, actualBody[key], "Unexpected value for key: %s", key)
		}
	}

	// Custom validation hook
	if tc.Validate != nil {
		tc.Validate(t, testCtx)
	}
}

// JSONResponse parses the response body as JSON.
func JSONResponse(t *testing.T, tc *TestContext) map[string]interface{} {
	t.Helper()

	var result map[string]interface{}
	err := json.Unmarshal(tc.ResponseBody(), &result)
	require.NoError(t, err, "Failed to parse JSON response")
	return result
}

// JSONResponseAs parses the response body into the provided struct.
func JSONResponseAs[T any](t *testing.T, tc *TestContext) T {
	t.Helper()

	var result T
	err := json.Unmarshal(tc.ResponseBody(), &result)
	require.NoError(t, err, "Failed to parse JSON response")
	return result
}

// AssertSuccessResponse asserts the response is a successful API response.
func AssertSuccessResponse(t *testing.T, tc *TestContext) {
	t.Helper()

	resp := JSONResponse(t, tc)
	assert.True(t, resp["success"].(bool), "Expected success to be true")
	assert.Nil(t, resp["error"], "Expected no error")
}

// AssertErrorResponse asserts the response is an error API response.
func AssertErrorResponse(t *testing.T, tc *TestContext, expectedCode string) {
	t.Helper()

	resp := JSONResponse(t, tc)
	assert.False(t, resp["success"].(bool), "Expected success to be false")

	errMap, ok := resp["error"].(map[string]interface{})
	require.True(t, ok, "Expected error object in response")
	assert.Equal(t, expectedCode, errMap["code"], "Unexpected error code")
}

// ToJSONReader converts a value to a JSON io.Reader.
func ToJSONReader(t *testing.T, v interface{}) io.Reader {
	t.Helper()

	data, err := json.Marshal(v)
	require.NoError(t, err, "Failed to marshal to JSON")
	return bytes.NewReader(data)
}
