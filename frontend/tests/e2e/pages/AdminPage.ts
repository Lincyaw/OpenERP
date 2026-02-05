import { type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * AdminPage - Page Object for Super Admin tenant management
 *
 * Handles:
 * - Tenant list page interactions
 * - Create, edit, delete tenant modals
 * - Suspend/activate tenant operations
 * - Change plan and update quota modals
 * - Audit logs page
 * - Navigation within admin module
 */
export class AdminPage extends BasePage {
  // Navigation selectors
  private readonly adminNavItem =
    '.semi-navigation-item:has-text("Tenants"), .semi-navigation-item:has-text("租户")'
  private readonly auditLogsNavItem =
    '.semi-navigation-item:has-text("Audit"), .semi-navigation-item:has-text("审计")'
  private readonly statsNavItem =
    '.semi-navigation-item:has-text("Stats"), .semi-navigation-item:has-text("统计")'

  // Tenant list page selectors
  private readonly pageTitle = '.semi-typography-h4, h4'
  private readonly createTenantButton =
    'button:has-text("Create Tenant"), button:has-text("创建租户")'
  private readonly refreshButton = 'button:has-text("Refresh"), button:has-text("刷新")'
  private readonly searchInput = 'input[placeholder*="Search"], input[placeholder*="搜索"]'
  private readonly statusFilter = '.semi-select:has-text("Status"), .semi-select:has-text("状态")'
  private readonly planFilter = '.semi-select:has-text("Plan"), .semi-select:has-text("套餐")'
  private readonly tenantTable = '.semi-table'
  private readonly tenantRow = '.semi-table-tbody .semi-table-row'
  private readonly actionDropdown = '.semi-dropdown'
  private readonly moreButton = 'button:has(.semi-icon-more), button[aria-label="More"]'

  // Modal selectors
  private readonly modal = '.semi-modal'
  private readonly modalTitle = '.semi-modal-header-title'
  private readonly modalOkButton = '.semi-modal-footer .semi-button-primary'
  private readonly modalCancelButton = '.semi-modal-footer .semi-button:not(.semi-button-primary)'
  private readonly modalCloseButton = '.semi-modal-close'

  // Form selectors
  private readonly nameInput = 'input[name="name"], #name'
  private readonly codeInput = 'input[name="code"], #code'
  private readonly emailInput = 'input[name="contact_email"], #contact_email'
  private readonly phoneInput = 'input[name="contact_phone"], #contact_phone'
  private readonly addressInput = 'textarea[name="address"], #address'
  private readonly planSelect = '.semi-select:has-text("Plan"), [data-testid="plan-select"]'
  private readonly trialDaysInput = 'input[name="trial_days"], #trial_days'

  // Tenant detail page selectors
  private readonly backButton = 'button:has-text("Back"), button:has-text("返回")'
  private readonly editButton = 'button:has-text("Edit"), button:has-text("编辑")'
  private readonly suspendButton = 'button:has-text("Suspend"), button:has-text("暂停")'
  private readonly activateButton = 'button:has-text("Activate"), button:has-text("激活")'
  private readonly deleteButton = 'button:has-text("Delete"), button:has-text("删除")'
  private readonly changePlanButton = 'button:has-text("Change Plan"), button:has-text("变更计划")'
  private readonly updateQuotaButton =
    'button:has-text("Update Quota"), button:has-text("更新配额")'

  // Status and plan tags
  private readonly statusTag = '.semi-tag'
  private readonly activeTag = '.semi-tag-green'
  private readonly suspendedTag = '.semi-tag-red'
  private readonly trialTag = '.semi-tag-orange'

  // Quota usage selectors
  private readonly quotaProgress = '.semi-progress'

  // Audit log selectors
  private readonly auditTable = '.semi-table'
  private readonly auditRow = '.semi-table-tbody .semi-table-row'
  private readonly auditActionFilter =
    '.semi-select:has-text("action"), .semi-select:has-text("操作")'
  private readonly auditDatePicker = '.semi-datepicker'

  constructor(page: Page) {
    super(page)
  }

  // ============================================================================
  // Navigation
  // ============================================================================

  /**
   * Navigate to tenant list page
   */
  async navigateToTenantList(): Promise<void> {
    await this.goto('/super-admin/tenants')
    await this.waitForPageLoad()
  }

  /**
   * Navigate to audit logs page
   */
  async navigateToAuditLogs(): Promise<void> {
    await this.goto('/super-admin/audit-logs')
    await this.waitForPageLoad()
  }

  /**
   * Navigate to tenant detail page
   */
  async navigateToTenantDetail(tenantId: string): Promise<void> {
    await this.goto(`/super-admin/tenants/${tenantId}`)
    await this.waitForPageLoad()
  }

  /**
   * Navigate to stats page
   */
  async navigateToStats(): Promise<void> {
    await this.goto('/super-admin/stats')
    await this.waitForPageLoad()
  }

  // ============================================================================
  // Tenant List Operations
  // ============================================================================

  /**
   * Search tenants by keyword
   */
  async searchTenants(keyword: string): Promise<void> {
    await this.page.fill(this.searchInput, keyword)
    await this.page.waitForTimeout(500) // Wait for debounce
    await this.waitForTableLoad()
  }

  /**
   * Filter tenants by status
   */
  async filterByStatus(status: 'active' | 'suspended' | 'inactive' | 'trial' | ''): Promise<void> {
    const statusSelect = this.page
      .locator('.semi-select')
      .filter({ hasText: /Status|状态|All Status/ })
      .first()
    await statusSelect.click()
    await this.page.waitForTimeout(200)

    if (status === '') {
      await this.page.click(
        '.semi-select-option:has-text("All Status"), .semi-select-option:has-text("全部状态")'
      )
    } else {
      await this.page.click(`.semi-select-option:has-text("${status}")`)
    }
    await this.waitForTableLoad()
  }

  /**
   * Filter tenants by plan
   */
  async filterByPlan(plan: 'free' | 'basic' | 'pro' | 'enterprise' | ''): Promise<void> {
    const planSelect = this.page
      .locator('.semi-select')
      .filter({ hasText: /Plan|套餐|All Plans/ })
      .first()
    await planSelect.click()
    await this.page.waitForTimeout(200)

    if (plan === '') {
      await this.page.click(
        '.semi-select-option:has-text("All Plans"), .semi-select-option:has-text("全部套餐")'
      )
    } else {
      await this.page.click(`.semi-select-option:has-text("${plan}")`)
    }
    await this.waitForTableLoad()
  }

  /**
   * Click create tenant button
   */
  async clickCreateTenant(): Promise<void> {
    await this.page.click(this.createTenantButton)
    await this.page.waitForSelector(this.modal, { state: 'visible' })
  }

  /**
   * Click refresh button
   */
  async clickRefresh(): Promise<void> {
    await this.page.click(this.refreshButton)
    await this.waitForTableLoad()
  }

  /**
   * Get tenant row by name
   */
  getTenantRowByName(name: string) {
    return this.page.locator(this.tenantRow).filter({ hasText: name })
  }

  /**
   * Click tenant name to view details
   */
  async clickTenantName(name: string): Promise<void> {
    const row = this.getTenantRowByName(name)
    await row.locator('.semi-typography-link, a').first().click()
    await this.waitForPageLoad()
  }

  /**
   * Open actions dropdown for a tenant
   */
  async openTenantActions(name: string): Promise<void> {
    const row = this.getTenantRowByName(name)
    await row.locator('button:has(.semi-icon-more), button[aria-label="More"]').click()
    await this.page.waitForSelector('.semi-dropdown-menu', { state: 'visible' })
  }

  /**
   * Click action in dropdown menu
   */
  async clickDropdownAction(
    action: 'View' | 'Edit' | 'Suspend' | 'Activate' | 'Delete'
  ): Promise<void> {
    const actionMap: Record<string, string> = {
      View: 'View|查看',
      Edit: 'Edit|编辑',
      Suspend: 'Suspend|暂停',
      Activate: 'Activate|激活',
      Delete: 'Delete|删除',
    }
    const pattern = actionMap[action]
    await this.page.click(
      `.semi-dropdown-item:has-text("${pattern.split('|')[0]}"), .semi-dropdown-item:has-text("${pattern.split('|')[1]}")`
    )
  }

  /**
   * Select multiple tenants by checkbox
   */
  async selectTenant(name: string): Promise<void> {
    const row = this.getTenantRowByName(name)
    await row.locator('.semi-checkbox').click()
  }

  /**
   * Click batch suspend button
   */
  async clickBatchSuspend(): Promise<void> {
    await this.page.click('button:has-text("Suspend Selected"), button:has-text("暂停选中")')
  }

  /**
   * Click batch activate button
   */
  async clickBatchActivate(): Promise<void> {
    await this.page.click('button:has-text("Activate Selected"), button:has-text("激活选中")')
  }

  // ============================================================================
  // Create Tenant Modal
  // ============================================================================

  /**
   * Fill create tenant form
   */
  async fillCreateTenantForm(data: {
    name: string
    code: string
    email?: string
    phone?: string
    address?: string
    plan?: 'free' | 'basic' | 'pro' | 'enterprise'
    trialDays?: number
  }): Promise<void> {
    // Wait for form to be visible
    await this.page.waitForSelector(this.modal, { state: 'visible' })

    await this.page.fill('input[name="name"], #name, input[placeholder*="名称"]', data.name)
    await this.page.fill('input[name="code"], #code, input[placeholder*="编码"]', data.code)

    if (data.email) {
      await this.page.fill(
        'input[name="contact_email"], #contact_email, input[type="email"]',
        data.email
      )
    }

    if (data.phone) {
      await this.page.fill(
        'input[name="contact_phone"], #contact_phone, input[type="tel"]',
        data.phone
      )
    }

    if (data.address) {
      await this.page.fill('textarea[name="address"], #address', data.address)
    }

    if (data.plan) {
      // Click plan select dropdown
      const planSelect = this.page
        .locator('.semi-select')
        .filter({ hasText: /Plan|套餐|free|basic|pro|enterprise/ })
        .first()
      await planSelect.click()
      await this.page.waitForTimeout(200)
      await this.page.click(`.semi-select-option:has-text("${data.plan}")`)
    }

    if (data.trialDays !== undefined) {
      await this.page.fill('input[name="trial_days"], #trial_days', data.trialDays.toString())
    }
  }

  /**
   * Submit create tenant form
   */
  async submitCreateTenant(): Promise<void> {
    await this.page.click(this.modalOkButton)
  }

  // ============================================================================
  // Edit Tenant Modal
  // ============================================================================

  /**
   * Fill edit tenant form
   */
  async fillEditTenantForm(data: {
    name?: string
    email?: string
    phone?: string
    address?: string
  }): Promise<void> {
    await this.page.waitForSelector(this.modal, { state: 'visible' })

    if (data.name) {
      await this.page.fill('input[name="name"], #name', data.name)
    }

    if (data.email) {
      await this.page.fill(
        'input[name="contact_email"], #contact_email, input[type="email"]',
        data.email
      )
    }

    if (data.phone) {
      await this.page.fill(
        'input[name="contact_phone"], #contact_phone, input[type="tel"]',
        data.phone
      )
    }

    if (data.address) {
      await this.page.fill('textarea[name="address"], #address', data.address)
    }
  }

  /**
   * Submit edit tenant form
   */
  async submitEditTenant(): Promise<void> {
    await this.page.click(this.modalOkButton)
  }

  // ============================================================================
  // Suspend/Activate Tenant
  // ============================================================================

  /**
   * Confirm suspend tenant in modal
   */
  async confirmSuspendTenant(reason?: string): Promise<void> {
    await this.page.waitForSelector(this.modal, { state: 'visible' })

    if (reason) {
      await this.page.fill('textarea[name="reason"], #reason, textarea', reason)
    }

    await this.page.click(this.modalOkButton)
  }

  /**
   * Confirm activate tenant in dialog
   */
  async confirmActivateTenant(): Promise<void> {
    // Semi Design Modal.confirm uses a different button selector
    await this.page.click('.semi-modal-footer .semi-button-primary')
  }

  // ============================================================================
  // Change Plan Modal
  // ============================================================================

  /**
   * Select new plan in change plan modal
   */
  async selectNewPlan(plan: 'free' | 'basic' | 'pro' | 'enterprise'): Promise<void> {
    await this.page.waitForSelector(this.modal, { state: 'visible' })

    // Click the plan option (typically radio or card)
    await this.page.click(
      `.semi-radio-addon:has-text("${plan}"), .plan-card:has-text("${plan}"), label:has-text("${plan}")`
    )
  }

  /**
   * Submit change plan
   */
  async submitChangePlan(): Promise<void> {
    await this.page.click(this.modalOkButton)
  }

  // ============================================================================
  // Update Quota Modal
  // ============================================================================

  /**
   * Fill quota values in update quota modal
   */
  async fillQuotaForm(data: {
    maxUsers?: number
    maxProducts?: number
    maxWarehouses?: number
  }): Promise<void> {
    await this.page.waitForSelector(this.modal, { state: 'visible' })

    if (data.maxUsers !== undefined) {
      await this.page.fill('input[name="max_users"], #max_users', data.maxUsers.toString())
    }

    if (data.maxProducts !== undefined) {
      await this.page.fill('input[name="max_products"], #max_products', data.maxProducts.toString())
    }

    if (data.maxWarehouses !== undefined) {
      await this.page.fill(
        'input[name="max_warehouses"], #max_warehouses',
        data.maxWarehouses.toString()
      )
    }
  }

  /**
   * Submit update quota
   */
  async submitUpdateQuota(): Promise<void> {
    await this.page.click(this.modalOkButton)
  }

  // ============================================================================
  // Delete Tenant Modal
  // ============================================================================

  /**
   * Confirm delete tenant (requires typing tenant name)
   */
  async confirmDeleteTenant(tenantName: string): Promise<void> {
    await this.page.waitForSelector(this.modal, { state: 'visible' })

    // Type tenant name for confirmation
    await this.page.fill(
      'input[placeholder*="name"], input[placeholder*="名称"], input[name="confirmName"]',
      tenantName
    )

    await this.page.click(this.modalOkButton)
  }

  // ============================================================================
  // Tenant Detail Page
  // ============================================================================

  /**
   * Click back button to return to list
   */
  async clickBack(): Promise<void> {
    await this.page.click(this.backButton)
    await this.waitForPageLoad()
  }

  /**
   * Click edit button on detail page
   */
  async clickEdit(): Promise<void> {
    await this.page.click(this.editButton)
    await this.page.waitForSelector(this.modal, { state: 'visible' })
  }

  /**
   * Click suspend button on detail page
   */
  async clickSuspend(): Promise<void> {
    await this.page.click(this.suspendButton)
    await this.page.waitForSelector(this.modal, { state: 'visible' })
  }

  /**
   * Click activate button on detail page
   */
  async clickActivate(): Promise<void> {
    await this.page.click(this.activateButton)
    // Activate uses Modal.confirm
    await this.page.waitForSelector('.semi-modal', { state: 'visible' })
  }

  /**
   * Click delete button on detail page
   */
  async clickDelete(): Promise<void> {
    await this.page.click(this.deleteButton)
    await this.page.waitForSelector(this.modal, { state: 'visible' })
  }

  /**
   * Click change plan button on detail page
   */
  async clickChangePlan(): Promise<void> {
    await this.page.click(this.changePlanButton)
    await this.page.waitForSelector(this.modal, { state: 'visible' })
  }

  /**
   * Click update quota button on detail page
   */
  async clickUpdateQuota(): Promise<void> {
    await this.page.click(this.updateQuotaButton)
    await this.page.waitForSelector(this.modal, { state: 'visible' })
  }

  /**
   * Get tenant status from detail page
   */
  async getTenantStatus(): Promise<string> {
    const statusTag = this.page
      .locator('.semi-tag')
      .filter({ hasText: /active|suspended|inactive|trial/ })
      .first()
    return (await statusTag.textContent()) || ''
  }

  /**
   * Get tenant plan from detail page
   */
  async getTenantPlan(): Promise<string> {
    const planTag = this.page
      .locator('.semi-tag')
      .filter({ hasText: /free|basic|pro|enterprise/ })
      .first()
    return (await planTag.textContent()) || ''
  }

  // ============================================================================
  // Audit Logs Page
  // ============================================================================

  /**
   * Filter audit logs by action
   */
  async filterAuditByAction(action: string): Promise<void> {
    const actionSelect = this.page
      .locator('.semi-select')
      .filter({ hasText: /action|操作/ })
      .first()
    await actionSelect.click()
    await this.page.waitForTimeout(200)
    await this.page.click(`.semi-select-option:has-text("${action}")`)
    await this.waitForTableLoad()
  }

  /**
   * Search audit logs
   */
  async searchAuditLogs(keyword: string): Promise<void> {
    await this.page.fill('input[placeholder*="Search"], input[placeholder*="搜索"]', keyword)
    await this.page.waitForTimeout(500) // Wait for debounce
    await this.waitForTableLoad()
  }

  /**
   * Get audit log rows count
   */
  async getAuditLogCount(): Promise<number> {
    return this.page.locator(this.auditRow).count()
  }

  // ============================================================================
  // Assertions
  // ============================================================================

  /**
   * Assert tenant list page is displayed
   */
  async assertOnTenantListPage(): Promise<void> {
    await expect(this.page).toHaveURL(/.*super-admin\/tenants$/)
    await expect(this.page.locator('text=Tenant Management, text=租户管理')).toBeVisible()
  }

  /**
   * Assert tenant detail page is displayed
   */
  async assertOnTenantDetailPage(): Promise<void> {
    await expect(this.page).toHaveURL(/.*super-admin\/tenants\/.*/)
    await expect(this.page.locator(this.backButton)).toBeVisible()
  }

  /**
   * Assert tenant exists in list
   */
  async assertTenantInList(name: string): Promise<void> {
    await expect(this.getTenantRowByName(name)).toBeVisible()
  }

  /**
   * Assert tenant not in list
   */
  async assertTenantNotInList(name: string): Promise<void> {
    await expect(this.getTenantRowByName(name)).not.toBeVisible()
  }

  /**
   * Assert tenant status
   */
  async assertTenantStatusIs(
    name: string,
    status: 'active' | 'suspended' | 'inactive' | 'trial'
  ): Promise<void> {
    const row = this.getTenantRowByName(name)
    await expect(row.locator('.semi-tag').filter({ hasText: status })).toBeVisible()
  }

  /**
   * Assert tenant plan
   */
  async assertTenantPlanIs(
    name: string,
    plan: 'free' | 'basic' | 'pro' | 'enterprise'
  ): Promise<void> {
    const row = this.getTenantRowByName(name)
    await expect(row.locator('.semi-tag').filter({ hasText: plan })).toBeVisible()
  }

  /**
   * Assert modal is visible
   */
  async assertModalVisible(): Promise<void> {
    await expect(this.page.locator(this.modal)).toBeVisible()
  }

  /**
   * Assert modal is hidden
   */
  async assertModalHidden(): Promise<void> {
    await expect(this.page.locator(this.modal)).not.toBeVisible()
  }

  /**
   * Assert success toast
   */
  async assertSuccessToast(message?: string): Promise<void> {
    const toastSelector = message
      ? `.semi-toast-content:has-text("${message}")`
      : '.semi-toast-content'
    await expect(this.page.locator(toastSelector).first()).toBeVisible()
  }

  /**
   * Assert on audit logs page
   */
  async assertOnAuditLogsPage(): Promise<void> {
    await expect(this.page).toHaveURL(/.*super-admin\/audit-logs.*/)
    await expect(this.page.locator('text=Audit Logs, text=审计日志')).toBeVisible()
  }
}
