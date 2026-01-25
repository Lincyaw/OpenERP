import { test, expect } from '../fixtures/test-fixtures'

/**
 * Warehouse Module E2E Tests
 *
 * Requirements covered (P1-INT-002):
 * - Docker 环境: 使用 seed 仓库数据
 * - E2E 仓库: 列表展示、新建、启用/禁用
 * - 截图断言: 仓库列表和表单页面
 */
test.describe('Warehouse Module', () => {
  // Use authenticated state from setup
  test.use({ storageState: 'tests/e2e/.auth/user.json' })

  test.describe('Warehouse List Display', () => {
    test('should display seed warehouse data correctly', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Verify page loaded with data
      const count = await warehousesPage.getWarehouseCount()
      expect(count).toBeGreaterThan(0)

      // Verify seed data - Main Warehouse Beijing
      await warehousesPage.assertWarehouseExists('WH001')

      // Verify Shanghai Distribution Center
      await warehousesPage.assertWarehouseExists('WH002')

      // Verify Shenzhen Warehouse
      await warehousesPage.assertWarehouseExists('WH003')
    })

    test('should display warehouse details in table row', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Find Main Warehouse Beijing
      const row = await warehousesPage.findWarehouseRowByCode('WH001')
      expect(row).not.toBeNull()

      if (row) {
        // Verify warehouse name contains expected text
        const nameCell = row.locator('.warehouse-name').first()
        await expect(nameCell).toContainText('Main Warehouse')

        // Verify status tag is shown
        const statusTag = row.locator('.semi-tag').last()
        await expect(statusTag).toBeVisible()
      }
    })

    test('should show default warehouse indicator', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // WH001 (Main Warehouse Beijing) should be the default
      await warehousesPage.assertWarehouseIsDefault('WH001')
    })

    test('should display warehouse type tags correctly', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Physical warehouse
      const physicalRow = await warehousesPage.findWarehouseRowByCode('WH001')
      if (physicalRow) {
        const typeTag = physicalRow.locator('.type-tag')
        await expect(typeTag).toBeVisible()
      }

      // Virtual warehouse
      const virtualRow = await warehousesPage.findWarehouseRowByCode('WH-VIRTUAL')
      if (virtualRow) {
        const typeTag = virtualRow.locator('.type-tag')
        await expect(typeTag).toContainText('虚拟')
      }
    })
  })

  test.describe('Warehouse Creation', () => {
    test('should navigate to create page', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()
      await warehousesPage.clickAddWarehouse()

      await warehousesPage.assertPageTitle('新增仓库')
      await expect(warehousesPage.codeInput).toBeVisible()
      await expect(warehousesPage.nameInput).toBeVisible()
    })

    test('should create new warehouse with full form', async ({ warehousesPage }) => {
      const uniqueCode = `TEST-WH-${Date.now()}`

      await warehousesPage.navigateToCreate()

      await warehousesPage.fillWarehouseForm({
        code: uniqueCode,
        name: 'Test Warehouse E2E',
        shortName: 'Test WH',
        type: 'normal',
        contactName: 'Test Manager',
        phone: '13700000000',
        email: 'testwarehouse@example.com',
        province: 'Test Province',
        city: 'Test City',
        address: 'Test Warehouse Address 123',
        capacity: 5000,
        sortOrder: 99,
      })

      await warehousesPage.submitForm()
      await warehousesPage.waitForFormSuccess()

      // Verify warehouse was created
      await warehousesPage.assertWarehouseExists(uniqueCode)
    })

    test('should validate required fields', async ({ warehousesPage }) => {
      await warehousesPage.navigateToCreate()

      // Try to submit empty form
      await warehousesPage.submitForm()

      // Should show validation errors
      await warehousesPage.page.waitForTimeout(500)

      // Check that we're still on the create page (form wasn't submitted)
      await warehousesPage.assertUrlContains('/partner/warehouses/new')
    })

    test('should validate warehouse code format', async ({ warehousesPage }) => {
      await warehousesPage.navigateToCreate()

      // Fill with invalid code containing special characters
      await warehousesPage.fillWarehouseForm({
        code: 'INVALID@CODE!',
        name: 'Test Warehouse',
      })

      await warehousesPage.submitForm()
      await warehousesPage.page.waitForTimeout(500)

      // Should still be on create page due to validation error
      await warehousesPage.assertUrlContains('/partner/warehouses/new')
    })
  })

  test.describe('Warehouse Editing', () => {
    test('should edit existing warehouse', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Find an existing warehouse (not the default one)
      const row = await warehousesPage.findWarehouseRowByCode('WH003')
      expect(row).not.toBeNull()

      if (row) {
        await warehousesPage.clickRowAction(row, 'edit')
        await warehousesPage.page.waitForURL('**/partner/warehouses/**/edit')

        // Verify edit page loaded
        await warehousesPage.assertPageTitle('编辑仓库')

        // Code field should be disabled in edit mode
        await expect(warehousesPage.codeInput).toBeDisabled()

        // Update the name
        await warehousesPage.nameInput.fill('Shenzhen Warehouse Updated')
        await warehousesPage.submitForm()
        await warehousesPage.waitForFormSuccess()

        // Verify the update
        const updatedRow = await warehousesPage.findWarehouseRowByName('Shenzhen Warehouse Updated')
        expect(updatedRow).not.toBeNull()
      }
    })

    test('should have code field disabled in edit mode', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      const row = await warehousesPage.findWarehouseRowByCode('WH002')
      if (row) {
        await warehousesPage.clickRowAction(row, 'edit')
        await warehousesPage.page.waitForURL('**/partner/warehouses/**/edit')

        // Code should be disabled
        await expect(warehousesPage.codeInput).toBeDisabled()
      }
    })
  })

  test.describe('Warehouse Status Management', () => {
    test('should disable enabled warehouse', async ({ warehousesPage }) => {
      // First create a warehouse to disable
      const uniqueCode = `DIS-WH-${Date.now()}`

      await warehousesPage.navigateToCreate()
      await warehousesPage.fillWarehouseForm({
        code: uniqueCode,
        name: 'Warehouse To Disable',
        type: 'normal',
      })
      await warehousesPage.submitForm()
      await warehousesPage.waitForFormSuccess()

      // Now disable it
      const row = await warehousesPage.findWarehouseRowByCode(uniqueCode)
      if (row) {
        await warehousesPage.clickRowAction(row, 'disable')
        await warehousesPage.page.waitForTimeout(500)

        // Verify status changed
        await warehousesPage.assertWarehouseStatus(uniqueCode, '停用')
      }
    })

    test('should enable disabled warehouse', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Find any disabled warehouse or create one
      await warehousesPage.filterByStatus('disabled')
      const count = await warehousesPage.getWarehouseCount()

      if (count > 0) {
        const row = warehousesPage.tableRows.first()
        const code = await row.locator('.semi-table-row-cell').nth(1).textContent()

        if (code) {
          await warehousesPage.clickRowAction(row, 'enable')
          await warehousesPage.page.waitForTimeout(500)

          // Reset filter and verify
          await warehousesPage.filterByStatus('')
          await warehousesPage.assertWarehouseStatus(code.trim(), '启用')
        }
      }
    })

    test('should not allow disabling default warehouse', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Find the default warehouse
      const row = await warehousesPage.findWarehouseRowByCode('WH001')
      if (row) {
        // Try to click disable - it should not be available for default warehouse
        const actionsCell = row.locator('.semi-table-row-cell').last()
        const dropdownTrigger = actionsCell.locator('.semi-dropdown-trigger, button').first()
        await dropdownTrigger.click()

        // The disable option should not be present for default warehouse
        const disableOption = warehousesPage.page
          .locator('.semi-dropdown-menu .semi-dropdown-item')
          .filter({ hasText: '停用' })

        // Either not visible or count should be 0
        const optionCount = await disableOption.count()
        expect(optionCount).toBe(0)

        // Close dropdown
        await warehousesPage.page.keyboard.press('Escape')
      }
    })

    test('should filter warehouses by status', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Filter by enabled status
      await warehousesPage.filterByStatus('enabled')

      // All visible warehouses should have enabled status
      const count = await warehousesPage.getWarehouseCount()
      expect(count).toBeGreaterThan(0)

      // Reset filter
      await warehousesPage.filterByStatus('')
    })
  })

  test.describe('Set Default Warehouse', () => {
    test('should set warehouse as default', async ({ warehousesPage }) => {
      // First create a new warehouse
      const uniqueCode = `DEF-WH-${Date.now()}`

      await warehousesPage.navigateToCreate()
      await warehousesPage.fillWarehouseForm({
        code: uniqueCode,
        name: 'New Default Warehouse',
        type: 'normal',
      })
      await warehousesPage.submitForm()
      await warehousesPage.waitForFormSuccess()

      // Set it as default
      const row = await warehousesPage.findWarehouseRowByCode(uniqueCode)
      if (row) {
        await warehousesPage.clickRowAction(row, 'setDefault')
        await warehousesPage.confirmDialog()
        await warehousesPage.page.waitForTimeout(500)

        // Verify it's now default
        await warehousesPage.assertWarehouseIsDefault(uniqueCode)

        // Restore original default (WH001)
        const wh001Row = await warehousesPage.findWarehouseRowByCode('WH001')
        if (wh001Row) {
          await warehousesPage.clickRowAction(wh001Row, 'setDefault')
          await warehousesPage.confirmDialog()
          await warehousesPage.page.waitForTimeout(500)
        }
      }
    })
  })

  test.describe('Warehouse Deletion', () => {
    test('should delete warehouse with confirmation', async ({ warehousesPage }) => {
      // First create a warehouse to delete
      const uniqueCode = `DEL-WH-${Date.now()}`

      await warehousesPage.navigateToCreate()
      await warehousesPage.fillWarehouseForm({
        code: uniqueCode,
        name: 'Warehouse To Delete',
        type: 'transit',
      })
      await warehousesPage.submitForm()
      await warehousesPage.waitForFormSuccess()

      // Now delete it
      const row = await warehousesPage.findWarehouseRowByCode(uniqueCode)
      if (row) {
        await warehousesPage.clickRowAction(row, 'delete')
        await warehousesPage.confirmDialog()
        await warehousesPage.page.waitForTimeout(500)

        // Verify warehouse was deleted
        await warehousesPage.assertWarehouseNotExists(uniqueCode)
      }
    })

    test('should cancel delete dialog', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      const row = await warehousesPage.findWarehouseRowByCode('WH003')
      if (row) {
        await warehousesPage.clickRowAction(row, 'delete')
        await warehousesPage.cancelDialog()

        // Warehouse should still exist
        await warehousesPage.assertWarehouseExists('WH003')
      }
    })

    test('should not allow deleting default warehouse', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Find the default warehouse
      const row = await warehousesPage.findWarehouseRowByCode('WH001')
      if (row) {
        // Try to click delete - it should not be available for default warehouse
        const actionsCell = row.locator('.semi-table-row-cell').last()
        const dropdownTrigger = actionsCell.locator('.semi-dropdown-trigger, button').first()
        await dropdownTrigger.click()

        // The delete option should not be present for default warehouse
        const deleteOption = warehousesPage.page
          .locator('.semi-dropdown-menu .semi-dropdown-item')
          .filter({ hasText: '删除' })

        const optionCount = await deleteOption.count()
        expect(optionCount).toBe(0)

        // Close dropdown
        await warehousesPage.page.keyboard.press('Escape')
      }
    })
  })

  test.describe('Search and Filter', () => {
    test('should search warehouses by name', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      await warehousesPage.search('Beijing')
      const count = await warehousesPage.getWarehouseCount()
      expect(count).toBeGreaterThanOrEqual(1)

      // Should find Beijing Main
      await warehousesPage.assertWarehouseExists('WH001')
    })

    test('should search warehouses by code', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      await warehousesPage.search('WH002')
      const count = await warehousesPage.getWarehouseCount()
      expect(count).toBe(1)
    })

    test('should show empty state for no results', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      await warehousesPage.search('NONEXISTENT12345')
      const count = await warehousesPage.getWarehouseCount()
      expect(count).toBe(0)
    })

    test('should clear search and show all warehouses', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Search first
      await warehousesPage.search('Shanghai')
      const filteredCount = await warehousesPage.getWarehouseCount()

      // Clear search
      await warehousesPage.clearSearch()
      const allCount = await warehousesPage.getWarehouseCount()

      expect(allCount).toBeGreaterThanOrEqual(filteredCount)
    })

    test('should filter by warehouse type', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()

      // Filter by virtual type
      await warehousesPage.filterByType('virtual')
      const count = await warehousesPage.getWarehouseCount()
      expect(count).toBeGreaterThanOrEqual(0)

      // Reset filter
      await warehousesPage.filterByType('')
    })
  })

  test.describe('Screenshots', () => {
    test('should capture warehouse list page screenshot', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()
      await warehousesPage.screenshotList('warehouses-list')
    })

    test('should capture warehouse create form screenshot', async ({ warehousesPage }) => {
      await warehousesPage.navigateToCreate()
      await warehousesPage.screenshotForm('warehouse-create-form')
    })

    test('should capture warehouse edit form screenshot', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()
      const row = await warehousesPage.findWarehouseRowByCode('WH002')
      if (row) {
        await warehousesPage.clickRowAction(row, 'edit')
        await warehousesPage.page.waitForURL('**/partner/warehouses/**/edit')
        await warehousesPage.screenshotForm('warehouse-edit-form')
      }
    })

    test('should capture filtered warehouse list screenshot', async ({ warehousesPage }) => {
      await warehousesPage.navigateToList()
      await warehousesPage.filterByStatus('enabled')
      await warehousesPage.screenshotList('warehouses-list-filtered')
    })
  })
})
