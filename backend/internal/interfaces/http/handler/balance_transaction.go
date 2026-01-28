package handler

import (
	partnerapp "github.com/erp/backend/internal/application/partner"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BalanceTransactionHandler handles balance transaction-related API endpoints
type BalanceTransactionHandler struct {
	BaseHandler
	balanceTxService *partnerapp.BalanceTransactionService
}

// NewBalanceTransactionHandler creates a new BalanceTransactionHandler
func NewBalanceTransactionHandler(balanceTxService *partnerapp.BalanceTransactionService) *BalanceTransactionHandler {
	return &BalanceTransactionHandler{
		balanceTxService: balanceTxService,
	}
}

// RechargeRequest represents a request to recharge customer balance
// @Description Request body for recharging customer balance
type RechargeRequest struct {
	Amount        float64 `json:"amount" binding:"required,gt=0" example:"1000.00"`
	PaymentMethod string  `json:"payment_method" binding:"required,oneof=CASH WECHAT ALIPAY BANK" example:"CASH"`
	Reference     string  `json:"reference" binding:"max=100" example:"RCH-20260124-001"`
	Remark        string  `json:"remark" binding:"max=500" example:"Customer deposit"`
}

// AdjustRequest represents a request to adjust customer balance
// @Description Request body for adjusting customer balance
type AdjustRequest struct {
	Amount     float64 `json:"amount" binding:"required,gt=0" example:"100.00"`
	IsIncrease bool    `json:"is_increase" example:"true"`
	Reference  string  `json:"reference" binding:"max=100" example:"ADJ-20260124-001"`
	Remark     string  `json:"remark" binding:"required,min=1,max=500" example:"Balance correction due to system error"`
}

// BalanceTransactionListFilter represents filter options for balance transaction list
// @Description Filter options for listing balance transactions
type BalanceTransactionListFilter struct {
	TransactionType string `form:"transaction_type" binding:"omitempty,oneof=RECHARGE CONSUME REFUND ADJUSTMENT EXPIRE" example:"RECHARGE"`
	SourceType      string `form:"source_type" binding:"omitempty,oneof=MANUAL SALES_ORDER SALES_RETURN RECEIPT_VOUCHER SYSTEM" example:"MANUAL"`
	DateFrom        string `form:"date_from" example:"2026-01-01"`
	DateTo          string `form:"date_to" example:"2026-01-31"`
	Page            int    `form:"page" binding:"min=1" example:"1"`
	PageSize        int    `form:"page_size" binding:"min=1,max=100" example:"20"`
}

// BalanceTransactionResponse represents a balance transaction in API responses
// @Description Balance transaction response
type BalanceTransactionResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID        string  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CustomerID      string  `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	TransactionType string  `json:"transaction_type" example:"RECHARGE"`
	Amount          float64 `json:"amount" example:"1000.00"`
	BalanceBefore   float64 `json:"balance_before" example:"500.00"`
	BalanceAfter    float64 `json:"balance_after" example:"1500.00"`
	SourceType      string  `json:"source_type" example:"MANUAL"`
	SourceID        *string `json:"source_id,omitempty" example:"ORD-001"`
	Reference       string  `json:"reference" example:"RCH-20260124-001"`
	Remark          string  `json:"remark" example:"Customer deposit"`
	OperatorID      *string `json:"operator_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440003"`
	TransactionDate string  `json:"transaction_date" example:"2026-01-24T10:00:00Z"`
	CreatedAt       string  `json:"created_at" example:"2026-01-24T10:00:00Z"`
}

// BalanceSummaryResponse represents customer balance summary
// @Description Customer balance summary response
type BalanceSummaryResponse struct {
	CustomerID     string  `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CurrentBalance float64 `json:"current_balance" example:"1500.00"`
	TotalRecharge  float64 `json:"total_recharge" example:"5000.00"`
	TotalConsume   float64 `json:"total_consume" example:"3500.00"`
	TotalRefund    float64 `json:"total_refund" example:"0.00"`
}

// Recharge godoc
// @ID           rechargeBalance
// @Summary      Recharge customer balance
// @Description  Add funds to a customer's prepaid balance
// @Tags         balance
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Param        request body RechargeRequest true "Recharge request"
// @Success      201 {object} APIResponse[BalanceTransactionResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/balance/recharge [post]
func (h *BalanceTransactionHandler) Recharge(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	var req RechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	amount := decimal.NewFromFloat(req.Amount)
	paymentMethod := partner.PaymentMethod(req.PaymentMethod)

	// Get operator ID from context if available
	var operatorID *uuid.UUID
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			operatorID = &uid
		}
	}

	transaction, err := h.balanceTxService.Recharge(
		c.Request.Context(),
		tenantID,
		customerID,
		amount,
		paymentMethod,
		req.Reference,
		req.Remark,
		operatorID,
	)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, transaction)
}

// Adjust godoc
// @ID           adjustBalance
// @Summary      Adjust customer balance
// @Description  Manually adjust a customer's balance (increase or decrease)
// @Tags         balance
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Param        request body AdjustRequest true "Adjust request"
// @Success      201 {object} APIResponse[BalanceTransactionResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/balance/adjust [post]
func (h *BalanceTransactionHandler) Adjust(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	var req AdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	amount := decimal.NewFromFloat(req.Amount)

	// Get operator ID from context if available
	var operatorID *uuid.UUID
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			operatorID = &uid
		}
	}

	transaction, err := h.balanceTxService.Adjust(
		c.Request.Context(),
		tenantID,
		customerID,
		amount,
		req.IsIncrease,
		req.Reference,
		req.Remark,
		operatorID,
	)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, transaction)
}

// GetBalance godoc
// @ID           getBalanceBalance
// @Summary      Get customer balance
// @Description  Get the current balance for a customer
// @Tags         balance
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Success      200 {object} APIResponse[BalanceData]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/balance [get]
func (h *BalanceTransactionHandler) GetBalance(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	balance, err := h.balanceTxService.GetBalance(c.Request.Context(), tenantID, customerID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, map[string]interface{}{
		"customer_id": customerID,
		"balance":     balance,
	})
}

// GetBalanceSummary godoc
// @ID           getBalanceBalanceSummary
// @Summary      Get customer balance summary
// @Description  Get balance summary including total recharge, consume, and refund
// @Tags         balance
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Success      200 {object} APIResponse[BalanceSummaryResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/balance/summary [get]
func (h *BalanceTransactionHandler) GetBalanceSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	summary, err := h.balanceTxService.GetBalanceSummary(c.Request.Context(), tenantID, customerID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, summary)
}

// ListTransactions godoc
// @ID           listBalanceTransactions
// @Summary      List balance transactions
// @Description  List balance transactions for a customer with optional filtering
// @Tags         balance
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Customer ID" format(uuid)
// @Param        transaction_type query string false "Transaction type" Enums(RECHARGE, CONSUME, REFUND, ADJUSTMENT, EXPIRE)
// @Param        source_type query string false "Source type" Enums(MANUAL, SALES_ORDER, SALES_RETURN, RECEIPT_VOUCHER, SYSTEM)
// @Param        date_from query string false "Start date (YYYY-MM-DD)"
// @Param        date_to query string false "End date (YYYY-MM-DD)"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Success      200 {object} APIResponse[[]BalanceTransactionResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/customers/{id}/balance/transactions [get]
func (h *BalanceTransactionHandler) ListTransactions(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	var filter BalanceTransactionListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	// Convert to application filter
	appFilter := partnerapp.BalanceTransactionListFilter{
		CustomerID:      &customerID,
		TransactionType: filter.TransactionType,
		SourceType:      filter.SourceType,
		DateFrom:        filter.DateFrom,
		DateTo:          filter.DateTo,
		Page:            filter.Page,
		PageSize:        filter.PageSize,
	}

	transactions, total, err := h.balanceTxService.ListByCustomer(
		c.Request.Context(),
		tenantID,
		customerID,
		appFilter,
	)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, transactions, total, filter.Page, filter.PageSize)
}

// GetTransaction godoc
// @ID           getBalanceTransaction
// @Summary      Get balance transaction by ID
// @Description  Get a specific balance transaction by its ID
// @Tags         balance
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Transaction ID" format(uuid)
// @Success      200 {object} APIResponse[BalanceTransactionResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /partner/balance/transactions/{id} [get]
func (h *BalanceTransactionHandler) GetTransaction(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	transactionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid transaction ID format")
		return
	}

	transaction, err := h.balanceTxService.GetByID(c.Request.Context(), tenantID, transactionID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, transaction)
}
