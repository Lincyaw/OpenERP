import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * WarehousesPage - Page Object for Warehouse management pages
 *
 * Covers:
 * - Warehouse list page (/partner/warehouses)
 * - Warehouse create page (/partner/warehouses/new)
 * - Warehouse edit page (/partner/warehouses/:id/edit)
 */
export class WarehousesPage extends BasePage {
  // List page elements
  readonly pageTitle: Locator
  readonly searchInput: Locator
  readonly statusFilter: Locator
  readonly typeFilter: Locator
  readonly addWarehouseButton: Locator
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
  readonly countryInput: Locator
  readonly provinceInput: Locator
  readonly cityInput: Locator
  readonly postalCodeInput: Locator
  readonly addressInput: Locator
  readonly capacityInput: Locator
  readonly sortOrderInput: Locator
  readonly isDefaultCheckbox: Locator
  readonly notesInput: Locator
  readonly submitButton: Locator
  readonly cancelButton: Locator

  constructor(page: Page) {
    super(page)

    // List page elements
    this.pageTitle = page.locator('.warehouses-header h4, .warehouse-form-header h4')
    this.searchInput = page.locator('.semi-input-wrapper input[placeholder*="搜索"]')
    this.statusFilter = page.locator('.warehouses-filter-container .semi-select').first()
    this.typeFilter = page.locator('.warehouses-filter-container .semi-select').nth(1)
    this.addWarehouseButton = page.locator('button').filter({ hasText: '新增仓库' })
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
      .filter({ has: page.locator('label:has-text("仓库类型")') })
      .locator('.semi-select')
    this.contactNameInput = page.locator('input[name="contact_name"]')
    this.phoneInput = page.locator('input[name="phone"]')
    this.emailInput = page.locator('input[name="email"]')
    this.countryInput = page.locator('input[name="country"]')
    this.provinceInput = page.locator('input[name="province"]')
    this.cityInput = page.locator('input[name="city"]')
    this.postalCodeInput = page.locator('input[name="postal_code"]')
    this.addressInput = page.locator('input[name="address"]')
    this.capacityInput = page.locator('input[name="capacity"]')
    this.sortOrderInput = page.locator('input[name="sort_order"]')
    this.isDefaultCheckbox = page.locator('.semi-checkbox').filter({ hasText: '默认仓库' })
    this.notesInput = page.locator('textarea[name="notes"]')
    this.submitButton = page.locator('button[type="submit"]')
    this.cancelButton = page.locator('button').filter({ hasText: '取消' })
  }

  /**
   * Navigate to warehouses list page
   */
  async navigateToList(): Promise<void> {
    await this.goto('/partner/warehouses')
    await this.waitForTableLoad()
  }

  /**
   * Navigate to create warehouse page
   */
  async navigateToCreate(): Promise<void> {
    await this.goto('/partner/warehouses/new')
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Navigate to edit warehouse page
   */
  async navigateToEdit(warehouseId: string): Promise<void> {
    await this.goto(`/partner/warehouses/${warehouseId}/edit`)
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Click add warehouse button to navigate to create page
   */
  async clickAddWarehouse(): Promise<void> {
    await this.addWarehouseButton.click()
    await this.page.waitForURL('**/partner/warehouses/new')
  }

  /**
   * Search for warehouses
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
  async filterByStatus(status: 'enabled' | 'disabled' | ''): Promise<void> {
    await this.statusFilter.click()
    const statusLabels: Record<string, string> = {
      '': '全部状态',
      enabled: '启用',
      disabled: '停用',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: statusLabels[status] }).click()
    await this.waitForTableLoad()
  }

  /**
   * Filter by type
   */
  async filterByType(type: 'normal' | 'virtual' | 'transit' | ''): Promise<void> {
    await this.typeFilter.click()
    const typeLabels: Record<string, string> = {
      '': '全部类型',
      normal: '普通仓库',
      virtual: '虚拟仓库',
      transit: '中转仓库',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: typeLabels[type] }).click()
    await this.waitForTableLoad()
  }

  /**
   * Get warehouse count from table
   */
  async getWarehouseCount(): Promise<number> {
    return this.tableRows.count()
  }

  /**
   * Get warehouse data from a row by index
   */
  async getWarehouseRowData(index: number): Promise<{
    code: string
    name: string
    type: string
    location: string
    sortOrder: string
    status: string
    isDefault: boolean
  }> {
    const row = this.tableRows.nth(index)
    const cells = row.locator('.semi-table-row-cell')

    // Check if default tag is present
    const hasDefaultTag = (await cells.nth(2).locator('.default-tag').count()) > 0

    return {
      code: (await cells.nth(1).textContent()) || '',
      name: (await cells.nth(2).locator('.warehouse-name').first().textContent()) || '',
      type: (await cells.nth(3).locator('.semi-tag').textContent()) || '',
      location: (await cells.nth(4).textContent()) || '',
      sortOrder: (await cells.nth(5).textContent()) || '',
      status: (await cells.nth(6).locator('.semi-tag').textContent()) || '',
      isDefault: hasDefaultTag,
    }
  }

  /**
   * Find warehouse row by code
   */
  async findWarehouseRowByCode(code: string): Promise<Locator | null> {
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
   * Find warehouse row by name
   */
  async findWarehouseRowByName(name: string): Promise<Locator | null> {
    const rows = await this.tableRows.all()
    for (const row of rows) {
      const nameCell = await row.locator('.warehouse-name').first().textContent()
      if (nameCell?.includes(name)) {
        return row
      }
    }
    return null
  }

  /**
   * Click action button on a warehouse row
   */
  async clickRowAction(
    row: Locator,
    action: 'view' | 'edit' | 'setDefault' | 'enable' | 'disable' | 'delete'
  ): Promise<void> {
    const actionLabels: Record<string, string> = {
      view: '查看',
      edit: '编辑',
      setDefault: '设为默认',
      enable: '启用',
      disable: '停用',
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
   * Fill warehouse form
   */
  async fillWarehouseForm(data: {
    code?: string
    name?: string
    shortName?: string
    type?: 'normal' | 'virtual' | 'transit'
    contactName?: string
    phone?: string
    email?: string
    province?: string
    city?: string
    address?: string
    capacity?: number
    sortOrder?: number
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
        .filter({ has: this.page.locator('label:has-text("仓库类型")') })
        .locator('.semi-select')
        .click()
      const typeLabels: Record<string, string> = {
        normal: '普通仓库',
        virtual: '虚拟仓库',
        transit: '中转仓库',
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
    if (data.province !== undefined) {
      await this.provinceInput.fill(data.province)
    }
    if (data.city !== undefined) {
      await this.cityInput.fill(data.city)
    }
    if (data.address !== undefined) {
      await this.addressInput.fill(data.address)
    }
    if (data.capacity !== undefined) {
      await this.capacityInput.fill(data.capacity.toString())
    }
    if (data.sortOrder !== undefined) {
      await this.sortOrderInput.fill(data.sortOrder.toString())
    }
  }

  /**
   * Submit warehouse form
   */
  async submitForm(): Promise<void> {
    await this.submitButton.click()
  }

  /**
   * Cancel warehouse form
   */
  async cancelForm(): Promise<void> {
    await this.cancelButton.click()
  }

  /**
   * Wait for form submission success
   */
  async waitForFormSuccess(): Promise<void> {
    await this.page.waitForURL('**/partner/warehouses')
    await this.waitForTableLoad()
  }

  /**
   * Select warehouse row checkbox
   */
  async selectWarehouseRow(row: Locator): Promise<void> {
    await row.locator('.semi-checkbox').click()
  }

  /**
   * Select all warehouses
   */
  async selectAllWarehouses(): Promise<void> {
    await this.page.locator('.semi-table-thead .semi-checkbox').click()
  }

  /**
   * Click bulk action
   */
  async clickBulkAction(action: 'enable' | 'disable'): Promise<void> {
    const labels: Record<string, string> = {
      enable: '批量启用',
      disable: '批量停用',
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
   * Assert warehouse exists in table
   */
  async assertWarehouseExists(code: string): Promise<void> {
    const row = await this.findWarehouseRowByCode(code)
    expect(row).not.toBeNull()
  }

  /**
   * Assert warehouse does not exist in table
   */
  async assertWarehouseNotExists(code: string): Promise<void> {
    const row = await this.findWarehouseRowByCode(code)
    expect(row).toBeNull()
  }

  /**
   * Assert warehouse status
   */
  async assertWarehouseStatus(code: string, expectedStatus: '启用' | '停用'): Promise<void> {
    const row = await this.findWarehouseRowByCode(code)
    expect(row).not.toBeNull()
    if (row) {
      const statusTag = row.locator('.semi-table-row-cell').nth(6).locator('.semi-tag')
      await expect(statusTag).toContainText(expectedStatus)
    }
  }

  /**
   * Assert warehouse is default
   */
  async assertWarehouseIsDefault(code: string): Promise<void> {
    const row = await this.findWarehouseRowByCode(code)
    expect(row).not.toBeNull()
    if (row) {
      const defaultTag = row.locator('.default-tag')
      await expect(defaultTag).toBeVisible()
    }
  }

  /**
   * Assert page title
   */
  async assertPageTitle(title: string): Promise<void> {
    await expect(this.pageTitle).toContainText(title)
  }

  /**
   * Take screenshot of warehouses list
   */
  async screenshotList(name: string = 'warehouses-list'): Promise<void> {
    await this.screenshot(name)
  }

  /**
   * Take screenshot of warehouse form
   */
  async screenshotForm(name: string = 'warehouse-form'): Promise<void> {
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
