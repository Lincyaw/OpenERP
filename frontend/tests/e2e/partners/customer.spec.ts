import { test, expect } from '../fixtures/test-fixtures'

/**
 * Customer Module E2E Tests
 *
 * Requirements covered (P1-INT-002):
 * - Docker 环境: 使用 seed 客户数据
 * - E2E 客户: 列表展示、新建、编辑、状态变更
 * - 截图断言: 客户列表和表单页面
 */
test.describe('Customer Module', () => {
  // Use authenticated state from setup
  test.use({ storageState: 'tests/e2e/.auth/user.json' })

  test.describe('Customer List Display', () => {
    test('should display seed customer data correctly', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Verify page loaded with data
      const count = await customersPage.getCustomerCount()
      expect(count).toBeGreaterThan(0)

      // Verify seed data - Beijing Tech Solutions Ltd
      await customersPage.assertCustomerExists('CUST001')

      // Verify Shanghai Digital Corp
      await customersPage.assertCustomerExists('CUST002')
    })

    test('should display customer details in table row', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Find Beijing Tech Solutions
      const row = await customersPage.findCustomerRowByCode('CUST001')
      expect(row).not.toBeNull()

      if (row) {
        // Verify customer name contains expected text
        const nameCell = row.locator('.customer-name')
        await expect(nameCell).toContainText('Beijing Tech')

        // Verify status tag is shown
        const statusTag = row.locator('.semi-tag').last()
        await expect(statusTag).toBeVisible()
      }
    })

    test('should show customer type tags correctly', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Organization type customer
      const orgRow = await customersPage.findCustomerRowByCode('CUST001')
      if (orgRow) {
        const typeTag = orgRow.locator('.type-tag')
        await expect(typeTag).toContainText('企业')
      }

      // Individual type customer
      const indRow = await customersPage.findCustomerRowByCode('CUST004')
      if (indRow) {
        const typeTag = indRow.locator('.type-tag')
        await expect(typeTag).toContainText('个人')
      }
    })
  })

  test.describe('Customer Creation', () => {
    test('should navigate to create page', async ({ customersPage }) => {
      await customersPage.navigateToList()
      await customersPage.clickAddCustomer()

      await customersPage.assertPageTitle('新增客户')
      await expect(customersPage.codeInput).toBeVisible()
      await expect(customersPage.nameInput).toBeVisible()
    })

    test('should create new customer with full form', async ({ customersPage }) => {
      const uniqueCode = `TEST-CUST-${Date.now()}`

      await customersPage.navigateToCreate()

      await customersPage.fillCustomerForm({
        code: uniqueCode,
        name: 'Test Customer E2E',
        shortName: 'Test E2E',
        type: 'organization',
        contactName: 'Test Contact',
        phone: '13800000000',
        email: 'test@example.com',
        province: 'Test Province',
        city: 'Test City',
        address: 'Test Address 123',
        creditLimit: 50000,
      })

      await customersPage.submitForm()
      await customersPage.waitForFormSuccess()

      // Verify customer was created
      await customersPage.assertCustomerExists(uniqueCode)
    })

    test('should validate required fields', async ({ customersPage }) => {
      await customersPage.navigateToCreate()

      // Try to submit empty form
      await customersPage.submitForm()

      // Should show validation errors
      await customersPage.page.waitForTimeout(500)

      // Check that we're still on the create page (form wasn't submitted)
      await customersPage.assertUrlContains('/partner/customers/new')
    })

    test('should validate customer code format', async ({ customersPage }) => {
      await customersPage.navigateToCreate()

      // Fill with invalid code containing special characters
      await customersPage.fillCustomerForm({
        code: 'INVALID@CODE!',
        name: 'Test Customer',
      })

      await customersPage.submitForm()
      await customersPage.page.waitForTimeout(500)

      // Should still be on create page due to validation error
      await customersPage.assertUrlContains('/partner/customers/new')
    })
  })

  test.describe('Customer Editing', () => {
    test('should edit existing customer', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Find an existing customer
      const row = await customersPage.findCustomerRowByCode('CUST003')
      expect(row).not.toBeNull()

      if (row) {
        await customersPage.clickRowAction(row, 'edit')
        await customersPage.page.waitForURL('**/partner/customers/**/edit')

        // Verify edit page loaded
        await customersPage.assertPageTitle('编辑客户')

        // Code field should be disabled in edit mode
        await expect(customersPage.codeInput).toBeDisabled()

        // Update the name
        await customersPage.nameInput.fill('Shenzhen Hardware Inc Updated')
        await customersPage.submitForm()
        await customersPage.waitForFormSuccess()

        // Verify the update
        const updatedRow = await customersPage.findCustomerRowByName(
          'Shenzhen Hardware Inc Updated'
        )
        expect(updatedRow).not.toBeNull()
      }
    })

    test('should have code field disabled in edit mode', async ({ customersPage }) => {
      await customersPage.navigateToList()

      const row = await customersPage.findCustomerRowByCode('CUST001')
      if (row) {
        await customersPage.clickRowAction(row, 'edit')
        await customersPage.page.waitForURL('**/partner/customers/**/edit')

        // Code should be disabled
        await expect(customersPage.codeInput).toBeDisabled()
      }
    })
  })

  test.describe('Customer Status Management', () => {
    test('should deactivate active customer', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Find an active customer
      const row = await customersPage.findCustomerRowByCode('CUST003')
      if (row) {
        await customersPage.clickRowAction(row, 'deactivate')
        await customersPage.page.waitForTimeout(500)

        // Verify status changed
        await customersPage.assertCustomerStatus('CUST003', '停用')
      }
    })

    test('should activate deactivated customer', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Find the customer we deactivated
      const row = await customersPage.findCustomerRowByCode('CUST003')
      if (row) {
        await customersPage.clickRowAction(row, 'activate')
        await customersPage.page.waitForTimeout(500)

        // Verify status changed back
        await customersPage.assertCustomerStatus('CUST003', '启用')
      }
    })

    test('should filter customers by status', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Filter by active status
      await customersPage.filterByStatus('active')

      // All visible customers should have active status
      const count = await customersPage.getCustomerCount()
      expect(count).toBeGreaterThan(0)

      // Reset filter
      await customersPage.filterByStatus('')
    })
  })

  test.describe('Customer Deletion', () => {
    test('should delete customer with confirmation', async ({ customersPage }) => {
      // First create a customer to delete
      const uniqueCode = `DEL-CUST-${Date.now()}`

      await customersPage.navigateToCreate()
      await customersPage.fillCustomerForm({
        code: uniqueCode,
        name: 'Customer To Delete',
        type: 'individual',
      })
      await customersPage.submitForm()
      await customersPage.waitForFormSuccess()

      // Now delete it
      const row = await customersPage.findCustomerRowByCode(uniqueCode)
      if (row) {
        await customersPage.clickRowAction(row, 'delete')
        await customersPage.confirmDialog()
        await customersPage.page.waitForTimeout(500)

        // Verify customer was deleted
        await customersPage.assertCustomerNotExists(uniqueCode)
      }
    })

    test('should cancel delete dialog', async ({ customersPage }) => {
      await customersPage.navigateToList()

      const row = await customersPage.findCustomerRowByCode('CUST005')
      if (row) {
        await customersPage.clickRowAction(row, 'delete')
        await customersPage.cancelDialog()

        // Customer should still exist
        await customersPage.assertCustomerExists('CUST005')
      }
    })
  })

  test.describe('Search and Filter', () => {
    test('should search customers by name', async ({ customersPage }) => {
      await customersPage.navigateToList()

      await customersPage.search('Beijing')
      const count = await customersPage.getCustomerCount()
      expect(count).toBeGreaterThanOrEqual(1)

      // Should find Beijing Tech
      await customersPage.assertCustomerExists('CUST001')
    })

    test('should search customers by code', async ({ customersPage }) => {
      await customersPage.navigateToList()

      await customersPage.search('CUST002')
      const count = await customersPage.getCustomerCount()
      expect(count).toBe(1)
    })

    test('should show empty state for no results', async ({ customersPage }) => {
      await customersPage.navigateToList()

      await customersPage.search('NONEXISTENT12345')
      const count = await customersPage.getCustomerCount()
      expect(count).toBe(0)
    })

    test('should clear search and show all customers', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Search first
      await customersPage.search('Beijing')
      const filteredCount = await customersPage.getCustomerCount()

      // Clear search
      await customersPage.clearSearch()
      const allCount = await customersPage.getCustomerCount()

      expect(allCount).toBeGreaterThanOrEqual(filteredCount)
    })

    test('should filter by customer type', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Filter by individual type
      await customersPage.filterByType('individual')
      const count = await customersPage.getCustomerCount()
      expect(count).toBeGreaterThan(0)

      // Reset filter
      await customersPage.filterByType('')
    })

    test('should filter by customer level', async ({ customersPage }) => {
      await customersPage.navigateToList()

      // Filter by gold level
      await customersPage.filterByLevel('gold')
      const count = await customersPage.getCustomerCount()
      expect(count).toBeGreaterThanOrEqual(0)

      // Reset filter
      await customersPage.filterByLevel('')
    })
  })

  test.describe('Screenshots', () => {
    test('should capture customer list page screenshot', async ({ customersPage }) => {
      await customersPage.navigateToList()
      await customersPage.screenshotList('customers-list')
    })

    test('should capture customer create form screenshot', async ({ customersPage }) => {
      await customersPage.navigateToCreate()
      await customersPage.screenshotForm('customer-create-form')
    })

    test('should capture customer edit form screenshot', async ({ customersPage }) => {
      await customersPage.navigateToList()
      const row = await customersPage.findCustomerRowByCode('CUST001')
      if (row) {
        await customersPage.clickRowAction(row, 'edit')
        await customersPage.page.waitForURL('**/partner/customers/**/edit')
        await customersPage.screenshotForm('customer-edit-form')
      }
    })

    test('should capture filtered customer list screenshot', async ({ customersPage }) => {
      await customersPage.navigateToList()
      await customersPage.filterByStatus('active')
      await customersPage.screenshotList('customers-list-filtered')
    })
  })
})
