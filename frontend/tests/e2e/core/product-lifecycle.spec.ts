import { test, expect } from '../fixtures/test-fixtures'
import { TEST_USERS } from '../fixtures/test-fixtures'

/**
 * Product Lifecycle Management E2E Tests
 *
 * Tests the complete product lifecycle including:
 * - Product creation and initial state
 * - Status transitions (draft → active → discontinued)
 * - Inventory impact of product status changes
 * - Multi-tenant isolation
 */
test.describe('Product Lifecycle Management', () => {
  test.beforeEach(async ({ page, authenticatedPage }) => {
    // Navigate to products page
    await page.goto('/catalog/products')
    await expect(page).toHaveURL(/.*\/catalog\/products/)
  })

  test('should create product with initial draft status', async ({ page, productsPage }) => {
    // Arrange
    const productCode = `TEST-${Date.now()}`
    const productName = 'Test Product Lifecycle'
    const purchasePrice = '100.00'
    const sellingPrice = '150.00'

    // Act - Create new product
    await productsPage.addProductButton.click()
    await expect(page).toHaveURL(/.*\/catalog\/products\/new/)

    // Fill product form
    await productsPage.codeInput.fill(productCode)
    await productsPage.nameInput.fill(productName)
    await productsPage.unitInput.fill('piece')
    await productsPage.barcodeInput.fill(`BAR${productCode}`)
    await productsPage.descriptionInput.fill('Test product for lifecycle management')
    await productsPage.purchasePriceInput.fill(purchasePrice)
    await productsPage.sellingPriceInput.fill(sellingPrice)
    await productsPage.minStockInput.fill('10')

    // Submit form
    await productsPage.submitButton.click()

    // Assert - Verify product created with draft status
    await expect(page).toHaveURL(/.*\/catalog\/products/)
    await expect(productsPage.successMessage).toContainText('创建成功')

    // Verify product appears in list
    await productsPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const productRow = productsPage.tableRows.filter({ hasText: productCode })
    await expect(productRow).toBeVisible()

    // Verify initial status is draft
    const statusCell = productRow.locator('.semi-table-cell').filter({ hasText: '草稿' })
    await expect(statusCell).toBeVisible()

    // Take screenshot for verification
    await page.screenshot({ path: 'artifacts/product-created-draft.png' })
  })

  test('should transition product from draft to active', async ({ page, productsPage }) => {
    // Arrange - Create a draft product first
    const productCode = `DRAFT-${Date.now()}`
    await productsPage.addProductButton.click()
    await productsPage.codeInput.fill(productCode)
    await productsPage.nameInput.fill('Draft to Active Test')
    await productsPage.unitInput.fill('piece')
    await productsPage.purchasePriceInput.fill('50.00')
    await productsPage.sellingPriceInput.fill('75.00')
    await productsPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Find the created product
    await productsPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const productRow = productsPage.tableRows.filter({ hasText: productCode })
    await expect(productRow).toBeVisible()

    // Act - Click edit and activate
    await productRow.locator('button').filter({ hasText: '编辑' }).click()
    await expect(page).toHaveURL(/.*\/catalog\/products\/.*\/edit/)

    // Change status to active
    const statusSelect = page.locator('.semi-select').filter({ hasText: '草稿' })
    await statusSelect.click()
    await page.locator('.semi-select-option').filter({ hasText: '启用' }).click()

    // Save changes
    await productsPage.submitButton.click()

    // Assert - Verify status changed
    await expect(page).toHaveURL(/.*\/catalog\/products/)
    await expect(productsPage.successMessage).toContainText('更新成功')

    // Verify status in list
    await productsPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const updatedRow = productsPage.tableRows.filter({ hasText: productCode })
    const statusCell = updatedRow.locator('.semi-table-cell').filter({ hasText: '启用' })
    await expect(statusCell).toBeVisible()

    await page.screenshot({ path: 'artifacts/product-activated.png' })
  })

  test('should prevent inventory transactions for draft products', async ({ page, productsPage, salesOrderPage }) => {
    // Arrange - Create a draft product
    const productCode = `NO-INV-${Date.now()}`
    await productsPage.addProductButton.click()
    await productsPage.codeInput.fill(productCode)
    await productsPage.nameInput.fill('Draft Product - No Inventory')
    await productsPage.unitInput.fill('piece')
    await productsPage.purchasePriceInput.fill('30.00')
    await productsPage.sellingPriceInput.fill('45.00')
    await productsPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Act - Try to create sales order with draft product
    await page.goto('/sales/orders/new')

    // Try to add the draft product
    await salesOrderPage.customerSelect.click()
    await page.locator('.semi-select-option').first().click()

    await salesOrderPage.addProductButton.click()
    const productSearch = page.locator('.semi-input').filter({ hasText: '请选择商品' })
    await productSearch.fill(productCode)

    // Assert - Draft product should not be available for selection
    await expect(page.locator('.semi-select-option')).not.toContainText(productCode)

    await page.screenshot({ path: 'artifacts/draft-product-not-available.png' })
  })

  test('should handle product discontinuation with existing inventory', async ({ page, productsPage, inventoryPage }) => {
    // Arrange - Create and activate a product first
    const productCode = `DISC-${Date.now()}`
    await productsPage.addProductButton.click()
    await productsPage.codeInput.fill(productCode)
    await productsPage.nameInput.fill('Product to Discontinue')
    await productsPage.unitInput.fill('piece')
    await productsPage.purchasePriceInput.fill('80.00')
    await productsPage.sellingPriceInput.fill('120.00')
    await productsPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Activate the product
    await productsPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')
    const productRow = productsPage.tableRows.filter({ hasText: productCode })
    await productRow.locator('button').filter({ hasText: '编辑' }).click()

    const statusSelect = page.locator('.semi-select').filter({ hasText: '草稿' })
    await statusSelect.click()
    await page.locator('.semi-select-option').filter({ hasText: '启用' }).click()
    await productsPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Add inventory through purchase order
    await page.goto('/purchase/orders/new')
    await page.locator('.semi-select').filter({ hasText: '请选择供应商' }).click()
    await page.locator('.semi-select-option').first().click()

    await page.locator('button').filter({ hasText: '添加商品' }).click()
    await page.locator('.semi-input').fill(productCode)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    await page.locator('.semi-input-number input').fill('100') // Quantity
    await page.locator('button').filter({ hasText: '保存' }).click()
    await page.locator('button').filter({ hasText: '提交' }).click()

    // Confirm purchase order
    await page.waitForLoadState('networkidle')
    await page.locator('button').filter({ hasText: '确认' }).click()

    // Act - Discontinue the product
    await page.goto('/catalog/products')
    await productsPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const activeRow = productsPage.tableRows.filter({ hasText: productCode })
    await activeRow.locator('button').filter({ hasText: '编辑' }).click()

    const statusSelect2 = page.locator('.semi-select').filter({ hasText: '启用' })
    await statusSelect2.click()
    await page.locator('.semi-select-option').filter({ hasText: '停用' }).click()
    await productsPage.submitButton.click()

    // Assert - Verify product discontinued but inventory remains
    await expect(productsPage.successMessage).toContainText('更新成功')

    // Check inventory still exists
    await page.goto('/inventory/stock')
    await page.locator('.semi-input').fill(productCode)
    await page.waitForLoadState('networkidle')

    const inventoryRow = page.locator('.semi-table-row').filter({ hasText: productCode })
    await expect(inventoryRow).toBeVisible()
    await expect(inventoryRow).toContainText('100') // Quantity should remain

    await page.screenshot({ path: 'artifacts/discontinued-with-inventory.png' })
  })

  test('should maintain tenant isolation for product operations', async ({ page, productsPage }) => {
    // This test verifies that products are isolated by tenant
    const productCode = `TENANT-${Date.now()}`

    // Create product as current tenant
    await productsPage.addProductButton.click()
    await productsPage.codeInput.fill(productCode)
    await productsPage.nameInput.fill('Tenant Isolation Test Product')
    await productsPage.unitInput.fill('piece')
    await productsPage.purchasePriceInput.fill('50.00')
    await productsPage.sellingPriceInput.fill('75.00')
    await productsPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Verify product exists for current tenant
    await productsPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const productRow = productsPage.tableRows.filter({ hasText: productCode })
    await expect(productRow).toBeVisible()

    // Logout and login as different user (should be different tenant)
    await page.locator('.semi-avatar').click()
    await page.locator('.semi-dropdown-item').filter({ hasText: '退出登录' }).click()

    // Login as sales user
    await page.goto('/auth/login')
    await page.locator('input[name="username"]').fill(TEST_USERS.sales.username)
    await page.locator('input[name="password"]').fill(TEST_USERS.sales.password)
    await page.locator('button[type="submit"]').click()

    // Navigate to products
    await page.goto('/catalog/products')
    await productsPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    // Product should not be visible to different tenant
    await expect(productsPage.emptyState).toBeVisible()

    await page.screenshot({ path: 'artifacts/tenant-isolation-verified.png' })
  })

  test('should validate required fields during product creation', async ({ page, productsPage }) => {
    // Act - Try to submit empty form
    await productsPage.addProductButton.click()
    await productsPage.submitButton.click()

    // Assert - Verify validation errors
    await expect(page.locator('.semi-form-field-error')).toContainText('请输入商品编码')
    await expect(page.locator('.semi-form-field-error')).toContainText('请输入商品名称')
    await expect(page.locator('.semi-form-field-error')).toContainText('请输入单位')

    // Fill only code and try to submit
    await productsPage.codeInput.fill('TEST-VALIDATION')
    await productsPage.submitButton.click()

    // Should still show errors for other required fields
    await expect(page.locator('.semi-form-field-error')).not.toContainText('请输入商品编码')
    await expect(page.locator('.semi-form-field-error')).toContainText('请输入商品名称')
    await expect(page.locator('.semi-form-field-error')).toContainText('请输入单位')

    await page.screenshot({ path: 'artifacts/product-validation-errors.png' })
  })

  test('should handle duplicate product code validation', async ({ page, productsPage }) => {
    // Arrange - Create first product
    const duplicateCode = `DUP-${Date.now()}`
    await productsPage.addProductButton.click()
    await productsPage.codeInput.fill(duplicateCode)
    await productsPage.nameInput.fill('First Product')
    await productsPage.unitInput.fill('piece')
    await productsPage.purchasePriceInput.fill('50.00')
    await productsPage.sellingPriceInput.fill('75.00')
    await productsPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Act - Try to create another product with same code
    await productsPage.addProductButton.click()
    await productsPage.codeInput.fill(duplicateCode)
    await productsPage.nameInput.fill('Duplicate Product')
    await productsPage.unitInput.fill('piece')
    await productsPage.purchasePriceInput.fill('60.00')
    await productsPage.sellingPriceInput.fill('90.00')
    await productsPage.submitButton.click()

    // Assert - Should show duplicate error
    await expect(page.locator('.semi-form-field-error')).toContainText('商品编码已存在')

    await page.screenshot({ path: 'artifacts/duplicate-code-error.png' })
  })
})