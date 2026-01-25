import { type Locator, type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * CustomersPage - Page Object for Customer management pages
 *
 * Covers:
 * - Customer list page (/partner/customers)
 * - Customer create page (/partner/customers/new)
 * - Customer edit page (/partner/customers/:id/edit)
 */
export class CustomersPage extends BasePage {
  // List page elements
  readonly pageTitle: Locator
  readonly searchInput: Locator
  readonly statusFilter: Locator
  readonly typeFilter: Locator
  readonly levelFilter: Locator
  readonly addCustomerButton: Locator
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
  readonly levelSelect: Locator
  readonly contactNameInput: Locator
  readonly phoneInput: Locator
  readonly emailInput: Locator
  readonly taxIdInput: Locator
  readonly countryInput: Locator
  readonly provinceInput: Locator
  readonly cityInput: Locator
  readonly postalCodeInput: Locator
  readonly addressInput: Locator
  readonly creditLimitInput: Locator
  readonly sortOrderInput: Locator
  readonly notesInput: Locator
  readonly submitButton: Locator
  readonly cancelButton: Locator

  constructor(page: Page) {
    super(page)

    // List page elements
    this.pageTitle = page.locator('.customers-header h4, .customer-form-header h4')
    this.searchInput = page.locator('.semi-input-wrapper input[placeholder*="搜索"]')
    this.statusFilter = page.locator('.semi-select').filter({ hasText: /状态/ }).first()
    this.typeFilter = page.locator('.semi-select').filter({ hasText: /类型/ }).first()
    this.levelFilter = page.locator('.semi-select').filter({ hasText: /等级/ }).first()
    this.addCustomerButton = page.locator('button').filter({ hasText: '新增客户' })
    this.refreshButton = page.locator('button').filter({ hasText: '刷新' })
    this.table = page.locator('.semi-table')
    this.tableRows = page.locator('.semi-table-tbody .semi-table-row')
    this.bulkActionBar = page.locator('.bulk-action-bar')
    this.emptyState = page.locator('.semi-table-empty')

    // Form elements - using name attributes set by React Hook Form
    this.codeInput = page.locator('input[name="code"]')
    this.nameInput = page.locator('input[name="name"]')
    this.shortNameInput = page.locator('input[name="short_name"]')
    this.typeSelect = page.locator('[data-field="type"] .semi-select, .semi-form-field').filter({ has: page.locator('label:has-text("客户类型")') }).locator('.semi-select')
    this.levelSelect = page.locator('[data-field="level"] .semi-select, .semi-form-field').filter({ has: page.locator('label:has-text("客户等级")') }).locator('.semi-select')
    this.contactNameInput = page.locator('input[name="contact_name"]')
    this.phoneInput = page.locator('input[name="phone"]')
    this.emailInput = page.locator('input[name="email"]')
    this.taxIdInput = page.locator('input[name="tax_id"]')
    this.countryInput = page.locator('input[name="country"]')
    this.provinceInput = page.locator('input[name="province"]')
    this.cityInput = page.locator('input[name="city"]')
    this.postalCodeInput = page.locator('input[name="postal_code"]')
    this.addressInput = page.locator('input[name="address"]')
    this.creditLimitInput = page.locator('input[name="credit_limit"]')
    this.sortOrderInput = page.locator('input[name="sort_order"]')
    this.notesInput = page.locator('textarea[name="notes"]')
    this.submitButton = page.locator('button[type="submit"]')
    this.cancelButton = page.locator('button').filter({ hasText: '取消' })
  }

  /**
   * Navigate to customers list page
   */
  async navigateToList(): Promise<void> {
    await this.goto('/partner/customers')
    await this.waitForTableLoad()
  }

  /**
   * Navigate to create customer page
   */
  async navigateToCreate(): Promise<void> {
    await this.goto('/partner/customers/new')
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Navigate to edit customer page
   */
  async navigateToEdit(customerId: string): Promise<void> {
    await this.goto(`/partner/customers/${customerId}/edit`)
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Click add customer button to navigate to create page
   */
  async clickAddCustomer(): Promise<void> {
    await this.addCustomerButton.click()
    await this.page.waitForURL('**/partner/customers/new')
  }

  /**
   * Search for customers
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
  async filterByStatus(status: 'active' | 'inactive' | 'suspended' | ''): Promise<void> {
    await this.page.locator('.customers-filter-container .semi-select').first().click()
    const statusLabels: Record<string, string> = {
      '': '全部状态',
      active: '启用',
      inactive: '停用',
      suspended: '暂停',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: statusLabels[status] }).click()
    await this.waitForTableLoad()
  }

  /**
   * Filter by type
   */
  async filterByType(type: 'individual' | 'organization' | ''): Promise<void> {
    await this.page.locator('.customers-filter-container .semi-select').nth(1).click()
    const typeLabels: Record<string, string> = {
      '': '全部类型',
      individual: '个人',
      organization: '企业/组织',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: typeLabels[type] }).click()
    await this.waitForTableLoad()
  }

  /**
   * Filter by level
   */
  async filterByLevel(level: 'normal' | 'silver' | 'gold' | 'platinum' | 'vip' | ''): Promise<void> {
    await this.page.locator('.customers-filter-container .semi-select').nth(2).click()
    const levelLabels: Record<string, string> = {
      '': '全部等级',
      normal: '普通',
      silver: '白银',
      gold: '黄金',
      platinum: '铂金',
      vip: 'VIP',
    }
    await this.page.locator('.semi-select-option').filter({ hasText: levelLabels[level] }).click()
    await this.waitForTableLoad()
  }

  /**
   * Get customer count from table
   */
  async getCustomerCount(): Promise<number> {
    return this.tableRows.count()
  }

  /**
   * Get customer data from a row by index
   */
  async getCustomerRowData(index: number): Promise<{
    code: string
    name: string
    type: string
    contact: string
    location: string
    level: string
    status: string
  }> {
    const row = this.tableRows.nth(index)
    const cells = row.locator('.semi-table-row-cell')

    return {
      code: (await cells.nth(1).textContent()) || '',
      name: (await cells.nth(2).locator('.customer-name').textContent()) || '',
      type: (await cells.nth(3).locator('.semi-tag').textContent()) || '',
      contact: (await cells.nth(4).textContent()) || '',
      location: (await cells.nth(5).textContent()) || '',
      level: (await cells.nth(6).locator('.semi-tag').textContent()) || '',
      status: (await cells.nth(7).locator('.semi-tag').textContent()) || '',
    }
  }

  /**
   * Find customer row by code
   */
  async findCustomerRowByCode(code: string): Promise<Locator | null> {
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
   * Find customer row by name
   */
  async findCustomerRowByName(name: string): Promise<Locator | null> {
    const rows = await this.tableRows.all()
    for (const row of rows) {
      const nameCell = await row.locator('.customer-name').textContent()
      if (nameCell?.includes(name)) {
        return row
      }
    }
    return null
  }

  /**
   * Click action button on a customer row
   */
  async clickRowAction(
    row: Locator,
    action: 'view' | 'edit' | 'balance' | 'activate' | 'deactivate' | 'suspend' | 'delete'
  ): Promise<void> {
    const actionLabels: Record<string, string> = {
      view: '查看',
      edit: '编辑',
      balance: '余额',
      activate: '启用',
      deactivate: '停用',
      suspend: '暂停',
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
   * Fill customer form
   */
  async fillCustomerForm(data: {
    code?: string
    name?: string
    shortName?: string
    type?: 'individual' | 'organization'
    contactName?: string
    phone?: string
    email?: string
    province?: string
    city?: string
    address?: string
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
      // Click the type select field
      await this.page.locator('.semi-form-field').filter({ has: this.page.locator('label:has-text("客户类型")') }).locator('.semi-select').click()
      const typeLabels: Record<string, string> = {
        individual: '个人',
        organization: '企业/组织',
      }
      await this.page.locator('.semi-select-option').filter({ hasText: typeLabels[data.type] }).click()
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
    if (data.creditLimit !== undefined) {
      await this.creditLimitInput.fill(data.creditLimit.toString())
    }
  }

  /**
   * Submit customer form
   */
  async submitForm(): Promise<void> {
    await this.submitButton.click()
  }

  /**
   * Cancel customer form
   */
  async cancelForm(): Promise<void> {
    await this.cancelButton.click()
  }

  /**
   * Wait for form submission success
   */
  async waitForFormSuccess(): Promise<void> {
    await this.page.waitForURL('**/partner/customers')
    await this.waitForTableLoad()
  }

  /**
   * Select customer row checkbox
   */
  async selectCustomerRow(row: Locator): Promise<void> {
    await row.locator('.semi-checkbox').click()
  }

  /**
   * Select all customers
   */
  async selectAllCustomers(): Promise<void> {
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
        '.semi-modal-footer button.semi-button-danger, .semi-modal-footer button.semi-button-primary, .semi-modal-footer button.semi-button-warning'
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
   * Assert customer exists in table
   */
  async assertCustomerExists(code: string): Promise<void> {
    const row = await this.findCustomerRowByCode(code)
    expect(row).not.toBeNull()
  }

  /**
   * Assert customer does not exist in table
   */
  async assertCustomerNotExists(code: string): Promise<void> {
    const row = await this.findCustomerRowByCode(code)
    expect(row).toBeNull()
  }

  /**
   * Assert customer status
   */
  async assertCustomerStatus(code: string, expectedStatus: '启用' | '停用' | '暂停'): Promise<void> {
    const row = await this.findCustomerRowByCode(code)
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
   * Take screenshot of customers list
   */
  async screenshotList(name: string = 'customers-list'): Promise<void> {
    await this.screenshot(name)
  }

  /**
   * Take screenshot of customer form
   */
  async screenshotForm(name: string = 'customer-form'): Promise<void> {
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
