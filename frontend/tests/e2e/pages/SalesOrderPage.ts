import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * SalesOrderPage - Page Object for sales order module E2E tests
 *
 * Provides methods for:
 * - Sales order list navigation and interactions
 * - Order filtering by status/customer/date
 * - Order creation with customer and product selection
 * - Order status operations (confirm, ship, complete, cancel)
 * - Order detail viewing
 * - Inventory verification during order lifecycle
 */
export class SalesOrderPage extends BasePage {
  // List page elements
  readonly pageTitle: Locator
  readonly newOrderButton: Locator
  readonly refreshButton: Locator
  readonly statusFilter: Locator
  readonly customerFilter: Locator
  readonly dateRangeFilter: Locator
  readonly searchInput: Locator
  readonly tableRows: Locator
  readonly tableBody: Locator

  // Form page elements
  readonly customerSelect: Locator
  readonly warehouseSelect: Locator
  readonly discountInput: Locator
  readonly remarkInput: Locator
  readonly addProductButton: Locator
  readonly itemsTable: Locator
  readonly submitButton: Locator
  readonly cancelButton: Locator

  // Summary elements
  readonly itemCountDisplay: Locator
  readonly subtotalDisplay: Locator
  readonly discountDisplay: Locator
  readonly totalDisplay: Locator

  // Detail page elements
  readonly orderNumberDisplay: Locator
  readonly orderStatusTag: Locator
  readonly confirmButton: Locator
  readonly shipButton: Locator
  readonly completeButton: Locator
  readonly cancelOrderButton: Locator
  readonly editButton: Locator
  readonly backButton: Locator
  readonly timeline: Locator

  // Ship modal elements
  readonly shipModal: Locator
  readonly shipWarehouseSelect: Locator
  readonly shipConfirmButton: Locator
  readonly shipCancelButton: Locator

  constructor(page: Page) {
    super(page)

    // List page
    this.pageTitle = page.locator('h4').filter({ hasText: '销售订单' })
    this.newOrderButton = page.locator('button').filter({ hasText: '新建订单' })
    this.refreshButton = page.locator('button').filter({ hasText: '刷新' })
    this.statusFilter = page
      .locator('.semi-select')
      .filter({ hasText: /状态|全部状态/ })
      .first()
    this.customerFilter = page
      .locator('.semi-select')
      .filter({ hasText: /客户|全部客户/ })
      .first()
    this.dateRangeFilter = page.locator('.semi-datepicker-range')
    this.searchInput = page.locator('input[placeholder*="搜索订单编号"]')
    this.tableRows = page.locator('.semi-table-tbody .semi-table-row')
    this.tableBody = page.locator('.semi-table-tbody')

    // Form page
    this.customerSelect = page
      .locator('.semi-select')
      .filter({ hasText: /搜索并选择客户|客户/ })
      .first()
    this.warehouseSelect = page
      .locator('.semi-select')
      .filter({ hasText: /发货仓库|选择发货仓库/ })
      .first()
    this.discountInput = page
      .locator('input')
      .filter({ has: page.locator('[suffix="%"]') })
      .first()
    this.remarkInput = page.locator('input[placeholder*="备注"]')
    this.addProductButton = page.locator('button').filter({ hasText: '添加商品' })
    this.itemsTable = page.locator('.items-table, .semi-table').first()
    this.submitButton = page.locator('button').filter({ hasText: /创建订单|保存/ })
    this.cancelButton = page.locator('button').filter({ hasText: '取消' }).first()

    // Summary
    this.itemCountDisplay = page.locator('.summary-item').filter({ hasText: '商品数量' })
    this.subtotalDisplay = page.locator('.summary-item').filter({ hasText: '小计' })
    this.discountDisplay = page.locator('.summary-item').filter({ hasText: '折扣' })
    this.totalDisplay = page.locator('.summary-item.total, .total-amount').first()

    // Detail page
    this.orderNumberDisplay = page.locator('.order-basic-info, .info-card').first()
    this.orderStatusTag = page.locator('.page-header .semi-tag, .header-left .semi-tag').first()
    this.confirmButton = page.locator('button').filter({ hasText: '确认订单' })
    this.shipButton = page.locator('button').filter({ hasText: '发货' })
    this.completeButton = page.locator('button').filter({ hasText: '完成' })
    this.cancelOrderButton = page.locator('button').filter({ hasText: /取消$|取消订单/ })
    this.editButton = page.locator('button').filter({ hasText: '编辑' })
    this.backButton = page.locator('button').filter({ hasText: '返回列表' })
    this.timeline = page.locator('.status-timeline, .semi-timeline')

    // Ship modal
    this.shipModal = page.locator('.semi-modal').filter({ hasText: /发货|选择仓库/ })
    this.shipWarehouseSelect = page.locator('.semi-modal .semi-select')
    this.shipConfirmButton = page
      .locator('.semi-modal .semi-button-primary')
      .filter({ hasText: /确认|发货/ })
    this.shipCancelButton = page.locator('.semi-modal button').filter({ hasText: '取消' })
  }

  // Navigation methods
  async navigateToList(): Promise<void> {
    await this.goto('/trade/sales')
    await this.waitForPageLoad()
    await this.waitForTableLoad()
  }

  async navigateToNewOrder(): Promise<void> {
    await this.goto('/trade/sales/new')
    await this.waitForPageLoad()
  }

  async navigateToDetail(orderId: string): Promise<void> {
    await this.goto(`/trade/sales/${orderId}`)
    await this.waitForPageLoad()
  }

  async navigateToEdit(orderId: string): Promise<void> {
    await this.goto(`/trade/sales/${orderId}/edit`)
    await this.waitForPageLoad()
  }

  // List page methods
  async getOrderCount(): Promise<number> {
    await this.waitForTableLoad()
    return this.tableRows.count()
  }

  async search(orderNumber: string): Promise<void> {
    await this.searchInput.fill(orderNumber)
    await this.page.waitForTimeout(500) // Debounce
    await this.waitForTableLoad()
  }

  async clearSearch(): Promise<void> {
    await this.searchInput.clear()
    await this.page.waitForTimeout(500)
    await this.waitForTableLoad()
  }

  async filterByStatus(status: string): Promise<void> {
    // Click status filter dropdown
    const statusSelect = this.page.locator('.semi-select').first()
    await statusSelect.click()
    await this.page.waitForTimeout(200)

    // Map status to Chinese labels
    const statusLabels: Record<string, string> = {
      '': '全部状态',
      draft: '草稿',
      confirmed: '已确认',
      shipped: '已发货',
      completed: '已完成',
      cancelled: '已取消',
    }
    const label = statusLabels[status] || status
    await this.page.locator('.semi-select-option').filter({ hasText: label }).click()

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async filterByCustomer(customerName: string): Promise<void> {
    // Find customer filter (second select typically)
    const customerSelect = this.page.locator('.semi-select').nth(1)
    await customerSelect.click()
    await this.page.waitForTimeout(200)

    if (customerName) {
      await this.page.locator('.semi-select-option').filter({ hasText: customerName }).click()
    } else {
      await this.page.locator('.semi-select-option').filter({ hasText: '全部客户' }).click()
    }

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async clickNewOrder(): Promise<void> {
    await this.newOrderButton.click()
    await this.page.waitForURL('**/trade/sales/new')
  }

  async refresh(): Promise<void> {
    await this.refreshButton.click()
    await this.waitForTableLoad()
  }

  async getOrderRow(index: number): Promise<Locator> {
    return this.tableRows.nth(index)
  }

  async getOrderRowByNumber(orderNumber: string): Promise<Locator | null> {
    await this.waitForTableLoad()
    const rows = this.tableRows
    const count = await rows.count()

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i)
      const text = await row.textContent()
      if (text?.includes(orderNumber)) {
        return row
      }
    }
    return null
  }

  async clickRowAction(
    row: Locator,
    action: 'view' | 'edit' | 'confirm' | 'ship' | 'complete' | 'cancel' | 'delete'
  ): Promise<void> {
    // Hover over the row to show actions
    await row.hover()
    await this.page.waitForTimeout(200)

    // Find the action button by text
    const actionLabels: Record<string, string> = {
      view: '查看',
      edit: '编辑',
      confirm: '确认',
      ship: '发货',
      complete: '完成',
      cancel: '取消',
      delete: '删除',
    }
    const actionText = actionLabels[action]

    // Try to click the action directly in the row
    const actionButton = row.locator('a, button, .semi-button').filter({ hasText: actionText })
    if (await actionButton.isVisible()) {
      await actionButton.click()
    } else {
      // Try to find in dropdown menu
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

  // Form page methods
  async selectCustomer(customerName: string): Promise<void> {
    // Find customer select input
    const customerSelect = this.page
      .locator('.semi-select')
      .filter({ hasText: /搜索并选择客户|客户/ })
      .first()
    await customerSelect.click()
    await this.page.waitForTimeout(200)

    // Type to search - Semi Design's combobox uses a textbox role
    const searchInput = this.page.getByRole('textbox').first()
    await searchInput.fill(customerName)
    await this.page.waitForTimeout(500) // Wait for search

    // Wait for option to appear
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: customerName })
      .first()
      .waitFor({ state: 'visible' })

    // Select the option
    await this.page.locator('.semi-select-option').filter({ hasText: customerName }).first().click()

    // Wait for selection to be applied - verify the customer name appears in the selected item
    await this.page.waitForTimeout(300)
  }

  async selectWarehouse(warehouseName: string): Promise<void> {
    const warehouseSelect = this.page
      .locator('.semi-select')
      .filter({ hasText: /发货仓库|选择发货仓库/ })
      .first()
    await warehouseSelect.click()
    await this.page.waitForTimeout(200)

    await this.page.locator('.semi-select-option').filter({ hasText: warehouseName }).click()
    await this.page.waitForTimeout(200)
  }

  async addProductRow(): Promise<void> {
    await this.addProductButton.click()
    await this.page.waitForTimeout(200)
  }

  async selectProductInRow(rowIndex: number, productName: string): Promise<void> {
    // Get the row in items table
    const row = this.page.locator('.semi-table-tbody .semi-table-row').nth(rowIndex)

    // Find the product select in this row
    const productSelect = row.locator('.semi-select').first()
    await productSelect.click()
    await this.page.waitForTimeout(200)

    // Type to search - Semi Design's combobox uses a textbox role
    const searchInput = this.page.getByRole('textbox').first()
    await searchInput.fill(productName)
    await this.page.waitForTimeout(500)

    // Select the option
    await this.page.locator('.semi-select-option').filter({ hasText: productName }).first().click()
    await this.page.waitForTimeout(200)
  }

  async setQuantityInRow(rowIndex: number, quantity: number): Promise<void> {
    const row = this.page.locator('.semi-table-tbody .semi-table-row').nth(rowIndex)
    // Find quantity input (typically 4th column based on column structure)
    const quantityInput = row.locator('.semi-input-number input').nth(1) // Second number input is quantity
    await quantityInput.clear()
    await quantityInput.fill(quantity.toString())
    await this.page.waitForTimeout(100)
  }

  async setUnitPriceInRow(rowIndex: number, price: number): Promise<void> {
    const row = this.page.locator('.semi-table-tbody .semi-table-row').nth(rowIndex)
    // First number input is unit price
    const priceInput = row.locator('.semi-input-number input').first()
    await priceInput.clear()
    await priceInput.fill(price.toString())
    await this.page.waitForTimeout(100)
  }

  async removeProductRow(rowIndex: number): Promise<void> {
    const row = this.page.locator('.semi-table-tbody .semi-table-row').nth(rowIndex)
    const deleteButton = row
      .locator('button')
      .filter({ has: this.page.locator('[class*="delete"]') })
    await deleteButton.click()

    // Confirm deletion if prompted
    const confirmButton = this.page
      .locator('.semi-popconfirm-footer button')
      .filter({ hasText: /确定|确认/ })
    if (await confirmButton.isVisible({ timeout: 1000 }).catch(() => false)) {
      await confirmButton.click()
    }
    await this.page.waitForTimeout(200)
  }

  async setDiscount(discountPercent: number): Promise<void> {
    const discountInput = this.page
      .locator('.semi-input-number')
      .filter({ has: this.page.locator('[suffix="%"]') })
      .locator('input')
    await discountInput.clear()
    await discountInput.fill(discountPercent.toString())
    await this.page.waitForTimeout(100)
  }

  async setRemark(remark: string): Promise<void> {
    await this.remarkInput.fill(remark)
  }

  async submitOrder(): Promise<void> {
    await this.submitButton.click()
  }

  async waitForOrderCreateSuccess(): Promise<void> {
    await Promise.race([
      this.page.waitForURL('**/trade/sales', { timeout: 10000 }),
      this.waitForToast('成功'),
    ])
  }

  // Detail page methods
  async getOrderStatus(): Promise<string> {
    const statusTag = this.page.locator('.page-header .semi-tag, .header-left .semi-tag').first()
    return (await statusTag.textContent()) || ''
  }

  async getOrderInfo(): Promise<{
    orderNumber: string
    customerName: string
    status: string
    itemCount: number
    totalAmount: string
    payableAmount: string
  }> {
    const infoCard = this.page.locator('.info-card, .order-basic-info')
    const text = (await infoCard.textContent()) || ''

    // Parse order info from the descriptions
    const orderNumberMatch = text.match(/订单编号[：:]?\s*(SO-[\w-]+)/)
    const customerMatch = text.match(/客户名称[：:]?\s*([^\s订单]+)/)
    const itemCountMatch = text.match(/商品数量[：:]?\s*(\d+)/)

    const status = await this.getOrderStatus()

    // Get amounts from summary section
    const amountSummary = this.page.locator('.amount-summary')
    const amountText = (await amountSummary.textContent()) || ''
    const totalMatch = amountText.match(/商品金额[：:]?\s*¥([\d,.]+)/)
    const payableMatch = amountText.match(/应付金额[：:]?\s*¥([\d,.]+)/)

    return {
      orderNumber: orderNumberMatch?.[1] || '',
      customerName: customerMatch?.[1] || '',
      status,
      itemCount: parseInt(itemCountMatch?.[1] || '0'),
      totalAmount: totalMatch?.[1] || '0.00',
      payableAmount: payableMatch?.[1] || '0.00',
    }
  }

  async confirmOrder(): Promise<void> {
    await this.confirmButton.click()

    // Handle confirmation modal
    await this.page.locator('.semi-modal').waitFor()
    await this.page.locator('.semi-modal-footer .semi-button-primary').click()

    await this.waitForToast('确认')
    await this.page.waitForTimeout(500)
  }

  async shipOrder(warehouseName?: string): Promise<void> {
    await this.shipButton.click()

    // Wait for ship modal
    await this.page.locator('.semi-modal').waitFor()

    // Select warehouse if specified
    if (warehouseName) {
      const warehouseSelect = this.page.locator('.semi-modal .semi-select')
      await warehouseSelect.click()
      await this.page.waitForTimeout(200)
      await this.page.locator('.semi-select-option').filter({ hasText: warehouseName }).click()
      await this.page.waitForTimeout(200)
    }

    // Confirm shipping
    await this.page.locator('.semi-modal-footer .semi-button-primary').click()

    await this.waitForToast('发货')
    await this.page.waitForTimeout(500)
  }

  async completeOrder(): Promise<void> {
    await this.completeButton.click()
    await this.waitForToast('完成')
    await this.page.waitForTimeout(500)
  }

  async cancelOrder(): Promise<void> {
    await this.cancelOrderButton.click()

    // Handle confirmation modal
    await this.page.locator('.semi-modal').waitFor()
    await this.page
      .locator('.semi-modal-footer .semi-button-primary, .semi-modal-footer .semi-button-danger')
      .click()

    await this.waitForToast('取消')
    await this.page.waitForTimeout(500)
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

  // Order item table methods
  async getOrderItems(): Promise<
    Array<{
      productCode: string
      productName: string
      unit: string
      quantity: string
      unitPrice: string
      amount: string
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
      quantity: string
      unitPrice: string
      amount: string
    }> = []

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i)
      const cells = row.locator('.semi-table-row-cell')

      items.push({
        productCode: (await cells.nth(1).textContent()) || '',
        productName: (await cells.nth(2).textContent()) || '',
        unit: (await cells.nth(3).textContent()) || '',
        quantity: (await cells.nth(4).textContent()) || '',
        unitPrice: (await cells.nth(5).textContent()) || '',
        amount: (await cells.nth(6).textContent()) || '',
      })
    }

    return items
  }

  // Assertions
  async assertOrderListDisplayed(): Promise<void> {
    await expect(this.page.locator('h4').filter({ hasText: '销售订单' })).toBeVisible()
  }

  async assertOrderFormDisplayed(): Promise<void> {
    // The form title is "创建销售订单" for create mode or "编辑销售订单" for edit mode
    await expect(
      this.page.locator('h4').filter({ hasText: /创建销售订单|编辑销售订单/ })
    ).toBeVisible()
  }

  async assertOrderDetailDisplayed(): Promise<void> {
    await expect(this.page.locator('h4').filter({ hasText: '订单详情' })).toBeVisible()
  }

  async assertOrderStatus(expectedStatus: string): Promise<void> {
    const statusLabels: Record<string, string> = {
      draft: '草稿',
      confirmed: '已确认',
      shipped: '已发货',
      completed: '已完成',
      cancelled: '已取消',
    }
    const expectedLabel = statusLabels[expectedStatus] || expectedStatus
    await expect(this.orderStatusTag).toContainText(expectedLabel)
  }

  async assertOrderExists(orderNumber: string): Promise<void> {
    const row = await this.getOrderRowByNumber(orderNumber)
    expect(row).not.toBeNull()
  }

  async assertOrderNotExists(orderNumber: string): Promise<void> {
    const row = await this.getOrderRowByNumber(orderNumber)
    expect(row).toBeNull()
  }

  async assertOrderInList(orderNumber: string, expectedStatus: string): Promise<void> {
    const row = await this.getOrderRowByNumber(orderNumber)
    expect(row).not.toBeNull()

    if (row) {
      const statusLabels: Record<string, string> = {
        draft: '草稿',
        confirmed: '已确认',
        shipped: '已发货',
        completed: '已完成',
        cancelled: '已取消',
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
  async screenshotOrderList(name: string): Promise<void> {
    await this.screenshot(`sales-orders/${name}`)
  }

  async screenshotOrderForm(name: string): Promise<void> {
    await this.screenshot(`sales-orders/${name}`)
  }

  async screenshotOrderDetail(name: string): Promise<void> {
    await this.screenshot(`sales-orders/${name}`)
  }
}
