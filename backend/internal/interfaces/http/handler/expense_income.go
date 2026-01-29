package handler

import (
	"net/http"
	"time"

	financeapp "github.com/erp/backend/internal/application/finance"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExpenseIncomeHandler handles expense and income related API endpoints
type ExpenseIncomeHandler struct {
	BaseHandler
	service *financeapp.ExpenseIncomeService
}

// NewExpenseIncomeHandler creates a new ExpenseIncomeHandler
func NewExpenseIncomeHandler(service *financeapp.ExpenseIncomeService) *ExpenseIncomeHandler {
	return &ExpenseIncomeHandler{
		service: service,
	}
}

// ===================== Request/Response DTOs =====================

// ExpenseRecordResponse represents an expense record in API responses
//
//	@Description	Expense record response
type ExpenseRecordResponse struct {
	ID              string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID        string     `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ExpenseNumber   string     `json:"expense_number" example:"EXP-2026-00001"`
	Category        string     `json:"category" example:"RENT"`
	CategoryName    string     `json:"category_name" example:"房租"`
	Amount          float64    `json:"amount" example:"5000.00"`
	Description     string     `json:"description" example:"2026年1月办公室租金"`
	IncurredAt      time.Time  `json:"incurred_at"`
	Status          string     `json:"status" example:"DRAFT"`
	PaymentStatus   string     `json:"payment_status" example:"UNPAID"`
	PaymentMethod   *string    `json:"payment_method,omitempty" example:"BANK_TRANSFER"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	Remark          string     `json:"remark,omitempty" example:"备注"`
	AttachmentURLs  string     `json:"attachment_urls,omitempty"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
	SubmittedBy     *string    `json:"submitted_by,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ApprovedBy      *string    `json:"approved_by,omitempty"`
	ApprovalRemark  string     `json:"approval_remark,omitempty"`
	RejectedAt      *time.Time `json:"rejected_at,omitempty"`
	RejectedBy      *string    `json:"rejected_by,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Version         int        `json:"version" example:"1"`
}

// CreateExpenseRecordRequest represents a request to create an expense record
//
//	@Description	Create expense record request
type CreateExpenseRecordRequest struct {
	Category       string    `json:"category" binding:"required" example:"RENT"`
	Amount         float64   `json:"amount" binding:"required,gt=0" example:"5000.00"`
	Description    string    `json:"description" binding:"required" example:"2026年1月办公室租金"`
	IncurredAt     time.Time `json:"incurred_at" binding:"required"`
	Remark         string    `json:"remark" example:"备注"`
	AttachmentURLs string    `json:"attachment_urls"`
}

// UpdateExpenseRecordRequest represents a request to update an expense record
//
//	@Description	Update expense record request
type UpdateExpenseRecordRequest struct {
	Category       string    `json:"category" binding:"required" example:"RENT"`
	Amount         float64   `json:"amount" binding:"required,gt=0" example:"5000.00"`
	Description    string    `json:"description" binding:"required" example:"2026年1月办公室租金"`
	IncurredAt     time.Time `json:"incurred_at" binding:"required"`
	Remark         string    `json:"remark" example:"备注"`
	AttachmentURLs string    `json:"attachment_urls"`
}

// ExpenseActionRequest represents a request to perform an action on an expense
//
//	@Description	Expense action request
type ExpenseActionRequest struct {
	Remark string `json:"remark" example:"审批意见"`
}

// MarkExpensePaidRequest represents a request to mark an expense as paid
//
//	@Description	Mark expense paid request
type MarkExpensePaidRequest struct {
	PaymentMethod string `json:"payment_method" binding:"required" example:"BANK_TRANSFER"`
}

// ExpenseListFilter represents filter parameters for expense list
//
//	@Description	Expense list filter
type ExpenseListFilter struct {
	Search        string `form:"search"`
	Category      string `form:"category"`
	Status        string `form:"status"`
	PaymentStatus string `form:"payment_status" json:"payment_status"`
	FromDate      string `form:"from_date" json:"from_date"`
	ToDate        string `form:"to_date" json:"to_date"`
	Page          int    `form:"page,omitempty" binding:"omitempty,min=1" example:"1"`
	PageSize      int    `form:"page_size,omitempty" binding:"omitempty,min=1,max=100" json:"page_size" example:"20"`
}

// ExpenseSummaryResponse represents expense summary statistics
//
//	@Description	Expense summary response
type ExpenseSummaryResponse struct {
	TotalApproved float64            `json:"total_approved" example:"100000.00"`
	TotalPending  int64              `json:"total_pending" example:"5"`
	ByCategory    map[string]float64 `json:"by_category"`
}

// ===================== Other Income DTOs =====================

// OtherIncomeRecordResponse represents an income record in API responses
//
//	@Description	Other income record response
type OtherIncomeRecordResponse struct {
	ID             string     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID       string     `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	IncomeNumber   string     `json:"income_number" example:"INC-2026-00001"`
	Category       string     `json:"category" example:"INVESTMENT"`
	CategoryName   string     `json:"category_name" example:"投资收益"`
	Amount         float64    `json:"amount" example:"10000.00"`
	Description    string     `json:"description" example:"理财产品收益"`
	ReceivedAt     time.Time  `json:"received_at"`
	Status         string     `json:"status" example:"DRAFT"`
	ReceiptStatus  string     `json:"receipt_status" example:"PENDING"`
	PaymentMethod  *string    `json:"payment_method,omitempty" example:"BANK_TRANSFER"`
	ActualReceived *time.Time `json:"actual_received,omitempty"`
	Remark         string     `json:"remark,omitempty" example:"备注"`
	AttachmentURLs string     `json:"attachment_urls,omitempty"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
	ConfirmedBy    *string    `json:"confirmed_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Version        int        `json:"version" example:"1"`
}

// CreateOtherIncomeRecordRequest represents a request to create an income record
//
//	@Description	Create other income record request
type CreateOtherIncomeRecordRequest struct {
	Category       string    `json:"category" binding:"required" example:"INVESTMENT"`
	Amount         float64   `json:"amount" binding:"required,gt=0" example:"10000.00"`
	Description    string    `json:"description" binding:"required" example:"理财产品收益"`
	ReceivedAt     time.Time `json:"received_at" binding:"required"`
	Remark         string    `json:"remark" example:"备注"`
	AttachmentURLs string    `json:"attachment_urls"`
}

// UpdateOtherIncomeRecordRequest represents a request to update an income record
//
//	@Description	Update other income record request
type UpdateOtherIncomeRecordRequest struct {
	Category       string    `json:"category" binding:"required" example:"INVESTMENT"`
	Amount         float64   `json:"amount" binding:"required,gt=0" example:"10000.00"`
	Description    string    `json:"description" binding:"required" example:"理财产品收益"`
	ReceivedAt     time.Time `json:"received_at" binding:"required"`
	Remark         string    `json:"remark" example:"备注"`
	AttachmentURLs string    `json:"attachment_urls"`
}

// IncomeActionRequest represents a request to perform an action on an income record
//
//	@Description	Income action request
type IncomeActionRequest struct {
	Remark string `json:"remark" example:"取消原因"`
}

// MarkIncomeReceivedRequest represents a request to mark income as received
//
//	@Description	Mark income received request
type MarkIncomeReceivedRequest struct {
	PaymentMethod string `json:"payment_method" binding:"required" example:"BANK_TRANSFER"`
}

// IncomeListFilter represents filter parameters for income list
//
//	@Description	Income list filter
type IncomeListFilter struct {
	Search        string `form:"search"`
	Category      string `form:"category"`
	Status        string `form:"status"`
	ReceiptStatus string `form:"receipt_status" json:"receipt_status"`
	FromDate      string `form:"from_date" json:"from_date"`
	ToDate        string `form:"to_date" json:"to_date"`
	Page          int    `form:"page,omitempty" binding:"omitempty,min=1" example:"1"`
	PageSize      int    `form:"page_size,omitempty" binding:"omitempty,min=1,max=100" json:"page_size" example:"20"`
}

// IncomeSummaryResponse represents income summary statistics
//
//	@Description	Income summary response
type IncomeSummaryResponse struct {
	TotalConfirmed float64            `json:"total_confirmed" example:"50000.00"`
	TotalDraft     int64              `json:"total_draft" example:"5"`
	ByCategory     map[string]float64 `json:"by_category"`
}

// CashFlowSummaryResponse represents cash flow summary
//
//	@Description	Cash flow summary response
type CashFlowSummaryResponse struct {
	TotalInflow  float64                     `json:"total_inflow" example:"50000.00"`
	TotalOutflow float64                     `json:"total_outflow" example:"30000.00"`
	NetCashFlow  float64                     `json:"net_cash_flow" example:"20000.00"`
	PeriodStart  string                      `json:"period_start" example:"2026-01-01"`
	PeriodEnd    string                      `json:"period_end" example:"2026-01-31"`
	IncomeItems  []ExpenseIncomeCashFlowItem `json:"income_items,omitempty"`
	ExpenseItems []ExpenseIncomeCashFlowItem `json:"expense_items,omitempty"`
}

// ExpenseIncomeCashFlowItem represents a cash flow item for expense/income
//
//	@Description	Cash flow item response
type ExpenseIncomeCashFlowItem struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type        string    `json:"type" example:"EXPENSE"`
	Category    string    `json:"category" example:"RENT"`
	Description string    `json:"description" example:"2026年1月办公室租金"`
	Amount      float64   `json:"amount" example:"5000.00"`
	Date        time.Time `json:"date"`
}

// ===================== Expense Handlers =====================

// ===================== Expense Handlers =====================

// ListExpenses godoc
// @ID           listExpensExpenses
//
//	@Summary		List expense records
//	@Description	Get a paginated list of expense records
//	@Tags			expenses
//	@Produce		json
//	@Param			search			query		string	false	"Search keyword"
//	@Param			category		query		string	false	"Filter by category"	Enums(RENT, UTILITIES, SALARY, OFFICE, TRAVEL, MARKETING, EQUIPMENT, MAINTENANCE, INSURANCE, TAX, OTHER)
//	@Param			status			query		string	false	"Filter by status"		Enums(DRAFT, PENDING, APPROVED, REJECTED, CANCELLED)
//	@Param			payment_status	query		string	false	"Filter by payment status"	Enums(UNPAID, PAID)
//	@Param			from_date		query		string	false	"Filter from date (YYYY-MM-DD)"
//	@Param			to_date			query		string	false	"Filter to date (YYYY-MM-DD)"
//	@Param			page			query		int		false	"Page number"			default(1)
//	@Param			page_size		query		int		false	"Page size"				default(20)
//	@Success		200				{object}	APIResponse[[]ExpenseRecordResponse]
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses [get]
func (h *ExpenseIncomeHandler) ListExpenses(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	var filter ExpenseListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	// Parse dates
	var fromDate, toDate *time.Time
	if filter.FromDate != "" {
		t, err := time.Parse("2006-01-02", filter.FromDate)
		if err == nil {
			fromDate = &t
		}
	}
	if filter.ToDate != "" {
		t, err := time.Parse("2006-01-02", filter.ToDate)
		if err == nil {
			// Set to end of day
			t = t.Add(24*time.Hour - time.Second)
			toDate = &t
		}
	}

	// Build service filter
	serviceFilter := financeapp.ExpenseRecordListFilter{
		Search:        filter.Search,
		Category:      filter.Category,
		Status:        filter.Status,
		PaymentStatus: filter.PaymentStatus,
		FromDate:      fromDate,
		ToDate:        toDate,
		Page:          filter.Page,
		PageSize:      filter.PageSize,
	}

	expenses, total, err := h.service.ListExpenseRecords(c.Request.Context(), tenantID, serviceFilter)
	if err != nil {
		h.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	// Convert to response format
	response := make([]ExpenseRecordResponse, len(expenses))
	for i, exp := range expenses {
		response[i] = h.toExpenseRecordResponse(exp)
	}

	h.SuccessWithMeta(c, response, total, filter.Page, filter.PageSize)
}

// GetExpense godoc
// @ID           getExpensExpense
//
//	@Summary		Get expense record by ID
//	@Description	Get a single expense record by its ID
//	@Tags			expenses
//	@Produce		json
//	@Param			id	path		string	true	"Expense ID"
//	@Success		200	{object}	APIResponse[ExpenseRecordResponse]
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/{id} [get]
func (h *ExpenseIncomeHandler) GetExpense(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID")
		return
	}

	expense, err := h.service.GetExpenseRecordByID(c.Request.Context(), tenantID, expenseID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toExpenseRecordResponse(*expense))
}

// CreateExpense godoc
// @ID           createExpensExpense
//
//	@Summary		Create expense record
//	@Description	Create a new expense record
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateExpenseRecordRequest	true	"Expense creation request"
//	@Success		201		{object}	APIResponse[ExpenseRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses [post]
func (h *ExpenseIncomeHandler) CreateExpense(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	userID, _ := getUserID(c)

	var req CreateExpenseRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	serviceReq := financeapp.CreateExpenseRecordRequest{
		Category:       req.Category,
		Amount:         decimal.NewFromFloat(req.Amount),
		Description:    req.Description,
		IncurredAt:     req.IncurredAt,
		Remark:         req.Remark,
		AttachmentURLs: req.AttachmentURLs,
	}
	if userID != uuid.Nil {
		serviceReq.CreatedBy = &userID
	}

	expense, err := h.service.CreateExpenseRecord(c.Request.Context(), tenantID, serviceReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.Response{
		Success: true,
		Data:    h.toExpenseRecordResponse(*expense),
	})
}

// UpdateExpense godoc
// @ID           updateExpensExpense
//
//	@Summary		Update expense record
//	@Description	Update an existing expense record (only draft status)
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Expense ID"
//	@Param			request	body		UpdateExpenseRecordRequest	true	"Expense update request"
//	@Success		200		{object}	APIResponse[ExpenseRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/{id} [put]
func (h *ExpenseIncomeHandler) UpdateExpense(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID")
		return
	}

	var req UpdateExpenseRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	serviceReq := financeapp.UpdateExpenseRecordRequest{
		Category:       req.Category,
		Amount:         decimal.NewFromFloat(req.Amount),
		Description:    req.Description,
		IncurredAt:     req.IncurredAt,
		Remark:         req.Remark,
		AttachmentURLs: req.AttachmentURLs,
	}

	expense, err := h.service.UpdateExpenseRecord(c.Request.Context(), tenantID, expenseID, serviceReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toExpenseRecordResponse(*expense))
}

// DeleteExpense godoc
// @ID           deleteExpensExpense
//
//	@Summary		Delete expense record
//	@Description	Delete an expense record (soft delete, only draft status)
//	@Tags			expenses
//	@Produce		json
//	@Param			id	path		string	true	"Expense ID"
//	@Success      200 {object} SuccessResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/{id} [delete]
func (h *ExpenseIncomeHandler) DeleteExpense(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID")
		return
	}

	if err := h.service.DeleteExpenseRecord(c.Request.Context(), tenantID, expenseID); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, nil)
}

// SubmitExpense godoc
// @ID           submitExpenseExpens
//
//	@Summary		Submit expense for approval
//	@Description	Submit an expense record for approval
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Expense ID"
//	@Success		200	{object}	APIResponse[ExpenseRecordResponse]
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/{id}/submit [post]
func (h *ExpenseIncomeHandler) SubmitExpense(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	userID, err := getUserID(c)
	if err != nil || userID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user")
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID")
		return
	}

	expense, err := h.service.SubmitExpenseRecord(c.Request.Context(), tenantID, expenseID, userID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toExpenseRecordResponse(*expense))
}

// ApproveExpense godoc
// @ID           approveExpenseExpens
//
//	@Summary		Approve expense
//	@Description	Approve an expense record
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Expense ID"
//	@Param			request	body		ExpenseActionRequest	false	"Approval remark"
//	@Success		200		{object}	APIResponse[ExpenseRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/{id}/approve [post]
func (h *ExpenseIncomeHandler) ApproveExpense(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	userID, err := getUserID(c)
	if err != nil || userID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user")
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID")
		return
	}

	var req ExpenseActionRequest
	c.ShouldBindJSON(&req) // Ignore error, remark is optional

	expense, err := h.service.ApproveExpenseRecord(c.Request.Context(), tenantID, expenseID, userID, req.Remark)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toExpenseRecordResponse(*expense))
}

// RejectExpense godoc
// @ID           rejectExpenseExpens
//
//	@Summary		Reject expense
//	@Description	Reject an expense record
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Expense ID"
//	@Param			request	body		ExpenseActionRequest	true	"Rejection reason"
//	@Success		200		{object}	APIResponse[ExpenseRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/{id}/reject [post]
func (h *ExpenseIncomeHandler) RejectExpense(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	userID, err := getUserID(c)
	if err != nil || userID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user")
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID")
		return
	}

	var req ExpenseActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	expense, err := h.service.RejectExpenseRecord(c.Request.Context(), tenantID, expenseID, userID, req.Remark)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toExpenseRecordResponse(*expense))
}

// CancelExpense godoc
// @ID           cancelExpenseExpens
//
//	@Summary		Cancel expense
//	@Description	Cancel an expense record
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Expense ID"
//	@Param			request	body		ExpenseActionRequest	true	"Cancel reason"
//	@Success		200		{object}	APIResponse[ExpenseRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/{id}/cancel [post]
func (h *ExpenseIncomeHandler) CancelExpense(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	userID, err := getUserID(c)
	if err != nil || userID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user")
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID")
		return
	}

	var req ExpenseActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	expense, err := h.service.CancelExpenseRecord(c.Request.Context(), tenantID, expenseID, userID, req.Remark)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toExpenseRecordResponse(*expense))
}

// MarkExpensePaid godoc
// @ID           markExpensePaidExpens
//
//	@Summary		Mark expense as paid
//	@Description	Mark an approved expense as paid
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Expense ID"
//	@Param			request	body		MarkExpensePaidRequest	true	"Payment method"
//	@Success		200		{object}	APIResponse[ExpenseRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/{id}/pay [post]
func (h *ExpenseIncomeHandler) MarkExpensePaid(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid expense ID")
		return
	}

	var req MarkExpensePaidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	expense, err := h.service.MarkExpenseAsPaid(c.Request.Context(), tenantID, expenseID, req.PaymentMethod)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toExpenseRecordResponse(*expense))
}

// GetExpensesSummary godoc
// @ID           getExpensExpensesSummary
//
//	@Summary		Get expenses summary
//	@Description	Get expense statistics summary
//	@Tags			expenses
//	@Produce		json
//	@Param			from_date	query		string	false	"From date (YYYY-MM-DD)"
//	@Param			to_date		query		string	false	"To date (YYYY-MM-DD)"
//	@Success		200			{object}	APIResponse[ExpenseSummaryResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/expenses/summary [get]
func (h *ExpenseIncomeHandler) GetExpensesSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	// Parse date range
	fromDateStr := c.Query("from_date")
	toDateStr := c.Query("to_date")

	var fromDate, toDate time.Time
	if fromDateStr != "" {
		t, err := time.Parse("2006-01-02", fromDateStr)
		if err == nil {
			fromDate = t
		}
	} else {
		// Default to first day of current month
		now := time.Now()
		fromDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	if toDateStr != "" {
		t, err := time.Parse("2006-01-02", toDateStr)
		if err == nil {
			toDate = t.Add(24*time.Hour - time.Second)
		}
	} else {
		// Default to now
		toDate = time.Now()
	}

	summary, err := h.service.GetExpenseSummary(c.Request.Context(), tenantID, fromDate, toDate)
	if err != nil {
		h.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response := ExpenseSummaryResponse{
		TotalApproved: summary.TotalApproved.InexactFloat64(),
		TotalPending:  summary.TotalPending,
		ByCategory:    make(map[string]float64),
	}

	for cat, amt := range summary.ByCategory {
		response.ByCategory[cat] = amt.InexactFloat64()
	}

	h.Success(c, response)
}

// ===================== Other Income Handlers =====================

// ===================== Other Income Handlers =====================

// ListIncomes godoc
// @ID           listIncomeIncomes
//
//	@Summary		List other income records
//	@Description	Get a paginated list of other income records
//	@Tags			incomes
//	@Produce		json
//	@Param			search			query		string	false	"Search keyword"
//	@Param			category		query		string	false	"Filter by category"	Enums(INVESTMENT, SUBSIDY, INTEREST, RENTAL, REFUND, COMPENSATION, ASSET_DISPOSAL, OTHER)
//	@Param			status			query		string	false	"Filter by status"		Enums(DRAFT, CONFIRMED, CANCELLED)
//	@Param			receipt_status	query		string	false	"Filter by receipt status"	Enums(PENDING, RECEIVED)
//	@Param			from_date		query		string	false	"Filter from date (YYYY-MM-DD)"
//	@Param			to_date			query		string	false	"Filter to date (YYYY-MM-DD)"
//	@Param			page			query		int		false	"Page number"			default(1)
//	@Param			page_size		query		int		false	"Page size"				default(20)
//	@Success		200				{object}	APIResponse[[]OtherIncomeRecordResponse]
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes [get]
func (h *ExpenseIncomeHandler) ListIncomes(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	var filter IncomeListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	// Parse dates
	var fromDate, toDate *time.Time
	if filter.FromDate != "" {
		t, err := time.Parse("2006-01-02", filter.FromDate)
		if err == nil {
			fromDate = &t
		}
	}
	if filter.ToDate != "" {
		t, err := time.Parse("2006-01-02", filter.ToDate)
		if err == nil {
			t = t.Add(24*time.Hour - time.Second)
			toDate = &t
		}
	}

	// Build service filter
	serviceFilter := financeapp.OtherIncomeRecordListFilter{
		Search:        filter.Search,
		Category:      filter.Category,
		Status:        filter.Status,
		ReceiptStatus: filter.ReceiptStatus,
		FromDate:      fromDate,
		ToDate:        toDate,
		Page:          filter.Page,
		PageSize:      filter.PageSize,
	}

	incomes, total, err := h.service.ListOtherIncomeRecords(c.Request.Context(), tenantID, serviceFilter)
	if err != nil {
		h.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	// Convert to response format
	response := make([]OtherIncomeRecordResponse, len(incomes))
	for i, inc := range incomes {
		response[i] = h.toIncomeRecordResponse(inc)
	}

	h.SuccessWithMeta(c, response, total, filter.Page, filter.PageSize)
}

// GetIncome godoc
// @ID           getIncomeIncome
//
//	@Summary		Get other income record by ID
//	@Description	Get a single income record by its ID
//	@Tags			incomes
//	@Produce		json
//	@Param			id	path		string	true	"Income ID"
//	@Success		200	{object}	APIResponse[OtherIncomeRecordResponse]
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes/{id} [get]
func (h *ExpenseIncomeHandler) GetIncome(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	incomeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid income ID")
		return
	}

	income, err := h.service.GetOtherIncomeRecordByID(c.Request.Context(), tenantID, incomeID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toIncomeRecordResponse(*income))
}

// CreateIncome godoc
// @ID           createIncomeIncome
//
//	@Summary		Create other income record
//	@Description	Create a new other income record
//	@Tags			incomes
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateOtherIncomeRecordRequest	true	"Income creation request"
//	@Success		201		{object}	APIResponse[OtherIncomeRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes [post]
func (h *ExpenseIncomeHandler) CreateIncome(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	userID, _ := getUserID(c)

	var req CreateOtherIncomeRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	serviceReq := financeapp.CreateOtherIncomeRecordRequest{
		Category:       req.Category,
		Amount:         decimal.NewFromFloat(req.Amount),
		Description:    req.Description,
		ReceivedAt:     req.ReceivedAt,
		Remark:         req.Remark,
		AttachmentURLs: req.AttachmentURLs,
	}
	if userID != uuid.Nil {
		serviceReq.CreatedBy = &userID
	}

	income, err := h.service.CreateOtherIncomeRecord(c.Request.Context(), tenantID, serviceReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.Response{
		Success: true,
		Data:    h.toIncomeRecordResponse(*income),
	})
}

// UpdateIncome godoc
// @ID           updateIncomeIncome
//
//	@Summary		Update other income record
//	@Description	Update an existing income record (only draft status)
//	@Tags			incomes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Income ID"
//	@Param			request	body		UpdateOtherIncomeRecordRequest	true	"Income update request"
//	@Success		200		{object}	APIResponse[OtherIncomeRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes/{id} [put]
func (h *ExpenseIncomeHandler) UpdateIncome(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	incomeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid income ID")
		return
	}

	var req UpdateOtherIncomeRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	serviceReq := financeapp.UpdateOtherIncomeRecordRequest{
		Category:       req.Category,
		Amount:         decimal.NewFromFloat(req.Amount),
		Description:    req.Description,
		ReceivedAt:     req.ReceivedAt,
		Remark:         req.Remark,
		AttachmentURLs: req.AttachmentURLs,
	}

	income, err := h.service.UpdateOtherIncomeRecord(c.Request.Context(), tenantID, incomeID, serviceReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toIncomeRecordResponse(*income))
}

// DeleteIncome godoc
// @ID           deleteIncomeIncome
//
//	@Summary		Delete other income record
//	@Description	Delete an income record (soft delete, only draft status)
//	@Tags			incomes
//	@Produce		json
//	@Param			id	path		string	true	"Income ID"
//	@Success      200 {object} SuccessResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes/{id} [delete]
func (h *ExpenseIncomeHandler) DeleteIncome(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	incomeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid income ID")
		return
	}

	if err := h.service.DeleteOtherIncomeRecord(c.Request.Context(), tenantID, incomeID); err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, nil)
}

// ConfirmIncome godoc
// @ID           confirmIncomeIncome
//
//	@Summary		Confirm income record
//	@Description	Confirm an income record
//	@Tags			incomes
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Income ID"
//	@Success		200	{object}	APIResponse[OtherIncomeRecordResponse]
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes/{id}/confirm [post]
func (h *ExpenseIncomeHandler) ConfirmIncome(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	userID, err := getUserID(c)
	if err != nil || userID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user")
		return
	}

	incomeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid income ID")
		return
	}

	income, err := h.service.ConfirmOtherIncomeRecord(c.Request.Context(), tenantID, incomeID, userID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toIncomeRecordResponse(*income))
}

// CancelIncome godoc
// @ID           cancelIncomeIncome
//
//	@Summary		Cancel income record
//	@Description	Cancel an income record
//	@Tags			incomes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Income ID"
//	@Param			request	body		IncomeActionRequest	true	"Cancel reason"
//	@Success		200		{object}	APIResponse[OtherIncomeRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes/{id}/cancel [post]
func (h *ExpenseIncomeHandler) CancelIncome(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	userID, err := getUserID(c)
	if err != nil || userID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user")
		return
	}

	incomeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid income ID")
		return
	}

	var req IncomeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	income, err := h.service.CancelOtherIncomeRecord(c.Request.Context(), tenantID, incomeID, userID, req.Remark)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toIncomeRecordResponse(*income))
}

// MarkIncomeReceived godoc
// @ID           markIncomeReceivedIncome
//
//	@Summary		Mark income as received
//	@Description	Mark a confirmed income as received
//	@Tags			incomes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Income ID"
//	@Param			request	body		MarkIncomeReceivedRequest	true	"Payment method"
//	@Success		200		{object}	APIResponse[OtherIncomeRecordResponse]
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes/{id}/receive [post]
func (h *ExpenseIncomeHandler) MarkIncomeReceived(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	incomeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid income ID")
		return
	}

	var req MarkIncomeReceivedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	income, err := h.service.MarkIncomeAsReceived(c.Request.Context(), tenantID, incomeID, req.PaymentMethod)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, h.toIncomeRecordResponse(*income))
}

// GetIncomesSummary godoc
// @ID           getIncomeIncomesSummary
//
//	@Summary		Get incomes summary
//	@Description	Get income statistics summary
//	@Tags			incomes
//	@Produce		json
//	@Param			from_date	query		string	false	"From date (YYYY-MM-DD)"
//	@Param			to_date		query		string	false	"To date (YYYY-MM-DD)"
//	@Success		200			{object}	APIResponse[IncomeSummaryResponse]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/incomes/summary [get]
func (h *ExpenseIncomeHandler) GetIncomesSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	// Parse date range
	fromDateStr := c.Query("from_date")
	toDateStr := c.Query("to_date")

	var fromDate, toDate time.Time
	if fromDateStr != "" {
		t, err := time.Parse("2006-01-02", fromDateStr)
		if err == nil {
			fromDate = t
		}
	} else {
		now := time.Now()
		fromDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	if toDateStr != "" {
		t, err := time.Parse("2006-01-02", toDateStr)
		if err == nil {
			toDate = t.Add(24*time.Hour - time.Second)
		}
	} else {
		toDate = time.Now()
	}

	summary, err := h.service.GetIncomeSummary(c.Request.Context(), tenantID, fromDate, toDate)
	if err != nil {
		h.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response := IncomeSummaryResponse{
		TotalConfirmed: summary.TotalConfirmed.InexactFloat64(),
		TotalDraft:     summary.TotalDraft,
		ByCategory:     make(map[string]float64),
	}

	for cat, amt := range summary.ByCategory {
		response.ByCategory[cat] = amt.InexactFloat64()
	}

	h.Success(c, response)
}

// GetCashFlow godoc
// @ID           getExpensCashFlow
//
//	@Summary		Get cash flow summary
//	@Description	Get combined expense and income cash flow summary
//	@Tags			expenses
//	@Produce		json
//	@Param			from_date		query		string	false	"From date (YYYY-MM-DD)"
//	@Param			to_date			query		string	false	"To date (YYYY-MM-DD)"
//	@Param			include_items	query		bool	false	"Include individual items"
//	@Success		200				{object}	APIResponse[CashFlowSummaryResponse]
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Security		BearerAuth
//	@Router			/finance/cash-flow [get]
func (h *ExpenseIncomeHandler) GetCashFlow(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil || tenantID == uuid.Nil {
		h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid tenant")
		return
	}

	// Parse date range
	fromDateStr := c.Query("from_date")
	toDateStr := c.Query("to_date")
	includeItems := c.Query("include_items") == "true"

	var fromDate, toDate time.Time
	if fromDateStr != "" {
		t, err := time.Parse("2006-01-02", fromDateStr)
		if err == nil {
			fromDate = t
		}
	} else {
		now := time.Now()
		fromDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	}

	if toDateStr != "" {
		t, err := time.Parse("2006-01-02", toDateStr)
		if err == nil {
			toDate = t.Add(24*time.Hour - time.Second)
		}
	} else {
		toDate = time.Now()
	}

	summary, err := h.service.GetCashFlowSummary(c.Request.Context(), tenantID, fromDate, toDate, includeItems)
	if err != nil {
		h.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response := CashFlowSummaryResponse{
		TotalInflow:  summary.TotalInflow.InexactFloat64(),
		TotalOutflow: summary.TotalOutflow.InexactFloat64(),
		NetCashFlow:  summary.NetCashFlow.InexactFloat64(),
		PeriodStart:  fromDate.Format("2006-01-02"),
		PeriodEnd:    toDate.Format("2006-01-02"),
	}

	if includeItems && len(summary.Items) > 0 {
		for _, item := range summary.Items {
			cashFlowItem := ExpenseIncomeCashFlowItem{
				ID:          item.ID.String(),
				Type:        item.Type,
				Category:    item.Category,
				Description: item.Description,
				Amount:      item.Amount.InexactFloat64(),
				Date:        item.Date,
			}
			if item.Direction == "INFLOW" {
				response.IncomeItems = append(response.IncomeItems, cashFlowItem)
			} else {
				response.ExpenseItems = append(response.ExpenseItems, cashFlowItem)
			}
		}
	}

	h.Success(c, response)
}

// ===================== Helper Functions =====================

func (h *ExpenseIncomeHandler) toExpenseRecordResponse(exp financeapp.ExpenseRecordResponse) ExpenseRecordResponse {
	var submittedBy, approvedBy, rejectedBy *string
	if exp.SubmittedBy != nil {
		s := exp.SubmittedBy.String()
		submittedBy = &s
	}
	if exp.ApprovedBy != nil {
		s := exp.ApprovedBy.String()
		approvedBy = &s
	}
	if exp.RejectedBy != nil {
		s := exp.RejectedBy.String()
		rejectedBy = &s
	}

	return ExpenseRecordResponse{
		ID:              exp.ID.String(),
		TenantID:        exp.TenantID.String(),
		ExpenseNumber:   exp.ExpenseNumber,
		Category:        exp.Category,
		CategoryName:    exp.CategoryName,
		Amount:          exp.Amount.InexactFloat64(),
		Description:     exp.Description,
		IncurredAt:      exp.IncurredAt,
		Status:          exp.Status,
		PaymentStatus:   exp.PaymentStatus,
		PaymentMethod:   exp.PaymentMethod,
		PaidAt:          exp.PaidAt,
		Remark:          exp.Remark,
		AttachmentURLs:  exp.AttachmentURLs,
		SubmittedAt:     exp.SubmittedAt,
		SubmittedBy:     submittedBy,
		ApprovedAt:      exp.ApprovedAt,
		ApprovedBy:      approvedBy,
		ApprovalRemark:  exp.ApprovalRemark,
		RejectedAt:      exp.RejectedAt,
		RejectedBy:      rejectedBy,
		RejectionReason: exp.RejectionReason,
		CreatedAt:       exp.CreatedAt,
		UpdatedAt:       exp.UpdatedAt,
		Version:         exp.Version,
	}
}

func (h *ExpenseIncomeHandler) toIncomeRecordResponse(inc financeapp.OtherIncomeRecordResponse) OtherIncomeRecordResponse {
	var confirmedBy *string
	if inc.ConfirmedBy != nil {
		s := inc.ConfirmedBy.String()
		confirmedBy = &s
	}

	return OtherIncomeRecordResponse{
		ID:             inc.ID.String(),
		TenantID:       inc.TenantID.String(),
		IncomeNumber:   inc.IncomeNumber,
		Category:       inc.Category,
		CategoryName:   inc.CategoryName,
		Amount:         inc.Amount.InexactFloat64(),
		Description:    inc.Description,
		ReceivedAt:     inc.ReceivedAt,
		Status:         inc.Status,
		ReceiptStatus:  inc.ReceiptStatus,
		PaymentMethod:  inc.PaymentMethod,
		ActualReceived: inc.ActualReceived,
		Remark:         inc.Remark,
		AttachmentURLs: inc.AttachmentURLs,
		ConfirmedAt:    inc.ConfirmedAt,
		ConfirmedBy:    confirmedBy,
		CreatedAt:      inc.CreatedAt,
		UpdatedAt:      inc.UpdatedAt,
		Version:        inc.Version,
	}
}

// RegisterRoutes registers all expense and income routes
func (h *ExpenseIncomeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Expense routes
	expenses := rg.Group("/expenses")
	{
		expenses.GET("", h.ListExpenses)
		expenses.GET("/summary", h.GetExpensesSummary)
		expenses.GET("/:id", h.GetExpense)
		expenses.POST("", h.CreateExpense)
		expenses.PUT("/:id", h.UpdateExpense)
		expenses.DELETE("/:id", h.DeleteExpense)
		expenses.POST("/:id/submit", h.SubmitExpense)
		expenses.POST("/:id/approve", h.ApproveExpense)
		expenses.POST("/:id/reject", h.RejectExpense)
		expenses.POST("/:id/cancel", h.CancelExpense)
		expenses.POST("/:id/pay", h.MarkExpensePaid)
	}

	// Income routes
	incomes := rg.Group("/incomes")
	{
		incomes.GET("", h.ListIncomes)
		incomes.GET("/summary", h.GetIncomesSummary)
		incomes.GET("/:id", h.GetIncome)
		incomes.POST("", h.CreateIncome)
		incomes.PUT("/:id", h.UpdateIncome)
		incomes.DELETE("/:id", h.DeleteIncome)
		incomes.POST("/:id/confirm", h.ConfirmIncome)
		incomes.POST("/:id/cancel", h.CancelIncome)
		incomes.POST("/:id/receive", h.MarkIncomeReceived)
	}

	// Cash flow
	rg.GET("/cash-flow", h.GetCashFlow)
}
