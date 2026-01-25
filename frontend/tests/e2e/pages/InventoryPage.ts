import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * InventoryPage - Page Object for inventory module E2E tests
 *
 * Provides methods for:
 * - Stock list navigation and interactions
 * - Stock filtering by warehouse/product/status
 * - Stock adjustment operations
 * - Transaction history viewing
 * - Stocktaking operations
 */
export class InventoryPage extends BasePage {
  // Current stock taking number (set after creation)
  currentStockTakingNumber: string = ''

  // Stock list page elements
  readonly stockListTitle: Locator
  readonly warehouseFilter: Locator
  readonly stockStatusFilter: Locator
  readonly searchInput: Locator
  readonly refreshButton: Locator
  readonly tableRows: Locator
  readonly tableBody: Locator

  // Stock adjustment page elements
  readonly warehouseSelect: Locator
  readonly productSelect: Locator
  readonly actualQuantityInput: Locator
  readonly adjustmentReasonSelect: Locator
  readonly notesInput: Locator
  readonly submitButton: Locator
  readonly cancelButton: Locator
  readonly adjustmentPreview: Locator

  // Transaction page elements
  readonly transactionTypeFilter: Locator
  readonly dateRangePicker: Locator
  readonly transactionTable: Locator

  constructor(page: Page) {
    super(page)

    // Stock list elements
    this.stockListTitle = page.locator('h4').filter({ hasText: '库存查询' })
    this.warehouseFilter = page
      .locator('.semi-select')
      .filter({ hasText: /仓库|全部仓库/ })
      .first()
    this.stockStatusFilter = page
      .locator('.semi-select')
      .filter({ hasText: /状态|全部状态/ })
      .first()
    this.searchInput = page.locator('input[placeholder*="搜索"]')
    this.refreshButton = page.locator('button').filter({ hasText: '刷新' })
    this.tableRows = page.locator('.semi-table-tbody .semi-table-row')
    this.tableBody = page.locator('.semi-table-tbody')

    // Stock adjustment elements
    this.warehouseSelect = page
      .locator('[data-testid="warehouse-select"], .semi-select')
      .filter({ has: page.locator('label:has-text("仓库")') })
    this.productSelect = page
      .locator('[data-testid="product-select"], .semi-select')
      .filter({ has: page.locator('label:has-text("商品")') })
    this.actualQuantityInput = page.locator(
      'input[name="actual_quantity"], input[placeholder*="实际"]'
    )
    this.adjustmentReasonSelect = page
      .locator('.semi-select')
      .filter({ has: page.locator('label:has-text("调整原因")') })
    this.notesInput = page.locator('textarea[name="source_id"], textarea[placeholder*="备注"]')
    this.submitButton = page.locator('button[type="submit"], button:has-text("确认调整")')
    this.cancelButton = page.locator('button:has-text("取消")')
    this.adjustmentPreview = page.locator('.adjustment-preview')

    // Transaction page elements
    this.transactionTypeFilter = page.locator('.semi-select').filter({ hasText: /类型|全部类型/ })
    this.dateRangePicker = page.locator('.semi-datepicker-range')
    this.transactionTable = page.locator('.semi-table')
  }

  // Navigation methods
  async navigateToStockList(): Promise<void> {
    await this.goto('/inventory/stock')
    await this.waitForPageLoad()
    await this.waitForTableLoad()
  }

  async navigateToStockDetail(itemId: string): Promise<void> {
    await this.goto(`/inventory/stock/${itemId}`)
    await this.waitForPageLoad()
  }

  async navigateToStockTransactions(itemId: string): Promise<void> {
    await this.goto(`/inventory/stock/${itemId}/transactions`)
    await this.waitForPageLoad()
    await this.waitForTableLoad()
  }

  async navigateToStockAdjust(warehouseId?: string, productId?: string): Promise<void> {
    let url = '/inventory/adjust'
    const params = new URLSearchParams()
    if (warehouseId) params.set('warehouse_id', warehouseId)
    if (productId) params.set('product_id', productId)
    if (params.toString()) url += `?${params.toString()}`

    await this.goto(url)
    await this.waitForPageLoad()
  }

  async navigateToStockTakingList(): Promise<void> {
    await this.goto('/inventory/stock-taking')
    await this.waitForPageLoad()
  }

  async navigateToStockTakingCreate(): Promise<void> {
    await this.goto('/inventory/stock-taking/new')
    await this.waitForPageLoad()
  }

  // Stock list methods
  async getStockCount(): Promise<number> {
    await this.waitForTableLoad()
    return this.tableRows.count()
  }

  async search(keyword: string): Promise<void> {
    await this.searchInput.fill(keyword)
    await this.page.waitForTimeout(500) // Debounce
    await this.waitForTableLoad()
  }

  async clearSearch(): Promise<void> {
    await this.searchInput.clear()
    await this.page.waitForTimeout(500)
    await this.waitForTableLoad()
  }

  async filterByWarehouse(warehouseName: string): Promise<void> {
    // Find the warehouse filter select within the toolbar filters section
    // First try to locate within the filters wrapper, fall back to first select
    const filtersWrapper = this.page.locator('.table-toolbar-filters')
    let warehouseSelect = filtersWrapper.locator('.semi-select').first()

    // If no filters wrapper, fall back to first select on page
    if (!(await filtersWrapper.count())) {
      warehouseSelect = this.page.locator('.semi-select').first()
    }

    // Ensure the select is visible before clicking
    await warehouseSelect.waitFor({ state: 'visible', timeout: 5000 })
    await warehouseSelect.click()

    // Wait for the dropdown portal to appear
    await this.page.waitForTimeout(300)

    // Wait for options to load (Semi uses portal for dropdown, so look globally)
    const optionToSelect = warehouseName
      ? this.page.locator('.semi-select-option').filter({ hasText: warehouseName })
      : this.page.locator('.semi-select-option').filter({ hasText: '全部仓库' })

    // Wait for options to appear and not be "暂无数据"
    try {
      await optionToSelect.waitFor({ state: 'visible', timeout: 10000 })
      await optionToSelect.click()
    } catch {
      // If option not found, check if dropdown is actually open
      const dropdownOptions = this.page.locator('.semi-select-option')
      const optionCount = await dropdownOptions.count()
      console.log(`Warehouse filter: ${optionCount} options found. Looking for: ${warehouseName}`)

      // Try clicking again with scroll
      if (optionCount > 0) {
        const optionTexts = await dropdownOptions.allTextContents()
        console.log(`Available options: ${optionTexts.join(', ')}`)
      }
      throw new Error(`Warehouse option "${warehouseName}" not found in dropdown`)
    }

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async filterByStockStatus(status: string): Promise<void> {
    // Find the status filter select (second select)
    const statusSelect = this.page.locator('.semi-select').nth(1)
    await statusSelect.click()

    // Select the option
    const optionText =
      status === 'has_stock'
        ? '有库存'
        : status === 'below_minimum'
          ? '低库存预警'
          : status === 'no_stock'
            ? '无库存'
            : '全部状态'

    const optionToSelect = this.page.locator('.semi-select-option').filter({ hasText: optionText })
    await optionToSelect.waitFor({ state: 'visible', timeout: 10000 })
    await optionToSelect.click()

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async refresh(): Promise<void> {
    await this.refreshButton.click()
    await this.waitForTableLoad()
  }

  async getInventoryRow(index: number): Promise<Locator> {
    return this.tableRows.nth(index)
  }

  async getInventoryRowByProductName(productName: string): Promise<Locator | null> {
    await this.waitForTableLoad()
    const rows = this.tableRows
    const count = await rows.count()

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i)
      const text = await row.textContent()
      if (text?.includes(productName)) {
        return row
      }
    }
    return null
  }

  async getQuantitiesFromRow(row: Locator): Promise<{
    available: number
    locked: number
    total: number
  }> {
    const cells = row.locator('.semi-table-row-cell')
    // Based on StockList.tsx columns order:
    // 0: warehouse, 1: product, 2: available, 3: locked, 4: total, 5: unit_cost, 6: total_value, 7: status, 8: updated_at

    const availableText = (await cells.nth(2).textContent()) || '0'
    const lockedText = (await cells.nth(3).textContent()) || '0'
    const totalText = (await cells.nth(4).textContent()) || '0'

    return {
      available: parseFloat(availableText.replace(/[^\d.-]/g, '')) || 0,
      locked: parseFloat(lockedText.replace(/[^\d.-]/g, '')) || 0,
      total: parseFloat(totalText.replace(/[^\d.-]/g, '')) || 0,
    }
  }

  async clickRowAction(row: Locator, action: 'view' | 'transactions' | 'adjust'): Promise<void> {
    // Hover over the row to show actions
    await row.hover()
    await this.page.waitForTimeout(200)

    // Click the action button
    const actionText =
      action === 'view' ? '查看明细' : action === 'transactions' ? '流水记录' : '库存调整'

    // Find the action dropdown/menu
    const actionButton = row
      .locator('.semi-dropdown-trigger, .semi-button')
      .filter({ hasText: '操作' })
    if (await actionButton.isVisible()) {
      await actionButton.click()
      await this.page.waitForTimeout(200)
      await this.page
        .locator('.semi-dropdown-menu .semi-dropdown-item')
        .filter({ hasText: actionText })
        .click()
    } else {
      // Actions might be direct links
      await row.locator(`a, button`).filter({ hasText: actionText }).click()
    }
  }

  // Stock adjustment methods
  async selectWarehouse(warehouseName: string): Promise<void> {
    // Find the form-field-wrapper containing the warehouse label, then find the select inside
    // The label might be in <label> tag or direct text node with class
    const wrapper = this.page.locator('.form-field-wrapper').filter({ hasText: '仓库' }).first()
    const select = wrapper.locator('.semi-select')
    await select.click()

    const optionToSelect = this.page
      .locator('.semi-select-option')
      .filter({ hasText: warehouseName })
    await optionToSelect.waitFor({ state: 'visible', timeout: 10000 })
    await optionToSelect.click()
    await this.page.waitForTimeout(300)
  }

  async selectProduct(productName: string): Promise<void> {
    // Find the form-field-wrapper containing the product label, then find the select inside
    // The label might be in <label> tag or direct text node with class
    const wrapper = this.page.locator('.form-field-wrapper').filter({ hasText: '商品' }).first()
    const select = wrapper.locator('.semi-select')
    await select.click()

    const optionToSelect = this.page.locator('.semi-select-option').filter({ hasText: productName })
    await optionToSelect.waitFor({ state: 'visible', timeout: 10000 })
    await optionToSelect.click()
    await this.page.waitForTimeout(300)
  }

  async fillAdjustmentForm(data: {
    actualQuantity: number
    reason: string
    notes?: string
  }): Promise<void> {
    // Fill actual quantity - Semi Design InputNumber uses spinbutton role
    const quantityWrapper = this.page
      .locator('.form-field-wrapper')
      .filter({ hasText: '实际数量' })
      .first()
    // Try spinbutton first (Semi Design InputNumber), fallback to regular input
    let quantityInput = quantityWrapper.locator('[role="spinbutton"]')
    if (!(await quantityInput.isVisible().catch(() => false))) {
      quantityInput = quantityWrapper.locator('.semi-input-number input, input').first()
    }
    await quantityInput.clear()
    await quantityInput.fill(data.actualQuantity.toString())

    // Select reason - find select in the form-field-wrapper with reason label
    const reasonWrapper = this.page
      .locator('.form-field-wrapper')
      .filter({ hasText: '调整原因' })
      .first()
    const reasonSelect = reasonWrapper.locator('.semi-select')
    await reasonSelect.click()
    await this.page.waitForTimeout(200)
    await this.page.locator('.semi-select-option').filter({ hasText: data.reason }).click()

    // Fill notes if provided
    if (data.notes) {
      const notesWrapper = this.page
        .locator('.form-field-wrapper')
        .filter({ hasText: '备注' })
        .first()
      const notesTextarea = notesWrapper.locator('textarea')
      await notesTextarea.fill(data.notes)
    }
  }

  async submitAdjustment(): Promise<void> {
    await this.submitButton.click()
  }

  async waitForAdjustmentSuccess(): Promise<void> {
    // Wait for success toast or redirect
    await this.page.waitForURL(/\/inventory\/stock/, { timeout: 10000 }).catch(async () => {
      // Alternative: wait for success toast
      await this.waitForToast('成功')
    })
  }

  async getAdjustmentPreview(): Promise<{
    currentQuantity: number
    newQuantity: number
    difference: number
  }> {
    const preview = this.page.locator('.adjustment-preview, .preview-row')
    const text = (await preview.textContent()) || ''

    // Parse the preview values
    const currentMatch = text.match(/当前数量[：:]?\s*([\d.]+)/)
    const newMatch = text.match(/调整后数量[：:]?\s*([\d.]+)/)
    const diffMatch = text.match(/变动数量[：:]?\s*([+-]?[\d.]+)/)

    return {
      currentQuantity: parseFloat(currentMatch?.[1] || '0'),
      newQuantity: parseFloat(newMatch?.[1] || '0'),
      difference: parseFloat(diffMatch?.[1] || '0'),
    }
  }

  // Transaction history methods
  async filterTransactionsByType(type: string): Promise<void> {
    const typeSelect = this.page.locator('.semi-select').filter({ hasText: /类型/ })
    await typeSelect.click()
    await this.page.waitForTimeout(200)

    const optionText =
      type === 'INBOUND'
        ? '入库'
        : type === 'OUTBOUND'
          ? '出库'
          : type === 'LOCK'
            ? '锁定'
            : type === 'UNLOCK'
              ? '解锁'
              : type === 'ADJUSTMENT'
                ? '调整'
                : '全部类型'
    await this.page.locator('.semi-select-option').filter({ hasText: optionText }).click()
    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async getTransactionCount(): Promise<number> {
    await this.waitForTableLoad()
    return this.tableRows.count()
  }

  async getTransactionRow(index: number): Promise<Locator> {
    return this.tableRows.nth(index)
  }

  async getTransactionDetails(row: Locator): Promise<{
    date: string
    type: string
    quantity: string
    balanceBefore: string
    balanceAfter: string
    sourceType: string
  }> {
    const cells = row.locator('.semi-table-row-cell')

    return {
      date: (await cells.nth(0).textContent()) || '',
      type: (await cells.nth(1).textContent()) || '',
      quantity: (await cells.nth(2).textContent()) || '',
      balanceBefore: (await cells.nth(3).textContent()) || '',
      balanceAfter: (await cells.nth(4).textContent()) || '',
      sourceType: (await cells.nth(7).textContent()) || '',
    }
  }

  // Assertion methods
  async assertStockListDisplayed(): Promise<void> {
    await expect(this.page.locator('h4').filter({ hasText: '库存查询' })).toBeVisible()
  }

  async assertStockExists(productName: string): Promise<void> {
    const row = await this.getInventoryRowByProductName(productName)
    expect(row).not.toBeNull()
  }

  async assertStockNotExists(productName: string): Promise<void> {
    const row = await this.getInventoryRowByProductName(productName)
    expect(row).toBeNull()
  }

  async assertQuantities(
    productName: string,
    expected: {
      available?: number
      locked?: number
      total?: number
    }
  ): Promise<void> {
    const row = await this.getInventoryRowByProductName(productName)
    expect(row).not.toBeNull()

    if (row) {
      const quantities = await this.getQuantitiesFromRow(row)

      if (expected.available !== undefined) {
        expect(quantities.available).toBeCloseTo(expected.available, 1)
      }
      if (expected.locked !== undefined) {
        expect(quantities.locked).toBeCloseTo(expected.locked, 1)
      }
      if (expected.total !== undefined) {
        expect(quantities.total).toBeCloseTo(expected.total, 1)
      }
    }
  }

  async assertLowStockWarning(productName: string): Promise<void> {
    const row = await this.getInventoryRowByProductName(productName)
    expect(row).not.toBeNull()

    if (row) {
      const warningTag = row.locator('.semi-tag').filter({ hasText: '低库存' })
      await expect(warningTag).toBeVisible()
    }
  }

  async assertTransactionType(row: Locator, expectedType: string): Promise<void> {
    const typeCell = row.locator('.semi-tag')
    await expect(typeCell).toContainText(expectedType)
  }

  // Screenshot methods
  async screenshotStockList(name: string): Promise<void> {
    await this.screenshot(`inventory/${name}`)
  }

  async screenshotAdjustment(name: string): Promise<void> {
    await this.screenshot(`inventory/${name}`)
  }

  async screenshotTransactions(name: string): Promise<void> {
    await this.screenshot(`inventory/${name}`)
  }

  async screenshotStockTaking(name: string): Promise<void> {
    await this.screenshot(`inventory/${name}`)
  }

  // ==================== Stock Taking Methods ====================

  /**
   * Navigate to stock taking list page
   */
  async navigateToStockTakingListPage(): Promise<void> {
    await this.goto('/inventory/stock-taking')
    await this.waitForPageLoad()
  }

  /**
   * Navigate to stock taking create page
   */
  async navigateToStockTakingCreatePage(): Promise<void> {
    await this.goto('/inventory/stock-taking/new')
    await this.waitForPageLoad()
  }

  /**
   * Navigate to stock taking execute page
   */
  async navigateToStockTakingExecute(stockTakingId: string): Promise<void> {
    await this.goto(`/inventory/stock-taking/${stockTakingId}/execute`)
    await this.waitForPageLoad()
  }

  /**
   * Click "新建盘点" button on stock taking list page
   */
  async clickNewStockTaking(): Promise<void> {
    const newButton = this.page.locator('button').filter({ hasText: '新建盘点' })
    await newButton.click()
    await this.waitForPageLoad()
  }

  /**
   * Select warehouse in stock taking create form
   */
  async selectStockTakingWarehouse(warehouseName: string): Promise<void> {
    // Find the form-field-wrapper containing the warehouse label
    const wrapper = this.page.locator('.form-field-wrapper').filter({ hasText: '仓库' }).first()
    const select = wrapper.locator('.semi-select')
    await select.click()
    await this.page.waitForTimeout(300)

    const optionToSelect = this.page
      .locator('.semi-select-option')
      .filter({ hasText: warehouseName })
    await optionToSelect.waitFor({ state: 'visible', timeout: 10000 })
    await optionToSelect.click()
    await this.page.waitForTimeout(500) // Wait for inventory to load
  }

  /**
   * Click "全部导入" button to import all inventory items
   */
  async clickImportAllProducts(): Promise<void> {
    const importButton = this.page.locator('button').filter({ hasText: '全部导入' })
    await importButton.click()
    await this.waitForToast('已导入')
  }

  /**
   * Click "选择商品" button to open product selection modal
   */
  async clickSelectProducts(): Promise<void> {
    const selectButton = this.page.locator('button').filter({ hasText: '选择商品' })
    await selectButton.click()
    await this.page.waitForTimeout(300)
  }

  /**
   * Select products in the product selection modal
   */
  async selectProductsInModal(productNames: string[]): Promise<void> {
    for (const productName of productNames) {
      const row = this.page
        .locator('.semi-table-tbody .semi-table-row')
        .filter({ hasText: productName })
      const checkbox = row.locator('.semi-checkbox')
      await checkbox.click()
    }
  }

  /**
   * Confirm product selection in modal
   */
  async confirmProductSelection(): Promise<void> {
    const confirmButton = this.page
      .locator('.semi-modal-footer button')
      .filter({ hasText: '确认选择' })
    await confirmButton.click()
    await this.page.waitForTimeout(300)
  }

  /**
   * Get the count of selected products in the create form
   */
  async getSelectedProductCount(): Promise<number> {
    const countText = await this.page.locator('text=/已选择 \\d+ 个商品/').textContent()
    const match = countText?.match(/已选择 (\d+) 个商品/)
    return match ? parseInt(match[1], 10) : 0
  }

  /**
   * Submit the stock taking create form
   */
  async submitStockTakingCreate(): Promise<void> {
    const submitButton = this.page
      .locator('button[type="submit"]')
      .filter({ hasText: '创建盘点单' })
    await submitButton.click()
  }

  /**
   * Wait for stock taking creation success and return the taking number
   * Also stores the taking number in currentStockTakingNumber for later use
   * @returns The taking number of the newly created stock taking
   */
  async waitForStockTakingCreateSuccess(): Promise<string> {
    await this.waitForToast('创建成功')
    // Wait for URL change and network to be idle to ensure redirect is complete
    await this.page.waitForURL(/\/inventory\/stock-taking$/, {
      timeout: 10000,
      waitUntil: 'networkidle',
    })
    // Extra wait for table to render
    await this.page.waitForTimeout(500)
    // Get the taking number from the first row (most recent)
    await this.waitForTableLoad()
    const firstRow = this.tableRows.first()
    const takingNumber = await this.getStockTakingNumberFromRow(firstRow)
    this.currentStockTakingNumber = takingNumber.trim()
    return this.currentStockTakingNumber
  }

  /**
   * Get the current stock taking row (by stored taking number)
   * @returns The row locator for the current stock taking
   */
  async getCurrentStockTakingRow(): Promise<Locator> {
    if (!this.currentStockTakingNumber) {
      throw new Error('No current stock taking number set. Create a stock taking first.')
    }
    const row = await this.findStockTakingByNumber(this.currentStockTakingNumber)
    if (!row) {
      throw new Error(`Stock taking with number ${this.currentStockTakingNumber} not found`)
    }
    return row
  }

  /**
   * Get the first stock taking row in the list
   */
  async getStockTakingRow(index: number): Promise<Locator> {
    await this.waitForTableLoad()
    return this.tableRows.nth(index)
  }

  /**
   * Get stock taking number from a row
   */
  async getStockTakingNumberFromRow(row: Locator): Promise<string> {
    const numberCell = row.locator('.semi-table-row-cell').first()
    return (await numberCell.textContent()) || ''
  }

  /**
   * Click execute action on a stock taking row
   */
  async clickStockTakingExecute(row: Locator): Promise<void> {
    // First hover to reveal action buttons
    await row.hover()
    await this.page.waitForTimeout(200)

    // Look for action dropdown or direct link
    const actionDropdown = row.locator('.semi-dropdown-trigger, button').filter({ hasText: '操作' })
    if (await actionDropdown.isVisible()) {
      await actionDropdown.click()
      await this.page.waitForTimeout(200)
      await this.page
        .locator('.semi-dropdown-menu .semi-dropdown-item')
        .filter({ hasText: '执行' })
        .click()
    } else {
      // Direct link
      await row.locator('a, button').filter({ hasText: '执行' }).click()
    }
    await this.waitForPageLoad()
  }

  /**
   * Click "开始盘点" button on execute page
   * If the button is not visible (already counting), skip and verify status
   */
  async clickStartCounting(): Promise<void> {
    const startButton = this.page.locator('button').filter({ hasText: '开始盘点' })
    // Try to wait for the button to appear (it should be visible in DRAFT status)
    try {
      await startButton.waitFor({ state: 'visible', timeout: 3000 })
      await startButton.click()
      await this.waitForToast('开始')
    } catch {
      // Button not found or timed out - might already be in COUNTING status
    }
    // Verify we're now in COUNTING status
    const status = await this.getStockTakingStatus()
    if (!status.includes('盘点中') && !status.includes('COUNTING')) {
      throw new Error(`Expected status to be COUNTING but got: ${status}`)
    }
  }

  /**
   * Enter actual quantity for a product in stock taking
   */
  async enterActualQuantity(productCode: string, quantity: number): Promise<void> {
    // Find the row with the product code
    const row = this.page
      .locator('.semi-table-tbody .semi-table-row')
      .filter({ hasText: productCode })

    // Find the InputNumber in that row (actual quantity column)
    const input = row.locator('.semi-input-number input, [role="spinbutton"]').first()
    await input.clear()
    await input.fill(quantity.toString())
    await this.page.waitForTimeout(200)
  }

  /**
   * Save all counts in stock taking
   */
  async clickSaveAllCounts(): Promise<void> {
    const saveButton = this.page.locator('button').filter({ hasText: '保存全部' })
    // Wait for button to be enabled (there are items to save)
    await saveButton.waitFor({ state: 'visible', timeout: 5000 })
    // The button might be disabled if no counts have changed
    const isEnabled = await saveButton.isEnabled()
    if (isEnabled) {
      await saveButton.click()
      await this.waitForToast('保存')
      // Wait for the save to complete and button state to update
      await this.page.waitForTimeout(500)
    }
  }

  /**
   * Submit stock taking for approval
   */
  async clickSubmitForApproval(): Promise<void> {
    const submitButton = this.page.locator('button').filter({ hasText: '提交审批' })
    // Wait for button to become enabled (all items must be counted)
    await submitButton.waitFor({ state: 'visible', timeout: 5000 })
    // Wait for button to be enabled using locator assertion
    await expect(submitButton).toBeEnabled({ timeout: 10000 })
    await submitButton.click()
    await this.page.waitForTimeout(300)
  }

  /**
   * Confirm submit for approval in modal
   */
  async confirmSubmitForApproval(): Promise<void> {
    // Find the modal and click confirm
    const modal = this.page.locator('.semi-modal')
    const confirmButton = modal.locator('button').filter({ hasText: '确认提交' })
    await confirmButton.click()
    await this.waitForToast('提交')
  }

  /**
   * Get the progress percentage from execute page
   */
  async getStockTakingProgress(): Promise<number> {
    const progressText = await this.page.locator('.semi-progress').textContent()
    const match = progressText?.match(/(\d+)%/)
    return match ? parseInt(match[1], 10) : 0
  }

  /**
   * Get the total difference amount from execute page
   */
  async getTotalDifferenceAmount(): Promise<string> {
    const diffElement = this.page.locator('.total-difference .total-value, .total-value')
    return (await diffElement.textContent()) || '0'
  }

  /**
   * Verify difference is calculated correctly for a product
   */
  async verifyDifferenceForProduct(productCode: string, expectedDiff: number): Promise<boolean> {
    const row = this.page
      .locator('.semi-table-tbody .semi-table-row')
      .filter({ hasText: productCode })

    // The difference quantity column (should be around index 5)
    const cells = row.locator('.semi-table-row-cell')
    const diffQtyText = await cells.nth(5).textContent()

    // Parse the difference value
    const diffMatch = diffQtyText?.match(/([+-]?[\d.]+)/)
    const actualDiff = diffMatch ? parseFloat(diffMatch[1]) : 0

    return Math.abs(actualDiff - expectedDiff) < 0.01
  }

  /**
   * Get stock taking status from the header
   */
  async getStockTakingStatus(): Promise<string> {
    const statusTag = this.page.locator('.stock-taking-execute-header .semi-tag')
    return (await statusTag.textContent()) || ''
  }

  /**
   * Filter stock taking list by status
   */
  async filterStockTakingByStatus(status: string): Promise<void> {
    // Find the status filter select
    const statusSelect = this.page.locator('.semi-select').filter({ hasText: /状态|盘点状态/ })
    await statusSelect.click()
    await this.page.waitForTimeout(200)

    const optionText =
      status === 'DRAFT'
        ? '草稿'
        : status === 'COUNTING'
          ? '盘点中'
          : status === 'PENDING_APPROVAL'
            ? '待审批'
            : status === 'APPROVED'
              ? '已通过'
              : status === 'REJECTED'
                ? '已拒绝'
                : status === 'CANCELLED'
                  ? '已取消'
                  : '全部状态'

    const option = this.page.locator('.semi-select-option').filter({ hasText: optionText })
    await option.waitFor({ state: 'visible', timeout: 5000 })
    await option.click()
    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  /**
   * Assert stock taking list is displayed
   */
  async assertStockTakingListDisplayed(): Promise<void> {
    await expect(this.page.locator('h4').filter({ hasText: '盘点管理' })).toBeVisible()
  }

  /**
   * Get the count of stock taking items in the list
   */
  async getStockTakingItemCount(): Promise<number> {
    await this.waitForTableLoad()
    return this.tableRows.count()
  }

  /**
   * Find a stock taking row by its number
   */
  async findStockTakingByNumber(takingNumber: string): Promise<Locator | null> {
    await this.waitForTableLoad()
    const rows = this.tableRows
    const count = await rows.count()

    for (let i = 0; i < count; i++) {
      const row = rows.nth(i)
      const text = await row.textContent()
      if (text?.includes(takingNumber)) {
        return row
      }
    }
    return null
  }
}
