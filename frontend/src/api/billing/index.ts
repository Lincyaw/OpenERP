/**
 * Billing API exports
 */
export * from './billing'

// Re-export types from models with simplified names for backward compatibility
export type {
  HandlerSubscriptionQuotaResponse as SubscriptionQuota,
  HandlerSubscriptionPlanResponse as SubscriptionPlan,
  HandlerCurrentSubscriptionResponse as CurrentSubscriptionResponse,
} from '../models'

// Note: The following types don't exist in the current API:
// - Invoice, InvoiceStatus, PaymentMethod, GetBillingHistoryParams
// - useGetBillingHistory, useGetBillingSummary, useSetDefaultPaymentMethod
// - useDeletePaymentMethod, downloadInvoicePdf
// These appear to be from a future billing history feature that hasn't been implemented yet.
