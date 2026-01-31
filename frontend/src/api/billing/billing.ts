/**
 * Billing API hooks
 *
 * Manual API hooks for tenant billing and invoice endpoints.
 * These endpoints are not yet in the auto-generated swagger docs.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type {
  QueryFunction,
  UseQueryOptions,
  UseQueryResult,
  UseMutationOptions,
  UseMutationResult,
} from '@tanstack/react-query'

import { customInstance } from '../../services/axios-instance'

// ============================================================================
// Types
// ============================================================================

/**
 * Invoice status
 */
export type InvoiceStatus = 'paid' | 'pending' | 'overdue' | 'cancelled' | 'refunded'

/**
 * Payment method type
 */
export type PaymentMethodType = 'card' | 'bank_transfer' | 'alipay' | 'wechat'

/**
 * Invoice item in billing history
 */
export interface Invoice {
  id: string
  invoice_number: string
  tenant_id: string
  amount: number
  currency: string
  status: InvoiceStatus
  description: string
  period_start: string
  period_end: string
  due_date: string
  paid_at?: string
  pdf_url?: string
  created_at: string
}

/**
 * Billing history response with pagination
 */
export interface BillingHistoryResponse {
  invoices: Invoice[]
  total: number
  page: number
  page_size: number
  has_more: boolean
}

/**
 * Payment method
 */
export interface PaymentMethod {
  id: string
  type: PaymentMethodType
  is_default: boolean
  last_four?: string
  brand?: string
  exp_month?: number
  exp_year?: number
  bank_name?: string
  account_last_four?: string
  created_at: string
}

/**
 * Payment methods response
 */
export interface PaymentMethodsResponse {
  payment_methods: PaymentMethod[]
  default_method_id?: string
}

/**
 * Upcoming billing information
 */
export interface UpcomingBilling {
  next_billing_date: string
  amount: number
  currency: string
  plan_name: string
  billing_cycle: 'monthly' | 'yearly'
  auto_renew: boolean
}

/**
 * Billing summary with upcoming and payment methods
 */
export interface BillingSummaryResponse {
  tenant_id: string
  tenant_name: string
  plan: string
  upcoming: UpcomingBilling | null
  payment_methods: PaymentMethod[]
  default_payment_method_id?: string
}

/**
 * API response wrapper
 */
export interface APIResponse<T> {
  success: boolean
  data: T
  error?: string
}

/**
 * Error response
 */
export interface ErrorResponse {
  success: boolean
  error: string
  message?: string
}

// ============================================================================
// Query parameters
// ============================================================================

export interface GetBillingHistoryParams {
  page?: number
  page_size?: number
  status?: InvoiceStatus
  start_date?: string
  end_date?: string
}

export interface SetDefaultPaymentMethodParams {
  payment_method_id: string
}

export interface DeletePaymentMethodParams {
  payment_method_id: string
}

// ============================================================================
// Response types
// ============================================================================

export type GetBillingHistoryResponse200 = {
  data: APIResponse<BillingHistoryResponse>
  status: 200
}

export type GetBillingHistoryResponse401 = {
  data: ErrorResponse
  status: 401
}

export type GetBillingHistoryResponse404 = {
  data: ErrorResponse
  status: 404
}

export type GetBillingHistoryResponse500 = {
  data: ErrorResponse
  status: 500
}

export type GetBillingHistoryResponseSuccess = GetBillingHistoryResponse200 & {
  headers: Headers
}

export type GetBillingHistoryResponseError = (
  | GetBillingHistoryResponse401
  | GetBillingHistoryResponse404
  | GetBillingHistoryResponse500
) & {
  headers: Headers
}

export type GetBillingHistoryResponse =
  | GetBillingHistoryResponseSuccess
  | GetBillingHistoryResponseError

export type GetBillingSummaryResponse200 = {
  data: APIResponse<BillingSummaryResponse>
  status: 200
}

export type GetBillingSummaryResponseSuccess = GetBillingSummaryResponse200 & {
  headers: Headers
}

export type GetBillingSummaryResponse =
  | GetBillingSummaryResponseSuccess
  | GetBillingHistoryResponseError

export type GetInvoicePdfResponse200 = {
  data: Blob
  status: 200
}

export type GetInvoicePdfResponseSuccess = GetInvoicePdfResponse200 & {
  headers: Headers
}

export type GetInvoicePdfResponse = GetInvoicePdfResponseSuccess | GetBillingHistoryResponseError

// ============================================================================
// API Functions
// ============================================================================

/**
 * Get billing history with pagination
 */
export const getBillingHistoryUrl = (params?: GetBillingHistoryParams) => {
  const normalizedParams = new URLSearchParams()

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : value.toString())
    }
  })

  const stringifiedParams = normalizedParams.toString()

  return stringifiedParams.length > 0
    ? `/identity/tenants/current/billing/history?${stringifiedParams}`
    : `/identity/tenants/current/billing/history`
}

export const getBillingHistory = async (
  params?: GetBillingHistoryParams,
  options?: RequestInit
): Promise<GetBillingHistoryResponse> => {
  return customInstance<GetBillingHistoryResponse>(getBillingHistoryUrl(params), {
    ...options,
    method: 'GET',
  })
}

/**
 * Get billing summary (upcoming billing + payment methods)
 */
export const getBillingSummaryUrl = () => `/identity/tenants/current/billing/summary`

export const getBillingSummary = async (
  options?: RequestInit
): Promise<GetBillingSummaryResponse> => {
  return customInstance<GetBillingSummaryResponse>(getBillingSummaryUrl(), {
    ...options,
    method: 'GET',
  })
}

/**
 * Download invoice PDF
 */
export const getInvoicePdfUrl = (invoiceId: string) =>
  `/identity/tenants/current/billing/invoices/${invoiceId}/pdf`

export const getInvoicePdf = async (
  invoiceId: string,
  options?: RequestInit
): Promise<GetInvoicePdfResponse> => {
  return customInstance<GetInvoicePdfResponse>(getInvoicePdfUrl(invoiceId), {
    ...options,
    method: 'GET',
    headers: {
      Accept: 'application/pdf',
    },
  })
}

/**
 * Set default payment method
 */
export const setDefaultPaymentMethodUrl = () =>
  `/identity/tenants/current/billing/payment-methods/default`

export const setDefaultPaymentMethod = async (
  params: SetDefaultPaymentMethodParams,
  options?: RequestInit
): Promise<GetBillingSummaryResponse> => {
  return customInstance<GetBillingSummaryResponse>(setDefaultPaymentMethodUrl(), {
    ...options,
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(params),
  })
}

/**
 * Delete payment method
 */
export const deletePaymentMethodUrl = (paymentMethodId: string) =>
  `/identity/tenants/current/billing/payment-methods/${paymentMethodId}`

export const deletePaymentMethod = async (
  paymentMethodId: string,
  options?: RequestInit
): Promise<{ data: APIResponse<{ success: boolean }>; status: number }> => {
  return customInstance(deletePaymentMethodUrl(paymentMethodId), {
    ...options,
    method: 'DELETE',
  })
}

// ============================================================================
// React Query Hooks
// ============================================================================

export const getGetBillingHistoryQueryKey = (params?: GetBillingHistoryParams) => {
  return ['getBillingHistory', ...(params ? [params] : [])] as const
}

export type GetBillingHistoryQueryResult = NonNullable<
  Awaited<ReturnType<typeof getBillingHistory>>
>
export type GetBillingHistoryQueryError = GetBillingHistoryResponseError

/**
 * Hook to get billing history
 */
export const useGetBillingHistory = <
  TData = Awaited<ReturnType<typeof getBillingHistory>>,
  TError = GetBillingHistoryResponseError,
>(
  params?: GetBillingHistoryParams,
  options?: {
    query?: Partial<UseQueryOptions<Awaited<ReturnType<typeof getBillingHistory>>, TError, TData>>
  }
): UseQueryResult<TData, TError> => {
  const { query: queryOptions } = options ?? {}

  const queryKey = queryOptions?.queryKey ?? getGetBillingHistoryQueryKey(params)

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getBillingHistory>>> = ({ signal }) =>
    getBillingHistory(params, { signal })

  return useQuery({
    queryKey,
    queryFn,
    ...queryOptions,
  })
}

export const getGetBillingSummaryQueryKey = () => {
  return ['getBillingSummary'] as const
}

export type GetBillingSummaryQueryResult = NonNullable<
  Awaited<ReturnType<typeof getBillingSummary>>
>
export type GetBillingSummaryQueryError = GetBillingHistoryResponseError

/**
 * Hook to get billing summary
 */
export const useGetBillingSummary = <
  TData = Awaited<ReturnType<typeof getBillingSummary>>,
  TError = GetBillingHistoryResponseError,
>(options?: {
  query?: Partial<UseQueryOptions<Awaited<ReturnType<typeof getBillingSummary>>, TError, TData>>
}): UseQueryResult<TData, TError> => {
  const { query: queryOptions } = options ?? {}

  const queryKey = queryOptions?.queryKey ?? getGetBillingSummaryQueryKey()

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getBillingSummary>>> = ({ signal }) =>
    getBillingSummary({ signal })

  return useQuery({
    queryKey,
    queryFn,
    ...queryOptions,
  })
}

/**
 * Hook to set default payment method
 */
export const useSetDefaultPaymentMethod = <TError = GetBillingHistoryResponseError>(options?: {
  mutation?: UseMutationOptions<
    Awaited<ReturnType<typeof setDefaultPaymentMethod>>,
    TError,
    SetDefaultPaymentMethodParams
  >
}): UseMutationResult<
  Awaited<ReturnType<typeof setDefaultPaymentMethod>>,
  TError,
  SetDefaultPaymentMethodParams
> => {
  const { mutation: mutationOptions } = options ?? {}
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (params: SetDefaultPaymentMethodParams) => setDefaultPaymentMethod(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getGetBillingSummaryQueryKey() })
    },
    ...mutationOptions,
  })
}

/**
 * Hook to delete payment method
 */
export const useDeletePaymentMethod = <TError = GetBillingHistoryResponseError>(options?: {
  mutation?: UseMutationOptions<Awaited<ReturnType<typeof deletePaymentMethod>>, TError, string>
}): UseMutationResult<Awaited<ReturnType<typeof deletePaymentMethod>>, TError, string> => {
  const { mutation: mutationOptions } = options ?? {}
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (paymentMethodId: string) => deletePaymentMethod(paymentMethodId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getGetBillingSummaryQueryKey() })
    },
    ...mutationOptions,
  })
}

/**
 * Utility function to download invoice PDF
 */
export const downloadInvoicePdf = async (invoiceId: string, invoiceNumber: string) => {
  try {
    const response = await getInvoicePdf(invoiceId)
    if (response.status === 200) {
      const blob = response.data as unknown as Blob
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `invoice-${invoiceNumber}.pdf`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
      return { success: true }
    }
    return { success: false, error: 'Failed to download invoice' }
  } catch (error) {
    return { success: false, error: String(error) }
  }
}
