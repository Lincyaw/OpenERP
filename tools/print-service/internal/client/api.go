package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Constants for API client limits.
const (
	// maxResponseBodySize is the maximum size of response body to read (10MB).
	maxResponseBodySize = 10 * 1024 * 1024
	// maxPDFSize is the maximum size of PDF files to download (100MB).
	maxPDFSize = 100 * 1024 * 1024
)

// APIClientConfig holds HTTP API client configuration.
type APIClientConfig struct {
	// BaseURL is the base URL of the ERP backend API.
	BaseURL string

	// Token is the authentication token.
	Token string

	// TenantID is the tenant identifier.
	TenantID string

	// Timeout is the request timeout.
	Timeout time.Duration

	// Logger is the logger instance.
	Logger *zap.Logger
}

// APIClient is an HTTP client for the ERP backend API.
type APIClient struct {
	config     APIClientConfig
	httpClient *http.Client
	logger     *zap.Logger
}

// NewAPIClient creates a new API client.
func NewAPIClient(config APIClientConfig) *APIClient {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &APIClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: config.Logger,
	}
}

// APIResponse represents a standard API response.
type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *APIError       `json:"error,omitempty"`
}

// APIError represents an API error.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// AutoPrintRule represents an auto-print rule from the API.
type AutoPrintRule struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	DocumentType string `json:"document_type"`
	TriggerEvent string `json:"trigger_event"`
	TemplateID   string `json:"template_id,omitempty"`
	AutoPrint    bool   `json:"auto_print"`
	Copies       int    `json:"copies"`
	PrinterName  string `json:"printer_name,omitempty"`
	Enabled      bool   `json:"enabled"`
}

// PrintTemplate represents a print template from the API.
type PrintTemplate struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	Name         string `json:"name"`
	DocumentType string `json:"document_type"`
	Content      string `json:"content"`
	PaperSize    string `json:"paper_size"`
	Orientation  string `json:"orientation"`
	IsDefault    bool   `json:"is_default"`
	Status       string `json:"status"`
}

// RenderRequest represents a template render request.
type RenderRequest struct {
	TemplateID   string `json:"template_id,omitempty"`
	DocumentType string `json:"document_type"`
	DocumentID   string `json:"document_id"`
}

// RenderResponse represents a template render response.
type RenderResponse struct {
	JobID  string `json:"job_id"`
	PdfURL string `json:"pdf_url"`
}

// PrintJob represents a print job from the API.
type PrintJob struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	DocumentType   string `json:"document_type"`
	DocumentID     string `json:"document_id"`
	DocumentNumber string `json:"document_number"`
	TemplateID     string `json:"template_id"`
	Status         string `json:"status"`
	PdfURL         string `json:"pdf_url,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	Copies         int    `json:"copies"`
	PrinterName    string `json:"printer_name,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// GetAutoPrintRule retrieves an auto-print rule by document type and trigger event.
func (c *APIClient) GetAutoPrintRule(ctx context.Context, docType, triggerEvent string) (*AutoPrintRule, error) {
	params := url.Values{}
	params.Set("document_type", docType)
	params.Set("trigger_event", triggerEvent)
	reqURL := fmt.Sprintf("%s/api/v1/printing/auto-print-rules?%s", c.config.BaseURL, params.Encode())

	resp, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	// Parse response as array and return first match
	var rules []AutoPrintRule
	if err := json.Unmarshal(resp.Data, &rules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
	}

	if len(rules) == 0 {
		return nil, nil // No matching rule
	}

	return &rules[0], nil
}

// GetAutoPrintRuleByID retrieves an auto-print rule by ID.
func (c *APIClient) GetAutoPrintRuleByID(ctx context.Context, id string) (*AutoPrintRule, error) {
	url := fmt.Sprintf("%s/api/v1/printing/auto-print-rules/%s", c.config.BaseURL, id)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var rule AutoPrintRule
	if err := json.Unmarshal(resp.Data, &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule: %w", err)
	}

	return &rule, nil
}

// RenderTemplate renders a document using a template and returns the PDF URL.
func (c *APIClient) RenderTemplate(ctx context.Context, req RenderRequest) (*RenderResponse, error) {
	url := fmt.Sprintf("%s/api/v1/printing/render", c.config.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	var renderResp RenderResponse
	if err := json.Unmarshal(resp.Data, &renderResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &renderResp, nil
}

// GetDocument retrieves document data for printing.
func (c *APIClient) GetDocument(ctx context.Context, docType, docID string) (json.RawMessage, error) {
	// Validate docID to prevent path traversal
	if strings.Contains(docID, "/") || strings.Contains(docID, "..") {
		return nil, fmt.Errorf("invalid document ID: %s", docID)
	}

	// Map document type to API endpoint
	endpoint := c.getDocumentEndpoint(docType)
	if endpoint == "" {
		return nil, fmt.Errorf("unknown document type: %s", docType)
	}

	reqURL := fmt.Sprintf("%s/api/v1/%s/%s", c.config.BaseURL, endpoint, url.PathEscape(docID))

	resp, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetPrintJob retrieves a print job by ID.
func (c *APIClient) GetPrintJob(ctx context.Context, jobID string) (*PrintJob, error) {
	url := fmt.Sprintf("%s/api/v1/printing/jobs/%s", c.config.BaseURL, jobID)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var job PrintJob
	if err := json.Unmarshal(resp.Data, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// GetTemplate retrieves a print template by ID.
func (c *APIClient) GetTemplate(ctx context.Context, templateID string) (*PrintTemplate, error) {
	url := fmt.Sprintf("%s/api/v1/printing/templates/%s", c.config.BaseURL, templateID)

	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var template PrintTemplate
	if err := json.Unmarshal(resp.Data, &template); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	return &template, nil
}

// GetDefaultTemplate retrieves the default template for a document type.
func (c *APIClient) GetDefaultTemplate(ctx context.Context, docType string) (*PrintTemplate, error) {
	params := url.Values{}
	params.Set("document_type", docType)
	params.Set("is_default", "true")
	reqURL := fmt.Sprintf("%s/api/v1/printing/templates?%s", c.config.BaseURL, params.Encode())

	resp, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	var templates []PrintTemplate
	if err := json.Unmarshal(resp.Data, &templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}

	if len(templates) == 0 {
		return nil, nil // No default template
	}

	return &templates[0], nil
}

// DownloadPDF downloads a PDF file from the given URL.
func (c *APIClient) DownloadPDF(ctx context.Context, pdfURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("X-Tenant-ID", c.config.TenantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Limit response size to prevent memory exhaustion
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPDFSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if int64(len(data)) >= maxPDFSize {
		return nil, fmt.Errorf("PDF exceeds maximum size limit of %d bytes", maxPDFSize)
	}

	return data, nil
}

// UpdateJobStatus updates the status of a print job.
func (c *APIClient) UpdateJobStatus(ctx context.Context, jobID, status, errorMsg string) error {
	url := fmt.Sprintf("%s/api/v1/printing/jobs/%s/status", c.config.BaseURL, jobID)

	body := map[string]string{
		"status": status,
	}
	if errorMsg != "" {
		body["error_message"] = errorMsg
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = c.doRequest(ctx, http.MethodPatch, url, bodyBytes)
	return err
}

// doRequest performs an HTTP request and returns the parsed response.
func (c *APIClient) doRequest(ctx context.Context, method, url string, body []byte) (*APIResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("X-Tenant-ID", c.config.TenantID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.logger.Debug("API request",
		zap.String("method", method),
		zap.String("url", url))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Limit response size to prevent memory exhaustion
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	c.logger.Debug("API response",
		zap.Int("status", resp.StatusCode),
		zap.Int("body_length", len(respBody)))

	// Parse response
	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		// If response is not JSON, wrap it
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if !apiResp.Success && apiResp.Error != nil {
		return nil, fmt.Errorf("API error [%s]: %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	// Check HTTP status
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return &apiResp, nil
}

// getDocumentEndpoint maps document type to API endpoint.
func (c *APIClient) getDocumentEndpoint(docType string) string {
	endpoints := map[string]string{
		"SALES_ORDER":        "trade/sales-orders",
		"SALES_DELIVERY":     "trade/sales-orders",
		"SALES_RECEIPT":      "trade/sales-orders",
		"SALES_RETURN":       "trade/sales-returns",
		"PURCHASE_ORDER":     "trade/purchase-orders",
		"PURCHASE_RECEIVING": "trade/purchase-orders",
		"PURCHASE_RETURN":    "trade/purchase-returns",
		"RECEIPT_VOUCHER":    "finance/receipt-vouchers",
		"PAYMENT_VOUCHER":    "finance/payment-vouchers",
		"STOCK_TAKING":       "inventory/stock-takings",
	}
	return endpoints[docType]
}

// HealthCheck performs a health check against the API.
func (c *APIClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
