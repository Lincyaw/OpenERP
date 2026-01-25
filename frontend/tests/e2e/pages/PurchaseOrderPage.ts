import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * PurchaseOrderPage - Page Object for purchase order module E2E tests
 *
 * Provides methods for:
 * - Purchase order list navigation and interactions
 * - Order filtering by status/supplier/date
 * - Order creation with supplier and product selection
 * - Order status operations (confirm, receive, cancel)
 * - Receiving operations (full and partial)
 * - Inventory verification during order lifecycle
 * - Accounts payable verification
 */
export class PurchaseOrderPage extends BasePage {
  // List page elements
  readonly pageTitle: Locator
  readonly newOrderButton: Locator
  readonly refreshButton: Locator
  readonly statusFilter: Locator
  readonly supplierFilter: Locator
  readonly dateRangeFilter: Locator
  readonly searchInput: Locator
  readonly tableRows: Locator
  readonly tableBody: Locator

  // Form page elements
  readonly supplierSelect: Locator
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
  readonly receiveButton: Locator
  readonly cancelOrderButton: Locator
  readonly editButton: Locator
  readonly backButton: Locator
  readonly receiveProgress: Locator

  // Receive page elements
  readonly receivePageTitle: Locator
  readonly receiveWarehouseSelect: Locator
  readonly receiveAllButton: Locator
  readonly clearAllButton: Locator
  readonly receiveSubmitButton: Locator
  readonly receiveCancelButton: Locator
  readonly receivableItemsTable: Locator

  constructor(page: Page) {
    super(page)

    // List page
    this.pageTitle = page.locator('h4').filter({ hasText: '采购订单' })
    this.newOrderButton = page.locator('button').filter({ hasText: '新建订单' })
    this.refreshButton = page.locator('button').filter({ hasText: '刷新' })
    this.statusFilter = page
      .locator('.semi-select')
      .filter({ hasText: /状态|全部状态/ })
      .first()
    this.supplierFilter = page
      .locator('.semi-select')
      .filter({ hasText: /供应商|全部供应商/ })
      .first()
    this.dateRangeFilter = page.locator('.semi-datepicker-range')
    this.searchInput = page.locator('input[placeholder*="搜索订单编号"]')
    this.tableRows = page.locator('.semi-table-tbody .semi-table-row')
    this.tableBody = page.locator('.semi-table-tbody')

    // Form page
    this.supplierSelect = page
      .locator('.semi-select')
      .filter({ hasText: /搜索并选择供应商|供应商/ })
      .first()
    this.warehouseSelect = page
      .locator('.semi-select')
      .filter({ hasText: /收货仓库|选择收货仓库/ })
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
    this.confirmButton = page.locator('button').filter({ hasText: '确认' })
    this.receiveButton = page.locator('button').filter({ hasText: '收货' })
    this.cancelOrderButton = page.locator('button').filter({ hasText: /取消$|取消订单/ })
    this.editButton = page.locator('button').filter({ hasText: '编辑' })
    this.backButton = page.locator('button').filter({ hasText: '返回列表' })
    this.receiveProgress = page.locator('.semi-progress')

    // Receive page
    this.receivePageTitle = page.locator('h4').filter({ hasText: '采购收货' })
    this.receiveWarehouseSelect = page.locator('.warehouse-selection .semi-select')
    this.receiveAllButton = page.locator('button').filter({ hasText: '全部收货' })
    this.clearAllButton = page.locator('button').filter({ hasText: '清空数量' })
    this.receiveSubmitButton = page.locator('button').filter({ hasText: '确认收货' })
    this.receiveCancelButton = page.locator('.actions-card button').filter({ hasText: '取消' })
    this.receivableItemsTable = page.locator('.items-card .semi-table')
  }

  // Navigation methods
  async navigateToList(): Promise<void> {
    await this.goto('/trade/purchase')
    await this.waitForPageLoad()
    await this.waitForTableLoad()
  }

  async navigateToNewOrder(): Promise<void> {
    await this.goto('/trade/purchase/new')
    await this.waitForPageLoad()
  }

  async navigateToDetail(orderId: string): Promise<void> {
    await this.goto(`/trade/purchase/${orderId}`)
    await this.waitForPageLoad()
  }

  async navigateToEdit(orderId: string): Promise<void> {
    await this.goto(`/trade/purchase/${orderId}/edit`)
    await this.waitForPageLoad()
  }

  async navigateToReceive(orderId: string): Promise<void> {
    await this.goto(`/trade/purchase/${orderId}/receive`)
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
      partial_received: '部分收货',
      completed: '已完成',
      cancelled: '已取消',
    }
    const label = statusLabels[status] || status
    await this.page.locator('.semi-select-option').filter({ hasText: label }).click()

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async filterBySupplier(supplierName: string): Promise<void> {
    // Find supplier filter (second select typically)
    const supplierSelect = this.page.locator('.semi-select').nth(1)
    await supplierSelect.click()
    await this.page.waitForTimeout(200)

    if (supplierName) {
      await this.page.locator('.semi-select-option').filter({ hasText: supplierName }).click()
    } else {
      await this.page.locator('.semi-select-option').filter({ hasText: '全部供应商' }).click()
    }

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async clickNewOrder(): Promise<void> {
    await this.newOrderButton.click()
    await this.page.waitForURL('**/trade/purchase/new')
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
    action: 'view' | 'edit' | 'confirm' | 'receive' | 'cancel' | 'delete'
  ): Promise<void> {
    // Scroll row into view first
    await row.scrollIntoViewIfNeeded()
    await this.page.waitForTimeout(100)

    // Hover over the row to show actions
    await row.hover()
    await this.page.waitForTimeout(200)

    // Find the action button by text
    const actionLabels: Record<string, string> = {
      view: '查看',
      edit: '编辑',
      confirm: '确认',
      receive: '收货',
      cancel: '取消',
      delete: '删除',
    }
    const actionText = actionLabels[action]

    // Try to click the action directly in the row
    const actionButton = row.locator('button.semi-button').filter({ hasText: actionText })
    const buttonVisible = await actionButton.isVisible().catch(() => false)

    if (buttonVisible) {
      await actionButton.click()
    } else {
      // Try to find in dropdown menu (more actions button is marked with data-testid)
      const moreButton = row.locator('[data-testid="table-row-more-actions"]')
      const moreButtonExists = await moreButton.isVisible().catch(() => false)

      if (moreButtonExists) {
        // Scroll more button into view to ensure dropdown positions correctly
        await moreButton.scrollIntoViewIfNeeded()
        await this.page.waitForTimeout(100)

        await moreButton.click()
        await this.page.waitForTimeout(300)

        // The dropdown menu uses menuitem role - use keyboard navigation
        // to ensure proper selection and React event handling
        const menuItem = this.page.getByRole('menuitem', { name: actionText })
        await menuItem.waitFor({ state: 'visible', timeout: 3000 })

        // Focus and click using keyboard (Enter key) to properly trigger React handlers
        await menuItem.focus()
        await this.page.keyboard.press('Enter')
      } else {
        throw new Error(
          `Action button "${actionText}" not found in row, and no more actions dropdown available`
        )
      }
    }

    await this.page.waitForTimeout(300)

    // Handle modal confirmation for actions that trigger modals (confirm, cancel, delete)
    if (['confirm', 'cancel', 'delete'].includes(action)) {
      // Wait for modal to appear
      const modal = this.page.locator('.semi-modal')
      await modal.waitFor({ state: 'visible', timeout: 5000 }).catch(() => {
        // Modal might not appear if action was directly executed
      })

      if (await modal.isVisible()) {
        // Click the confirm button in modal (primary or danger button)
        const confirmBtn = modal.locator(
          '.semi-modal-footer .semi-button-primary, .semi-modal-footer .semi-button-danger'
        )
        await confirmBtn.click()

        // Wait for modal to close
        await modal.waitFor({ state: 'hidden', timeout: 5000 }).catch(() => {})
        await this.page.waitForTimeout(500)
      }
    }
  }

  // Form page methods
  async selectSupplier(supplierName: string): Promise<void> {
    // Find supplier select input
    const supplierSelect = this.page
      .locator('.semi-select')
      .filter({ hasText: /搜索并选择供应商|供应商/ })
      .first()
    await supplierSelect.click()
    await this.page.waitForTimeout(300)

    // Type to search - Semi Design's combobox uses a textbox role
    // When the select opens, it creates an input for searching
    const searchInput = this.page.getByRole('textbox').first()
    await searchInput.fill(supplierName)
    await this.page.waitForTimeout(800) // Wait for debounced search API call

    // Wait for option to appear (increase timeout for API response)
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: supplierName })
      .first()
      .waitFor({ state: 'visible', timeout: 10000 })

    // Select the option
    await this.page.locator('.semi-select-option').filter({ hasText: supplierName }).first().click()
    await this.page.waitForTimeout(200)
  }

  async selectWarehouse(warehouseName: string): Promise<void> {
    const warehouseSelect = this.page
      .locator('.semi-select')
      .filter({ hasText: /收货仓库|选择收货仓库/ })
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
    await this.page.waitForTimeout(300)

    // Type to search - Semi Design's combobox uses a textbox role
    // When the select opens, it creates an input for searching
    const searchInput = this.page.getByRole('textbox').first()
    await searchInput.fill(productName)
    await this.page.waitForTimeout(800) // Wait for debounced search API call

    // Wait for option to appear (increase timeout for API response)
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: productName })
      .first()
      .waitFor({ state: 'visible', timeout: 10000 })

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

  async setUnitCostInRow(rowIndex: number, cost: number): Promise<void> {
    const row = this.page.locator('.semi-table-tbody .semi-table-row').nth(rowIndex)
    // First number input is unit cost
    const costInput = row.locator('.semi-input-number input').first()
    await costInput.clear()
    await costInput.fill(cost.toString())
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
    // Wait for button to be enabled and visible
    await this.submitButton.waitFor({ state: 'visible' })
    await expect(this.submitButton).toBeEnabled()

    // Scroll into view
    await this.submitButton.scrollIntoViewIfNeeded()
    await this.page.waitForTimeout(200)

    // Try to click and wait for network request
    const [response] = await Promise.all([
      this.page
        .waitForResponse(
          (resp) => resp.url().includes('/trade/purchase') && resp.request().method() === 'POST',
          { timeout: 15000 }
        )
        .catch(() => null), // Don't fail if no request is made
      this.submitButton.click({ force: true }),
    ])

    // If a response was received, check for errors
    if (response) {
      const status = response.status()
      if (status >= 400) {
        const body = await response.text().catch(() => 'Unable to read response body')
        throw new Error(`Order creation API failed with status ${status}: ${body}`)
      }
    } else {
      // Wait a bit to see if there's an error toast
      await this.page.waitForTimeout(1000)
    }
  }

  async waitForOrderCreateSuccess(): Promise<void> {
    // Wait for BOTH success indication AND navigation away from /new page
    // First wait for any toast or navigation indication
    try {
      await Promise.race([
        this.page.waitForURL(/\/trade\/purchase(?!\/new)/, { timeout: 20000 }),
        this.page.locator('.semi-toast-content').first().waitFor({ timeout: 20000 }),
      ])
    } catch {
      throw new Error('Order creation did not complete - no navigation or toast within 20 seconds')
    }

    // If still on /new page, wait for navigation to complete
    const currentUrl = this.page.url()
    if (currentUrl.endsWith('/new')) {
      await this.page.waitForURL(/\/trade\/purchase(?!\/new)/, { timeout: 10000 })
    }

    // Check for error toast
    const toast = this.page.locator('.semi-toast-content').first()
    if (await toast.isVisible()) {
      const toastText = await toast.textContent()
      if (
        toastText?.includes('错误') ||
        toastText?.includes('失败') ||
        toastText?.includes('Error')
      ) {
        throw new Error(`Order creation failed with error: ${toastText}`)
      }
    }

    // Wait for page to stabilize
    await this.page.waitForLoadState('domcontentloaded')
    await this.page.waitForTimeout(500)
  }

  // Detail page methods
  async getOrderStatus(): Promise<string> {
    const statusTag = this.page.locator('.page-header .semi-tag, .header-left .semi-tag').first()
    return (await statusTag.textContent()) || ''
  }

  async getOrderInfo(): Promise<{
    orderNumber: string
    supplierName: string
    status: string
    itemCount: number
    totalAmount: string
    payableAmount: string
  }> {
    const infoCard = this.page.locator('.info-card, .order-basic-info, .semi-descriptions')
    const text = (await infoCard.textContent()) || ''

    // Parse order info from the descriptions
    const orderNumberMatch = text.match(/订单编号[：:]?\s*(PO-[\w-]+)/)
    const supplierMatch = text.match(/供应商[：:]?\s*([^\s订单]+)/)
    const itemCountMatch = text.match(/商品数量[：:]?\s*(\d+)/)

    const status = await this.getOrderStatus()

    // Get amounts from summary section
    const amountText = text
    const totalMatch = amountText.match(/订单金额[：:]?\s*¥([\d,.]+)/)
    const payableMatch = amountText.match(/应付金额[：:]?\s*¥([\d,.]+)/)

    return {
      orderNumber: orderNumberMatch?.[1] || '',
      supplierName: supplierMatch?.[1] || '',
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

  async clickReceiveButton(): Promise<void> {
    await this.receiveButton.click()
    await this.page.waitForURL('**/receive')
    await this.waitForPageLoad()
  }

  async getReceiveProgress(): Promise<number> {
    const progressText = await this.receiveProgress.textContent()
    const match = progressText?.match(/(\d+)%/)
    return match ? parseInt(match[1]) : 0
  }

  // Receive page methods
  async selectReceiveWarehouse(warehouseName: string): Promise<void> {
    await this.receiveWarehouseSelect.click()
    await this.page.waitForTimeout(200)
    await this.page.locator('.semi-select-option').filter({ hasText: warehouseName }).click()
    await this.page.waitForTimeout(200)
  }

  async clickReceiveAll(): Promise<void> {
    await this.receiveAllButton.click()
    await this.page.waitForTimeout(200)
  }

  async clickClearAll(): Promise<void> {
    await this.clearAllButton.click()
    await this.page.waitForTimeout(200)
  }

  async setReceiveQuantity(rowIndex: number, quantity: number): Promise<void> {
    const row = this.receivableItemsTable.locator('.semi-table-row').nth(rowIndex)
    // Find the "本次收货数量" input (usually the input number in the row)
    const quantityInput = row.locator('.semi-input-number input')
    await quantityInput.clear()
    await quantityInput.fill(quantity.toString())
    await this.page.waitForTimeout(100)
  }

  async setBatchNumber(rowIndex: number, batchNumber: string): Promise<void> {
    const row = this.receivableItemsTable.locator('.semi-table-row').nth(rowIndex)
    const batchInput = row.locator('.semi-input').filter({ has: this.page.locator('input') })
    await batchInput.locator('input').fill(batchNumber)
    await this.page.waitForTimeout(100)
  }

  async submitReceive(): Promise<void> {
    await this.receiveSubmitButton.click()
  }

  async waitForReceiveSuccess(): Promise<void> {
    await Promise.race([
      this.page.waitForURL('**/trade/purchase', { timeout: 10000 }),
      this.waitForToast('收货'),
    ])
    await this.page.waitForTimeout(500)
  }

  async getReceivableItems(): Promise<
    Array<{
      productCode: string
      productName: string
      orderedQuantity: string
      receivedQuantity: string
      remainingQuantity: string
      unitCost: string
    }>
  > {
    const rows = this.receivableItemsTable.locator('.semi-table-row')
    const count = await rows.count()
    const items: Array<{
      productCode: string
      productName: string
      orderedQuantity: string
      receivedQuantity: string
      remainingQuantity: string
      unitCost: string
    }> = []

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i)
      const cells = row.locator('.semi-table-row-cell')

      items.push({
        productCode: (await cells.nth(0).textContent()) || '',
        productName: (await cells.nth(1).textContent()) || '',
        orderedQuantity: (await cells.nth(3).textContent()) || '',
        receivedQuantity: (await cells.nth(4).textContent()) || '',
        remainingQuantity: (await cells.nth(5).textContent()) || '',
        unitCost: (await cells.nth(6).textContent()) || '',
      })
    }

    return items
  }

  // Assertions
  async assertOrderListDisplayed(): Promise<void> {
    await expect(this.page.locator('h4').filter({ hasText: '采购订单' })).toBeVisible()
  }

  async assertOrderFormDisplayed(): Promise<void> {
    await expect(
      this.page.locator('h4').filter({ hasText: /新建采购订单|编辑采购订单/ })
    ).toBeVisible()
  }

  async assertOrderDetailDisplayed(): Promise<void> {
    await expect(this.page.locator('h4').filter({ hasText: '订单详情' })).toBeVisible()
  }

  async assertReceivePageDisplayed(): Promise<void> {
    await expect(this.receivePageTitle).toBeVisible()
  }

  async assertOrderStatus(expectedStatus: string): Promise<void> {
    const statusLabels: Record<string, string> = {
      draft: '草稿',
      confirmed: '已确认',
      partial_received: '部分收货',
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
        partial_received: '部分收货',
        completed: '已完成',
        cancelled: '已取消',
      }
      const expectedLabel = statusLabels[expectedStatus] || expectedStatus
      await expect(row.locator('.semi-tag')).toContainText(expectedLabel)
    }
  }

  async assertReceiveProgressVisible(): Promise<void> {
    await expect(this.receiveProgress).toBeVisible()
  }

  // Screenshot methods
  async screenshotOrderList(name: string): Promise<void> {
    await this.screenshot(`purchase-orders/${name}`)
  }

  async screenshotOrderForm(name: string): Promise<void> {
    await this.screenshot(`purchase-orders/${name}`)
  }

  async screenshotOrderDetail(name: string): Promise<void> {
    await this.screenshot(`purchase-orders/${name}`)
  }

  async screenshotReceivePage(name: string): Promise<void> {
    await this.screenshot(`purchase-orders/${name}`)
  }
}
