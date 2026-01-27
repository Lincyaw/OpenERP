import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * SalesReturnPage - Page Object for sales return module E2E tests
 *
 * Provides methods for:
 * - Sales return list navigation and interactions
 * - Return filtering by status/customer/date
 * - Return creation from sales order
 * - Return status operations (submit, approve, reject, complete, cancel)
 * - Return detail viewing
 * - Inventory verification during return lifecycle
 */
export class SalesReturnPage extends BasePage {
  // List page elements
  readonly pageTitle: Locator
  readonly newReturnButton: Locator
  readonly approvalButton: Locator
  readonly refreshButton: Locator
  readonly statusFilter: Locator
  readonly customerFilter: Locator
  readonly dateRangeFilter: Locator
  readonly searchInput: Locator
  readonly tableRows: Locator
  readonly tableBody: Locator

  // Form page elements
  readonly orderSelect: Locator
  readonly orderSearchInput: Locator
  readonly customerDisplay: Locator
  readonly reasonInput: Locator
  readonly remarkInput: Locator
  readonly itemsTable: Locator
  readonly submitButton: Locator
  readonly saveButton: Locator
  readonly cancelButton: Locator

  // Detail page elements
  readonly returnNumberDisplay: Locator
  readonly returnStatusTag: Locator
  readonly submitForApprovalButton: Locator
  readonly approveButton: Locator
  readonly rejectButton: Locator
  readonly completeButton: Locator
  readonly cancelReturnButton: Locator
  readonly editButton: Locator
  readonly backButton: Locator
  readonly timeline: Locator

  // Approval page elements
  readonly approvalList: Locator
  readonly approvalModal: Locator
  readonly approvalNoteInput: Locator
  readonly approvalConfirmButton: Locator
  readonly approvalRejectButton: Locator
  readonly rejectReasonInput: Locator

  // Modal elements
  readonly modalElement: Locator
  readonly confirmModalOkButton: Locator
  readonly confirmModalCancelButton: Locator

  constructor(page: Page) {
    super(page)

    // List page - use class context for better specificity
    this.pageTitle = page.locator('.sales-returns-header h4, .sales-return-form-header h4').first()
    this.newReturnButton = page
      .locator('.table-toolbar-right button')
      .filter({ hasText: '新建退货' })
    this.approvalButton = page.locator('.table-toolbar-right button').filter({ hasText: '审批' })
    this.refreshButton = page.locator('.table-toolbar-right button').filter({ hasText: '刷新' })
    // Status filter is in .table-toolbar-filters, NOT the first .semi-select on page
    this.statusFilter = page.locator('.table-toolbar-filters .semi-select').first()
    this.customerFilter = page.locator('.table-toolbar-filters .semi-select').nth(1)
    this.dateRangeFilter = page.locator('.table-toolbar-filters .semi-datepicker')
    // Search input uses class from TableToolbar
    this.searchInput = page.locator('.table-toolbar-search input')
    this.tableRows = page.locator('.semi-table-tbody .semi-table-row')
    this.tableBody = page.locator('.semi-table-tbody')

    // Form page
    this.orderSelect = page
      .locator('.semi-select')
      .filter({ hasText: /选择销售订单|原订单/ })
      .first()
    this.orderSearchInput = page.getByRole('textbox').first()
    this.customerDisplay = page.locator('.customer-info, .order-info')
    this.reasonInput = page.locator('textarea[placeholder*="退货原因"]').first()
    this.remarkInput = page.locator('textarea[placeholder*="备注"]')
    this.itemsTable = page.locator('.items-table, .semi-table').first()
    this.submitButton = page.locator('button').filter({ hasText: /提交|创建退货单/ })
    this.saveButton = page.locator('button').filter({ hasText: /保存|暂存/ })
    this.cancelButton = page.locator('button').filter({ hasText: '取消' }).first()

    // Detail page
    this.returnNumberDisplay = page.locator('.return-basic-info, .info-card').first()
    this.returnStatusTag = page.locator('.page-header .semi-tag, .header-left .semi-tag').first()
    this.submitForApprovalButton = page.locator('button').filter({ hasText: '提交审批' })
    this.approveButton = page.locator('button').filter({ hasText: '审批通过' })
    this.rejectButton = page.locator('button').filter({ hasText: '拒绝' })
    this.completeButton = page.locator('button').filter({ hasText: '完成退货' })
    this.cancelReturnButton = page.locator('button').filter({ hasText: /取消$|取消退货/ })
    this.editButton = page.locator('button').filter({ hasText: '编辑' })
    this.backButton = page.locator('button').filter({ hasText: '返回列表' })
    this.timeline = page.locator('.status-timeline, .semi-timeline')

    // Approval page
    this.approvalList = page.locator('.approval-list, .semi-table')
    this.approvalModal = page.locator('.semi-modal').filter({ hasText: /审批|批准/ })
    this.approvalNoteInput = page.locator('.semi-modal textarea').first()
    this.approvalConfirmButton = page.locator('.semi-modal button').filter({ hasText: /通过|确认/ })
    this.approvalRejectButton = page.locator('.semi-modal button').filter({ hasText: '拒绝' })
    this.rejectReasonInput = page.locator('.semi-modal textarea')

    // Modal
    this.modalElement = page.locator('.semi-modal')
    this.confirmModalOkButton = page.locator('.semi-modal-footer .semi-button-primary')
    this.confirmModalCancelButton = page
      .locator('.semi-modal-footer button')
      .filter({ hasText: '取消' })
  }

  // Navigation methods
  async navigateToList(): Promise<void> {
    await this.goto('/trade/sales-returns')
    await this.waitForPageLoad()
    // Wait for table toolbar to render
    await this.page.locator('.table-toolbar').waitFor({ state: 'visible', timeout: 15000 })
    await this.waitForTableLoad()
  }

  async navigateToNewReturn(): Promise<void> {
    await this.goto('/trade/sales-returns/new')
    await this.waitForPageLoad()
  }

  async navigateToDetail(returnId: string): Promise<void> {
    await this.goto(`/trade/sales-returns/${returnId}`)
    await this.waitForPageLoad()
  }

  async navigateToApproval(): Promise<void> {
    await this.goto('/trade/sales-returns/approval')
    await this.waitForPageLoad()
  }

  // List page methods
  async getReturnCount(): Promise<number> {
    await this.waitForTableLoad()
    return this.tableRows.count()
  }

  async search(returnNumber: string): Promise<void> {
    const searchInput = this.page.locator('.table-toolbar-search input')
    await searchInput.fill(returnNumber)
    await this.page.waitForTimeout(500) // Debounce
    await this.waitForTableLoad()
  }

  async clearSearch(): Promise<void> {
    const searchInput = this.page.locator('.table-toolbar-search input')
    await searchInput.clear()
    await this.page.waitForTimeout(500)
    await this.waitForTableLoad()
  }

  async filterByStatus(status: string): Promise<void> {
    // Use the specific filters container
    const statusSelect = this.page.locator('.table-toolbar-filters .semi-select').first()
    await statusSelect.click()
    await this.page.waitForTimeout(300)

    // Wait for dropdown to appear
    await this.page.locator('.semi-select-option-list').waitFor({ state: 'visible', timeout: 5000 })

    const statusLabels: Record<string, string> = {
      '': '全部状态',
      DRAFT: '草稿',
      PENDING: '待审批',
      APPROVED: '已审批',
      REJECTED: '已拒绝',
      COMPLETED: '已完成',
      CANCELLED: '已取消',
    }
    const label = statusLabels[status] || status
    await this.page.locator('.semi-select-option').filter({ hasText: label }).click()

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async filterByCustomer(customerName: string): Promise<void> {
    const customerSelect = this.page.locator('.table-toolbar-filters .semi-select').nth(1)
    await customerSelect.click()
    await this.page.waitForTimeout(300)

    // Wait for dropdown to appear
    await this.page.locator('.semi-select-option-list').waitFor({ state: 'visible', timeout: 5000 })

    if (customerName) {
      await this.page.locator('.semi-select-option').filter({ hasText: customerName }).click()
    } else {
      await this.page.locator('.semi-select-option').filter({ hasText: '全部客户' }).click()
    }

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async clickNewReturn(): Promise<void> {
    await this.newReturnButton.click()
    await this.page.waitForURL('**/trade/sales-returns/new')
  }

  async clickApproval(): Promise<void> {
    await this.approvalButton.click()
    await this.page.waitForURL('**/trade/sales-returns/approval')
  }

  async refresh(): Promise<void> {
    await this.refreshButton.click()
    await this.waitForTableLoad()
  }

  async getReturnRow(index: number): Promise<Locator> {
    return this.tableRows.nth(index)
  }

  async getReturnRowByNumber(returnNumber: string): Promise<Locator | null> {
    await this.waitForTableLoad()
    const rows = this.tableRows
    const count = await rows.count()

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i)
      const text = await row.textContent()
      if (text?.includes(returnNumber)) {
        return row
      }
    }
    return null
  }

  async clickRowAction(
    row: Locator,
    action: 'view' | 'submit' | 'approve' | 'reject' | 'complete' | 'cancel' | 'delete'
  ): Promise<void> {
    await row.hover()
    await this.page.waitForTimeout(200)

    const actionLabels: Record<string, string> = {
      view: '查看',
      submit: '提交',
      approve: '审批',
      reject: '拒绝',
      complete: '完成',
      cancel: '取消',
      delete: '删除',
    }
    const actionText = actionLabels[action]

    const actionButton = row.locator('a, button, .semi-button').filter({ hasText: actionText })
    if (await actionButton.isVisible()) {
      await actionButton.click()
    } else {
      const moreButton = row
        .locator('.semi-dropdown-trigger, button')
        .filter({ hasText: /更多|操作/ })
      if (await moreButton.isVisible()) {
        await moreButton.click()
        await this.page.waitForTimeout(200)
        await this.page
          .locator('.semi-dropdown-menu .semi-dropdown-item')
          .filter({ hasText: actionText })
          .click()
      }
    }
  }

  async viewReturnFromRow(row: Locator): Promise<void> {
    // Click return number link or view action
    const returnNumberLink = row.locator('.return-number, a').first()
    if (await returnNumberLink.isVisible()) {
      await returnNumberLink.click()
    } else {
      await this.clickRowAction(row, 'view')
    }
    await this.page.waitForURL('**/trade/sales-returns/**')
  }

  // Form page methods - Creating return from order
  async selectSalesOrder(orderNumber: string): Promise<void> {
    // Find order select input
    const orderSelect = this.page
      .locator('.semi-select')
      .filter({ hasText: /选择.*订单|订单/ })
      .first()
    await orderSelect.click()
    await this.page.waitForTimeout(200)

    // Type to search
    const searchInput = this.page.locator('.semi-select-option-list input[type="text"]')
    if (await searchInput.isVisible({ timeout: 1000 }).catch(() => false)) {
      await searchInput.fill(orderNumber)
    } else {
      await this.page.keyboard.type(orderNumber)
    }
    await this.page.waitForTimeout(500)

    // Select the order option
    const option = this.page.locator('.semi-select-option').filter({ hasText: orderNumber }).first()
    await option.waitFor({ state: 'visible', timeout: 5000 })
    await option.click()
    await this.page.waitForTimeout(300)
  }

  async setReturnReason(reason: string): Promise<void> {
    const reasonInput = this.page
      .locator('textarea')
      .filter({ has: this.page.locator('[placeholder*="原因"]') })
      .first()
    if (await reasonInput.isVisible().catch(() => false)) {
      await reasonInput.fill(reason)
    } else {
      // Try finding by placeholder
      const textarea = this.page
        .locator('textarea[placeholder*="原因"], textarea[placeholder*="退货"]')
        .first()
      await textarea.fill(reason)
    }
  }

  async setReturnRemark(remark: string): Promise<void> {
    const remarkInput = this.page.locator('textarea[placeholder*="备注"]').first()
    if (await remarkInput.isVisible().catch(() => false)) {
      await remarkInput.fill(remark)
    }
  }

  async selectReturnItem(productName: string, returnQuantity: number): Promise<void> {
    // Find the item row by product name
    const rows = this.page.locator('.semi-table-tbody .semi-table-row')
    const count = await rows.count()

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i)
      const text = await row.textContent()
      if (text?.includes(productName)) {
        // Check the checkbox or set quantity
        const checkbox = row.locator('.semi-checkbox input')
        if (await checkbox.isVisible().catch(() => false)) {
          await checkbox.check()
        }

        // Set return quantity
        const quantityInput = row.locator('.semi-input-number input').first()
        if (await quantityInput.isVisible().catch(() => false)) {
          await quantityInput.clear()
          await quantityInput.fill(returnQuantity.toString())
        }
        break
      }
    }
    await this.page.waitForTimeout(100)
  }

  async setReturnQuantityInRow(rowIndex: number, quantity: number): Promise<void> {
    const row = this.page.locator('.semi-table-tbody .semi-table-row').nth(rowIndex)
    const quantityInput = row.locator('.semi-input-number input').first()
    await quantityInput.clear()
    await quantityInput.fill(quantity.toString())
    await this.page.waitForTimeout(100)
  }

  async submitReturn(): Promise<void> {
    const submitBtn = this.page
      .locator('button')
      .filter({ hasText: /创建|提交|保存/ })
      .first()
    await submitBtn.click()
  }

  async waitForReturnCreateSuccess(): Promise<void> {
    await Promise.race([
      this.page.waitForURL('**/trade/sales-returns', { timeout: 15000 }),
      this.page.waitForURL('**/trade/sales-returns/**', { timeout: 15000 }),
      this.waitForToast('成功'),
    ])
    await this.page.waitForLoadState('domcontentloaded', { timeout: 5000 }).catch(() => {})
  }

  // Detail page methods
  async getReturnStatus(): Promise<string> {
    const statusTag = this.page.locator('.page-header .semi-tag, .header-left .semi-tag').first()
    return (await statusTag.textContent()) || ''
  }

  async getReturnInfo(): Promise<{
    returnNumber: string
    orderNumber: string
    customerName: string
    status: string
    itemCount: number
    totalRefund: string
  }> {
    const infoCard = this.page.locator('.info-card, .return-basic-info')
    const text = (await infoCard.textContent()) || ''

    const returnNumberMatch = text.match(/退货单号[：:]?\s*(SR-[\w-]+)/)
    const orderNumberMatch = text.match(/原订单[：:]?\s*(SO-[\w-]+)/)
    const customerMatch = text.match(/客户[名称]?[：:]?\s*([^\s退货]+)/)
    const itemCountMatch = text.match(/商品数量[：:]?\s*(\d+)/)

    const status = await this.getReturnStatus()

    const amountSummary = this.page.locator('.amount-summary, .refund-summary')
    const amountText = (await amountSummary.textContent()) || ''
    const refundMatch = amountText.match(/退款金额[：:]?\s*¥([\d,.]+)/)

    return {
      returnNumber: returnNumberMatch?.[1] || '',
      orderNumber: orderNumberMatch?.[1] || '',
      customerName: customerMatch?.[1] || '',
      status,
      itemCount: parseInt(itemCountMatch?.[1] || '0'),
      totalRefund: refundMatch?.[1] || '0.00',
    }
  }

  async submitForApproval(): Promise<void> {
    await this.submitForApprovalButton.click()

    // Handle confirmation modal
    await this.modalElement.waitFor({ state: 'visible', timeout: 5000 })
    await this.confirmModalOkButton.click()

    await this.waitForToast('提交')
    await this.page.waitForTimeout(500)
  }

  async approveReturn(note?: string): Promise<void> {
    await this.approveButton.click()

    // Handle approval modal
    await this.modalElement.waitFor({ state: 'visible', timeout: 5000 })

    if (note) {
      const noteInput = this.page.locator('.semi-modal textarea').first()
      if (await noteInput.isVisible().catch(() => false)) {
        await noteInput.fill(note)
      }
    }

    await this.confirmModalOkButton.click()
    await this.waitForToast('审批')
    await this.page.waitForTimeout(500)
  }

  async rejectReturn(reason: string): Promise<void> {
    await this.rejectButton.click()

    // Handle rejection modal
    await this.modalElement.waitFor({ state: 'visible', timeout: 5000 })

    const reasonInput = this.page.locator('.semi-modal textarea').first()
    if (await reasonInput.isVisible().catch(() => false)) {
      await reasonInput.fill(reason)
    }

    const rejectBtn = this.page.locator('.semi-modal button').filter({ hasText: '拒绝' }).first()
    await rejectBtn.click()

    await this.waitForToast('拒绝')
    await this.page.waitForTimeout(500)
  }

  async completeReturn(): Promise<void> {
    await this.completeButton.click()

    // Handle confirmation modal if exists
    const modal = this.page.locator('.semi-modal')
    if (await modal.isVisible({ timeout: 2000 }).catch(() => false)) {
      await this.confirmModalOkButton.click()
    }

    await this.waitForToast('完成')
    await this.page.waitForTimeout(500)
  }

  async cancelReturn(reason?: string): Promise<void> {
    await this.cancelReturnButton.click()

    // Handle confirmation modal
    await this.modalElement.waitFor({ state: 'visible', timeout: 5000 })

    if (reason) {
      const reasonInput = this.page.locator('.semi-modal textarea').first()
      if (await reasonInput.isVisible().catch(() => false)) {
        await reasonInput.fill(reason)
      }
    }

    const cancelConfirmBtn = this.page.locator(
      '.semi-modal-footer .semi-button-primary, .semi-modal-footer .semi-button-danger'
    )
    await cancelConfirmBtn.click()

    await this.waitForToast('取消')
    await this.page.waitForTimeout(500)
  }

  async goBackToList(): Promise<void> {
    await this.backButton.click()
    await this.page.waitForURL('**/trade/sales-returns')
  }

  async getTimelineEvents(): Promise<string[]> {
    const timeline = this.page.locator('.semi-timeline-item, .status-timeline .semi-timeline-item')
    const count = await timeline.count()
    const events: string[] = []

    for (let i = 0; i < count; i++) {
      const text = await timeline.nth(i).textContent()
      if (text) events.push(text)
    }

    return events
  }

  // Approval page methods
  async getPendingApprovalCount(): Promise<number> {
    await this.waitForTableLoad()
    return this.tableRows.count()
  }

  async approveFromList(returnNumber: string, note?: string): Promise<void> {
    const row = await this.getReturnRowByNumber(returnNumber)
    if (!row) throw new Error(`Return ${returnNumber} not found`)

    await this.clickRowAction(row, 'approve')

    // Handle modal
    await this.modalElement.waitFor({ state: 'visible', timeout: 5000 })
    if (note) {
      const noteInput = this.page.locator('.semi-modal textarea').first()
      if (await noteInput.isVisible().catch(() => false)) {
        await noteInput.fill(note)
      }
    }
    await this.confirmModalOkButton.click()
    await this.waitForToast('审批')
  }

  async rejectFromList(returnNumber: string, reason: string): Promise<void> {
    const row = await this.getReturnRowByNumber(returnNumber)
    if (!row) throw new Error(`Return ${returnNumber} not found`)

    await this.clickRowAction(row, 'reject')

    // Handle modal
    await this.modalElement.waitFor({ state: 'visible', timeout: 5000 })
    const reasonInput = this.page.locator('.semi-modal textarea').first()
    if (await reasonInput.isVisible().catch(() => false)) {
      await reasonInput.fill(reason)
    }

    const rejectBtn = this.page.locator('.semi-modal button').filter({ hasText: '拒绝' })
    await rejectBtn.click()
    await this.waitForToast('拒绝')
  }

  // Return item table methods
  async getReturnItems(): Promise<
    Array<{
      productCode: string
      productName: string
      unit: string
      returnQuantity: string
      unitPrice: string
      refundAmount: string
    }>
  > {
    const rows = this.page.locator(
      '.items-card .semi-table-tbody .semi-table-row, .semi-table-tbody .semi-table-row'
    )
    const count = await rows.count()
    const items: Array<{
      productCode: string
      productName: string
      unit: string
      returnQuantity: string
      unitPrice: string
      refundAmount: string
    }> = []

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i)
      const cells = row.locator('.semi-table-row-cell')

      items.push({
        productCode: (await cells.nth(1).textContent()) || '',
        productName: (await cells.nth(2).textContent()) || '',
        unit: (await cells.nth(3).textContent()) || '',
        returnQuantity: (await cells.nth(4).textContent()) || '',
        unitPrice: (await cells.nth(5).textContent()) || '',
        refundAmount: (await cells.nth(6).textContent()) || '',
      })
    }

    return items
  }

  // Assertions
  async assertReturnListDisplayed(): Promise<void> {
    // Wait for page load with specific header element
    await expect(this.page.locator('.sales-returns-header h4')).toBeVisible({ timeout: 15000 })
  }

  async assertReturnFormDisplayed(): Promise<void> {
    await expect(this.page.locator('.sales-return-form-header h4')).toBeVisible({ timeout: 15000 })
  }

  async assertReturnDetailDisplayed(): Promise<void> {
    await expect(this.page.locator('h4').filter({ hasText: /退货.*详情|退货单/ })).toBeVisible()
  }

  async assertReturnStatus(expectedStatus: string): Promise<void> {
    const statusLabels: Record<string, string> = {
      DRAFT: '草稿',
      PENDING: '待审批',
      APPROVED: '已审批',
      REJECTED: '已拒绝',
      COMPLETED: '已完成',
      CANCELLED: '已取消',
    }
    const expectedLabel = statusLabels[expectedStatus] || expectedStatus
    await expect(this.returnStatusTag).toContainText(expectedLabel)
  }

  async assertReturnExists(returnNumber: string): Promise<void> {
    const row = await this.getReturnRowByNumber(returnNumber)
    expect(row).not.toBeNull()
  }

  async assertReturnNotExists(returnNumber: string): Promise<void> {
    const row = await this.getReturnRowByNumber(returnNumber)
    expect(row).toBeNull()
  }

  async assertReturnInList(returnNumber: string, expectedStatus: string): Promise<void> {
    const row = await this.getReturnRowByNumber(returnNumber)
    expect(row).not.toBeNull()

    if (row) {
      const statusLabels: Record<string, string> = {
        DRAFT: '草稿',
        PENDING: '待审批',
        APPROVED: '已审批',
        REJECTED: '已拒绝',
        COMPLETED: '已完成',
        CANCELLED: '已取消',
      }
      const expectedLabel = statusLabels[expectedStatus] || expectedStatus
      await expect(row.locator('.semi-tag')).toContainText(expectedLabel)
    }
  }

  async assertTimelineContains(eventText: string): Promise<void> {
    const timeline = this.page.locator('.semi-timeline')
    await expect(timeline).toContainText(eventText)
  }

  // Screenshot methods
  async screenshotReturnList(name: string): Promise<void> {
    await this.screenshot(`sales-returns/${name}`)
  }

  async screenshotReturnForm(name: string): Promise<void> {
    await this.screenshot(`sales-returns/${name}`)
  }

  async screenshotReturnDetail(name: string): Promise<void> {
    await this.screenshot(`sales-returns/${name}`)
  }
}
