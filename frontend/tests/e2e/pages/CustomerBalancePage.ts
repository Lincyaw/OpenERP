import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * CustomerBalancePage - Page Object for Customer Balance management
 *
 * Covers:
 * - Customer balance page (/partner/customers/:id/balance)
 * - Balance summary display
 * - Recharge modal
 * - Transaction history
 */
export class CustomerBalancePage extends BasePage {
  // Page header elements
  readonly backButton: Locator
  readonly pageTitle: Locator

  // Balance summary card elements
  readonly customerName: Locator
  readonly customerCode: Locator
  readonly currentBalance: Locator
  readonly totalRecharge: Locator
  readonly totalConsume: Locator
  readonly totalRefund: Locator
  readonly rechargeButton: Locator

  // Transaction history elements
  readonly transactionTypeFilter: Locator
  readonly sourceTypeFilter: Locator
  readonly dateRangeFilter: Locator
  readonly refreshButton: Locator
  readonly transactionTable: Locator
  readonly transactionRows: Locator

  // Recharge modal elements
  readonly rechargeModal: Locator
  readonly amountInput: Locator
  readonly referenceInput: Locator
  readonly remarkInput: Locator
  readonly confirmRechargeButton: Locator
  readonly cancelRechargeButton: Locator
  readonly balancePreview: Locator

  constructor(page: Page) {
    super(page)

    // Page header elements
    this.backButton = page.locator('button').filter({ hasText: '返回客户列表' })
    this.pageTitle = page.locator('.balance-page-header h4')

    // Balance summary card elements
    this.customerName = page.locator('.customer-name-text')
    this.customerCode = page.locator('.customer-code-tag')
    this.currentBalance = page.locator('.balance-card.current-balance .balance-card-value')
    this.totalRecharge = page.locator('.balance-card.total-recharge .balance-card-value')
    this.totalConsume = page.locator('.balance-card.total-consume .balance-card-value')
    this.totalRefund = page.locator('.balance-card.total-refund .balance-card-value')
    this.rechargeButton = page.locator('button').filter({ hasText: '充值' })

    // Transaction history elements
    this.transactionTypeFilter = page.locator(
      '.transactions-filter-container .semi-select'
    ).first()
    this.sourceTypeFilter = page.locator('.transactions-filter-container .semi-select').nth(1)
    this.dateRangeFilter = page.locator('.transactions-filter-container .semi-datepicker')
    this.refreshButton = page.locator('button').filter({ hasText: '刷新' })
    this.transactionTable = page.locator('.transactions-card .semi-table')
    this.transactionRows = page.locator('.transactions-card .semi-table-tbody .semi-table-row')

    // Recharge modal elements
    // Note: Semi Design Modal applies className to the modal wrapper which may be hidden initially
    // We need to target the visible modal content container
    this.rechargeModal = page.locator('.semi-modal-content').filter({ has: page.locator('.recharge-modal-content') })
    this.amountInput = page.locator('.recharge-modal-content .amount-input input')
    this.referenceInput = page.locator('.recharge-modal-content input').nth(1)
    this.remarkInput = page.locator('.recharge-modal-content textarea')
    this.confirmRechargeButton = page.locator('.semi-modal-footer button.semi-button-primary')
    this.cancelRechargeButton = page.locator('.semi-modal-footer button.semi-button-tertiary')
    this.balancePreview = page.locator('.balance-preview')
  }

  /**
   * Navigate to customer balance page
   */
  async navigateToBalance(customerId: string): Promise<void> {
    await this.goto(`/partner/customers/${customerId}/balance`)
    await this.page.waitForLoadState('networkidle')
    // Wait for balance summary to load - need to wait for text content, not just visibility
    // The element exists immediately but is empty until API returns data
    await this.page.waitForFunction(
      (selector) => {
        const el = document.querySelector(selector)
        return el && el.textContent && el.textContent.trim().length > 0
      },
      '.balance-card.current-balance .balance-card-value',
      { timeout: 15000 }
    )
  }

  /**
   * Get current balance value
   */
  async getCurrentBalance(): Promise<number> {
    const text = await this.currentBalance.textContent()
    return this.parseCurrency(text)
  }

  /**
   * Get total recharge value
   */
  async getTotalRecharge(): Promise<number> {
    const text = await this.totalRecharge.textContent()
    return this.parseCurrency(text)
  }

  /**
   * Get total consume value
   */
  async getTotalConsume(): Promise<number> {
    const text = await this.totalConsume.textContent()
    return this.parseCurrency(text)
  }

  /**
   * Get total refund value
   */
  async getTotalRefund(): Promise<number> {
    const text = await this.totalRefund.textContent()
    return this.parseCurrency(text)
  }

  /**
   * Parse currency string to number
   */
  private parseCurrency(text: string | null): number {
    if (!text) return 0
    // Remove currency symbol and thousands separators
    const cleaned = text.replace(/[¥￥,\s]/g, '')
    return parseFloat(cleaned) || 0
  }

  /**
   * Open recharge modal
   */
  async openRechargeModal(): Promise<void> {
    await this.rechargeButton.click()
    await this.rechargeModal.waitFor({ state: 'visible', timeout: 5000 })
  }

  /**
   * Fill recharge form
   */
  async fillRechargeForm(data: {
    amount: number
    reference?: string
    remark?: string
  }): Promise<void> {
    // Fill amount
    await this.amountInput.fill(data.amount.toString())

    // Fill optional reference
    if (data.reference) {
      await this.referenceInput.fill(data.reference)
    }

    // Fill optional remark
    if (data.remark) {
      await this.remarkInput.fill(data.remark)
    }
  }

  /**
   * Submit recharge form
   */
  async submitRecharge(): Promise<void> {
    await this.confirmRechargeButton.click()
    // Wait for modal to close
    await this.rechargeModal.waitFor({ state: 'hidden', timeout: 10000 })
    // Wait for balance to refresh
    await this.page.waitForTimeout(1000)
  }

  /**
   * Cancel recharge
   */
  async cancelRecharge(): Promise<void> {
    await this.cancelRechargeButton.click()
    await this.rechargeModal.waitFor({ state: 'hidden', timeout: 5000 })
  }

  /**
   * Verify balance preview shows correct values
   */
  async verifyBalancePreview(
    currentBalance: number,
    rechargeAmount: number
  ): Promise<void> {
    // Wait for preview to appear
    await expect(this.balancePreview).toBeVisible({ timeout: 5000 })
    const expectedAfter = currentBalance + rechargeAmount

    // Wait for the balance-after element to have text content
    const balanceAfterLocator = this.balancePreview.locator('.balance-preview-total .balance-after')
    await expect(balanceAfterLocator).toBeVisible({ timeout: 3000 })

    // Check the balance after text
    const balanceAfterText = await balanceAfterLocator.textContent()
    const actualAfter = this.parseCurrency(balanceAfterText)
    expect(actualAfter).toBeCloseTo(expectedAfter, 2)
  }

  /**
   * Get transaction count
   */
  async getTransactionCount(): Promise<number> {
    return this.transactionRows.count()
  }

  /**
   * Filter transactions by type
   */
  async filterByTransactionType(
    type: 'RECHARGE' | 'CONSUME' | 'REFUND' | 'ADJUSTMENT' | 'EXPIRE' | ''
  ): Promise<void> {
    const typeLabels: Record<string, string> = {
      '': '全部类型',
      RECHARGE: '充值',
      CONSUME: '消费',
      REFUND: '退款',
      ADJUSTMENT: '调整',
      EXPIRE: '过期',
    }

    await this.transactionTypeFilter.click()
    await this.page.locator('.semi-select-option-list').waitFor({ state: 'visible' })
    await this.page.waitForTimeout(300)
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: typeLabels[type] })
      .first()
      .click()
    await this.waitForTableLoad()
  }

  /**
   * Filter transactions by source type
   */
  async filterBySourceType(
    source: 'MANUAL' | 'SALES_ORDER' | 'SALES_RETURN' | 'RECEIPT_VOUCHER' | 'SYSTEM' | ''
  ): Promise<void> {
    const sourceLabels: Record<string, string> = {
      '': '全部来源',
      MANUAL: '手动操作',
      SALES_ORDER: '销售订单',
      SALES_RETURN: '销售退货',
      RECEIPT_VOUCHER: '收款单',
      SYSTEM: '系统',
    }

    await this.sourceTypeFilter.click()
    await this.page.locator('.semi-select-option-list').waitFor({ state: 'visible' })
    await this.page.waitForTimeout(300)
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: sourceLabels[source] })
      .first()
      .click()
    await this.waitForTableLoad()
  }

  /**
   * Find transaction row by type
   */
  async findTransactionByType(
    type: 'RECHARGE' | 'CONSUME' | 'REFUND' | 'ADJUSTMENT' | 'EXPIRE'
  ): Promise<Locator | null> {
    const typeLabels: Record<string, string> = {
      RECHARGE: '充值',
      CONSUME: '消费',
      REFUND: '退款',
      ADJUSTMENT: '调整',
      EXPIRE: '过期',
    }

    const rows = await this.transactionRows.all()
    for (const row of rows) {
      const typeTag = row.locator('.semi-tag')
      const typeText = await typeTag.textContent()
      if (typeText?.includes(typeLabels[type])) {
        return row
      }
    }
    return null
  }

  /**
   * Verify latest transaction
   */
  async verifyLatestTransaction(data: {
    type: 'RECHARGE' | 'CONSUME' | 'REFUND' | 'ADJUSTMENT' | 'EXPIRE'
    amount: number
  }): Promise<void> {
    const typeLabels: Record<string, string> = {
      RECHARGE: '充值',
      CONSUME: '消费',
      REFUND: '退款',
      ADJUSTMENT: '调整',
      EXPIRE: '过期',
    }

    // Get the first row (latest transaction)
    const firstRow = this.transactionRows.first()
    await expect(firstRow).toBeVisible()

    // Verify type
    const typeTag = firstRow.locator('.semi-tag')
    await expect(typeTag).toContainText(typeLabels[data.type])

    // Verify amount (with sign)
    const amountCell = firstRow.locator('.semi-table-row-cell').nth(2)
    const amountText = await amountCell.textContent()
    const amount = this.parseCurrency(amountText)
    expect(amount).toBeCloseTo(data.amount, 2)
  }

  /**
   * Go back to customer list
   */
  async goBackToList(): Promise<void> {
    await this.backButton.click()
    await this.page.waitForURL('**/partner/customers')
  }

  /**
   * Refresh balance and transactions
   */
  async refresh(): Promise<void> {
    await this.refreshButton.click()
    await this.page.waitForTimeout(500)
    await this.waitForTableLoad()
  }

  /**
   * Assert customer info displayed
   */
  async assertCustomerInfo(name: string, code?: string): Promise<void> {
    await expect(this.customerName).toContainText(name)
    if (code) {
      await expect(this.customerCode).toContainText(code)
    }
  }

  /**
   * Assert balance value
   */
  async assertBalance(expectedBalance: number): Promise<void> {
    const actualBalance = await this.getCurrentBalance()
    expect(actualBalance).toBeCloseTo(expectedBalance, 2)
  }

  /**
   * Take screenshot of balance page
   */
  async screenshotBalancePage(name: string = 'customer-balance'): Promise<void> {
    await this.screenshot(name)
  }

  /**
   * Take screenshot of recharge modal
   */
  async screenshotRechargeModal(name: string = 'recharge-modal'): Promise<void> {
    await this.rechargeModal.screenshot({ path: `screenshots/${name}.png` })
  }
}
