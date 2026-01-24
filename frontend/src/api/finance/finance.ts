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
  }
}

// Export types for external use
export * from './types'
