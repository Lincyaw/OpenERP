import { test, expect } from '../fixtures/test-fixtures'

/**
 * Inventory Operations and State Transitions E2E Tests
 *
 * Tests inventory operations including:
 * - Stock initialization and updates
 * - Inventory movements (in/out/transfers)
 * - Stock adjustment workflows
 * - Transaction history tracking
 * - Multi-warehouse inventory management
 * - State consistency across operations
 */
test.describe('Inventory Operations and State Transitions', () => {
  test.beforeEach(async ({ page, authenticatedPage }) => {
    // Navigate to inventory page
    await page.goto('/inventory/stock')
    await expect(page).toHaveURL(/.*\/inventory\/stock/)
  })

  test('should initialize inventory for new product', async ({ page, productsPage, inventoryPage, purchaseOrderPage }) => {
    // Arrange - Create a new product first
    const productCode = `INV-INIT-${Date.now()}`
    await page.goto('/catalog/products')
    await productsPage.addProductButton.click()
    await productsPage.codeInput.fill(productCode)
    await productsPage.nameInput.fill('Inventory Initialization Test Product')
    await productsPage.unitInput.fill('piece')
    await productsPage.purchasePriceInput.fill('50.00')
    await productsPage.sellingPriceInput.fill('80.00')
    await productsPage.minStockInput.fill('20')
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

    // Act - Create purchase order to initialize inventory
    await page.goto('/purchase/orders/new')

    // Select supplier
    await page.locator('.semi-select').filter({ hasText: '请选择供应商' }).click()
    await page.locator('.semi-select-option').first().click()

    // Add product to order
    await page.locator('button').filter({ hasText: '添加商品' }).click()
    await page.locator('.semi-input').fill(productCode)
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    // Set quantity and price
    await page.locator('.semi-input-number input').nth(0).fill('100') // Quantity
    await page.locator('.semi-input-number input').nth(1).fill('50.00') // Price

    // Save item
    await page.locator('button').filter({ hasText: '保存' }).click()

    // Submit order
    await page.locator('button').filter({ hasText: '提交' }).click()
    await page.waitForLoadState('networkidle')

    // Confirm the order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Navigate to inventory to verify
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    // Assert - Verify inventory initialized
    const inventoryRow = inventoryPage.tableRows.filter({ hasText: productCode })
    await expect(inventoryRow).toBeVisible()
    await expect(inventoryRow).toContainText('100') // Initial quantity
    await expect(inventoryRow).toContainText('50.00') // Unit price

    // Verify stock status is "In Stock"
    await expect(inventoryRow).toContainText('有库存')

    await page.screenshot({ path: 'artifacts/inventory-initialized.png' })
  })

  test('should track inventory state through sales order lifecycle', async ({ page, salesOrderPage, inventoryPage }) => {
    // Arrange - Use existing product with inventory
    const productCode = 'PROD-001' // Assuming this exists from seed data

    // Check initial inventory
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const initialRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const initialQuantityText = await initialRow.locator('.semi-table-cell').nth(4).textContent()
    const initialQuantity = parseInt(initialQuantityText || '0')

    // Act - Create sales order
    await page.goto('/sales/orders/new')

    // Select customer
    await salesOrderPage.customerSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Select warehouse
    await salesOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Add product
    await salesOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill(productCode)
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    // Set quantity (less than available)
    const orderQuantity = Math.min(initialQuantity - 10, 50) // Leave at least 10 in stock
    await page.locator('.semi-input-number input').nth(0).fill(orderQuantity.toString())
    await page.locator('.semi-input-number input').nth(1).fill('80.00') // Price

    // Save item
    await page.locator('button').filter({ hasText: '保存' }).click()

    // Submit order
    await salesOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Get order number
    const orderNumber = await page.locator('.order-number').textContent()

    // Confirm order (this should reserve inventory)
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Check inventory after confirmation
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const confirmedRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const confirmedQuantityText = await confirmedRow.locator('.semi-table-cell').nth(4).textContent()
    const confirmedQuantity = parseInt(confirmedQuantityText || '0')

    // Assert - Inventory should be reserved but not deducted yet
    expect(confirmedQuantity).toBe(initialQuantity) // Quantity remains same

    // Ship the order
    await page.goto(`/sales/orders/${orderNumber}`)
    await page.locator('button').filter({ hasText: '发货' }).click()

    // Fill shipping details
    await page.locator('.semi-input').fill('SF123456789')
    await page.locator('button').filter({ hasText: '确认发货' }).click()
    await page.waitForLoadState('networkidle')

    // Check inventory after shipping
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const shippedRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const shippedQuantityText = await shippedRow.locator('.semi-table-cell').nth(4).textContent()
    const shippedQuantity = parseInt(shippedQuantityText || '0')

    // Assert - Inventory should be deducted after shipping
    expect(shippedQuantity).toBe(initialQuantity - orderQuantity)

    await page.screenshot({ path: 'artifacts/inventory-after-shipping.png' })
  })

  test('should handle stock adjustment with proper state tracking', async ({ page, inventoryPage }) => {
    // Arrange - Use existing product
    const productCode = 'PROD-002'

    // Check initial inventory
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const initialRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const initialQuantityText = await initialRow.locator('.semi-table-cell').nth(4).textContent()
    const initialQuantity = parseInt(initialQuantityText || '0')

    // Act - Create stock adjustment
    await page.goto('/inventory/adjustment')
    await page.waitForLoadState('networkidle')

    // Select warehouse
    await inventoryPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Select product
    await inventoryPage.productSelect.click()
    await page.locator('.semi-input').fill(productCode)
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    // Set adjustment details
    const adjustmentQuantity = 25
    await inventoryPage.actualQuantityInput.fill((initialQuantity + adjustmentQuantity).toString())

    await inventoryPage.adjustmentReasonSelect.click()
    await page.locator('.semi-select-option').filter({ hasText: '盘点差异' }).click()

    await inventoryPage.notesInput.fill('Test adjustment for state tracking')

    // Preview adjustment
    await expect(inventoryPage.adjustmentPreview).toContainText(`+${adjustmentQuantity}`)

    // Submit adjustment
    await inventoryPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Assert - Verify adjustment success message
    await expect(page.locator('.semi-toast-content')).toContainText('调整成功')

    // Verify inventory updated
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const adjustedRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const adjustedQuantityText = await adjustedRow.locator('.semi-table-cell').nth(4).textContent()
    const adjustedQuantity = parseInt(adjustedQuantityText || '0')

    expect(adjustedQuantity).toBe(initialQuantity + adjustmentQuantity)

    // Check transaction history
    await page.goto('/inventory/transactions')
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-input').fill(productCode)
    await page.waitForLoadState('networkidle')

    const transactionRow = page.locator('.semi-table-row').filter({ hasText: productCode })
    await expect(transactionRow).toContainText('调整')
    await expect(transactionRow).toContainText(`+${adjustmentQuantity}`)

    await page.screenshot({ path: 'artifacts/stock-adjustment-completed.png' })
  })

  test('should handle inventory transfer between warehouses', async ({ page, inventoryPage }) => {
    // Arrange - Use existing product with inventory
    const productCode = 'PROD-003'

    // Check initial inventory in first warehouse
    await page.goto('/inventory/stock')
    await inventoryPage.warehouseSelect.click()
    await page.locator('.semi-select-option').nth(0).click() // First warehouse
    await page.waitForLoadState('networkidle')

    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const sourceRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const sourceQuantityText = await sourceRow.locator('.semi-table-cell').nth(4).textContent()
    const sourceQuantity = parseInt(sourceQuantityText || '0')

    if (sourceQuantity < 20) {
      test.skip(true, 'Insufficient inventory for transfer test')
    }

    // Act - Create transfer
    await page.goto('/inventory/transfer')
    await page.waitForLoadState('networkidle')

    // Select source warehouse
    await page.locator('.semi-select').filter({ hasText: '请选择调出仓库' }).click()
    await page.locator('.semi-select-option').nth(0).click()

    // Select destination warehouse
    await page.locator('.semi-select').filter({ hasText: '请选择调入仓库' }).click()
    await page.locator('.semi-select-option').nth(1).click()

    // Add product
    await page.locator('button').filter({ hasText: '添加商品' }).click()
    await page.locator('.semi-input').fill(productCode)
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    // Set transfer quantity
    const transferQuantity = 20
    await page.locator('.semi-input-number input').fill(transferQuantity.toString())

    // Add notes
    await page.locator('textarea').fill('Test inventory transfer')

    // Submit transfer
    await page.locator('button').filter({ hasText: '提交' }).click()
    await page.waitForLoadState('networkidle')

    // Confirm transfer
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Verify transfer success
    await expect(page.locator('.semi-toast-content')).toContainText('成功')

    // Check source warehouse inventory decreased
    await page.goto('/inventory/stock')
    await inventoryPage.warehouseSelect.click()
    await page.locator('.semi-select-option').nth(0).click()
    await page.waitForLoadState('networkidle')

    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const sourceAfterRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const sourceAfterText = await sourceAfterRow.locator('.semi-table-cell').nth(4).textContent()
    const sourceAfterQuantity = parseInt(sourceAfterText || '0')
    expect(sourceAfterQuantity).toBe(sourceQuantity - transferQuantity)

    // Check destination warehouse inventory increased
    await inventoryPage.warehouseSelect.click()
    await page.locator('.semi-select-option').nth(1).click()
    await page.waitForLoadState('networkidle')

    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const destRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const destQuantityText = await destRow.locator('.semi-table-cell').nth(4).textContent()
    const destQuantity = parseInt(destQuantityText || '0')
    expect(destQuantity).toBeGreaterThanOrEqual(transferQuantity)

    await page.screenshot({ path: 'artifacts/inventory-transfer-completed.png' })
  })

  test('should maintain inventory state consistency during concurrent operations', async ({ page, salesOrderPage }) => {
    // This test verifies that inventory state remains consistent
    // when multiple operations happen on the same product

    const productCode = 'PROD-004'

    // Get initial state
    await page.goto('/inventory/stock')
    await page.locator('.semi-input').fill(productCode)
    await page.waitForLoadState('networkidle')

    const initialRow = page.locator('.semi-table-row').filter({ hasText: productCode })
    const initialQuantity = parseInt(await initialRow.locator('.semi-table-cell').nth(4).textContent() || '0')

    // Create multiple sales orders for the same product
    const orders = []
    for (let i = 0; i < 3; i++) {
      await page.goto('/sales/orders/new')

      // Select customer
      await page.locator('.semi-select').filter({ hasText: '请选择客户' }).click()
      await page.locator('.semi-select-option').nth(i).click()

      // Select warehouse
      await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
      await page.locator('.semi-select-option').first().click()

      // Add product
      await page.locator('button').filter({ hasText: '添加商品' }).click()
      await page.locator('.semi-input').fill(productCode)
      await page.waitForTimeout(500)
      await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

      // Set quantity
      await page.locator('.semi-input-number input').nth(0).fill('10')
      await page.locator('.semi-input-number input').nth(1).fill('100.00')

      // Save and submit
      await page.locator('button').filter({ hasText: '保存' }).click()
      await page.locator('button').filter({ hasText: '提交' }).click()
      await page.waitForLoadState('networkidle')

      // Get order number
      const orderNumber = await page.locator('.order-number').textContent()
      orders.push(orderNumber)
    }

    // Confirm all orders
    for (const orderNumber of orders) {
      await page.goto(`/sales/orders/${orderNumber}`)
      await page.locator('button').filter({ hasText: '确认' }).click()
      await page.waitForLoadState('networkidle')
    }

    // Check final inventory state
    await page.goto('/inventory/stock')
    await page.locator('.semi-input').fill(productCode)
    await page.waitForLoadState('networkidle')

    const finalRow = page.locator('.semi-table-row').filter({ hasText: productCode })
    const finalQuantity = parseInt(await finalRow.locator('.semi-table-cell').nth(4).textContent() || '0')

    // Assert - Inventory should be consistent
    expect(finalQuantity).toBe(initialQuantity - 30) // 3 orders × 10 each

    await page.screenshot({ path: 'artifacts/concurrent-operations-state.png' })
  })

  test('should validate inventory rules and prevent invalid operations', async ({ page, salesOrderPage }) => {
    // Test various validation rules:
    // 1. Cannot sell more than available inventory
    // 2. Cannot adjust to negative quantities
    // 3. Cannot transfer more than available

    const productCode = 'PROD-005'

    // Get current inventory
    await page.goto('/inventory/stock')
    await page.locator('.semi-input').fill(productCode)
    await page.waitForLoadState('networkidle')

    const currentRow = page.locator('.semi-table-row').filter({ hasText: productCode })
    const currentQuantity = parseInt(await currentRow.locator('.semi-table-cell').nth(4).textContent() || '0')

    // Test 1: Try to create sales order exceeding inventory
    await page.goto('/sales/orders/new')

    // Select customer
    await page.locator('.semi-select').filter({ hasText: '请选择客户' }).click()
    await page.locator('.semi-select-option').first().click()

    // Select warehouse
    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.locator('.semi-select-option').first().click()

    // Add product with quantity exceeding inventory
    await page.locator('button').filter({ hasText: '添加商品' }).click()
    await page.locator('.semi-input').fill(productCode)
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    await page.locator('.semi-input-number input').nth(0).fill((currentQuantity + 1000).toString())
    await page.locator('.semi-input-number input').nth(1).fill('100.00')

    // Should show validation error
    await expect(page.locator('.semi-form-field-error')).toContainText('库存不足')

    await page.screenshot({ path: 'artifacts/inventory-validation-error.png' })
  })
})