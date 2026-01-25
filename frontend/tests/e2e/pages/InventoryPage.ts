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
    // Find the warehouse filter select
    const warehouseSelect = this.page.locator('.semi-select').first()
    await warehouseSelect.click()
    await this.page.waitForTimeout(200)

    // Select the option
    if (warehouseName) {
      await this.page.locator('.semi-select-option').filter({ hasText: warehouseName }).click()
    } else {
      await this.page.locator('.semi-select-option').filter({ hasText: '全部仓库' }).click()
    }

    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  async filterByStockStatus(status: string): Promise<void> {
    // Find the status filter select (second select)
    const statusSelect = this.page.locator('.semi-select').nth(1)
    await statusSelect.click()
    await this.page.waitForTimeout(200)

    // Select the option
    const optionText =
      status === 'has_stock'
        ? '有库存'
        : status === 'below_minimum'
          ? '低库存预警'
          : status === 'no_stock'
            ? '无库存'
            : '全部状态'
    await this.page.locator('.semi-select-option').filter({ hasText: optionText }).click()

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
    const select = this.page
      .locator('.semi-select')
      .filter({ has: this.page.locator('label:has-text("仓库")') })
      .locator('.semi-select-selection')
    await select.click()
    await this.page.waitForTimeout(200)
    await this.page.locator('.semi-select-option').filter({ hasText: warehouseName }).click()
    await this.page.waitForTimeout(300)
  }

  async selectProduct(productName: string): Promise<void> {
    const select = this.page
      .locator('.semi-select')
      .filter({ has: this.page.locator('label:has-text("商品")') })
      .locator('.semi-select-selection')
    await select.click()
    await this.page.waitForTimeout(200)
    await this.page.locator('.semi-select-option').filter({ hasText: productName }).click()
    await this.page.waitForTimeout(300)
  }

  async fillAdjustmentForm(data: {
    actualQuantity: number
    reason: string
    notes?: string
  }): Promise<void> {
    // Fill actual quantity
    const quantityInput = this.page
      .locator('input')
      .filter({ has: this.page.locator('[placeholder*="实际"]') })
      .first()
      .or(this.page.locator('input[type="number"]').first())
    await quantityInput.fill(data.actualQuantity.toString())

    // Select reason
    const reasonSelect = this.page
      .locator('.semi-select')
      .filter({ has: this.page.locator('label:has-text("调整原因")') })
      .locator('.semi-select-selection')
    await reasonSelect.click()
    await this.page.waitForTimeout(200)
    await this.page.locator('.semi-select-option').filter({ hasText: data.reason }).click()

    // Fill notes if provided
    if (data.notes) {
      const notesTextarea = this.page.locator('textarea')
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
}
