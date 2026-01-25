import { test, expect } from '../fixtures/test-fixtures'

/**
 * Supplier Module E2E Tests
 *
 * Requirements covered (P1-INT-002):
 * - Docker 环境: 使用 seed 供应商数据
 * - E2E 供应商: 列表展示、新建、编辑、状态变更
 * - 截图断言: 供应商列表和表单页面
 */
test.describe('Supplier Module', () => {
  // Use authenticated state from setup
  test.use({ storageState: 'tests/e2e/.auth/user.json' })

  test.describe('Supplier List Display', () => {
    test('should display seed supplier data correctly', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Verify page loaded with data
      const count = await suppliersPage.getSupplierCount()
      expect(count).toBeGreaterThan(0)

      // Verify seed data - Apple China Distribution
      await suppliersPage.assertSupplierExists('SUP001')

      // Verify Samsung Electronics China
      await suppliersPage.assertSupplierExists('SUP002')

      // Verify Xiaomi Technology Ltd
      await suppliersPage.assertSupplierExists('SUP003')
    })

    test('should display supplier details in table row', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Find Apple China Distribution
      const row = await suppliersPage.findSupplierRowByCode('SUP001')
      expect(row).not.toBeNull()

      if (row) {
        // Verify supplier name contains expected text
        const nameCell = row.locator('.supplier-name')
        await expect(nameCell).toContainText('Apple China')

        // Verify status tag is shown
        const statusTag = row.locator('.semi-tag').last()
        await expect(statusTag).toBeVisible()
      }
    })

    test('should display supplier rating correctly', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Apple China should have 5-star rating
      const row = await suppliersPage.findSupplierRowByCode('SUP001')
      if (row) {
        const ratingComponent = row.locator('.semi-rating')
        await expect(ratingComponent).toBeVisible()
      }
    })
  })

  test.describe('Supplier Creation', () => {
    test('should navigate to create page', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()
      await suppliersPage.clickAddSupplier()

      await suppliersPage.assertPageTitle('新增供应商')
      await expect(suppliersPage.codeInput).toBeVisible()
      await expect(suppliersPage.nameInput).toBeVisible()
    })

    test('should create new supplier with full form', async ({ suppliersPage }) => {
      const uniqueCode = `TEST-SUP-${Date.now()}`

      await suppliersPage.navigateToCreate()

      await suppliersPage.fillSupplierForm({
        code: uniqueCode,
        name: 'Test Supplier E2E',
        shortName: 'Test E2E',
        type: 'distributor',
        contactName: 'Test Contact',
        phone: '13900000000',
        email: 'testsupplier@example.com',
        taxId: '912345678901234567',
        bankName: 'Test Bank',
        bankAccount: '1234567890123456789',
        province: 'Test Province',
        city: 'Test City',
        address: 'Test Supplier Address 123',
        creditDays: 30,
        creditLimit: 100000,
      })

      await suppliersPage.submitForm()
      await suppliersPage.waitForFormSuccess()

      // Verify supplier was created
      await suppliersPage.assertSupplierExists(uniqueCode)
    })

    test('should validate required fields', async ({ suppliersPage }) => {
      await suppliersPage.navigateToCreate()

      // Try to submit empty form
      await suppliersPage.submitForm()

      // Should show validation errors
      await suppliersPage.page.waitForTimeout(500)

      // Check that we're still on the create page (form wasn't submitted)
      await suppliersPage.assertUrlContains('/partner/suppliers/new')
    })

    test('should validate supplier code format', async ({ suppliersPage }) => {
      await suppliersPage.navigateToCreate()

      // Fill with invalid code containing special characters
      await suppliersPage.fillSupplierForm({
        code: 'INVALID@CODE!',
        name: 'Test Supplier',
      })

      await suppliersPage.submitForm()
      await suppliersPage.page.waitForTimeout(500)

      // Should still be on create page due to validation error
      await suppliersPage.assertUrlContains('/partner/suppliers/new')
    })
  })

  test.describe('Supplier Editing', () => {
    test('should edit existing supplier', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Find an existing supplier
      const row = await suppliersPage.findSupplierRowByCode('SUP005')
      expect(row).not.toBeNull()

      if (row) {
        await suppliersPage.clickRowAction(row, 'edit')
        await suppliersPage.page.waitForURL('**/partner/suppliers/**/edit')

        // Verify edit page loaded
        await suppliersPage.assertPageTitle('编辑供应商')

        // Code field should be disabled in edit mode
        await expect(suppliersPage.codeInput).toBeDisabled()

        // Update the name
        await suppliersPage.nameInput.fill('General Supplies Trading Updated')
        await suppliersPage.submitForm()
        await suppliersPage.waitForFormSuccess()

        // Verify the update
        const updatedRow = await suppliersPage.findSupplierRowByName('General Supplies Trading Updated')
        expect(updatedRow).not.toBeNull()
      }
    })

    test('should have code field disabled in edit mode', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      const row = await suppliersPage.findSupplierRowByCode('SUP001')
      if (row) {
        await suppliersPage.clickRowAction(row, 'edit')
        await suppliersPage.page.waitForURL('**/partner/suppliers/**/edit')

        // Code should be disabled
        await expect(suppliersPage.codeInput).toBeDisabled()
      }
    })
  })

  test.describe('Supplier Status Management', () => {
    test('should deactivate active supplier', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Find an active supplier
      const row = await suppliersPage.findSupplierRowByCode('SUP005')
      if (row) {
        await suppliersPage.clickRowAction(row, 'deactivate')
        await suppliersPage.page.waitForTimeout(500)

        // Verify status changed
        await suppliersPage.assertSupplierStatus('SUP005', '停用')
      }
    })

    test('should activate deactivated supplier', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Find the supplier we deactivated
      const row = await suppliersPage.findSupplierRowByCode('SUP005')
      if (row) {
        await suppliersPage.clickRowAction(row, 'activate')
        await suppliersPage.page.waitForTimeout(500)

        // Verify status changed back
        await suppliersPage.assertSupplierStatus('SUP005', '启用')
      }
    })

    test('should block supplier with confirmation', async ({ suppliersPage }) => {
      // First create a supplier to block
      const uniqueCode = `BLOCK-SUP-${Date.now()}`

      await suppliersPage.navigateToCreate()
      await suppliersPage.fillSupplierForm({
        code: uniqueCode,
        name: 'Supplier To Block',
        type: 'distributor',
      })
      await suppliersPage.submitForm()
      await suppliersPage.waitForFormSuccess()

      // Now block it
      const row = await suppliersPage.findSupplierRowByCode(uniqueCode)
      if (row) {
        await suppliersPage.clickRowAction(row, 'block')
        await suppliersPage.confirmDialog()
        await suppliersPage.page.waitForTimeout(500)

        // Verify status changed to blocked
        await suppliersPage.assertSupplierStatus(uniqueCode, '拉黑')
      }
    })

    test('should filter suppliers by status', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Filter by active status
      await suppliersPage.filterByStatus('active')

      // All visible suppliers should have active status
      const count = await suppliersPage.getSupplierCount()
      expect(count).toBeGreaterThan(0)

      // Reset filter
      await suppliersPage.filterByStatus('')
    })
  })

  test.describe('Supplier Deletion', () => {
    test('should delete supplier with confirmation', async ({ suppliersPage }) => {
      // First create a supplier to delete
      const uniqueCode = `DEL-SUP-${Date.now()}`

      await suppliersPage.navigateToCreate()
      await suppliersPage.fillSupplierForm({
        code: uniqueCode,
        name: 'Supplier To Delete',
        type: 'retailer',
      })
      await suppliersPage.submitForm()
      await suppliersPage.waitForFormSuccess()

      // Now delete it
      const row = await suppliersPage.findSupplierRowByCode(uniqueCode)
      if (row) {
        await suppliersPage.clickRowAction(row, 'delete')
        await suppliersPage.confirmDialog()
        await suppliersPage.page.waitForTimeout(500)

        // Verify supplier was deleted
        await suppliersPage.assertSupplierNotExists(uniqueCode)
      }
    })

    test('should cancel delete dialog', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      const row = await suppliersPage.findSupplierRowByCode('SUP004')
      if (row) {
        await suppliersPage.clickRowAction(row, 'delete')
        await suppliersPage.cancelDialog()

        // Supplier should still exist
        await suppliersPage.assertSupplierExists('SUP004')
      }
    })
  })

  test.describe('Search and Filter', () => {
    test('should search suppliers by name', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      await suppliersPage.search('Apple')
      const count = await suppliersPage.getSupplierCount()
      expect(count).toBeGreaterThanOrEqual(1)

      // Should find Apple China
      await suppliersPage.assertSupplierExists('SUP001')
    })

    test('should search suppliers by code', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      await suppliersPage.search('SUP002')
      const count = await suppliersPage.getSupplierCount()
      expect(count).toBe(1)
    })

    test('should show empty state for no results', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      await suppliersPage.search('NONEXISTENT12345')
      const count = await suppliersPage.getSupplierCount()
      expect(count).toBe(0)
    })

    test('should clear search and show all suppliers', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Search first
      await suppliersPage.search('Samsung')
      const filteredCount = await suppliersPage.getSupplierCount()

      // Clear search
      await suppliersPage.clearSearch()
      const allCount = await suppliersPage.getSupplierCount()

      expect(allCount).toBeGreaterThanOrEqual(filteredCount)
    })

    test('should filter by supplier type', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()

      // Filter by manufacturer type
      await suppliersPage.filterByType('manufacturer')
      const count = await suppliersPage.getSupplierCount()
      expect(count).toBeGreaterThan(0)

      // Reset filter
      await suppliersPage.filterByType('')
    })
  })

  test.describe('Screenshots', () => {
    test('should capture supplier list page screenshot', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()
      await suppliersPage.screenshotList('suppliers-list')
    })

    test('should capture supplier create form screenshot', async ({ suppliersPage }) => {
      await suppliersPage.navigateToCreate()
      await suppliersPage.screenshotForm('supplier-create-form')
    })

    test('should capture supplier edit form screenshot', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()
      const row = await suppliersPage.findSupplierRowByCode('SUP001')
      if (row) {
        await suppliersPage.clickRowAction(row, 'edit')
        await suppliersPage.page.waitForURL('**/partner/suppliers/**/edit')
        await suppliersPage.screenshotForm('supplier-edit-form')
      }
    })

    test('should capture filtered supplier list screenshot', async ({ suppliersPage }) => {
      await suppliersPage.navigateToList()
      await suppliersPage.filterByStatus('active')
      await suppliersPage.screenshotList('suppliers-list-filtered')
    })
  })
})
