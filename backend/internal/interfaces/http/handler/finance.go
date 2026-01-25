package handler

import (
	"time"

	financeapp "github.com/erp/backend/internal/application/finance"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FinanceHandler handles finance-related API endpoints
type FinanceHandler struct {
	BaseHandler
	financeService *financeapp.FinanceService
}

// NewFinanceHandler creates a new FinanceHandler
func NewFinanceHandler(financeService *financeapp.FinanceService) *FinanceHandler {
	return &FinanceHandler{
		financeService: financeService,
	}
}

// ===================== Request/Response DTOs =====================

// AccountReceivableResponse represents an account receivable in API responses
// @Description Account receivable response
type AccountReceivableResponse struct {
	ID                string                  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID          string                  `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ReceivableNumber  string                  `json:"receivable_number" example:"AR-2026-00001"`
	CustomerID        string                  `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	CustomerName      string                  `json:"customer_name" example:"张三"`
	SourceType        string                  `json:"source_type" example:"SALES_ORDER"`
	SourceID          string                  `json:"source_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	SourceNumber      string                  `json:"source_number" example:"SO-2026-00001"`
	TotalAmount       float64                 `json:"total_amount" example:"1000.00"`
	PaidAmount        float64                 `json:"paid_amount" example:"500.00"`
	OutstandingAmount float64                 `json:"outstanding_amount" example:"500.00"`
	Status            string                  `json:"status" example:"PARTIAL"`
	DueDate           *time.Time              `json:"due_date,omitempty"`
	PaymentRecords    []PaymentRecordResponse `json:"payment_records,omitempty"`
	Remark            string                  `json:"remark,omitempty" example:"备注"`
	PaidAt            *time.Time              `json:"paid_at,omitempty"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	Version           int                     `json:"version" example:"1"`
}

// PaymentRecordResponse represents a payment record in API responses
// @Description Payment record response
type PaymentRecordResponse struct {
	ID               string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReceiptVoucherID string    `json:"receipt_voucher_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Amount           float64   `json:"amount" example:"500.00"`
	AppliedAt        time.Time `json:"applied_at"`
	Remark           string    `json:"remark,omitempty" example:"收款记录"`
}

// AccountPayableResponse represents an account payable in API responses
// @Description Account payable response
type AccountPayableResponse struct {
	ID                string                         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID          string                         `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	PayableNumber     string                         `json:"payable_number" example:"AP-2026-00001"`
	SupplierID        string                         `json:"supplier_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	SupplierName      string                         `json:"supplier_name" example:"供应商A"`
	SourceType        string                         `json:"source_type" example:"PURCHASE_ORDER"`
	SourceID          string                         `json:"source_id" example:"550e8400-e29b-41d4-a716-446655440003"`
	SourceNumber      string                         `json:"source_number" example:"PO-2026-00001"`
	TotalAmount       float64                        `json:"total_amount" example:"2000.00"`
	PaidAmount        float64                        `json:"paid_amount" example:"1000.00"`
	OutstandingAmount float64                        `json:"outstanding_amount" example:"1000.00"`
	Status            string                         `json:"status" example:"PARTIAL"`
	DueDate           *time.Time                     `json:"due_date,omitempty"`
	PaymentRecords    []PayablePaymentRecordResponse `json:"payment_records,omitempty"`
	Remark            string                         `json:"remark,omitempty" example:"备注"`
	PaidAt            *time.Time                     `json:"paid_at,omitempty"`
	CreatedAt         time.Time                      `json:"created_at"`
	UpdatedAt         time.Time                      `json:"updated_at"`
	Version           int                            `json:"version" example:"1"`
}

// PayablePaymentRecordResponse represents a payment record for payable in API responses
// @Description Payable payment record response
type PayablePaymentRecordResponse struct {
	ID               string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PaymentVoucherID string    `json:"payment_voucher_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Amount           float64   `json:"amount" example:"1000.00"`
	AppliedAt        time.Time `json:"applied_at"`
	Remark           string    `json:"remark,omitempty" example:"付款记录"`
}

// ReceiptVoucherResponse represents a receipt voucher in API responses
// @Description Receipt voucher response
type ReceiptVoucherResponse struct {
	ID                string                         `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID          string                         `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	VoucherNumber     string                         `json:"voucher_number" example:"RV-2026-00001"`
	CustomerID        string                         `json:"customer_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	CustomerName      string                         `json:"customer_name" example:"张三"`
	Amount            float64                        `json:"amount" example:"1000.00"`
	AllocatedAmount   float64                        `json:"allocated_amount" example:"500.00"`
	UnallocatedAmount float64                        `json:"unallocated_amount" example:"500.00"`
	PaymentMethod     string                         `json:"payment_method" example:"CASH"`
	PaymentReference  string                         `json:"payment_reference,omitempty" example:"银行转账-12345"`
	Status            string                         `json:"status" example:"CONFIRMED"`
	ReceiptDate       time.Time                      `json:"receipt_date"`
	Allocations       []ReceivableAllocationResponse `json:"allocations,omitempty"`
	Remark            string                         `json:"remark,omitempty" example:"备注"`
	ConfirmedAt       *time.Time                     `json:"confirmed_at,omitempty"`
	CreatedAt         time.Time                      `json:"created_at"`
	UpdatedAt         time.Time                      `json:"updated_at"`
	Version           int                            `json:"version" example:"1"`
}

// ReceivableAllocationResponse represents a receivable allocation in API responses
// @Description Receivable allocation response
type ReceivableAllocationResponse struct {
	ID               string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReceivableID     string    `json:"receivable_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ReceivableNumber string    `json:"receivable_number" example:"AR-2026-00001"`
	Amount           float64   `json:"amount" example:"500.00"`
	AllocatedAt      time.Time `json:"allocated_at"`
	Remark           string    `json:"remark,omitempty" example:"核销备注"`
}

// PaymentVoucherResponse represents a payment voucher in API responses
// @Description Payment voucher response
type PaymentVoucherResponse struct {
	ID                string                      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID          string                      `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	VoucherNumber     string                      `json:"voucher_number" example:"PV-2026-00001"`
	SupplierID        string                      `json:"supplier_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	SupplierName      string                      `json:"supplier_name" example:"供应商A"`
	Amount            float64                     `json:"amount" example:"2000.00"`
	AllocatedAmount   float64                     `json:"allocated_amount" example:"1000.00"`
	UnallocatedAmount float64                     `json:"unallocated_amount" example:"1000.00"`
	PaymentMethod     string                      `json:"payment_method" example:"BANK_TRANSFER"`
	PaymentReference  string                      `json:"payment_reference,omitempty" example:"银行转账-67890"`
	Status            string                      `json:"status" example:"CONFIRMED"`
	PaymentDate       time.Time                   `json:"payment_date"`
	Allocations       []PayableAllocationResponse `json:"allocations,omitempty"`
	Remark            string                      `json:"remark,omitempty" example:"备注"`
	ConfirmedAt       *time.Time                  `json:"confirmed_at,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
	Version           int                         `json:"version" example:"1"`
}

// PayableAllocationResponse represents a payable allocation in API responses
// @Description Payable allocation response
type PayableAllocationResponse struct {
	ID            string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PayableID     string    `json:"payable_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	PayableNumber string    `json:"payable_number" example:"AP-2026-00001"`
	Amount        float64   `json:"amount" example:"1000.00"`
	AllocatedAt   time.Time `json:"allocated_at"`
	Remark        string    `json:"remark,omitempty" example:"核销备注"`
}

// CreateReceiptVoucherRequest represents a request to create a receipt voucher
// @Description Request body for creating a receipt voucher
type CreateReceiptVoucherRequest struct {
	CustomerID       string  `json:"customer_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CustomerName     string  `json:"customer_name" binding:"required,min=1,max=200" example:"张三"`
	Amount           float64 `json:"amount" binding:"required,gt=0" example:"1000.00"`
	PaymentMethod    string  `json:"payment_method" binding:"required" example:"CASH"`
	PaymentReference string  `json:"payment_reference" example:"银行转账-12345"`
	ReceiptDate      string  `json:"receipt_date" binding:"required" example:"2026-01-24"`
	Remark           string  `json:"remark" example:"收款备注"`
}

// CreatePaymentVoucherRequest represents a request to create a payment voucher
// @Description Request body for creating a payment voucher
type CreatePaymentVoucherRequest struct {
	SupplierID       string  `json:"supplier_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	SupplierName     string  `json:"supplier_name" binding:"required,min=1,max=200" example:"供应商A"`
	Amount           float64 `json:"amount" binding:"required,gt=0" example:"2000.00"`
	PaymentMethod    string  `json:"payment_method" binding:"required" example:"BANK_TRANSFER"`
	PaymentReference string  `json:"payment_reference" example:"银行转账-67890"`
	PaymentDate      string  `json:"payment_date" binding:"required" example:"2026-01-24"`
	Remark           string  `json:"remark" example:"付款备注"`
}

// CancelVoucherRequest represents a request to cancel a voucher
// @Description Request body for cancelling a voucher
type CancelVoucherRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=500" example:"客户取消"`
}

// ReconcileRequest represents a request to reconcile a voucher
// @Description Request body for reconciling a voucher
type ReconcileRequest struct {
	StrategyType      string                         `json:"strategy_type" binding:"required" example:"FIFO"`
	ManualAllocations []ManualAllocationInputRequest `json:"manual_allocations,omitempty"`
}

// ManualAllocationInputRequest represents a manual allocation input
// @Description Manual allocation input for reconciliation
type ManualAllocationInputRequest struct {
	TargetID string  `json:"target_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Amount   float64 `json:"amount" binding:"required,gt=0" example:"500.00"`
}

// ReceivableSummaryResponse represents a summary of receivables
// @Description Receivable summary response
type ReceivableSummaryResponse struct {
	TotalOutstanding float64 `json:"total_outstanding" example:"50000.00"`
	TotalOverdue     float64 `json:"total_overdue" example:"10000.00"`
	PendingCount     int64   `json:"pending_count" example:"10"`
	PartialCount     int64   `json:"partial_count" example:"5"`
	OverdueCount     int64   `json:"overdue_count" example:"3"`
}

// PayableSummaryResponse represents a summary of payables
// @Description Payable summary response
type PayableSummaryResponse struct {
	TotalOutstanding float64 `json:"total_outstanding" example:"80000.00"`
	TotalOverdue     float64 `json:"total_overdue" example:"20000.00"`
	PendingCount     int64   `json:"pending_count" example:"15"`
	PartialCount     int64   `json:"partial_count" example:"8"`
	OverdueCount     int64   `json:"overdue_count" example:"5"`
}

// ReconcileReceiptResultResponse represents the result of reconciling a receipt voucher
// @Description Reconcile receipt result response
type ReconcileReceiptResultResponse struct {
	Voucher              ReceiptVoucherResponse      `json:"voucher"`
	UpdatedReceivables   []AccountReceivableResponse `json:"updated_receivables"`
	TotalReconciled      float64                     `json:"total_reconciled" example:"500.00"`
	RemainingUnallocated float64                     `json:"remaining_unallocated" example:"500.00"`
	FullyReconciled      bool                        `json:"fully_reconciled" example:"false"`
}

// ReconcilePaymentResultResponse represents the result of reconciling a payment voucher
// @Description Reconcile payment result response
type ReconcilePaymentResultResponse struct {
	Voucher              PaymentVoucherResponse   `json:"voucher"`
	UpdatedPayables      []AccountPayableResponse `json:"updated_payables"`
	TotalReconciled      float64                  `json:"total_reconciled" example:"1000.00"`
	RemainingUnallocated float64                  `json:"remaining_unallocated" example:"1000.00"`
	FullyReconciled      bool                     `json:"fully_reconciled" example:"false"`
}

// ===================== Account Receivable Handlers =====================

// ListReceivables godoc
// @Summary      List account receivables
// @Description  Retrieve a paginated list of account receivables with filtering
// @Tags         finance-receivables
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (receivable number, customer name)"
// @Param        customer_id query string false "Customer ID" format(uuid)
// @Param        status query string false "Status" Enums(PENDING, PARTIAL, PAID, REVERSED, CANCELLED)
// @Param        source_type query string false "Source type" Enums(SALES_ORDER, SALES_RETURN, MANUAL)
// @Param        from_date query string false "From date (ISO 8601)" format(date)
// @Param        to_date query string false "To date (ISO 8601)" format(date)
// @Param        overdue query boolean false "Filter overdue only"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Success      200 {object} dto.Response{data=[]AccountReceivableResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receivables [get]
func (h *FinanceHandler) ListReceivables(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter financeapp.AccountReceivableListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	receivables, total, err := h.financeService.ListReceivables(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, toAccountReceivableResponses(receivables), total, filter.Page, filter.PageSize)
}

// GetReceivableByID godoc
// @Summary      Get account receivable by ID
// @Description  Retrieve an account receivable by its ID
// @Tags         finance-receivables
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Receivable ID" format(uuid)
// @Success      200 {object} dto.Response{data=AccountReceivableResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receivables/{id} [get]
func (h *FinanceHandler) GetReceivableByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	receivableID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid receivable ID format")
		return
	}

	receivable, err := h.financeService.GetReceivableByID(c.Request.Context(), tenantID, receivableID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAccountReceivableResponse(receivable))
}

// GetReceivableSummary godoc
// @Summary      Get receivables summary
// @Description  Get summary statistics for account receivables
// @Tags         finance-receivables
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Success      200 {object} dto.Response{data=ReceivableSummaryResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receivables/summary [get]
func (h *FinanceHandler) GetReceivableSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	summary, err := h.financeService.GetReceivableSummary(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, ReceivableSummaryResponse{
		TotalOutstanding: summary.TotalOutstanding.InexactFloat64(),
		TotalOverdue:     summary.TotalOverdue.InexactFloat64(),
		PendingCount:     summary.PendingCount,
		PartialCount:     summary.PartialCount,
		OverdueCount:     summary.OverdueCount,
	})
}

// ===================== Account Payable Handlers =====================

// ListPayables godoc
// @Summary      List account payables
// @Description  Retrieve a paginated list of account payables with filtering
// @Tags         finance-payables
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (payable number, supplier name)"
// @Param        supplier_id query string false "Supplier ID" format(uuid)
// @Param        status query string false "Status" Enums(PENDING, PARTIAL, PAID, REVERSED, CANCELLED)
// @Param        source_type query string false "Source type" Enums(PURCHASE_ORDER, PURCHASE_RETURN, MANUAL)
// @Param        from_date query string false "From date (ISO 8601)" format(date)
// @Param        to_date query string false "To date (ISO 8601)" format(date)
// @Param        overdue query boolean false "Filter overdue only"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Success      200 {object} dto.Response{data=[]AccountPayableResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payables [get]
func (h *FinanceHandler) ListPayables(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter financeapp.AccountPayableListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	payables, total, err := h.financeService.ListPayables(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, toAccountPayableResponses(payables), total, filter.Page, filter.PageSize)
}

// GetPayableByID godoc
// @Summary      Get account payable by ID
// @Description  Retrieve an account payable by its ID
// @Tags         finance-payables
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Payable ID" format(uuid)
// @Success      200 {object} dto.Response{data=AccountPayableResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payables/{id} [get]
func (h *FinanceHandler) GetPayableByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	payableID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid payable ID format")
		return
	}

	payable, err := h.financeService.GetPayableByID(c.Request.Context(), tenantID, payableID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toAccountPayableResponse(payable))
}

// GetPayableSummary godoc
// @Summary      Get payables summary
// @Description  Get summary statistics for account payables
// @Tags         finance-payables
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Success      200 {object} dto.Response{data=PayableSummaryResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payables/summary [get]
func (h *FinanceHandler) GetPayableSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	summary, err := h.financeService.GetPayableSummary(c.Request.Context(), tenantID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, PayableSummaryResponse{
		TotalOutstanding: summary.TotalOutstanding.InexactFloat64(),
		TotalOverdue:     summary.TotalOverdue.InexactFloat64(),
		PendingCount:     summary.PendingCount,
		PartialCount:     summary.PartialCount,
		OverdueCount:     summary.OverdueCount,
	})
}

// ===================== Receipt Voucher Handlers =====================

// CreateReceiptVoucher godoc
// @Summary      Create a receipt voucher
// @Description  Create a new receipt voucher for customer payment
// @Tags         finance-receipts
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreateReceiptVoucherRequest true "Receipt voucher creation request"
// @Success      201 {object} dto.Response{data=ReceiptVoucherResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receipts [post]
func (h *FinanceHandler) CreateReceiptVoucher(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreateReceiptVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		h.BadRequest(c, "Invalid customer ID format")
		return
	}

	receiptDate, err := parseDateTime(req.ReceiptDate)
	if err != nil {
		h.BadRequest(c, "Invalid receipt date format")
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	appReq := financeapp.CreateReceiptVoucherRequest{
		CustomerID:       customerID,
		CustomerName:     req.CustomerName,
		Amount:           decimal.NewFromFloat(req.Amount),
		PaymentMethod:    req.PaymentMethod,
		PaymentReference: req.PaymentReference,
		ReceiptDate:      receiptDate,
		Remark:           req.Remark,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		appReq.CreatedBy = &userID
	}

	voucher, err := h.financeService.CreateReceiptVoucher(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, toReceiptVoucherResponse(voucher))
}

// ListReceiptVouchers godoc
// @Summary      List receipt vouchers
// @Description  Retrieve a paginated list of receipt vouchers with filtering
// @Tags         finance-receipts
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (voucher number, customer name)"
// @Param        customer_id query string false "Customer ID" format(uuid)
// @Param        status query string false "Status" Enums(DRAFT, CONFIRMED, ALLOCATED, CANCELLED)
// @Param        payment_method query string false "Payment method" Enums(CASH, BANK_TRANSFER, WECHAT, ALIPAY, CHECK, BALANCE, OTHER)
// @Param        from_date query string false "From date (ISO 8601)" format(date)
// @Param        to_date query string false "To date (ISO 8601)" format(date)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Success      200 {object} dto.Response{data=[]ReceiptVoucherResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receipts [get]
func (h *FinanceHandler) ListReceiptVouchers(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter financeapp.ReceiptVoucherListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	vouchers, total, err := h.financeService.ListReceiptVouchers(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, toReceiptVoucherResponses(vouchers), total, filter.Page, filter.PageSize)
}

// GetReceiptVoucherByID godoc
// @Summary      Get receipt voucher by ID
// @Description  Retrieve a receipt voucher by its ID
// @Tags         finance-receipts
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Receipt Voucher ID" format(uuid)
// @Success      200 {object} dto.Response{data=ReceiptVoucherResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receipts/{id} [get]
func (h *FinanceHandler) GetReceiptVoucherByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	voucherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid voucher ID format")
		return
	}

	voucher, err := h.financeService.GetReceiptVoucherByID(c.Request.Context(), tenantID, voucherID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toReceiptVoucherResponse(voucher))
}

// ConfirmReceiptVoucher godoc
// @Summary      Confirm a receipt voucher
// @Description  Confirm a receipt voucher to allow reconciliation
// @Tags         finance-receipts
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Receipt Voucher ID" format(uuid)
// @Success      200 {object} dto.Response{data=ReceiptVoucherResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receipts/{id}/confirm [post]
func (h *FinanceHandler) ConfirmReceiptVoucher(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	voucherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid voucher ID format")
		return
	}

	// TODO: Get actual user ID from auth context
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	voucher, err := h.financeService.ConfirmReceiptVoucher(c.Request.Context(), tenantID, voucherID, userID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toReceiptVoucherResponse(voucher))
}

// CancelReceiptVoucher godoc
// @Summary      Cancel a receipt voucher
// @Description  Cancel a receipt voucher (only if not yet allocated)
// @Tags         finance-receipts
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Receipt Voucher ID" format(uuid)
// @Param        request body CancelVoucherRequest true "Cancel request"
// @Success      200 {object} dto.Response{data=ReceiptVoucherResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receipts/{id}/cancel [post]
func (h *FinanceHandler) CancelReceiptVoucher(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	voucherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid voucher ID format")
		return
	}

	var req CancelVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// TODO: Get actual user ID from auth context
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	voucher, err := h.financeService.CancelReceiptVoucher(c.Request.Context(), tenantID, voucherID, userID, req.Reason)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toReceiptVoucherResponse(voucher))
}

// ReconcileReceiptVoucher godoc
// @Summary      Reconcile a receipt voucher
// @Description  Reconcile a receipt voucher to outstanding receivables
// @Tags         finance-receipts
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Receipt Voucher ID" format(uuid)
// @Param        request body ReconcileRequest true "Reconcile request"
// @Success      200 {object} dto.Response{data=ReconcileReceiptResultResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/receipts/{id}/reconcile [post]
func (h *FinanceHandler) ReconcileReceiptVoucher(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	voucherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid voucher ID format")
		return
	}

	var req ReconcileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	var manualAllocs []financeapp.ManualAllocationRequest
	for _, ma := range req.ManualAllocations {
		targetID, err := uuid.Parse(ma.TargetID)
		if err != nil {
			h.BadRequest(c, "Invalid target ID format in manual allocations")
			return
		}
		manualAllocs = append(manualAllocs, financeapp.ManualAllocationRequest{
			TargetID: targetID,
			Amount:   decimal.NewFromFloat(ma.Amount),
		})
	}

	appReq := financeapp.ReconcileReceiptRequest{
		VoucherID:         voucherID,
		StrategyType:      req.StrategyType,
		ManualAllocations: manualAllocs,
	}

	result, err := h.financeService.ReconcileReceipt(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toReconcileReceiptResultResponse(result))
}

// ===================== Payment Voucher Handlers =====================

// CreatePaymentVoucher godoc
// @Summary      Create a payment voucher
// @Description  Create a new payment voucher for supplier payment
// @Tags         finance-payments
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        request body CreatePaymentVoucherRequest true "Payment voucher creation request"
// @Success      201 {object} dto.Response{data=PaymentVoucherResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payments [post]
func (h *FinanceHandler) CreatePaymentVoucher(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var req CreatePaymentVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	supplierID, err := uuid.Parse(req.SupplierID)
	if err != nil {
		h.BadRequest(c, "Invalid supplier ID format")
		return
	}

	paymentDate, err := parseDateTime(req.PaymentDate)
	if err != nil {
		h.BadRequest(c, "Invalid payment date format")
		return
	}

	// Get user ID from JWT context (optional, for data scope)
	userID, _ := getUserID(c)

	appReq := financeapp.CreatePaymentVoucherRequest{
		SupplierID:       supplierID,
		SupplierName:     req.SupplierName,
		Amount:           decimal.NewFromFloat(req.Amount),
		PaymentMethod:    req.PaymentMethod,
		PaymentReference: req.PaymentReference,
		PaymentDate:      paymentDate,
		Remark:           req.Remark,
	}

	// Set CreatedBy for data scope filtering
	if userID != uuid.Nil {
		appReq.CreatedBy = &userID
	}

	voucher, err := h.financeService.CreatePaymentVoucher(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Created(c, toPaymentVoucherResponse(voucher))
}

// ListPaymentVouchers godoc
// @Summary      List payment vouchers
// @Description  Retrieve a paginated list of payment vouchers with filtering
// @Tags         finance-payments
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        search query string false "Search term (voucher number, supplier name)"
// @Param        supplier_id query string false "Supplier ID" format(uuid)
// @Param        status query string false "Status" Enums(DRAFT, CONFIRMED, ALLOCATED, CANCELLED)
// @Param        payment_method query string false "Payment method" Enums(CASH, BANK_TRANSFER, WECHAT, ALIPAY, CHECK, BALANCE, OTHER)
// @Param        from_date query string false "From date (ISO 8601)" format(date)
// @Param        to_date query string false "To date (ISO 8601)" format(date)
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20) maximum(100)
// @Success      200 {object} dto.Response{data=[]PaymentVoucherResponse,meta=dto.Meta}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payments [get]
func (h *FinanceHandler) ListPaymentVouchers(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	var filter financeapp.PaymentVoucherListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	vouchers, total, err := h.financeService.ListPaymentVouchers(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.SuccessWithMeta(c, toPaymentVoucherResponses(vouchers), total, filter.Page, filter.PageSize)
}

// GetPaymentVoucherByID godoc
// @Summary      Get payment voucher by ID
// @Description  Retrieve a payment voucher by its ID
// @Tags         finance-payments
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Payment Voucher ID" format(uuid)
// @Success      200 {object} dto.Response{data=PaymentVoucherResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payments/{id} [get]
func (h *FinanceHandler) GetPaymentVoucherByID(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	voucherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid voucher ID format")
		return
	}

	voucher, err := h.financeService.GetPaymentVoucherByID(c.Request.Context(), tenantID, voucherID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPaymentVoucherResponse(voucher))
}

// ConfirmPaymentVoucher godoc
// @Summary      Confirm a payment voucher
// @Description  Confirm a payment voucher to allow reconciliation
// @Tags         finance-payments
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Payment Voucher ID" format(uuid)
// @Success      200 {object} dto.Response{data=PaymentVoucherResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payments/{id}/confirm [post]
func (h *FinanceHandler) ConfirmPaymentVoucher(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	voucherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid voucher ID format")
		return
	}

	// TODO: Get actual user ID from auth context
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	voucher, err := h.financeService.ConfirmPaymentVoucher(c.Request.Context(), tenantID, voucherID, userID)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPaymentVoucherResponse(voucher))
}

// CancelPaymentVoucher godoc
// @Summary      Cancel a payment voucher
// @Description  Cancel a payment voucher (only if not yet allocated)
// @Tags         finance-payments
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Payment Voucher ID" format(uuid)
// @Param        request body CancelVoucherRequest true "Cancel request"
// @Success      200 {object} dto.Response{data=PaymentVoucherResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payments/{id}/cancel [post]
func (h *FinanceHandler) CancelPaymentVoucher(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	voucherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid voucher ID format")
		return
	}

	var req CancelVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	// TODO: Get actual user ID from auth context
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	voucher, err := h.financeService.CancelPaymentVoucher(c.Request.Context(), tenantID, voucherID, userID, req.Reason)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toPaymentVoucherResponse(voucher))
}

// ReconcilePaymentVoucher godoc
// @Summary      Reconcile a payment voucher
// @Description  Reconcile a payment voucher to outstanding payables
// @Tags         finance-payments
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID header string false "Tenant ID (optional for dev)"
// @Param        id path string true "Payment Voucher ID" format(uuid)
// @Param        request body ReconcileRequest true "Reconcile request"
// @Success      200 {object} dto.Response{data=ReconcilePaymentResultResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      404 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Security     BearerAuth
// @Router       /finance/payments/{id}/reconcile [post]
func (h *FinanceHandler) ReconcilePaymentVoucher(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID")
		return
	}

	voucherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.BadRequest(c, "Invalid voucher ID format")
		return
	}

	var req ReconcileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, err.Error())
		return
	}

	var manualAllocs []financeapp.ManualAllocationRequest
	for _, ma := range req.ManualAllocations {
		targetID, err := uuid.Parse(ma.TargetID)
		if err != nil {
			h.BadRequest(c, "Invalid target ID format in manual allocations")
			return
		}
		manualAllocs = append(manualAllocs, financeapp.ManualAllocationRequest{
			TargetID: targetID,
			Amount:   decimal.NewFromFloat(ma.Amount),
		})
	}

	appReq := financeapp.ReconcilePaymentRequest{
		VoucherID:         voucherID,
		StrategyType:      req.StrategyType,
		ManualAllocations: manualAllocs,
	}

	result, err := h.financeService.ReconcilePayment(c.Request.Context(), tenantID, appReq)
	if err != nil {
		h.HandleDomainError(c, err)
		return
	}

	h.Success(c, toReconcilePaymentResultResponse(result))
}

// ===================== Response Conversion Functions =====================

func toAccountReceivableResponse(r *financeapp.AccountReceivableResponse) AccountReceivableResponse {
	paymentRecords := make([]PaymentRecordResponse, len(r.PaymentRecords))
	for i, pr := range r.PaymentRecords {
		paymentRecords[i] = PaymentRecordResponse{
			ID:               pr.ID.String(),
			ReceiptVoucherID: pr.ReceiptVoucherID.String(),
			Amount:           pr.Amount.InexactFloat64(),
			AppliedAt:        pr.AppliedAt,
			Remark:           pr.Remark,
		}
	}

	return AccountReceivableResponse{
		ID:                r.ID.String(),
		TenantID:          r.TenantID.String(),
		ReceivableNumber:  r.ReceivableNumber,
		CustomerID:        r.CustomerID.String(),
		CustomerName:      r.CustomerName,
		SourceType:        r.SourceType,
		SourceID:          r.SourceID.String(),
		SourceNumber:      r.SourceNumber,
		TotalAmount:       r.TotalAmount.InexactFloat64(),
		PaidAmount:        r.PaidAmount.InexactFloat64(),
		OutstandingAmount: r.OutstandingAmount.InexactFloat64(),
		Status:            r.Status,
		DueDate:           r.DueDate,
		PaymentRecords:    paymentRecords,
		Remark:            r.Remark,
		PaidAt:            r.PaidAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		Version:           r.Version,
	}
}

func toAccountReceivableResponses(receivables []financeapp.AccountReceivableResponse) []AccountReceivableResponse {
	responses := make([]AccountReceivableResponse, len(receivables))
	for i, r := range receivables {
		responses[i] = toAccountReceivableResponse(&r)
	}
	return responses
}

func toAccountPayableResponse(p *financeapp.AccountPayableResponse) AccountPayableResponse {
	paymentRecords := make([]PayablePaymentRecordResponse, len(p.PaymentRecords))
	for i, pr := range p.PaymentRecords {
		paymentRecords[i] = PayablePaymentRecordResponse{
			ID:               pr.ID.String(),
			PaymentVoucherID: pr.PaymentVoucherID.String(),
			Amount:           pr.Amount.InexactFloat64(),
			AppliedAt:        pr.AppliedAt,
			Remark:           pr.Remark,
		}
	}

	return AccountPayableResponse{
		ID:                p.ID.String(),
		TenantID:          p.TenantID.String(),
		PayableNumber:     p.PayableNumber,
		SupplierID:        p.SupplierID.String(),
		SupplierName:      p.SupplierName,
		SourceType:        p.SourceType,
		SourceID:          p.SourceID.String(),
		SourceNumber:      p.SourceNumber,
		TotalAmount:       p.TotalAmount.InexactFloat64(),
		PaidAmount:        p.PaidAmount.InexactFloat64(),
		OutstandingAmount: p.OutstandingAmount.InexactFloat64(),
		Status:            p.Status,
		DueDate:           p.DueDate,
		PaymentRecords:    paymentRecords,
		Remark:            p.Remark,
		PaidAt:            p.PaidAt,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		Version:           p.Version,
	}
}

func toAccountPayableResponses(payables []financeapp.AccountPayableResponse) []AccountPayableResponse {
	responses := make([]AccountPayableResponse, len(payables))
	for i, p := range payables {
		responses[i] = toAccountPayableResponse(&p)
	}
	return responses
}

func toReceiptVoucherResponse(v *financeapp.ReceiptVoucherResponse) ReceiptVoucherResponse {
	allocations := make([]ReceivableAllocationResponse, len(v.Allocations))
	for i, a := range v.Allocations {
		allocations[i] = ReceivableAllocationResponse{
			ID:               a.ID.String(),
			ReceivableID:     a.ReceivableID.String(),
			ReceivableNumber: a.ReceivableNumber,
			Amount:           a.Amount.InexactFloat64(),
			AllocatedAt:      a.AllocatedAt,
			Remark:           a.Remark,
		}
	}

	return ReceiptVoucherResponse{
		ID:                v.ID.String(),
		TenantID:          v.TenantID.String(),
		VoucherNumber:     v.VoucherNumber,
		CustomerID:        v.CustomerID.String(),
		CustomerName:      v.CustomerName,
		Amount:            v.Amount.InexactFloat64(),
		AllocatedAmount:   v.AllocatedAmount.InexactFloat64(),
		UnallocatedAmount: v.UnallocatedAmount.InexactFloat64(),
		PaymentMethod:     v.PaymentMethod,
		PaymentReference:  v.PaymentReference,
		Status:            v.Status,
		ReceiptDate:       v.ReceiptDate,
		Allocations:       allocations,
		Remark:            v.Remark,
		ConfirmedAt:       v.ConfirmedAt,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
		Version:           v.Version,
	}
}

func toReceiptVoucherResponses(vouchers []financeapp.ReceiptVoucherResponse) []ReceiptVoucherResponse {
	responses := make([]ReceiptVoucherResponse, len(vouchers))
	for i, v := range vouchers {
		responses[i] = toReceiptVoucherResponse(&v)
	}
	return responses
}

func toPaymentVoucherResponse(v *financeapp.PaymentVoucherResponse) PaymentVoucherResponse {
	allocations := make([]PayableAllocationResponse, len(v.Allocations))
	for i, a := range v.Allocations {
		allocations[i] = PayableAllocationResponse{
			ID:            a.ID.String(),
			PayableID:     a.PayableID.String(),
			PayableNumber: a.PayableNumber,
			Amount:        a.Amount.InexactFloat64(),
			AllocatedAt:   a.AllocatedAt,
			Remark:        a.Remark,
		}
	}

	return PaymentVoucherResponse{
		ID:                v.ID.String(),
		TenantID:          v.TenantID.String(),
		VoucherNumber:     v.VoucherNumber,
		SupplierID:        v.SupplierID.String(),
		SupplierName:      v.SupplierName,
		Amount:            v.Amount.InexactFloat64(),
		AllocatedAmount:   v.AllocatedAmount.InexactFloat64(),
		UnallocatedAmount: v.UnallocatedAmount.InexactFloat64(),
		PaymentMethod:     v.PaymentMethod,
		PaymentReference:  v.PaymentReference,
		Status:            v.Status,
		PaymentDate:       v.PaymentDate,
		Allocations:       allocations,
		Remark:            v.Remark,
		ConfirmedAt:       v.ConfirmedAt,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
		Version:           v.Version,
	}
}

func toPaymentVoucherResponses(vouchers []financeapp.PaymentVoucherResponse) []PaymentVoucherResponse {
	responses := make([]PaymentVoucherResponse, len(vouchers))
	for i, v := range vouchers {
		responses[i] = toPaymentVoucherResponse(&v)
	}
	return responses
}

func toReconcileReceiptResultResponse(r *financeapp.ReconcileReceiptResult) ReconcileReceiptResultResponse {
	updatedReceivables := make([]AccountReceivableResponse, len(r.UpdatedReceivables))
	for i, recv := range r.UpdatedReceivables {
		updatedReceivables[i] = toAccountReceivableResponse(&recv)
	}

	return ReconcileReceiptResultResponse{
		Voucher:              toReceiptVoucherResponse(r.Voucher),
		UpdatedReceivables:   updatedReceivables,
		TotalReconciled:      r.TotalReconciled.InexactFloat64(),
		RemainingUnallocated: r.RemainingUnallocated.InexactFloat64(),
		FullyReconciled:      r.FullyReconciled,
	}
}

func toReconcilePaymentResultResponse(r *financeapp.ReconcilePaymentResult) ReconcilePaymentResultResponse {
	updatedPayables := make([]AccountPayableResponse, len(r.UpdatedPayables))
	for i, pay := range r.UpdatedPayables {
		updatedPayables[i] = toAccountPayableResponse(&pay)
	}

	return ReconcilePaymentResultResponse{
		Voucher:              toPaymentVoucherResponse(r.Voucher),
		UpdatedPayables:      updatedPayables,
		TotalReconciled:      r.TotalReconciled.InexactFloat64(),
		RemainingUnallocated: r.RemainingUnallocated.InexactFloat64(),
		FullyReconciled:      r.FullyReconciled,
	}
}

// Suppress unused import warning
var _ = dto.Response{}
