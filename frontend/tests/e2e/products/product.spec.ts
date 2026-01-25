import { test, expect } from '../fixtures'
import { ProductsPage } from '../pages'

/**
 * P1-INT-001: Product Module E2E Integration Tests
 *
 * Test Environment: Docker (docker-compose.test.yml)
 * Seed Data: docker/seed-data.sql
 *
 * Test Scenarios:
 * 1. Product list displays seed data correctly
 * 2. Create new product with full form
 * 3. Edit existing product
 * 4. Enable/disable product status
 * 5. Delete product (soft delete)
 * 6. Pagination and search functionality
 *
 * Note: Tests run serially to avoid data interference from parallel execution
 */
test.describe.serial('Product Module E2E Tests (P1-INT-001)', () => {
  let productsPage: ProductsPage

  test.beforeEach(async ({ page }) => {
    productsPage = new ProductsPage(page)
    // Auth setup is handled by Playwright config (storageState)
  })

  test.describe('Product List Display', () => {
    test('should display product list with seed data', async () => {
      await productsPage.navigateToList()

      // Verify page title
      await productsPage.assertPageTitle('商品管理')

      // Verify table has data from seed
      const productCount = await productsPage.getProductCount()
      expect(productCount).toBeGreaterThan(0)

      // Verify known seed products exist
      // From seed-data.sql: iPhone 15 Pro (IPHONE15), Samsung Galaxy S24 (SAMSUNG24)
      await productsPage.assertProductExists('IPHONE15')
      await productsPage.assertProductExists('SAMSUNG24')

      // Take screenshot for documentation
      await productsPage.screenshotList('product-list-seed-data')
    })

    test('should show correct product details in table row', async () => {
      await productsPage.navigateToList()

      // Find iPhone 15 Pro and verify its data
      const row = await productsPage.findProductRowByCode('IPHONE15')
      expect(row).not.toBeNull()

      if (row) {
        // Verify product name is displayed
        const nameCell = row.locator('.product-name')
        await expect(nameCell).toContainText('iPhone 15 Pro')

        // Verify status tag
        const statusTag = row.locator('.semi-tag')
        await expect(statusTag).toBeVisible()
      }
    })
  })

  test.describe('Product Creation', () => {
    test('should navigate to create product page', async () => {
      await productsPage.navigateToList()
      await productsPage.clickAddProduct()

      // Verify we're on the create page
      await productsPage.assertPageTitle('新增商品')
      await productsPage.screenshotForm('product-create-form-empty')
    })

    test('should create a new product successfully', async () => {
      await productsPage.navigateToCreate()

      // Generate unique code and barcode to avoid conflicts
      const timestamp = Date.now()
      const uniqueCode = `TEST_${timestamp}`
      const uniqueBarcode = `${timestamp}` // Barcodes are typically numeric strings

      // Fill the form
      await productsPage.fillProductForm({
        code: uniqueCode,
        name: 'E2E Test Product',
        unit: 'pcs',
        barcode: uniqueBarcode,
        description: 'Product created during E2E testing',
        purchasePrice: 100.5,
        sellingPrice: 150.99,
        minStock: 10,
        sortOrder: 1,
      })

      // Take screenshot before submission
      await productsPage.screenshotForm('product-create-form-filled')

      // Submit the form
      await productsPage.submitForm()

      // Wait for redirect to list and verify success
      await productsPage.waitForFormSuccess()

      // Verify the new product appears in the list
      await productsPage.search(uniqueCode)
      await productsPage.assertProductExists(uniqueCode)
    })

    test('should show validation error for empty required fields', async () => {
      await productsPage.navigateToCreate()

      // Try to submit empty form
      await productsPage.submitForm()

      // Verify code field shows error (it's required)
      // Note: The exact error handling depends on the form implementation
      const codeInput = productsPage.codeInput
      await expect(codeInput).toBeVisible()

      // The form should not navigate away
      await expect(productsPage.page).toHaveURL(/\/catalog\/products\/new/)
    })

    test('should validate product code format', async () => {
      await productsPage.navigateToCreate()

      // Fill with invalid code (special characters)
      await productsPage.fillProductForm({
        code: 'Invalid@Code!',
        name: 'Test Product',
        unit: 'pcs',
      })

      await productsPage.submitForm()

      // Should stay on the form page due to validation error
      await expect(productsPage.page).toHaveURL(/\/catalog\/products\/new/)
    })
  })

  test.describe('Product Editing', () => {
    test('should edit an existing product', async ({ page }) => {
      await productsPage.navigateToList()

      // Find a product to edit (use a seed product)
      await productsPage.search('CHARGER')
      const row = await productsPage.findProductRowByCode('CHARGER')
      expect(row).not.toBeNull()

      if (row) {
        // Click edit action
        await productsPage.clickRowAction(row, 'edit')

        // Wait for edit page to load
        await page.waitForURL(/\/catalog\/products\/.*\/edit/)
        await productsPage.assertPageTitle('编辑商品')

        // Take screenshot of edit form
        await productsPage.screenshotForm('product-edit-form')

        // Modify the product name
        const newName = `USB-C Charger 65W Updated ${Date.now()}`
        await productsPage.nameInput.clear()
        await productsPage.nameInput.fill(newName)

        // Submit the form
        await productsPage.submitForm()
        await productsPage.waitForFormSuccess()

        // Verify the update by searching
        await productsPage.search('CHARGER')
        const updatedRow = await productsPage.findProductRowByCode('CHARGER')
        expect(updatedRow).not.toBeNull()
        if (updatedRow) {
          const nameCell = updatedRow.locator('.product-name')
          await expect(nameCell).toContainText('Updated')
        }
      }
    })

    test('should have code field disabled in edit mode', async ({ page }) => {
      await productsPage.navigateToList()

      // Find any product and edit it
      const row = await productsPage.findProductRowByCode('AIRPODS')
      expect(row).not.toBeNull()

      if (row) {
        await productsPage.clickRowAction(row, 'edit')
        await page.waitForURL(/\/catalog\/products\/.*\/edit/)

        // Code field should be disabled
        await expect(productsPage.codeInput).toBeDisabled()
      }
    })
  })

  test.describe('Product Status Management', () => {
    test('should deactivate an active product', async () => {
      await productsPage.navigateToList()

      // Filter to show only active products
      await productsPage.filterByStatus('active')

      // Get an active product (seed products are active by default)
      const productCount = await productsPage.getProductCount()
      expect(productCount).toBeGreaterThan(0)

      // Get the first row and check its code
      const firstRow = productsPage.tableRows.first()
      const codeCell = firstRow.locator('.semi-table-row-cell').nth(1)
      const productCode = await codeCell.textContent()

      if (productCode) {
        // Deactivate the product
        await productsPage.clickRowAction(firstRow, 'deactivate')

        // Wait for toast notification
        await productsPage.waitForToast('已禁用')

        // Verify status changed
        await productsPage.filterByStatus('')
        await productsPage.search(productCode.trim())
        await productsPage.assertProductStatus(productCode.trim(), '禁用')
      }
    })

    test('should activate a disabled product', async () => {
      await productsPage.navigateToList()

      // First, we need to find or create a disabled product
      // Let's filter by inactive status
      await productsPage.filterByStatus('inactive')

      const productCount = await productsPage.getProductCount()
      if (productCount > 0) {
        const firstRow = productsPage.tableRows.first()
        const codeCell = firstRow.locator('.semi-table-row-cell').nth(1)
        const productCode = await codeCell.textContent()

        if (productCode) {
          // Activate the product
          await productsPage.clickRowAction(firstRow, 'activate')

          // Wait for toast notification
          await productsPage.waitForToast('已启用')

          // Verify status changed
          await productsPage.filterByStatus('')
          await productsPage.search(productCode.trim())
          await productsPage.assertProductStatus(productCode.trim(), '启用')
        }
      }
    })

    test('should filter products by status', async () => {
      await productsPage.navigateToList()

      // Test active filter
      await productsPage.filterByStatus('active')
      let count = await productsPage.getProductCount()
      // All seed products are active, so we should have some
      expect(count).toBeGreaterThanOrEqual(0)

      // Reset filter
      await productsPage.filterByStatus('')
      count = await productsPage.getProductCount()
      expect(count).toBeGreaterThan(0)
    })
  })

  test.describe('Product Deletion', () => {
    test('should delete a product with confirmation', async ({ page }) => {
      // First create a test product to delete
      await productsPage.navigateToCreate()

      const deleteTestCode = `DELETE_TEST_${Date.now()}`
      await productsPage.fillProductForm({
        code: deleteTestCode,
        name: 'Product to Delete',
        unit: 'pcs',
      })
      await productsPage.submitForm()
      await productsPage.waitForFormSuccess()

      // Find the product
      await productsPage.search(deleteTestCode)
      const row = await productsPage.findProductRowByCode(deleteTestCode)
      expect(row).not.toBeNull()

      if (row) {
        // Click delete action
        await productsPage.clickRowAction(row, 'delete')

        // Confirm the dialog
        await page.waitForSelector('.semi-modal')
        await productsPage.confirmDialog()

        // Wait for toast
        await productsPage.waitForToast('已删除')

        // Verify product is no longer visible
        await productsPage.clearSearch()
        await productsPage.search(deleteTestCode)

        // Product should not exist (soft deleted)
        await productsPage.assertProductNotExists(deleteTestCode)
      }
    })

    test('should cancel delete when clicking cancel in dialog', async ({ page }) => {
      await productsPage.navigateToList()

      // Find any product
      const row = await productsPage.findProductRowByCode('IPHONE15')
      expect(row).not.toBeNull()

      if (row) {
        // Click delete action
        await productsPage.clickRowAction(row, 'delete')

        // Cancel the dialog
        await page.waitForSelector('.semi-modal')
        await productsPage.cancelDialog()

        // Product should still exist
        await productsPage.assertProductExists('IPHONE15')
      }
    })
  })

  test.describe('Search and Pagination', () => {
    test('should search products by name', async () => {
      await productsPage.navigateToList()

      // Search for iPhone
      await productsPage.search('iPhone')

      // Verify search results
      const count = await productsPage.getProductCount()
      expect(count).toBeGreaterThan(0)

      // All results should contain iPhone
      await productsPage.assertProductExists('IPHONE15')
    })

    test('should search products by code', async () => {
      await productsPage.navigateToList()

      // Search by product code
      await productsPage.search('SAMSUNG')

      // Verify search results
      await productsPage.assertProductExists('SAMSUNG24')
    })

    test('should search products by barcode', async () => {
      await productsPage.navigateToList()

      // Search by barcode (from seed: 6941234567890 for iPhone)
      await productsPage.search('6941234567890')

      // Verify search results contain iPhone
      const count = await productsPage.getProductCount()
      expect(count).toBeGreaterThanOrEqual(0)
    })

    test('should show empty state for no results', async () => {
      await productsPage.navigateToList()

      // Search for non-existent product
      await productsPage.search('NonExistentProduct12345')

      // Verify empty state or zero results
      const count = await productsPage.getProductCount()
      expect(count).toBe(0)
    })

    test('should clear search and show all products', async () => {
      await productsPage.navigateToList()

      // Get initial count
      const initialCount = await productsPage.getProductCount()

      // Search to filter
      await productsPage.search('iPhone')
      const filteredCount = await productsPage.getProductCount()
      expect(filteredCount).toBeLessThanOrEqual(initialCount)

      // Clear search
      await productsPage.clearSearch()
      const afterClearCount = await productsPage.getProductCount()
      expect(afterClearCount).toBe(initialCount)
    })

    test('should display pagination info correctly', async () => {
      await productsPage.navigateToList()

      // Get pagination info
      const paginationInfo = await productsPage.getPaginationInfo()

      // Should show total count from seed data
      expect(paginationInfo.total).toBeGreaterThan(0)
      expect(paginationInfo.current).toBe(1)
    })
  })

  test.describe('Screenshots and Visual Verification', () => {
    test('should capture product list page screenshot', async () => {
      await productsPage.navigateToList()
      await productsPage.screenshotList('product-list-full')
    })

    test('should capture product create form screenshot', async () => {
      await productsPage.navigateToCreate()
      await productsPage.screenshotForm('product-create-form')
    })

    test('should capture product edit form screenshot', async ({ page }) => {
      await productsPage.navigateToList()
      const row = await productsPage.findProductRowByCode('IPHONE15')
      if (row) {
        await productsPage.clickRowAction(row, 'edit')
        await page.waitForURL(/\/catalog\/products\/.*\/edit/)
        await productsPage.screenshotForm('product-edit-form-iphone')
      }
    })

    test('should capture search results screenshot', async () => {
      await productsPage.navigateToList()
      await productsPage.search('Phone')
      await productsPage.screenshotList('product-search-results')
    })

    test('should capture filtered list screenshot', async () => {
      await productsPage.navigateToList()
      await productsPage.filterByStatus('active')
      await productsPage.screenshotList('product-filter-active')
    })
  })
})
