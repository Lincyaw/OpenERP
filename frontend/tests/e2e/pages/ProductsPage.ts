import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * ProductsPage - Page Object for Product management pages
 *
 * Covers:
 * - Product list page (/catalog/products)
 * - Product create page (/catalog/products/new)
 * - Product edit page (/catalog/products/:id/edit)
 */
export class ProductsPage extends BasePage {
  // List page elements
  readonly pageTitle: Locator
  readonly searchInput: Locator
  readonly statusFilter: Locator
  readonly addProductButton: Locator
  readonly refreshButton: Locator
  readonly table: Locator
  readonly tableRows: Locator
  readonly bulkActionBar: Locator
  readonly emptyState: Locator

  // Form page elements
  readonly codeInput: Locator
  readonly nameInput: Locator
  readonly unitInput: Locator
  readonly barcodeInput: Locator
  readonly descriptionInput: Locator
  readonly purchasePriceInput: Locator
  readonly sellingPriceInput: Locator
  readonly minStockInput: Locator
  readonly sortOrderInput: Locator
  readonly submitButton: Locator
  readonly cancelButton: Locator

  constructor(page: Page) {
    super(page)

    // List page elements
    this.pageTitle = page.locator('.products-header h4, .product-form-header h4')
    this.searchInput = page.locator('.semi-input-wrapper input[placeholder*="搜索"]')
    this.statusFilter = page.locator('.semi-select').filter({ hasText: /状态/ })
    this.addProductButton = page.locator('button').filter({ hasText: '新增商品' })
    this.refreshButton = page.locator('button').filter({ hasText: '刷新' })
    this.table = page.locator('.semi-table')
    this.tableRows = page.locator('.semi-table-tbody .semi-table-row')
    this.bulkActionBar = page.locator('.bulk-action-bar')
    this.emptyState = page.locator('.semi-table-empty')

    // Form elements - using name attributes set by React Hook Form
    this.codeInput = page.locator('input[name="code"]')
    this.nameInput = page.locator('input[name="name"]')
    this.unitInput = page.locator('input[name="unit"]')
    this.barcodeInput = page.locator('input[name="barcode"]')
    this.descriptionInput = page.locator('textarea[name="description"]')
    this.purchasePriceInput = page.locator('input[name="purchase_price"]')
    this.sellingPriceInput = page.locator('input[name="selling_price"]')
    this.minStockInput = page.locator('input[name="min_stock"]')
    this.sortOrderInput = page.locator('input[name="sort_order"]')
    // Semi UI Button with htmlType="submit" renders with type="submit"
    this.submitButton = page.locator('button[type="submit"]')
    this.cancelButton = page.locator('button').filter({ hasText: '取消' })
  }

  /**
   * Navigate to products list page
   */
  async navigateToList(): Promise<void> {
    await this.goto('/catalog/products')
    await this.waitForTableLoad()
  }

  /**
   * Navigate to create product page
   */
  async navigateToCreate(): Promise<void> {
    await this.goto('/catalog/products/new')
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Navigate to edit product page
   */
  async navigateToEdit(productId: string): Promise<void> {
    await this.goto(`/catalog/products/${productId}/edit`)
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Click add product button to navigate to create page
   */
  async clickAddProduct(): Promise<void> {
    await this.addProductButton.click()
    await this.page.waitForURL('**/catalog/products/new')
  }

  /**
   * Search for products
   */
  async search(keyword: string): Promise<void> {
    await this.searchInput.fill(keyword)
    // Wait for debounce and API response
    await this.page.waitForTimeout(500)
    await this.waitForTableLoad()
  }

  /**
   * Clear search
   */
  async clearSearch(): Promise<void> {
    await this.searchInput.clear()
    await this.page.waitForTimeout(500)
    await this.waitForTableLoad()
  }

  /**
   * Filter by status
   */
  async filterByStatus(status: 'active' | 'inactive' | 'discontinued' | ''): Promise<void> {
    // Click the status filter Select
    const statusSelect = this.page.locator('.semi-select').first()
    await statusSelect.click()

    // Wait for dropdown to appear
    await this.page.waitForTimeout(300)

    const statusLabels: Record<string, string> = {
      '': '全部状态',
      active: '启用',
      inactive: '禁用',
      discontinued: '停售',
    }

    // Semi Select options appear in a portal/overlay with class semi-select-option-list
    const optionList = this.page.locator('.semi-select-option-list')
    await optionList.waitFor({ state: 'visible', timeout: 5000 })

    // Click the option
    await optionList
      .locator('.semi-select-option')
      .filter({ hasText: statusLabels[status] })
      .click()

    // Wait for the dropdown to close and table to reload
    await this.page.waitForTimeout(300)
    await this.waitForTableLoad()
  }

  /**
   * Get product count from table
   */
  async getProductCount(): Promise<number> {
    return this.tableRows.count()
  }

  /**
   * Get product data from a row by index
   */
  async getProductRowData(index: number): Promise<{
    code: string
    name: string
    unit: string
    purchasePrice: string
    sellingPrice: string
    status: string
  }> {
    const row = this.tableRows.nth(index)
    const cells = row.locator('.semi-table-row-cell')

    return {
      code: (await cells.nth(1).textContent()) || '',
      name: (await cells.nth(2).locator('.product-name').textContent()) || '',
      unit: (await cells.nth(3).textContent()) || '',
      purchasePrice: (await cells.nth(4).textContent()) || '',
      sellingPrice: (await cells.nth(5).textContent()) || '',
      status: (await cells.nth(6).locator('.semi-tag').textContent()) || '',
    }
  }

  /**
   * Find product row by code
   */
  async findProductRowByCode(code: string): Promise<Locator | null> {
    const rows = await this.tableRows.all()
    for (const row of rows) {
      const codeCell = await row.locator('.semi-table-row-cell').nth(1).textContent()
      if (codeCell?.includes(code)) {
        return row
      }
    }
    return null
  }

  /**
   * Find product row by name
   */
  async findProductRowByName(name: string): Promise<Locator | null> {
    const rows = await this.tableRows.all()
    for (const row of rows) {
      const nameCell = await row.locator('.product-name').textContent()
      if (nameCell?.includes(name)) {
        return row
      }
    }
    return null
  }

  /**
   * Click action button on a product row
   */
  async clickRowAction(
    row: Locator,
    action: 'view' | 'edit' | 'activate' | 'deactivate' | 'discontinue' | 'delete'
  ): Promise<void> {
    const actionLabels: Record<string, string> = {
      view: '查看',
      edit: '编辑',
      activate: '启用',
      deactivate: '禁用',
      discontinue: '停售',
      delete: '删除',
    }

    // For 'view' and 'edit', they are direct buttons, not in dropdown
    if (action === 'view' || action === 'edit') {
      const actionButton = row.locator('button').filter({ hasText: actionLabels[action] })
      await actionButton.click()
      return
    }

    // For other actions, click the "more" dropdown trigger button
    const moreButton = row.locator('[data-testid="table-row-more-actions"]')
    await moreButton.click()

    // Wait for dropdown menu to appear
    await this.page.waitForTimeout(300)

    // Click the action in dropdown menu
    await this.page.locator('.semi-dropdown-item').filter({ hasText: actionLabels[action] }).click()
  }

  /**
   * Fill product form
   */
  async fillProductForm(data: {
    code?: string
    name?: string
    unit?: string
    barcode?: string
    description?: string
    purchasePrice?: number
    sellingPrice?: number
    minStock?: number
    sortOrder?: number
  }): Promise<void> {
    if (data.code !== undefined) {
      await this.codeInput.fill(data.code)
    }
    if (data.name !== undefined) {
      await this.nameInput.fill(data.name)
    }
    if (data.unit !== undefined) {
      await this.unitInput.fill(data.unit)
    }
    if (data.barcode !== undefined) {
      await this.barcodeInput.fill(data.barcode)
    }
    if (data.description !== undefined) {
      await this.descriptionInput.fill(data.description)
    }
    if (data.purchasePrice !== undefined) {
      await this.purchasePriceInput.fill(data.purchasePrice.toString())
    }
    if (data.sellingPrice !== undefined) {
      await this.sellingPriceInput.fill(data.sellingPrice.toString())
    }
    if (data.minStock !== undefined) {
      await this.minStockInput.fill(data.minStock.toString())
    }
    if (data.sortOrder !== undefined) {
      await this.sortOrderInput.fill(data.sortOrder.toString())
    }
  }

  /**
   * Submit product form
   */
  async submitForm(): Promise<void> {
    // Use getByRole for more reliable button selection
    // The submit button in create mode shows "新建", in edit mode shows "保存"
    const submitBtn = this.page.getByRole('button', { name: /新建|保存|Create|Save/i })
    await submitBtn.click()
  }

  /**
   * Cancel product form
   */
  async cancelForm(): Promise<void> {
    await this.cancelButton.click()
  }

  /**
   * Wait for form submission success
   */
  async waitForFormSuccess(): Promise<void> {
    await this.page.waitForURL('**/catalog/products')
    await this.waitForTableLoad()
  }

  /**
   * Select product row checkbox
   */
  async selectProductRow(row: Locator): Promise<void> {
    await row.locator('.semi-checkbox').click()
  }

  /**
   * Select all products
   */
  async selectAllProducts(): Promise<void> {
    await this.page.locator('.semi-table-thead .semi-checkbox').click()
  }

  /**
   * Click bulk action
   */
  async clickBulkAction(action: 'activate' | 'deactivate'): Promise<void> {
    const labels: Record<string, string> = {
      activate: '批量启用',
      deactivate: '批量禁用',
    }
    await this.bulkActionBar.locator('.semi-tag').filter({ hasText: labels[action] }).click()
  }

  /**
   * Confirm modal dialog
   */
  async confirmDialog(): Promise<void> {
    await this.page
      .locator(
        '.semi-modal-footer button.semi-button-danger, .semi-modal-footer button.semi-button-primary'
      )
      .click()
  }

  /**
   * Cancel modal dialog
   */
  async cancelDialog(): Promise<void> {
    await this.page.locator('.semi-modal-footer button').filter({ hasText: '取消' }).click()
  }

  /**
   * Assert product exists in table
   */
  async assertProductExists(code: string): Promise<void> {
    const row = await this.findProductRowByCode(code)
    expect(row).not.toBeNull()
  }

  /**
   * Assert product does not exist in table
   */
  async assertProductNotExists(code: string): Promise<void> {
    const row = await this.findProductRowByCode(code)
    expect(row).toBeNull()
  }

  /**
   * Assert product status
   */
  async assertProductStatus(code: string, expectedStatus: '启用' | '禁用' | '停售'): Promise<void> {
    const row = await this.findProductRowByCode(code)
    expect(row).not.toBeNull()
    if (row) {
      const statusTag = row.locator('.semi-tag')
      await expect(statusTag).toContainText(expectedStatus)
    }
  }

  /**
   * Assert form validation error
   */
  async assertFormError(fieldName: string, errorMessage: string): Promise<void> {
    const errorElement = this.page.locator(
      `.semi-form-field[data-field="${fieldName}"] .semi-form-field-error-message, input[name="${fieldName}"] ~ .semi-form-field-error-message`
    )
    await expect(errorElement).toContainText(errorMessage)
  }

  /**
   * Assert page title
   */
  async assertPageTitle(title: string): Promise<void> {
    await expect(this.pageTitle).toContainText(title)
  }

  /**
   * Take screenshot of products list
   */
  async screenshotList(name: string = 'products-list'): Promise<void> {
    await this.screenshot(name)
  }

  /**
   * Take screenshot of product form
   */
  async screenshotForm(name: string = 'product-form'): Promise<void> {
    await this.screenshot(name)
  }

  /**
   * Wait for pagination and check current page
   */
  async getPaginationInfo(): Promise<{ current: number; total: number }> {
    // Get total from pagination info text - look for the specific class
    const paginationInfo = this.page.locator('.data-table-pagination-info')
    const infoText = await paginationInfo.textContent()
    // Matches patterns like "共 10 条记录" or "共 10 条" or "Total: 10 records"
    const totalMatch = infoText?.match(/(\d+)/)
    const total = totalMatch ? parseInt(totalMatch[1], 10) : 0

    // Current page is shown in the active pagination item
    // Semi Pagination uses .semi-page-item-active
    const currentPageElement = this.page.locator('.semi-page-item-active')
    const isVisible = await currentPageElement.isVisible().catch(() => false)
    const currentPage = isVisible ? await currentPageElement.textContent() : '1'
    const current = currentPage ? parseInt(currentPage, 10) : 1

    return { current, total }
  }

  /**
   * Go to specific page
   */
  async goToPage(page: number): Promise<void> {
    await this.page.locator('.semi-page-item').filter({ hasText: page.toString() }).click()
    await this.waitForTableLoad()
  }

  /**
   * Change page size
   */
  async changePageSize(size: 10 | 20 | 50 | 100): Promise<void> {
    await this.page.locator('.semi-page-switch').click()
    await this.page
      .locator('.semi-select-option')
      .filter({ hasText: `${size} 条/页` })
      .click()
    await this.waitForTableLoad()
  }
}
