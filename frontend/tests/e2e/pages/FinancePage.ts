import { type Page, type Locator, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * FinancePage - Page Object for Finance module E2E tests
 *
 * Covers:
 * - Accounts Receivable list and operations
 * - Accounts Payable list and operations
 * - Receipt Vouchers (customer payments)
 * - Payment Vouchers (supplier payments)
 * - Reconciliation operations
 */
export class FinancePage extends BasePage {
  // Receivables page locators
  readonly receivablesTitle: Locator
  readonly receivablesTable: Locator
  readonly receivablesSearchInput: Locator
  readonly statusFilter: Locator
  readonly sourceTypeFilter: Locator
  readonly overdueFilter: Locator
  readonly dateRangePicker: Locator

  // Payables page locators
  readonly payablesTitle: Locator
  readonly payablesTable: Locator
  readonly payablesSearchInput: Locator

  // Receipt voucher form locators
  readonly receiptVoucherTitle: Locator
  readonly customerSelect: Locator
  readonly amountInput: Locator
  readonly paymentMethodSelect: Locator
  readonly receiptDatePicker: Locator
  readonly paymentReferenceInput: Locator
  readonly remarkInput: Locator

  // Reconcile page locators
  readonly reconcileTitle: Locator
  readonly fifoModeButton: Locator
  readonly manualModeButton: Locator
  readonly reconcileTable: Locator
  readonly confirmReconcileButton: Locator

  // Summary card locators
  readonly totalOutstandingCard: Locator
  readonly totalOverdueCard: Locator
  readonly pendingCountCard: Locator

  constructor(page: Page) {
    super(page)

    // Receivables locators
    this.receivablesTitle = page.locator('h4').filter({ hasText: '应收账款' })
    this.receivablesTable = page.locator('.semi-table')
    this.receivablesSearchInput = page.locator(
      'input[placeholder*="搜索"], input[placeholder*="单据编号"]'
    )
    this.statusFilter = page.locator('.semi-select').filter({ hasText: /状态/ }).first()
    this.sourceTypeFilter = page.locator('.semi-select').filter({ hasText: /来源/ }).first()
    this.overdueFilter = page.locator('.semi-select').filter({ hasText: /逾期/ }).first()
    this.dateRangePicker = page.locator('.semi-datepicker')

    // Payables locators
    this.payablesTitle = page.locator('h4').filter({ hasText: '应付账款' })
    this.payablesTable = page.locator('.semi-table')
    this.payablesSearchInput = page.locator(
      'input[placeholder*="搜索"], input[placeholder*="单据编号"]'
    )

    // Receipt voucher form locators
    this.receiptVoucherTitle = page.locator('h4').filter({ hasText: '新增收款单' })
    this.customerSelect = page.locator('.customer-select-wrapper .semi-select')
    this.amountInput = page.locator('input[placeholder*="收款金额"]')
    this.paymentMethodSelect = page.locator('.semi-select').filter({ hasText: /收款方式/ })
    this.receiptDatePicker = page.locator('.semi-datepicker').first()
    this.paymentReferenceInput = page.locator('input[placeholder*="交易流水号"]')
    this.remarkInput = page.locator('textarea[placeholder*="备注"]')

    // Reconcile page locators
    this.reconcileTitle = page.locator('h4').filter({ hasText: '收款核销' })
    this.fifoModeButton = page.getByRole('button', { name: /自动核销|FIFO/ })
    this.manualModeButton = page.getByRole('button', { name: /手动核销/ })
    this.reconcileTable = page.locator('.receivables-table, .semi-table')
    this.confirmReconcileButton = page.getByRole('button', { name: '确认核销' })

    // Summary cards
    this.totalOutstandingCard = page.locator('.summary-item').filter({ hasText: '待收总额' })
    this.totalOverdueCard = page.locator('.summary-item').filter({ hasText: '逾期总额' })
    this.pendingCountCard = page.locator('.summary-item').filter({ hasText: '待收款单' })
  }

  // =========== Navigation ===========

  async navigateToReceivables(): Promise<void> {
    await this.goto('/finance/receivables')
    await this.waitForPageLoad()
    await this.page.waitForSelector('.semi-table', { timeout: 10000 })
  }

  async navigateToPayables(): Promise<void> {
    await this.goto('/finance/payables')
    await this.waitForPageLoad()
    await this.page.waitForSelector('.semi-table', { timeout: 10000 })
  }

  async navigateToNewReceiptVoucher(customerId?: string): Promise<void> {
    const url = customerId
      ? `/finance/receipts/new?customer_id=${customerId}`
      : '/finance/receipts/new'
    await this.goto(url)
    await this.waitForPageLoad()
  }

  async navigateToNewPaymentVoucher(supplierId?: string): Promise<void> {
    const url = supplierId
      ? `/finance/payments/new?supplier_id=${supplierId}`
      : '/finance/payments/new'
    await this.goto(url)
    await this.waitForPageLoad()
  }

  async navigateToReceiptReconcile(voucherId: string): Promise<void> {
    await this.goto(`/finance/receipts/${voucherId}/reconcile`)
    await this.waitForPageLoad()
  }

  async navigateToPaymentReconcile(voucherId: string): Promise<void> {
    await this.goto(`/finance/payments/${voucherId}/reconcile`)
    await this.waitForPageLoad()
  }

  // =========== Receivables Page ===========

  async getReceivableCount(): Promise<number> {
    await this.waitForTableLoad()
    const rows = this.page.locator('.semi-table-tbody .semi-table-row')
    return rows.count()
  }

  async searchReceivables(keyword: string): Promise<void> {
    await this.receivablesSearchInput.fill(keyword)
    await this.page.keyboard.press('Enter')
    await this.waitForTableLoad()
  }

  async filterByStatus(
    status: 'pending' | 'partial' | 'paid' | 'reversed' | 'cancelled' | ''
  ): Promise<void> {
    await this.page
      .locator('.semi-select')
      .filter({ hasText: /状态筛选|全部状态/ })
      .click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const statusMap: Record<string, string> = {
      pending: '待收款',
      partial: '部分收款',
      paid: '已收款',
      reversed: '已冲红',
      cancelled: '已取消',
      '': '全部状态',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: statusMap[status] }).click()
    await this.waitForTableLoad()
  }

  async filterBySourceType(
    sourceType: 'sales_order' | 'sales_return' | 'manual' | ''
  ): Promise<void> {
    await this.page
      .locator('.semi-select')
      .filter({ hasText: /来源筛选|全部来源/ })
      .click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const sourceMap: Record<string, string> = {
      sales_order: '销售订单',
      sales_return: '销售退货',
      manual: '手工录入',
      '': '全部来源',
    }
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: sourceMap[sourceType] })
      .click()
    await this.waitForTableLoad()
  }

  async clickCollectButton(receivableNumber: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: receivableNumber })
    await row.locator('button, .semi-button').filter({ hasText: '收款' }).click()
  }

  async getReceivableRowData(rowIndex: number): Promise<{
    number: string
    customerName: string
    totalAmount: string
    paidAmount: string
    outstandingAmount: string
    status: string
  }> {
    const row = this.page.locator('.semi-table-tbody .semi-table-row').nth(rowIndex)
    const cells = row.locator('.semi-table-row-cell')

    return {
      number: (await cells.nth(0).textContent()) || '',
      customerName: (await cells.nth(1).textContent()) || '',
      totalAmount: (await cells.nth(3).textContent()) || '',
      paidAmount: (await cells.nth(4).textContent()) || '',
      outstandingAmount: (await cells.nth(5).textContent()) || '',
      status: (await cells.nth(7).textContent()) || '',
    }
  }

  async getSummaryValues(): Promise<{
    totalOutstanding: string
    totalOverdue: string
    pendingCount: string
  }> {
    const totalOutstanding =
      (await this.page
        .locator('.summary-item')
        .filter({ hasText: '待收总额' })
        .locator('.summary-value')
        .textContent()) || '0'
    const totalOverdue =
      (await this.page
        .locator('.summary-item')
        .filter({ hasText: '逾期总额' })
        .locator('.summary-value')
        .textContent()) || '0'
    const pendingCount =
      (await this.page
        .locator('.summary-item')
        .filter({ hasText: '待收款单' })
        .locator('.summary-value')
        .textContent()) || '0'

    return { totalOutstanding, totalOverdue, pendingCount }
  }

  // =========== Payables Page ===========

  async getPayableCount(): Promise<number> {
    await this.waitForTableLoad()
    const rows = this.page.locator('.semi-table-tbody .semi-table-row')
    return rows.count()
  }

  async searchPayables(keyword: string): Promise<void> {
    await this.payablesSearchInput.fill(keyword)
    await this.page.keyboard.press('Enter')
    await this.waitForTableLoad()
  }

  async filterPayablesByStatus(
    status: 'pending' | 'partial' | 'paid' | 'reversed' | 'cancelled' | ''
  ): Promise<void> {
    await this.page
      .locator('.semi-select')
      .filter({ hasText: /状态筛选|全部状态/ })
      .click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const statusMap: Record<string, string> = {
      pending: '待付款',
      partial: '部分付款',
      paid: '已付款',
      reversed: '已冲红',
      cancelled: '已取消',
      '': '全部状态',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: statusMap[status] }).click()
    await this.waitForTableLoad()
  }

  async clickPayButton(payableNumber: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: payableNumber })
    await row.locator('button, .semi-button').filter({ hasText: '付款' }).click()
  }

  async getPayableRowData(rowIndex: number): Promise<{
    number: string
    supplierName: string
    totalAmount: string
    paidAmount: string
    outstandingAmount: string
    status: string
  }> {
    const row = this.page.locator('.semi-table-tbody .semi-table-row').nth(rowIndex)
    const cells = row.locator('.semi-table-row-cell')

    return {
      number: (await cells.nth(0).textContent()) || '',
      supplierName: (await cells.nth(1).textContent()) || '',
      totalAmount: (await cells.nth(3).textContent()) || '',
      paidAmount: (await cells.nth(4).textContent()) || '',
      outstandingAmount: (await cells.nth(5).textContent()) || '',
      status: (await cells.nth(7).textContent()) || '',
    }
  }

  // =========== Receipt Voucher Creation ===========

  async selectCustomer(customerName: string): Promise<void> {
    // Click on customer select and search
    const customerSelect = this.page
      .locator('.customer-select-wrapper .semi-select, .semi-select')
      .filter({ hasText: /客户/ })
      .first()
    await customerSelect.click()
    await this.page.locator('.semi-select input').fill(customerName)
    await this.page.waitForTimeout(500) // Wait for search results
    await this.page.locator('.semi-select-option').filter({ hasText: customerName }).first().click()
  }

  async fillReceiptAmount(amount: number): Promise<void> {
    const amountInput = this.page.locator('.semi-input-number input').first()
    await amountInput.fill(amount.toString())
  }

  async selectPaymentMethod(
    method: 'cash' | 'bank_transfer' | 'wechat' | 'alipay' | 'check' | 'balance' | 'other'
  ): Promise<void> {
    const methodSelect = this.page
      .locator('label')
      .filter({ hasText: '收款方式' })
      .locator('..')
      .locator('.semi-select')
    await methodSelect.click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const methodMap: Record<string, string> = {
      cash: '现金',
      bank_transfer: '银行转账',
      wechat: '微信支付',
      alipay: '支付宝',
      check: '支票',
      balance: '余额抵扣',
      other: '其他',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: methodMap[method] }).click()
  }

  async fillPaymentReference(reference: string): Promise<void> {
    const referenceInput = this.page.locator('input[placeholder*="交易流水号"]')
    await referenceInput.fill(reference)
  }

  async fillReceiptRemark(remark: string): Promise<void> {
    const remarkInput = this.page.locator('textarea[placeholder*="备注"]')
    await remarkInput.fill(remark)
  }

  async submitReceiptVoucher(): Promise<void> {
    const submitButton = this.page.getByRole('button', { name: '创建' })
    await submitButton.click()
    await this.page.waitForResponse(
      (response) =>
        response.url().includes('/finance/receipts') && response.request().method() === 'POST'
    )
  }

  // =========== Payment Voucher Creation ===========

  async selectSupplier(supplierName: string): Promise<void> {
    const supplierSelect = this.page
      .locator('.supplier-select-wrapper .semi-select, .semi-select')
      .filter({ hasText: /供应商/ })
      .first()
    await supplierSelect.click()
    await this.page.locator('.semi-select input').fill(supplierName)
    await this.page.waitForTimeout(500)
    await this.page.locator('.semi-select-option').filter({ hasText: supplierName }).first().click()
  }

  async fillPaymentAmount(amount: number): Promise<void> {
    const amountInput = this.page.locator('.semi-input-number input').first()
    await amountInput.fill(amount.toString())
  }

  async selectPaymentVoucherMethod(
    method: 'cash' | 'bank_transfer' | 'wechat' | 'alipay' | 'check' | 'other'
  ): Promise<void> {
    const methodSelect = this.page
      .locator('label')
      .filter({ hasText: '付款方式' })
      .locator('..')
      .locator('.semi-select')
    await methodSelect.click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const methodMap: Record<string, string> = {
      cash: '现金',
      bank_transfer: '银行转账',
      wechat: '微信支付',
      alipay: '支付宝',
      check: '支票',
      other: '其他',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: methodMap[method] }).click()
  }

  async submitPaymentVoucher(): Promise<void> {
    const submitButton = this.page.getByRole('button', { name: '创建' })
    await submitButton.click()
    await this.page.waitForResponse(
      (response) =>
        response.url().includes('/finance/payments') && response.request().method() === 'POST'
    )
  }

  // =========== Reconciliation ===========

  async selectFIFOMode(): Promise<void> {
    await this.fifoModeButton.click()
  }

  async selectManualMode(): Promise<void> {
    await this.manualModeButton.click()
  }

  async selectReceivableForReconcile(receivableNumber: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: receivableNumber })
    await row.locator('.semi-checkbox').click()
  }

  async setManualReconcileAmount(receivableNumber: string, amount: number): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: receivableNumber })
    const amountInput = row.locator('.semi-input-number input')
    await amountInput.fill(amount.toString())
  }

  async getReconcilePreviewAmount(): Promise<string> {
    const previewAmount = this.page
      .locator('.reconcile-summary .summary-item')
      .filter({ hasText: /预计核销|已选核销/ })
      .locator('text=¥')
    return (await previewAmount.textContent()) || '0'
  }

  async confirmReconcile(): Promise<void> {
    await this.confirmReconcileButton.click()
    await this.page.waitForResponse(
      (response) => response.url().includes('/reconcile') && response.request().method() === 'POST'
    )
  }

  async getReconcileResult(): Promise<{
    success: boolean
    totalReconciled: string
    remainingUnallocated: string
  }> {
    // Wait for result page
    await this.page.waitForSelector('.reconcile-result-card, .semi-banner', { timeout: 5000 })

    const hasBanner = await this.page.locator('.semi-banner-success, .semi-banner-info').isVisible()
    const totalReconciled =
      (await this.page
        .locator('.result-summary')
        .locator('text=本次核销')
        .locator('..')
        .textContent()) || '0'
    const remainingUnallocated =
      (await this.page
        .locator('.result-summary')
        .locator('text=剩余未核销')
        .locator('..')
        .textContent()) || '0'

    return {
      success: hasBanner,
      totalReconciled,
      remainingUnallocated,
    }
  }

  // =========== Assertions ===========

  async assertReceivablesPageLoaded(): Promise<void> {
    await expect(this.receivablesTitle).toBeVisible()
    await expect(this.receivablesTable).toBeVisible()
  }

  async assertPayablesPageLoaded(): Promise<void> {
    await expect(this.payablesTitle).toBeVisible()
    await expect(this.payablesTable).toBeVisible()
  }

  async assertReceiptVoucherFormLoaded(): Promise<void> {
    await expect(this.receiptVoucherTitle).toBeVisible()
  }

  async assertReconcilePageLoaded(): Promise<void> {
    await expect(this.reconcileTitle).toBeVisible()
  }

  async assertReceivableExists(receivableNumber: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: receivableNumber })
    await expect(row).toBeVisible()
  }

  async assertPayableExists(payableNumber: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: payableNumber })
    await expect(row).toBeVisible()
  }

  async assertReceivableStatus(receivableNumber: string, status: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: receivableNumber })
    const statusTag = row.locator('.semi-tag')
    await expect(statusTag).toContainText(status)
  }

  async assertPayableStatus(payableNumber: string, status: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: payableNumber })
    const statusTag = row.locator('.semi-tag')
    await expect(statusTag).toContainText(status)
  }

  // =========== Screenshots ===========

  async takeReceivablesScreenshot(name: string = 'receivables'): Promise<void> {
    await this.screenshot(`finance/${name}`)
  }

  async takePayablesScreenshot(name: string = 'payables'): Promise<void> {
    await this.screenshot(`finance/${name}`)
  }

  async takeReceiptVoucherScreenshot(name: string = 'receipt-voucher'): Promise<void> {
    await this.screenshot(`finance/${name}`)
  }

  async takeReconcileScreenshot(name: string = 'reconcile'): Promise<void> {
    await this.screenshot(`finance/${name}`)
  }
}
