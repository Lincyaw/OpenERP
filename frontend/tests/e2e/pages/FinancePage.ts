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
    // Wait for table toolbar to render first, then table
    await this.page
      .locator('.table-toolbar')
      .waitFor({ state: 'visible', timeout: 15000 })
      .catch(() => {})
    await this.page
      .waitForSelector('.semi-table, .semi-table-empty', { timeout: 15000 })
      .catch(() => {})
  }

  async navigateToPayables(): Promise<void> {
    await this.goto('/finance/payables')
    await this.waitForPageLoad()
    // Wait for table toolbar to render first, then table
    await this.page
      .locator('.table-toolbar')
      .waitFor({ state: 'visible', timeout: 15000 })
      .catch(() => {})
    await this.page
      .waitForSelector('.semi-table, .semi-table-empty', { timeout: 15000 })
      .catch(() => {})
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
    // Semi Design Select with filter + remote creates a searchable input inside
    const customerSelectWrapper = this.page.locator('.customer-select-wrapper')
    const select = customerSelectWrapper.locator('.semi-select')

    // Click to open and focus the select
    await select.click()
    await this.page.waitForTimeout(300)

    // The filter-enabled select has an input inside for searching
    // Use getByRole('textbox') to find the search input
    const searchInput = this.page.getByRole('textbox').first()
    await searchInput.fill(customerName)
    await this.page.waitForTimeout(1000) // Wait for API search results

    // Wait for option to appear and click
    const option = this.page
      .locator('.semi-select-option')
      .filter({ hasText: customerName })
      .first()
    await option.waitFor({ state: 'visible', timeout: 8000 })
    await option.click()
    await this.page.waitForTimeout(300)
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
    // Semi Design Select with filter + remote creates a searchable input inside
    const supplierSelectWrapper = this.page.locator('.supplier-select-wrapper')
    const select = supplierSelectWrapper.locator('.semi-select')

    // Click to open and focus the select
    await select.click()
    await this.page.waitForTimeout(300)

    // The filter-enabled select has an input inside for searching
    const searchInput = this.page.getByRole('textbox').first()
    await searchInput.fill(supplierName)
    await this.page.waitForTimeout(1000) // Wait for API search results

    // Wait for option to appear and click
    const option = this.page
      .locator('.semi-select-option')
      .filter({ hasText: supplierName })
      .first()
    await option.waitFor({ state: 'visible', timeout: 8000 })
    await option.click()
    await this.page.waitForTimeout(300)
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

  // =========== Expenses Page ===========

  async navigateToExpenses(): Promise<void> {
    await this.goto('/finance/expenses')
    await this.waitForPageLoad()
    await this.page.waitForSelector('.semi-table', { timeout: 10000 })
  }

  async navigateToNewExpense(): Promise<void> {
    await this.goto('/finance/expenses/new')
    await this.waitForPageLoad()
  }

  async assertExpensesPageLoaded(): Promise<void> {
    const title = this.page.locator('h4').filter({ hasText: '费用管理' })
    await expect(title).toBeVisible()
    await expect(this.page.locator('.semi-table')).toBeVisible()
  }

  async getExpenseCount(): Promise<number> {
    await this.waitForTableLoad()
    const rows = this.page.locator('.semi-table-tbody .semi-table-row')
    return rows.count()
  }

  async assertExpenseExists(expenseNumber: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: expenseNumber })
    await expect(row).toBeVisible()
  }

  async assertExpenseStatus(expenseNumber: string, status: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: expenseNumber })
    const statusTag = row.locator('.semi-tag').first()
    await expect(statusTag).toContainText(status)
  }

  async filterExpensesByCategory(
    category: 'RENT' | 'UTILITIES' | 'SALARY' | 'OFFICE' | 'TRAVEL' | 'MARKETING' | 'OTHER' | ''
  ): Promise<void> {
    await this.page
      .locator('.semi-select')
      .filter({ hasText: /分类筛选|全部分类/ })
      .click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const categoryMap: Record<string, string> = {
      RENT: '房租',
      UTILITIES: '水电费',
      SALARY: '工资',
      OFFICE: '办公费',
      TRAVEL: '差旅费',
      MARKETING: '市场营销',
      OTHER: '其他费用',
      '': '全部分类',
    }
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: categoryMap[category] })
      .click()
    await this.waitForTableLoad()
  }

  async filterExpensesByStatus(
    status: 'DRAFT' | 'PENDING' | 'APPROVED' | 'REJECTED' | 'CANCELLED' | ''
  ): Promise<void> {
    // Find the status select - second select in the filter area
    const statusSelect = this.page.locator('.expenses-filter-container .semi-select').nth(1)
    await statusSelect.click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const statusMap: Record<string, string> = {
      DRAFT: '草稿',
      PENDING: '待审批',
      APPROVED: '已审批',
      REJECTED: '已拒绝',
      CANCELLED: '已取消',
      '': '全部状态',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: statusMap[status] }).click()
    await this.waitForTableLoad()
  }

  async getExpenseSummaryValues(): Promise<{
    totalApproved: string
    totalPending: string
  }> {
    await this.page.waitForSelector('.expenses-summary', { timeout: 5000 })
    const totalApproved =
      (await this.page
        .locator('.summary-item')
        .filter({ hasText: /已审批/ })
        .locator('.summary-value')
        .textContent()) || '-'
    const totalPending =
      (await this.page
        .locator('.summary-item')
        .filter({ hasText: /待审批/ })
        .locator('.summary-value')
        .textContent()) || '-'

    return { totalApproved, totalPending }
  }

  async clickNewExpenseButton(): Promise<void> {
    const newButton = this.page.getByRole('button', { name: /新增费用|新增/ })
    await newButton.click()
    await this.page.waitForURL(/\/finance\/expenses\/new/)
  }

  async fillExpenseForm(data: {
    category: string
    amount: number
    description: string
    incurredAt?: Date
  }): Promise<void> {
    // Select category
    const categorySelect = this.page.locator('.semi-select').filter({ hasText: /请选择分类/ })
    await categorySelect.click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })
    await this.page.locator('.semi-select-option').filter({ hasText: data.category }).click()

    // Fill amount
    const amountInput = this.page.locator('.semi-input-number input')
    await amountInput.fill(data.amount.toString())

    // Fill description
    const descriptionInput = this.page.locator('textarea').first()
    await descriptionInput.fill(data.description)
  }

  async submitExpenseForm(): Promise<void> {
    const submitButton = this.page.getByRole('button', { name: /提交|创建|保存/ })
    await submitButton.click()
  }

  async clickExpenseAction(
    expenseNumber: string,
    action: 'edit' | 'submit' | 'approve' | 'delete'
  ): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: expenseNumber })

    // Look for action buttons or dropdown
    const actionButton = row.locator('button, .semi-button').filter({
      hasText:
        action === 'edit'
          ? '编辑'
          : action === 'submit'
            ? '提交'
            : action === 'approve'
              ? '审批'
              : '删除',
    })

    if (await actionButton.isVisible()) {
      await actionButton.click()
    } else {
      // Try dropdown menu
      const moreButton = row
        .locator('.semi-dropdown-trigger, button')
        .filter({ hasText: /更多|操作/ })
        .first()
      if (await moreButton.isVisible()) {
        await moreButton.click()
        await this.page.waitForSelector('.semi-dropdown-menu', { state: 'visible' })
        const menuItem = this.page.locator('.semi-dropdown-item').filter({
          hasText:
            action === 'edit'
              ? '编辑'
              : action === 'submit'
                ? '提交'
                : action === 'approve'
                  ? '审批'
                  : '删除',
        })
        await menuItem.click()
      }
    }
  }

  async takeExpensesScreenshot(name: string = 'expenses'): Promise<void> {
    await this.screenshot(`finance/${name}`)
  }

  // =========== Other Incomes Page ===========

  async navigateToOtherIncomes(): Promise<void> {
    await this.goto('/finance/incomes')
    await this.waitForPageLoad()
    await this.page.waitForSelector('.semi-table', { timeout: 10000 })
  }

  async navigateToNewIncome(): Promise<void> {
    await this.goto('/finance/incomes/new')
    await this.waitForPageLoad()
  }

  async assertIncomesPageLoaded(): Promise<void> {
    const title = this.page.locator('h4').filter({ hasText: /其他收入管理|收入记录/ })
    await expect(title).toBeVisible()
    await expect(this.page.locator('.semi-table')).toBeVisible()
  }

  async getIncomeCount(): Promise<number> {
    await this.waitForTableLoad()
    const rows = this.page.locator('.semi-table-tbody .semi-table-row')
    return rows.count()
  }

  async assertIncomeExists(incomeNumber: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: incomeNumber })
    await expect(row).toBeVisible()
  }

  async assertIncomeStatus(incomeNumber: string, status: string): Promise<void> {
    const row = this.page.locator('.semi-table-row').filter({ hasText: incomeNumber })
    const statusTag = row.locator('.semi-tag').first()
    await expect(statusTag).toContainText(status)
  }

  async filterIncomesByCategory(
    category: 'INVESTMENT' | 'SUBSIDY' | 'INTEREST' | 'RENTAL' | 'REFUND' | 'OTHER' | ''
  ): Promise<void> {
    await this.page
      .locator('.semi-select')
      .filter({ hasText: /分类筛选|全部分类/ })
      .click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const categoryMap: Record<string, string> = {
      INVESTMENT: '投资收益',
      SUBSIDY: '补贴收入',
      INTEREST: '利息收入',
      RENTAL: '租金收入',
      REFUND: '退款收入',
      OTHER: '其他收入',
      '': '全部分类',
    }
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: categoryMap[category] })
      .click()
    await this.waitForTableLoad()
  }

  async getIncomeSummaryValues(): Promise<{
    totalConfirmed: string
    totalPending: string
  }> {
    await this.page.waitForSelector('.incomes-summary, .summary-descriptions', { timeout: 5000 })
    const totalConfirmed =
      (await this.page
        .locator('.summary-item')
        .filter({ hasText: /已确认/ })
        .locator('.summary-value')
        .textContent()) || '-'
    const totalPending =
      (await this.page
        .locator('.summary-item')
        .filter({ hasText: /待确认/ })
        .locator('.summary-value')
        .textContent()) || '-'

    return { totalConfirmed, totalPending }
  }

  async clickNewIncomeButton(): Promise<void> {
    const newButton = this.page.getByRole('button', { name: /新增收入|新增/ })
    await newButton.click()
    await this.page.waitForURL(/\/finance\/incomes\/new/)
  }

  async fillIncomeForm(data: {
    category: string
    amount: number
    description: string
    incomeDate?: Date
  }): Promise<void> {
    // Select category
    const categorySelect = this.page.locator('.semi-select').filter({ hasText: /请选择分类/ })
    await categorySelect.click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })
    await this.page.locator('.semi-select-option').filter({ hasText: data.category }).click()

    // Fill amount
    const amountInput = this.page.locator('.semi-input-number input')
    await amountInput.fill(data.amount.toString())

    // Fill description
    const descriptionInput = this.page.locator('textarea').first()
    await descriptionInput.fill(data.description)
  }

  async submitIncomeForm(): Promise<void> {
    const submitButton = this.page.getByRole('button', { name: /提交|创建|保存/ })
    await submitButton.click()
  }

  async takeIncomesScreenshot(name: string = 'incomes'): Promise<void> {
    await this.screenshot(`finance/${name}`)
  }

  // =========== Cash Flow Page ===========

  async navigateToCashFlow(): Promise<void> {
    await this.goto('/finance/cash-flow')
    await this.waitForPageLoad()
    await this.page.waitForSelector('.semi-table, .cash-flow-page', { timeout: 10000 })
  }

  async assertCashFlowPageLoaded(): Promise<void> {
    const title = this.page.locator('h4').filter({ hasText: /收支流水|现金流/ })
    await expect(title).toBeVisible()
  }

  async getCashFlowCount(): Promise<number> {
    await this.waitForTableLoad()
    const rows = this.page.locator('.semi-table-tbody .semi-table-row')
    return rows.count()
  }

  async filterCashFlowByType(type: 'income' | 'expense' | ''): Promise<void> {
    await this.page
      .locator('.semi-select')
      .filter({ hasText: /类型筛选|全部类型/ })
      .click()
    await this.page.waitForSelector('.semi-select-option-list', { state: 'visible' })

    const typeMap: Record<string, string> = {
      income: '收入',
      expense: '支出',
      '': '全部类型',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: typeMap[type] }).click()
    await this.waitForTableLoad()
  }

  async getCashFlowSummary(): Promise<{
    totalIncome: string
    totalExpense: string
    netBalance: string
  }> {
    const totalIncome =
      (await this.page
        .locator('.summary-item, .cash-flow-summary')
        .filter({ hasText: /总收入/ })
        .locator('.summary-value, .amount')
        .textContent()) || '-'
    const totalExpense =
      (await this.page
        .locator('.summary-item, .cash-flow-summary')
        .filter({ hasText: /总支出/ })
        .locator('.summary-value, .amount')
        .textContent()) || '-'
    const netBalance =
      (await this.page
        .locator('.summary-item, .cash-flow-summary')
        .filter({ hasText: /净额|结余/ })
        .locator('.summary-value, .amount')
        .textContent()) || '-'

    return { totalIncome, totalExpense, netBalance }
  }

  async takeCashFlowScreenshot(name: string = 'cash-flow'): Promise<void> {
    await this.screenshot(`finance/${name}`)
  }
}
