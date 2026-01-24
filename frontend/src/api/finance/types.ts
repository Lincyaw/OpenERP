/**
 * Finance API Types
 *
 * These types match the backend handler DTOs defined in:
 * backend/internal/interfaces/http/handler/finance.go
 *
 * TODO: Replace with auto-generated types once OpenAPI client is generated
 */

// ===================== Account Receivable Types =====================

export type AccountReceivableStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'REVERSED' | 'CANCELLED'

export type ReceivableSourceType = 'SALES_ORDER' | 'SALES_RETURN' | 'MANUAL'

export interface PaymentRecord {
  id: string
  receipt_voucher_id: string
  amount: number
  applied_at: string
  remark?: string
}

export interface AccountReceivable {
  id: string
  tenant_id: string
  receivable_number: string
  customer_id: string
  customer_name: string
  source_type: ReceivableSourceType
  source_id: string
  source_number: string
  total_amount: number
  paid_amount: number
  outstanding_amount: number
  status: AccountReceivableStatus
  due_date?: string
  payment_records?: PaymentRecord[]
  remark?: string
  paid_at?: string
  created_at: string
  updated_at: string
  version: number
}

export interface ReceivableSummary {
  total_outstanding: number
  total_overdue: number
  pending_count: number
  partial_count: number
  overdue_count: number
}

// ===================== Account Payable Types =====================

export type AccountPayableStatus = 'PENDING' | 'PARTIAL' | 'PAID' | 'REVERSED' | 'CANCELLED'

export type PayableSourceType = 'PURCHASE_ORDER' | 'PURCHASE_RETURN' | 'MANUAL'

export interface PayablePaymentRecord {
  id: string
  payment_voucher_id: string
  amount: number
  applied_at: string
  remark?: string
}

export interface AccountPayable {
  id: string
  tenant_id: string
  payable_number: string
  supplier_id: string
  supplier_name: string
  source_type: PayableSourceType
  source_id: string
  source_number: string
  total_amount: number
  paid_amount: number
  outstanding_amount: number
  status: AccountPayableStatus
  due_date?: string
  payment_records?: PayablePaymentRecord[]
  remark?: string
  paid_at?: string
  created_at: string
  updated_at: string
  version: number
}

export interface PayableSummary {
  total_outstanding: number
  total_overdue: number
  pending_count: number
  partial_count: number
  overdue_count: number
}

// ===================== Receipt Voucher Types =====================

export type ReceiptVoucherStatus = 'DRAFT' | 'CONFIRMED' | 'ALLOCATED' | 'CANCELLED'

export type PaymentMethod =
  | 'CASH'
  | 'BANK_TRANSFER'
  | 'WECHAT'
  | 'ALIPAY'
  | 'CHECK'
  | 'BALANCE'
  | 'OTHER'

export interface ReceivableAllocation {
  id: string
  receivable_id: string
  receivable_number: string
  amount: number
  allocated_at: string
  remark?: string
}

export interface ReceiptVoucher {
  id: string
  tenant_id: string
  voucher_number: string
  customer_id: string
  customer_name: string
  amount: number
  allocated_amount: number
  unallocated_amount: number
  payment_method: PaymentMethod
  payment_reference?: string
  status: ReceiptVoucherStatus
  receipt_date: string
  allocations?: ReceivableAllocation[]
  remark?: string
  confirmed_at?: string
  created_at: string
  updated_at: string
  version: number
}

// ===================== Payment Voucher Types =====================

export type PaymentVoucherStatus = 'DRAFT' | 'CONFIRMED' | 'ALLOCATED' | 'CANCELLED'

export interface PayableAllocation {
  id: string
  payable_id: string
  payable_number: string
  amount: number
  allocated_at: string
  remark?: string
}

export interface PaymentVoucher {
  id: string
  tenant_id: string
  voucher_number: string
  supplier_id: string
  supplier_name: string
  amount: number
  allocated_amount: number
  unallocated_amount: number
  payment_method: PaymentMethod
  payment_reference?: string
  status: PaymentVoucherStatus
  payment_date: string
  allocations?: PayableAllocation[]
  remark?: string
  confirmed_at?: string
  created_at: string
  updated_at: string
  version: number
}

// ===================== Request/Filter Types =====================

export interface GetReceivablesParams {
  search?: string
  customer_id?: string
  status?: AccountReceivableStatus
  source_type?: ReceivableSourceType
  from_date?: string
  to_date?: string
  overdue?: boolean
  page?: number
  page_size?: number
}

export interface GetPayablesParams {
  search?: string
  supplier_id?: string
  status?: AccountPayableStatus
  source_type?: PayableSourceType
  from_date?: string
  to_date?: string
  overdue?: boolean
  page?: number
  page_size?: number
}

export interface GetReceiptVouchersParams {
  search?: string
  customer_id?: string
  status?: ReceiptVoucherStatus
  payment_method?: PaymentMethod
  from_date?: string
  to_date?: string
  page?: number
  page_size?: number
}

export interface GetPaymentVouchersParams {
  search?: string
  supplier_id?: string
  status?: PaymentVoucherStatus
  payment_method?: PaymentMethod
  from_date?: string
  to_date?: string
  page?: number
  page_size?: number
}

export interface CreateReceiptVoucherRequest {
  customer_id: string
  customer_name: string
  amount: number
  payment_method: PaymentMethod
  payment_reference?: string
  receipt_date: string
  remark?: string
}

export interface CreatePaymentVoucherRequest {
  supplier_id: string
  supplier_name: string
  amount: number
  payment_method: PaymentMethod
  payment_reference?: string
  payment_date: string
  remark?: string
}

export interface CancelVoucherRequest {
  reason: string
}

export interface ManualAllocationInput {
  target_id: string
  amount: number
}

export interface ReconcileRequest {
  strategy_type: 'FIFO' | 'MANUAL'
  manual_allocations?: ManualAllocationInput[]
}

// ===================== Response Types =====================

export interface ApiMeta {
  page: number
  page_size: number
  total: number
  total_pages: number
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: string
  meta?: ApiMeta
}

export interface ReconcileReceiptResult {
  voucher: ReceiptVoucher
  updated_receivables: AccountReceivable[]
  total_reconciled: number
  remaining_unallocated: number
  fully_reconciled: boolean
}

export interface ReconcilePaymentResult {
  voucher: PaymentVoucher
  updated_payables: AccountPayable[]
  total_reconciled: number
  remaining_unallocated: number
  fully_reconciled: boolean
}

// ===================== Expense Record Types =====================

export type ExpenseCategory =
  | 'RENT'
  | 'UTILITIES'
  | 'SALARY'
  | 'OFFICE'
  | 'TRAVEL'
  | 'MARKETING'
  | 'EQUIPMENT'
  | 'MAINTENANCE'
  | 'INSURANCE'
  | 'TAX'
  | 'OTHER'

export type ExpenseStatus = 'DRAFT' | 'PENDING' | 'APPROVED' | 'REJECTED' | 'CANCELLED'

export type ExpensePaymentStatus = 'UNPAID' | 'PAID'

export interface ExpenseRecord {
  id: string
  tenant_id: string
  expense_number: string
  category: ExpenseCategory
  category_name: string
  amount: number
  description: string
  incurred_at: string
  status: ExpenseStatus
  payment_status: ExpensePaymentStatus
  payment_method?: PaymentMethod
  paid_at?: string
  remark?: string
  attachment_urls?: string
  submitted_at?: string
  submitted_by?: string
  approved_at?: string
  approved_by?: string
  approval_remark?: string
  rejected_at?: string
  rejected_by?: string
  rejection_reason?: string
  created_at: string
  updated_at: string
  version: number
}

export interface GetExpenseRecordsParams {
  search?: string
  category?: ExpenseCategory
  status?: ExpenseStatus
  payment_status?: ExpensePaymentStatus
  from_date?: string
  to_date?: string
  page?: number
  page_size?: number
}

export interface CreateExpenseRecordRequest {
  category: ExpenseCategory
  amount: number
  description: string
  incurred_at: string
  remark?: string
  attachment_urls?: string
}

export interface UpdateExpenseRecordRequest {
  category: ExpenseCategory
  amount: number
  description: string
  incurred_at: string
  remark?: string
  attachment_urls?: string
}

export interface ApproveExpenseRequest {
  remark?: string
}

export interface RejectExpenseRequest {
  reason: string
}

export interface CancelExpenseRequest {
  reason: string
}

export interface MarkExpensePaidRequest {
  payment_method: PaymentMethod
}

export interface ExpenseSummary {
  total_approved: number
  total_pending: number
  by_category: Record<string, number>
}

// ===================== Other Income Record Types =====================

export type IncomeCategory =
  | 'INVESTMENT'
  | 'SUBSIDY'
  | 'INTEREST'
  | 'RENTAL'
  | 'REFUND'
  | 'COMPENSATION'
  | 'ASSET_DISPOSAL'
  | 'OTHER'

export type IncomeStatus = 'DRAFT' | 'CONFIRMED' | 'CANCELLED'

export type IncomeReceiptStatus = 'PENDING' | 'RECEIVED'

export interface OtherIncomeRecord {
  id: string
  tenant_id: string
  income_number: string
  category: IncomeCategory
  category_name: string
  amount: number
  description: string
  received_at: string
  status: IncomeStatus
  receipt_status: IncomeReceiptStatus
  payment_method?: PaymentMethod
  actual_received?: string
  remark?: string
  attachment_urls?: string
  confirmed_at?: string
  confirmed_by?: string
  created_at: string
  updated_at: string
  version: number
}

export interface GetOtherIncomeRecordsParams {
  search?: string
  category?: IncomeCategory
  status?: IncomeStatus
  receipt_status?: IncomeReceiptStatus
  from_date?: string
  to_date?: string
  page?: number
  page_size?: number
}

export interface CreateOtherIncomeRecordRequest {
  category: IncomeCategory
  amount: number
  description: string
  received_at: string
  remark?: string
  attachment_urls?: string
}

export interface UpdateOtherIncomeRecordRequest {
  category: IncomeCategory
  amount: number
  description: string
  received_at: string
  remark?: string
  attachment_urls?: string
}

export interface CancelIncomeRequest {
  reason: string
}

export interface MarkIncomeReceivedRequest {
  payment_method: PaymentMethod
}

export interface IncomeSummary {
  total_confirmed: number
  total_draft: number
  by_category: Record<string, number>
}

// ===================== Cash Flow Types =====================

export type CashFlowDirection = 'INFLOW' | 'OUTFLOW'

export type CashFlowItemType = 'EXPENSE' | 'INCOME' | 'RECEIPT' | 'PAYMENT'

export interface CashFlowItem {
  id: string
  type: CashFlowItemType
  category: string
  number: string
  description: string
  amount: number
  date: string
  direction: CashFlowDirection
}

export interface CashFlowSummary {
  total_inflow: number
  total_outflow: number
  net_cash_flow: number
  expense_total: number
  income_total: number
  items?: CashFlowItem[]
}
