/**
 * Finance API Service
 *
 * Manual API service for finance module.
 * TODO: Replace with auto-generated orval client once OpenAPI is available
 */
import { customInstance } from '../../services/axios-instance'
import type {
  AccountReceivable,
  AccountPayable,
  ReceiptVoucher,
  PaymentVoucher,
  ReceivableSummary,
  PayableSummary,
  GetReceivablesParams,
  GetPayablesParams,
  GetReceiptVouchersParams,
  GetPaymentVouchersParams,
  CreateReceiptVoucherRequest,
  CreatePaymentVoucherRequest,
  CancelVoucherRequest,
  ReconcileRequest,
  ReconcileReceiptResult,
  ReconcilePaymentResult,
  ExpenseRecord,
  ExpenseSummary,
  GetExpenseRecordsParams,
  CreateExpenseRecordRequest,
  UpdateExpenseRecordRequest,
  ApproveExpenseRequest,
  RejectExpenseRequest,
  CancelExpenseRequest,
  MarkExpensePaidRequest,
  OtherIncomeRecord,
  IncomeSummary,
  GetOtherIncomeRecordsParams,
  CreateOtherIncomeRecordRequest,
  UpdateOtherIncomeRecordRequest,
  CancelIncomeRequest,
  MarkIncomeReceivedRequest,
  CashFlowSummary,
  ApiResponse,
} from './types'

export const getFinanceApi = () => {
  // ===================== Account Receivable APIs =====================

  /**
   * List account receivables with filtering
   */
  const getFinanceReceivables = (params?: GetReceivablesParams) => {
    return customInstance<ApiResponse<AccountReceivable[]>>({
      url: '/finance/receivables',
      method: 'GET',
      params,
    })
  }

  /**
   * Get account receivable by ID
   */
  const getFinanceReceivablesId = (id: string) => {
    return customInstance<ApiResponse<AccountReceivable>>({
      url: `/finance/receivables/${id}`,
      method: 'GET',
    })
  }

  /**
   * Get receivables summary
   */
  const getFinanceReceivablesSummary = () => {
    return customInstance<ApiResponse<ReceivableSummary>>({
      url: '/finance/receivables/summary',
      method: 'GET',
    })
  }

  // ===================== Account Payable APIs =====================

  /**
   * List account payables with filtering
   */
  const getFinancePayables = (params?: GetPayablesParams) => {
    return customInstance<ApiResponse<AccountPayable[]>>({
      url: '/finance/payables',
      method: 'GET',
      params,
    })
  }

  /**
   * Get account payable by ID
   */
  const getFinancePayablesId = (id: string) => {
    return customInstance<ApiResponse<AccountPayable>>({
      url: `/finance/payables/${id}`,
      method: 'GET',
    })
  }

  /**
   * Get payables summary
   */
  const getFinancePayablesSummary = () => {
    return customInstance<ApiResponse<PayableSummary>>({
      url: '/finance/payables/summary',
      method: 'GET',
    })
  }

  // ===================== Receipt Voucher APIs =====================

  /**
   * List receipt vouchers with filtering
   */
  const getFinanceReceipts = (params?: GetReceiptVouchersParams) => {
    return customInstance<ApiResponse<ReceiptVoucher[]>>({
      url: '/finance/receipts',
      method: 'GET',
      params,
    })
  }

  /**
   * Get receipt voucher by ID
   */
  const getFinanceReceiptsId = (id: string) => {
    return customInstance<ApiResponse<ReceiptVoucher>>({
      url: `/finance/receipts/${id}`,
      method: 'GET',
    })
  }

  /**
   * Create a new receipt voucher
   */
  const postFinanceReceipts = (request: CreateReceiptVoucherRequest) => {
    return customInstance<ApiResponse<ReceiptVoucher>>({
      url: '/finance/receipts',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Confirm a receipt voucher
   */
  const postFinanceReceiptsIdConfirm = (id: string) => {
    return customInstance<ApiResponse<ReceiptVoucher>>({
      url: `/finance/receipts/${id}/confirm`,
      method: 'POST',
    })
  }

  /**
   * Cancel a receipt voucher
   */
  const postFinanceReceiptsIdCancel = (id: string, request: CancelVoucherRequest) => {
    return customInstance<ApiResponse<ReceiptVoucher>>({
      url: `/finance/receipts/${id}/cancel`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Reconcile a receipt voucher
   */
  const postFinanceReceiptsIdReconcile = (id: string, request: ReconcileRequest) => {
    return customInstance<ApiResponse<ReconcileReceiptResult>>({
      url: `/finance/receipts/${id}/reconcile`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  // ===================== Payment Voucher APIs =====================

  /**
   * List payment vouchers with filtering
   */
  const getFinancePayments = (params?: GetPaymentVouchersParams) => {
    return customInstance<ApiResponse<PaymentVoucher[]>>({
      url: '/finance/payments',
      method: 'GET',
      params,
    })
  }

  /**
   * Get payment voucher by ID
   */
  const getFinancePaymentsId = (id: string) => {
    return customInstance<ApiResponse<PaymentVoucher>>({
      url: `/finance/payments/${id}`,
      method: 'GET',
    })
  }

  /**
   * Create a new payment voucher
   */
  const postFinancePayments = (request: CreatePaymentVoucherRequest) => {
    return customInstance<ApiResponse<PaymentVoucher>>({
      url: '/finance/payments',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Confirm a payment voucher
   */
  const postFinancePaymentsIdConfirm = (id: string) => {
    return customInstance<ApiResponse<PaymentVoucher>>({
      url: `/finance/payments/${id}/confirm`,
      method: 'POST',
    })
  }

  /**
   * Cancel a payment voucher
   */
  const postFinancePaymentsIdCancel = (id: string, request: CancelVoucherRequest) => {
    return customInstance<ApiResponse<PaymentVoucher>>({
      url: `/finance/payments/${id}/cancel`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Reconcile a payment voucher
   */
  const postFinancePaymentsIdReconcile = (id: string, request: ReconcileRequest) => {
    return customInstance<ApiResponse<ReconcilePaymentResult>>({
      url: `/finance/payments/${id}/reconcile`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  // ===================== Expense Record APIs =====================

  /**
   * List expense records with filtering
   */
  const getFinanceExpenses = (params?: GetExpenseRecordsParams) => {
    return customInstance<ApiResponse<ExpenseRecord[]>>({
      url: '/finance/expenses',
      method: 'GET',
      params,
    })
  }

  /**
   * Get expense record by ID
   */
  const getFinanceExpensesId = (id: string) => {
    return customInstance<ApiResponse<ExpenseRecord>>({
      url: `/finance/expenses/${id}`,
      method: 'GET',
    })
  }

  /**
   * Create a new expense record
   */
  const postFinanceExpenses = (request: CreateExpenseRecordRequest) => {
    return customInstance<ApiResponse<ExpenseRecord>>({
      url: '/finance/expenses',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Update an expense record
   */
  const putFinanceExpensesId = (id: string, request: UpdateExpenseRecordRequest) => {
    return customInstance<ApiResponse<ExpenseRecord>>({
      url: `/finance/expenses/${id}`,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Delete an expense record
   */
  const deleteFinanceExpensesId = (id: string) => {
    return customInstance<ApiResponse<void>>({
      url: `/finance/expenses/${id}`,
      method: 'DELETE',
    })
  }

  /**
   * Submit an expense record for approval
   */
  const postFinanceExpensesIdSubmit = (id: string) => {
    return customInstance<ApiResponse<ExpenseRecord>>({
      url: `/finance/expenses/${id}/submit`,
      method: 'POST',
    })
  }

  /**
   * Approve an expense record
   */
  const postFinanceExpensesIdApprove = (id: string, request?: ApproveExpenseRequest) => {
    return customInstance<ApiResponse<ExpenseRecord>>({
      url: `/finance/expenses/${id}/approve`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Reject an expense record
   */
  const postFinanceExpensesIdReject = (id: string, request: RejectExpenseRequest) => {
    return customInstance<ApiResponse<ExpenseRecord>>({
      url: `/finance/expenses/${id}/reject`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Cancel an expense record
   */
  const postFinanceExpensesIdCancel = (id: string, request: CancelExpenseRequest) => {
    return customInstance<ApiResponse<ExpenseRecord>>({
      url: `/finance/expenses/${id}/cancel`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Mark an expense as paid
   */
  const postFinanceExpensesIdPay = (id: string, request: MarkExpensePaidRequest) => {
    return customInstance<ApiResponse<ExpenseRecord>>({
      url: `/finance/expenses/${id}/pay`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Get expense summary
   */
  const getFinanceExpensesSummary = (params?: { from_date?: string; to_date?: string }) => {
    return customInstance<ApiResponse<ExpenseSummary>>({
      url: '/finance/expenses/summary',
      method: 'GET',
      params,
    })
  }

  // ===================== Other Income Record APIs =====================

  /**
   * List other income records with filtering
   */
  const getFinanceIncomes = (params?: GetOtherIncomeRecordsParams) => {
    return customInstance<ApiResponse<OtherIncomeRecord[]>>({
      url: '/finance/incomes',
      method: 'GET',
      params,
    })
  }

  /**
   * Get other income record by ID
   */
  const getFinanceIncomesId = (id: string) => {
    return customInstance<ApiResponse<OtherIncomeRecord>>({
      url: `/finance/incomes/${id}`,
      method: 'GET',
    })
  }

  /**
   * Create a new other income record
   */
  const postFinanceIncomes = (request: CreateOtherIncomeRecordRequest) => {
    return customInstance<ApiResponse<OtherIncomeRecord>>({
      url: '/finance/incomes',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Update an other income record
   */
  const putFinanceIncomesId = (id: string, request: UpdateOtherIncomeRecordRequest) => {
    return customInstance<ApiResponse<OtherIncomeRecord>>({
      url: `/finance/incomes/${id}`,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Delete an other income record
   */
  const deleteFinanceIncomesId = (id: string) => {
    return customInstance<ApiResponse<void>>({
      url: `/finance/incomes/${id}`,
      method: 'DELETE',
    })
  }

  /**
   * Confirm an other income record
   */
  const postFinanceIncomesIdConfirm = (id: string) => {
    return customInstance<ApiResponse<OtherIncomeRecord>>({
      url: `/finance/incomes/${id}/confirm`,
      method: 'POST',
    })
  }

  /**
   * Cancel an other income record
   */
  const postFinanceIncomesIdCancel = (id: string, request: CancelIncomeRequest) => {
    return customInstance<ApiResponse<OtherIncomeRecord>>({
      url: `/finance/incomes/${id}/cancel`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Mark an income as received
   */
  const postFinanceIncomesIdReceive = (id: string, request: MarkIncomeReceivedRequest) => {
    return customInstance<ApiResponse<OtherIncomeRecord>>({
      url: `/finance/incomes/${id}/receive`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Get income summary
   */
  const getFinanceIncomesSummary = (params?: { from_date?: string; to_date?: string }) => {
    return customInstance<ApiResponse<IncomeSummary>>({
      url: '/finance/incomes/summary',
      method: 'GET',
      params,
    })
  }

  // ===================== Cash Flow APIs =====================

  /**
   * Get cash flow summary
   */
  const getFinanceCashFlow = (params?: {
    from_date?: string
    to_date?: string
    include_items?: boolean
  }) => {
    return customInstance<ApiResponse<CashFlowSummary>>({
      url: '/finance/cash-flow',
      method: 'GET',
      params,
    })
  }

  return {
    // Account Receivable
    getFinanceReceivables,
    getFinanceReceivablesId,
    getFinanceReceivablesSummary,
    // Account Payable
    getFinancePayables,
    getFinancePayablesId,
    getFinancePayablesSummary,
    // Receipt Voucher
    getFinanceReceipts,
    getFinanceReceiptsId,
    postFinanceReceipts,
    postFinanceReceiptsIdConfirm,
    postFinanceReceiptsIdCancel,
    postFinanceReceiptsIdReconcile,
    // Payment Voucher
    getFinancePayments,
    getFinancePaymentsId,
    postFinancePayments,
    postFinancePaymentsIdConfirm,
    postFinancePaymentsIdCancel,
    postFinancePaymentsIdReconcile,
    // Expense Record
    getFinanceExpenses,
    getFinanceExpensesId,
    postFinanceExpenses,
    putFinanceExpensesId,
    deleteFinanceExpensesId,
    postFinanceExpensesIdSubmit,
    postFinanceExpensesIdApprove,
    postFinanceExpensesIdReject,
    postFinanceExpensesIdCancel,
    postFinanceExpensesIdPay,
    getFinanceExpensesSummary,
    // Other Income Record
    getFinanceIncomes,
    getFinanceIncomesId,
    postFinanceIncomes,
    putFinanceIncomesId,
    deleteFinanceIncomesId,
    postFinanceIncomesIdConfirm,
    postFinanceIncomesIdCancel,
    postFinanceIncomesIdReceive,
    getFinanceIncomesSummary,
    // Cash Flow
    getFinanceCashFlow,
  }
}

// Export types for external use
export * from './types'
