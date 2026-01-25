import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * SuppliersPage - Page Object for Supplier management pages
 *
 * Covers:
 * - Supplier list page (/partner/suppliers)
 * - Supplier create page (/partner/suppliers/new)
 * - Supplier edit page (/partner/suppliers/:id/edit)
 */
export class SuppliersPage extends BasePage {
  // List page elements
  readonly pageTitle: Locator
  readonly searchInput: Locator
  readonly statusFilter: Locator
  readonly typeFilter: Locator
  readonly addSupplierButton: Locator
  readonly refreshButton: Locator
  readonly table: Locator
  readonly tableRows: Locator
  readonly bulkActionBar: Locator
  readonly emptyState: Locator

  // Form page elements
  readonly codeInput: Locator
  readonly nameInput: Locator
  readonly shortNameInput: Locator
  readonly typeSelect: Locator
  readonly contactNameInput: Locator
  readonly phoneInput: Locator
  readonly emailInput: Locator
  readonly taxIdInput: Locator
  readonly bankNameInput: Locator
  readonly bankAccountInput: Locator
  readonly countryInput: Locator
  readonly provinceInput: Locator
  readonly cityInput: Locator
  readonly postalCodeInput: Locator
  readonly addressInput: Locator
  readonly creditDaysInput: Locator
  readonly creditLimitInput: Locator
  readonly ratingInput: Locator
  readonly sortOrderInput: Locator
  readonly notesInput: Locator
  readonly submitButton: Locator
  readonly cancelButton: Locator

  constructor(page: Page) {
    super(page)

    // List page elements
    this.pageTitle = page.locator('.suppliers-header h4, .supplier-form-header h4')
    this.searchInput = page.locator('.semi-input-wrapper input[placeholder*="搜索"]')
    this.statusFilter = page.locator('.suppliers-filter-container .semi-select').first()
    this.typeFilter = page.locator('.suppliers-filter-container .semi-select').nth(1)
    this.addSupplierButton = page.locator('button').filter({ hasText: '新增供应商' })
    this.refreshButton = page.locator('button').filter({ hasText: '刷新' })
    this.table = page.locator('.semi-table')
    this.tableRows = page.locator('.semi-table-tbody .semi-table-row')
    this.bulkActionBar = page.locator('.bulk-action-bar')
    this.emptyState = page.locator('.semi-table-empty')

    // Form elements
    this.codeInput = page.locator('input[name="code"]')
    this.nameInput = page.locator('input[name="name"]')
    this.shortNameInput = page.locator('input[name="short_name"]')
    this.typeSelect = page
      .locator('.semi-form-field')
      .filter({ has: page.locator('label:has-text("供应商类型")') })
      .locator('.semi-select')
    this.contactNameInput = page.locator('input[name="contact_name"]')
    this.phoneInput = page.locator('input[name="phone"]')
    this.emailInput = page.locator('input[name="email"]')
    this.taxIdInput = page.locator('input[name="tax_id"]')
    this.bankNameInput = page.locator('input[name="bank_name"]')
    this.bankAccountInput = page.locator('input[name="bank_account"]')
    this.countryInput = page.locator('input[name="country"]')
    this.provinceInput = page.locator('input[name="province"]')
    this.cityInput = page.locator('input[name="city"]')
    this.postalCodeInput = page.locator('input[name="postal_code"]')
    this.addressInput = page.locator('input[name="address"]')
    this.creditDaysInput = page.locator('input[name="credit_days"]')
    this.creditLimitInput = page.locator('input[name="credit_limit"]')
    this.ratingInput = page.locator('.semi-rating')
    this.sortOrderInput = page.locator('input[name="sort_order"]')
    this.notesInput = page.locator('textarea[name="notes"]')
    this.submitButton = page.locator('button[type="submit"]')
    this.cancelButton = page.locator('button').filter({ hasText: '取消' })
  }

  /**
   * Navigate to suppliers list page
   */
  async navigateToList(): Promise<void> {
    await this.goto('/partner/suppliers')
    await this.waitForTableLoad()
  }

  /**
   * Navigate to create supplier page
   */
  async navigateToCreate(): Promise<void> {
    await this.goto('/partner/suppliers/new')
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Navigate to edit supplier page
   */
  async navigateToEdit(supplierId: string): Promise<void> {
    await this.goto(`/partner/suppliers/${supplierId}/edit`)
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Click add supplier button to navigate to create page
   */
  async clickAddSupplier(): Promise<void> {
    await this.addSupplierButton.click()
    await this.page.waitForURL('**/partner/suppliers/new')
  }

  /**
   * Search for suppliers
   */
  async search(keyword: string): Promise<void> {
    await this.searchInput.fill(keyword)
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
  async filterByStatus(status: 'active' | 'inactive' | 'blocked' | ''): Promise<void> {
    await this.statusFilter.click()
    const statusLabels: Record<string, string> = {
      '': '全部状态',
      active: '启用',
      inactive: '停用',
      blocked: '拉黑',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: statusLabels[status] }).click()
    await this.waitForTableLoad()
  }

  /**
   * Filter by type
   */
  async filterByType(
    type: 'manufacturer' | 'distributor' | 'retailer' | 'service' | ''
  ): Promise<void> {
    await this.typeFilter.click()
    const typeLabels: Record<string, string> = {
      '': '全部类型',
      manufacturer: '生产商',
      distributor: '经销商',
      retailer: '零售商',
      service: '服务商',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: typeLabels[type] }).click()
    await this.waitForTableLoad()
  }

  /**
   * Get supplier count from table
   */
  async getSupplierCount(): Promise<number> {
    return this.tableRows.count()
  }

  /**
   * Get supplier data from a row by index
   */
  async getSupplierRowData(index: number): Promise<{
    code: string
    name: string
    contact: string
    location: string
    rating: number
    creditDays: string
    status: string
  }> {
    const row = this.tableRows.nth(index)
    const cells = row.locator('.semi-table-row-cell')

    // Parse rating from the Rating component
    const ratingStars = await cells.nth(5).locator('.semi-rating-star-full').count()

    return {
      code: (await cells.nth(1).textContent()) || '',
      name: (await cells.nth(2).locator('.supplier-name').textContent()) || '',
      contact: (await cells.nth(3).textContent()) || '',
      location: (await cells.nth(4).textContent()) || '',
      rating: ratingStars,
      creditDays: (await cells.nth(6).textContent()) || '',
      status: (await cells.nth(7).locator('.semi-tag').textContent()) || '',
    }
  }

  /**
   * Find supplier row by code
   */
  async findSupplierRowByCode(code: string): Promise<Locator | null> {
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
   * Find supplier row by name
   */
  async findSupplierRowByName(name: string): Promise<Locator | null> {
    const rows = await this.tableRows.all()
    for (const row of rows) {
      const nameCell = await row.locator('.supplier-name').textContent()
      if (nameCell?.includes(name)) {
        return row
      }
    }
    return null
  }

  /**
   * Click action button on a supplier row
   */
  async clickRowAction(
    row: Locator,
    action: 'view' | 'edit' | 'activate' | 'deactivate' | 'block' | 'delete'
  ): Promise<void> {
    const actionLabels: Record<string, string> = {
      view: '查看',
      edit: '编辑',
      activate: '启用',
      deactivate: '停用',
      block: '拉黑',
      delete: '删除',
    }

    // Click the dropdown trigger in the actions column
    const actionsCell = row.locator('.semi-table-row-cell').last()
    const dropdownTrigger = actionsCell.locator('.semi-dropdown-trigger, button').first()
    await dropdownTrigger.click()

    // Click the action in dropdown menu
    await this.page
      .locator('.semi-dropdown-menu .semi-dropdown-item')
      .filter({ hasText: actionLabels[action] })
      .click()
  }

  /**
   * Fill supplier form
   */
  async fillSupplierForm(data: {
    code?: string
    name?: string
    shortName?: string
    type?: 'manufacturer' | 'distributor' | 'retailer' | 'service'
    contactName?: string
    phone?: string
    email?: string
    taxId?: string
    bankName?: string
    bankAccount?: string
    province?: string
    city?: string
    address?: string
    creditDays?: number
    creditLimit?: number
  }): Promise<void> {
    if (data.code !== undefined) {
      await this.codeInput.fill(data.code)
    }
    if (data.name !== undefined) {
      await this.nameInput.fill(data.name)
    }
    if (data.shortName !== undefined) {
      await this.shortNameInput.fill(data.shortName)
    }
    if (data.type !== undefined) {
      await this.page
        .locator('.semi-form-field')
        .filter({ has: this.page.locator('label:has-text("供应商类型")') })
        .locator('.semi-select')
        .click()
      const typeLabels: Record<string, string> = {
        manufacturer: '生产商',
        distributor: '经销商',
        retailer: '零售商',
        service: '服务商',
      }
      await this.page
        .locator('.semi-select-option')
        .filter({ hasText: typeLabels[data.type] })
        .click()
    }
    if (data.contactName !== undefined) {
      await this.contactNameInput.fill(data.contactName)
    }
    if (data.phone !== undefined) {
      await this.phoneInput.fill(data.phone)
    }
    if (data.email !== undefined) {
      await this.emailInput.fill(data.email)
    }
    if (data.taxId !== undefined) {
      await this.taxIdInput.fill(data.taxId)
    }
    if (data.bankName !== undefined) {
      await this.bankNameInput.fill(data.bankName)
    }
    if (data.bankAccount !== undefined) {
      await this.bankAccountInput.fill(data.bankAccount)
    }
    if (data.province !== undefined) {
      await this.provinceInput.fill(data.province)
    }
    if (data.city !== undefined) {
      await this.cityInput.fill(data.city)
    }
    if (data.address !== undefined) {
      await this.addressInput.fill(data.address)
    }
    if (data.creditDays !== undefined) {
      await this.creditDaysInput.fill(data.creditDays.toString())
    }
    if (data.creditLimit !== undefined) {
      await this.creditLimitInput.fill(data.creditLimit.toString())
    }
  }

  /**
   * Submit supplier form
   */
  async submitForm(): Promise<void> {
    await this.submitButton.click()
  }

  /**
   * Cancel supplier form
   */
  async cancelForm(): Promise<void> {
    await this.cancelButton.click()
  }

  /**
   * Wait for form submission success
   */
  async waitForFormSuccess(): Promise<void> {
    await this.page.waitForURL('**/partner/suppliers')
    await this.waitForTableLoad()
  }

  /**
   * Select supplier row checkbox
   */
  async selectSupplierRow(row: Locator): Promise<void> {
    await row.locator('.semi-checkbox').click()
  }

  /**
   * Select all suppliers
   */
  async selectAllSuppliers(): Promise<void> {
    await this.page.locator('.semi-table-thead .semi-checkbox').click()
  }

  /**
   * Click bulk action
   */
  async clickBulkAction(action: 'activate' | 'deactivate'): Promise<void> {
    const labels: Record<string, string> = {
      activate: '批量启用',
      deactivate: '批量停用',
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
   * Assert supplier exists in table
   */
  async assertSupplierExists(code: string): Promise<void> {
    const row = await this.findSupplierRowByCode(code)
    expect(row).not.toBeNull()
  }

  /**
   * Assert supplier does not exist in table
   */
  async assertSupplierNotExists(code: string): Promise<void> {
    const row = await this.findSupplierRowByCode(code)
    expect(row).toBeNull()
  }

  /**
   * Assert supplier status
   */
  async assertSupplierStatus(
    code: string,
    expectedStatus: '启用' | '停用' | '拉黑'
  ): Promise<void> {
    const row = await this.findSupplierRowByCode(code)
    expect(row).not.toBeNull()
    if (row) {
      const statusTag = row.locator('.semi-table-row-cell').nth(7).locator('.semi-tag')
      await expect(statusTag).toContainText(expectedStatus)
    }
  }

  /**
   * Assert page title
   */
  async assertPageTitle(title: string): Promise<void> {
    await expect(this.pageTitle).toContainText(title)
  }

  /**
   * Take screenshot of suppliers list
   */
  async screenshotList(name: string = 'suppliers-list'): Promise<void> {
    await this.screenshot(name)
  }

  /**
   * Take screenshot of supplier form
   */
  async screenshotForm(name: string = 'supplier-form'): Promise<void> {
    await this.screenshot(name)
  }

  /**
   * Wait for pagination and check current page
   */
  async getPaginationInfo(): Promise<{ current: number; total: number }> {
    const paginationText = await this.page.locator('.semi-page').textContent()
    const totalMatch = paginationText?.match(/共\s*(\d+)\s*条/)
    const total = totalMatch ? parseInt(totalMatch[1], 10) : 0

    const currentPage = await this.page.locator('.semi-page-item-active').textContent()
    const current = currentPage ? parseInt(currentPage, 10) : 1

    return { current, total }
  }
}
