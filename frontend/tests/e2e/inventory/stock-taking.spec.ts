import { test, expect } from '../fixtures'
import { InventoryPage } from '../pages'

/**
 * P2-INT-002: Stock Taking Module E2E Integration Tests (盘点功能联调)
 *
 * Test Environment: Docker (docker-compose.test.yml)
 * Seed Data: docker/seed-data.sql
 *
 * Seed Data Summary (from seed-data.sql):
 * - 10 inventory items across 3 warehouses
 * - Warehouses: 北京主仓 (WH001), 上海配送中心 (WH002), 深圳仓库 (WH003)
 *
 * Test Scenarios (P2-INT-002 Requirements):
 * 1. Create stock taking document - select warehouse and product range
 * 2. System auto-imports current inventory as book quantity
 * 3. Enter actual counted quantities
 * 4. Verify difference calculation (gain/loss)
 * 5. Submit stock taking for approval
 * 6. Approval leads to automatic inventory adjustment
 * 7. Video recording: Complete stock taking flow
 */
test.describe('Stock Taking Module E2E Tests (P2-INT-002)', () => {
  let inventoryPage: InventoryPage

  test.beforeEach(async ({ page }) => {
    inventoryPage = new InventoryPage(page)
  })

  test.describe('Stock Taking List Display', () => {
    test('should display stock taking list page', async () => {
      await inventoryPage.navigateToStockTakingListPage()

      // Verify page title
      await inventoryPage.assertStockTakingListDisplayed()

      // Take screenshot
      await inventoryPage.screenshotStockTaking('stock-taking-list-page')
    })

    test('should have "新建盘点" button visible', async () => {
      await inventoryPage.navigateToStockTakingListPage()

      const newButton = inventoryPage.page.locator('button').filter({ hasText: '新建盘点' })
      await expect(newButton).toBeVisible()
    })

    test('should navigate to create page when clicking new button', async ({ page }) => {
      await inventoryPage.navigateToStockTakingListPage()
      await inventoryPage.clickNewStockTaking()

      await expect(page).toHaveURL(/\/inventory\/stock-taking\/new/)
      await inventoryPage.screenshotStockTaking('stock-taking-create-page')
    })
  })

  test.describe('Stock Taking Creation', () => {
    test('should display create form with warehouse select', async () => {
      await inventoryPage.navigateToStockTakingCreatePage()

      // Verify warehouse select is visible
      const warehouseWrapper = inventoryPage.page
        .locator('.form-field-wrapper')
        .filter({ hasText: '仓库' })
      await expect(warehouseWrapper).toBeVisible()

      // Verify date picker is visible
      const dateWrapper = inventoryPage.page
        .locator('.form-field-wrapper')
        .filter({ hasText: '盘点日期' })
      await expect(dateWrapper).toBeVisible()

      await inventoryPage.screenshotStockTaking('stock-taking-create-form')
    })

    test('should show empty state before selecting warehouse', async () => {
      await inventoryPage.navigateToStockTakingCreatePage()

      // Should show "请先选择仓库" message
      const emptyMessage = inventoryPage.page.locator('text=请先选择仓库')
      await expect(emptyMessage).toBeVisible()
    })

    test('should load inventory items after selecting warehouse', async () => {
      await inventoryPage.navigateToStockTakingCreatePage()

      // Select warehouse
      await inventoryPage.selectStockTakingWarehouse('北京主仓')

      // Wait for inventory to load
      await inventoryPage.page.waitForTimeout(1000)

      // Should show product selection buttons
      const importAllButton = inventoryPage.page.locator('button').filter({ hasText: '全部导入' })
      await expect(importAllButton).toBeVisible()

      await inventoryPage.screenshotStockTaking('stock-taking-warehouse-selected')
    })

    test('should import all products when clicking "全部导入"', async () => {
      await inventoryPage.navigateToStockTakingCreatePage()

      // Select warehouse
      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await inventoryPage.page.waitForTimeout(1000)

      // Click import all
      await inventoryPage.clickImportAllProducts()

      // Verify products are imported
      const selectedCount = await inventoryPage.getSelectedProductCount()
      expect(selectedCount).toBeGreaterThan(0)

      await inventoryPage.screenshotStockTaking('stock-taking-products-imported')
    })

    test('should successfully create stock taking document', async ({ page }) => {
      await inventoryPage.navigateToStockTakingCreatePage()

      // Select warehouse
      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await inventoryPage.page.waitForTimeout(1000)

      // Import all products
      await inventoryPage.clickImportAllProducts()

      // Verify products imported
      const selectedCount = await inventoryPage.getSelectedProductCount()
      expect(selectedCount).toBeGreaterThan(0)

      // Submit the form
      await inventoryPage.submitStockTakingCreate()

      // Wait for success and redirect
      await inventoryPage.waitForStockTakingCreateSuccess()

      // Should be back on list page
      await expect(page).toHaveURL(/\/inventory\/stock-taking$/)
    })
  })

  test.describe('Stock Taking Execution', () => {
    // These tests create a stock taking first, then test the execution flow
    test.beforeEach(async () => {
      // Create a new stock taking for each test
      await inventoryPage.navigateToStockTakingCreatePage()
      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await inventoryPage.page.waitForTimeout(1000)
      await inventoryPage.clickImportAllProducts()
      await inventoryPage.submitStockTakingCreate()
      await inventoryPage.waitForStockTakingCreateSuccess()
    })

    test('should display new stock taking in list with DRAFT status', async () => {
      await inventoryPage.navigateToStockTakingListPage()

      // Get the current stock taking row (by stored taking number)
      const currentRow = await inventoryPage.getCurrentStockTakingRow()

      // Verify status is DRAFT (草稿)
      const statusTag = currentRow.locator('.semi-tag').filter({ hasText: '草稿' })
      await expect(statusTag).toBeVisible()

      await inventoryPage.screenshotStockTaking('stock-taking-draft-status')
    })

    test('should navigate to execute page and display items', async ({ page }) => {
      await inventoryPage.navigateToStockTakingListPage()

      // Click execute on the current stock taking row
      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)

      // Verify we're on the execute page
      await expect(page).toHaveURL(/\/inventory\/stock-taking\/[^/]+/)

      // Verify items table is displayed
      const itemsTable = inventoryPage.page.locator('.semi-table')
      await expect(itemsTable).toBeVisible()

      await inventoryPage.screenshotStockTaking('stock-taking-execute-page')
    })

    test('should start counting when clicking "开始盘点"', async () => {
      await inventoryPage.navigateToStockTakingListPage()

      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)

      // Click start counting
      await inventoryPage.clickStartCounting()

      // Status should change to COUNTING (盘点中)
      const status = await inventoryPage.getStockTakingStatus()
      expect(status).toContain('盘点中')

      await inventoryPage.screenshotStockTaking('stock-taking-counting-started')
    })

    test('should allow entering actual quantities', async () => {
      await inventoryPage.navigateToStockTakingListPage()

      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)

      // Start counting first
      await inventoryPage.clickStartCounting()

      // Find a product row and enter quantity
      const itemRows = inventoryPage.page.locator('.semi-table-tbody .semi-table-row')
      const firstItemRow = itemRows.first()
      const productCodeCell = firstItemRow.locator('.semi-table-row-cell').first()
      const productCode = await productCodeCell.textContent()

      if (productCode) {
        // Enter an actual quantity
        await inventoryPage.enterActualQuantity(productCode.trim(), 50)
      }

      await inventoryPage.screenshotStockTaking('stock-taking-quantity-entered')
    })

    test('should calculate difference after entering quantity', async () => {
      await inventoryPage.navigateToStockTakingListPage()

      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)

      // Start counting
      await inventoryPage.clickStartCounting()

      // Get the first item and its system quantity
      const itemRows = inventoryPage.page.locator('.semi-table-tbody .semi-table-row')
      const firstItemRow = itemRows.first()
      const cells = firstItemRow.locator('.semi-table-row-cell')

      const productCodeCell = cells.first()
      const productCode = (await productCodeCell.textContent())?.trim() || ''

      // Get system quantity (column index 3)
      const systemQtyText = (await cells.nth(3).textContent()) || '0'
      const systemQty = parseFloat(systemQtyText.replace(/[^\d.-]/g, '')) || 0

      // Enter a different quantity
      const actualQty = systemQty + 10 // 10 units gain
      await inventoryPage.enterActualQuantity(productCode, actualQty)
      await inventoryPage.page.waitForTimeout(300)

      // Verify difference is calculated (column index 5 should show +10)
      const diffText = await cells.nth(5).textContent()
      expect(diffText).toContain('+')

      await inventoryPage.screenshotStockTaking('stock-taking-difference-calculated')
    })

    test('should save counts and update progress', async () => {
      await inventoryPage.navigateToStockTakingListPage()

      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)

      // Start counting
      await inventoryPage.clickStartCounting()

      // Enter quantities for all items
      const itemRows = inventoryPage.page.locator('.semi-table-tbody .semi-table-row')
      const itemCount = await itemRows.count()

      for (let i = 0; i < itemCount; i++) {
        const row = itemRows.nth(i)
        const cells = row.locator('.semi-table-row-cell')
        const productCode = ((await cells.first().textContent()) || '').trim()

        // Get system quantity and use same value (no difference)
        const systemQtyText = (await cells.nth(3).textContent()) || '0'
        const systemQty = parseFloat(systemQtyText.replace(/[^\d.-]/g, '')) || 0

        await inventoryPage.enterActualQuantity(productCode, systemQty)
        await inventoryPage.page.waitForTimeout(100)
      }

      // Save all counts
      await inventoryPage.clickSaveAllCounts()

      // Progress should be 100%
      const progress = await inventoryPage.getStockTakingProgress()
      expect(progress).toBe(100)

      await inventoryPage.screenshotStockTaking('stock-taking-all-counted')
    })

    test('should submit for approval when all items are counted', async () => {
      await inventoryPage.navigateToStockTakingListPage()

      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)

      // Start counting
      await inventoryPage.clickStartCounting()

      // Enter quantities for all items
      const itemRows = inventoryPage.page.locator('.semi-table-tbody .semi-table-row')
      const itemCount = await itemRows.count()

      for (let i = 0; i < itemCount; i++) {
        const row = itemRows.nth(i)
        const cells = row.locator('.semi-table-row-cell')
        const productCode = ((await cells.first().textContent()) || '').trim()
        const systemQtyText = (await cells.nth(3).textContent()) || '0'
        const systemQty = parseFloat(systemQtyText.replace(/[^\d.-]/g, '')) || 0
        await inventoryPage.enterActualQuantity(productCode, systemQty)
        await inventoryPage.page.waitForTimeout(100)
      }

      // Save all counts
      await inventoryPage.clickSaveAllCounts()

      // Submit for approval
      await inventoryPage.clickSubmitForApproval()
      await inventoryPage.confirmSubmitForApproval()

      // Status should change to PENDING_APPROVAL (待审批)
      await inventoryPage.page.waitForTimeout(500)
      const status = await inventoryPage.getStockTakingStatus()
      expect(status).toContain('待审批')

      await inventoryPage.screenshotStockTaking('stock-taking-pending-approval')
    })
  })

  test.describe('Stock Taking with Differences', () => {
    test('should correctly show gain (盘盈) when actual > system', async () => {
      // Create stock taking
      await inventoryPage.navigateToStockTakingCreatePage()
      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await inventoryPage.page.waitForTimeout(1000)
      await inventoryPage.clickImportAllProducts()
      await inventoryPage.submitStockTakingCreate()
      await inventoryPage.waitForStockTakingCreateSuccess()

      // Execute
      await inventoryPage.navigateToStockTakingListPage()
      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)
      await inventoryPage.clickStartCounting()

      // Enter higher quantity for first item
      const itemRows = inventoryPage.page.locator('.semi-table-tbody .semi-table-row')
      const firstItemRow = itemRows.first()
      const cells = firstItemRow.locator('.semi-table-row-cell')
      const productCode = ((await cells.first().textContent()) || '').trim()
      const systemQtyText = (await cells.nth(3).textContent()) || '0'
      const systemQty = parseFloat(systemQtyText.replace(/[^\d.-]/g, '')) || 0

      // Add 20 units (gain)
      const actualQty = systemQty + 20
      await inventoryPage.enterActualQuantity(productCode, actualQty)
      await inventoryPage.page.waitForTimeout(300)

      // Verify difference shows positive value
      const diffCell = cells.nth(5)
      const diffText = await diffCell.textContent()
      expect(diffText).toMatch(/\+.*20/)

      // Verify positive styling (green/positive class)
      const hasPositiveClass = await diffCell.locator('.diff-positive').isVisible()
      expect(hasPositiveClass).toBe(true)

      await inventoryPage.screenshotStockTaking('stock-taking-gain')
    })

    test('should correctly show loss (盘亏) when actual < system', async () => {
      // Create stock taking
      await inventoryPage.navigateToStockTakingCreatePage()
      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await inventoryPage.page.waitForTimeout(1000)
      await inventoryPage.clickImportAllProducts()
      await inventoryPage.submitStockTakingCreate()
      await inventoryPage.waitForStockTakingCreateSuccess()

      // Execute
      await inventoryPage.navigateToStockTakingListPage()
      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)
      await inventoryPage.clickStartCounting()

      // Enter lower quantity for first item
      const itemRows = inventoryPage.page.locator('.semi-table-tbody .semi-table-row')
      const firstItemRow = itemRows.first()
      const cells = firstItemRow.locator('.semi-table-row-cell')
      const productCode = ((await cells.first().textContent()) || '').trim()
      const systemQtyText = (await cells.nth(3).textContent()) || '0'
      const systemQty = parseFloat(systemQtyText.replace(/[^\d.-]/g, '')) || 0

      // Subtract 15 units (loss)
      const actualQty = Math.max(0, systemQty - 15)
      await inventoryPage.enterActualQuantity(productCode, actualQty)
      await inventoryPage.page.waitForTimeout(300)

      // Verify difference shows negative value
      const diffCell = cells.nth(5)
      const diffText = await diffCell.textContent()
      expect(diffText).toMatch(/-/)

      // Verify negative styling (red/negative class)
      const hasNegativeClass = await diffCell.locator('.diff-negative').isVisible()
      expect(hasNegativeClass).toBe(true)

      await inventoryPage.screenshotStockTaking('stock-taking-loss')
    })
  })

  test.describe('Video Recording - Complete Stock Taking Flow', () => {
    test('should record complete stock taking workflow', async ({ page }) => {
      // Step 1: Navigate to stock taking list
      await inventoryPage.navigateToStockTakingListPage()
      await page.waitForTimeout(1000)
      await inventoryPage.screenshotStockTaking('video-1-list-page')

      // Step 2: Click new stock taking
      await inventoryPage.clickNewStockTaking()
      await page.waitForTimeout(1000)
      await inventoryPage.screenshotStockTaking('video-2-create-page')

      // Step 3: Select warehouse
      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await page.waitForTimeout(1500)
      await inventoryPage.screenshotStockTaking('video-3-warehouse-selected')

      // Step 4: Import all products
      await inventoryPage.clickImportAllProducts()
      await page.waitForTimeout(1000)
      await inventoryPage.screenshotStockTaking('video-4-products-imported')

      // Step 5: Create stock taking
      await inventoryPage.submitStockTakingCreate()
      await inventoryPage.waitForStockTakingCreateSuccess()
      await page.waitForTimeout(1000)
      await inventoryPage.screenshotStockTaking('video-5-created-success')

      // Step 6: Execute stock taking (use current stock taking to ensure isolation)
      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)
      await page.waitForTimeout(1000)
      await inventoryPage.screenshotStockTaking('video-6-execute-page')

      // Step 7: Start counting
      await inventoryPage.clickStartCounting()
      await page.waitForTimeout(1000)
      await inventoryPage.screenshotStockTaking('video-7-counting-started')

      // Step 8: Enter quantities for all items
      const itemRows = page.locator('.semi-table-tbody .semi-table-row')
      const itemCount = await itemRows.count()

      for (let i = 0; i < itemCount; i++) {
        const row = itemRows.nth(i)
        const cells = row.locator('.semi-table-row-cell')
        const productCode = ((await cells.first().textContent()) || '').trim()
        const systemQtyText = (await cells.nth(3).textContent()) || '0'
        const systemQty = parseFloat(systemQtyText.replace(/[^\d.-]/g, '')) || 0

        // Add small variance for demonstration
        const variance = i === 0 ? 5 : i === 1 ? -3 : 0
        await inventoryPage.enterActualQuantity(productCode, systemQty + variance)
        await page.waitForTimeout(200)
      }
      await inventoryPage.screenshotStockTaking('video-8-quantities-entered')

      // Step 9: Save all counts
      await inventoryPage.clickSaveAllCounts()
      await page.waitForTimeout(1500)
      await inventoryPage.screenshotStockTaking('video-9-counts-saved')

      // Step 10: Submit for approval
      await inventoryPage.clickSubmitForApproval()
      await page.waitForTimeout(500)
      await inventoryPage.screenshotStockTaking('video-10-submit-modal')

      await inventoryPage.confirmSubmitForApproval()
      await page.waitForTimeout(1500)
      await inventoryPage.screenshotStockTaking('video-11-submitted')
    })
  })

  test.describe('Stock Taking Filtering', () => {
    test('should filter stock takings by status', async () => {
      // First create a stock taking
      await inventoryPage.navigateToStockTakingCreatePage()
      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await inventoryPage.page.waitForTimeout(1000)
      await inventoryPage.clickImportAllProducts()
      await inventoryPage.submitStockTakingCreate()
      await inventoryPage.waitForStockTakingCreateSuccess()

      // Now filter by DRAFT status
      await inventoryPage.filterStockTakingByStatus('DRAFT')

      // Should have at least one DRAFT item
      const count = await inventoryPage.getStockTakingItemCount()
      expect(count).toBeGreaterThan(0)

      // All items should have DRAFT status
      const rows = inventoryPage.page.locator('.semi-table-tbody .semi-table-row')
      const firstRow = rows.first()
      const statusTag = firstRow.locator('.semi-tag').filter({ hasText: '草稿' })
      await expect(statusTag).toBeVisible()

      await inventoryPage.screenshotStockTaking('stock-taking-filter-draft')
    })

    test('should filter stock takings by warehouse', async () => {
      // Create stock takings for different warehouses
      await inventoryPage.navigateToStockTakingCreatePage()
      await inventoryPage.selectStockTakingWarehouse('深圳仓库')
      await inventoryPage.page.waitForTimeout(1000)
      await inventoryPage.clickImportAllProducts()
      await inventoryPage.submitStockTakingCreate()
      await inventoryPage.waitForStockTakingCreateSuccess()

      // Filter by warehouse
      const warehouseSelect = inventoryPage.page.locator('.semi-select').first()
      await warehouseSelect.click()
      await inventoryPage.page.waitForTimeout(200)

      const warehouseOption = inventoryPage.page
        .locator('.semi-select-option')
        .filter({ hasText: '深圳仓库' })
      await warehouseOption.click()
      await inventoryPage.page.waitForTimeout(500)

      // Should have at least one item from Shenzhen warehouse
      const count = await inventoryPage.getStockTakingItemCount()
      expect(count).toBeGreaterThan(0)

      await inventoryPage.screenshotStockTaking('stock-taking-filter-warehouse')
    })
  })

  test.describe('Screenshots for Documentation', () => {
    test('should capture stock taking list page', async () => {
      await inventoryPage.navigateToStockTakingListPage()
      await inventoryPage.screenshotStockTaking('doc-stock-taking-list')
    })

    test('should capture stock taking create page', async () => {
      await inventoryPage.navigateToStockTakingCreatePage()
      await inventoryPage.screenshotStockTaking('doc-stock-taking-create-empty')

      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await inventoryPage.page.waitForTimeout(1000)
      await inventoryPage.screenshotStockTaking('doc-stock-taking-create-warehouse')

      await inventoryPage.clickImportAllProducts()
      await inventoryPage.screenshotStockTaking('doc-stock-taking-create-products')
    })

    test('should capture stock taking execute page', async () => {
      // Create and navigate to execute page
      await inventoryPage.navigateToStockTakingCreatePage()
      await inventoryPage.selectStockTakingWarehouse('北京主仓')
      await inventoryPage.page.waitForTimeout(1000)
      await inventoryPage.clickImportAllProducts()
      await inventoryPage.submitStockTakingCreate()
      await inventoryPage.waitForStockTakingCreateSuccess()

      const currentRow = await inventoryPage.getCurrentStockTakingRow()
      await inventoryPage.clickStockTakingExecute(currentRow)
      await inventoryPage.screenshotStockTaking('doc-stock-taking-execute-draft')

      await inventoryPage.clickStartCounting()
      await inventoryPage.screenshotStockTaking('doc-stock-taking-execute-counting')
    })
  })
})
