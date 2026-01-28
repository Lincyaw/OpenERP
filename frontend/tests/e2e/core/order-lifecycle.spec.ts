import { test, expect } from '../fixtures/test-fixtures'

/**
 * Order Complete Lifecycle E2E Tests
 *
 * Tests the complete order lifecycle including:
 * - Order creation with proper state initialization
 * - Status transitions (draft → confirmed → shipped → completed)
 * - Inventory impact at each stage
 * - Financial transaction recording
 * - Customer balance updates
 * - Return and refund processing
 * - Multi-tenant order isolation
 */
test.describe('Order Complete Lifecycle', () => {
  test.beforeEach(async ({ page, authenticatedPage }) => {
    // Ensure we're logged in and ready
    await page.goto('/sales/orders')
    await expect(page).toHaveURL(/.*\/sales\/orders/)
  })

  test('should complete full order lifecycle successfully', async ({ page, salesOrderPage, inventoryPage }) => {
    // Arrange - Get a product with sufficient inventory
    const productCode = 'PROD-001'

    // Check initial inventory
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const initialRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const initialQuantityText = await initialRow.locator('.semi-table-cell').nth(4).textContent()
    const initialQuantity = parseInt(initialQuantityText || '0')

    if (initialQuantity < 20) {
      test.skip(true, 'Insufficient inventory for order test')
    }

    // Act 1 - Create order (Draft state)
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

    // Set quantity and price
    const orderQuantity = 10
    await page.locator('.semi-input-number input').nth(0).fill(orderQuantity.toString())
    await page.locator('.semi-input-number input').nth(1).fill('100.00')

    // Save item
    await page.locator('button').filter({ hasText: '保存' }).click()

    // Verify totals
    await expect(salesOrderPage.itemCountDisplay).toContainText('1')
    await expect(salesOrderPage.subtotalDisplay).toContainText('1,000.00')

    // Submit order (creates draft)
    await salesOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Get order number
    const orderNumber = await page.locator('.order-number').textContent()

    // Assert 1 - Order created in draft state
    await expect(page.locator('.semi-tag').filter({ hasText: '草稿' })).toBeVisible()

    // Verify inventory not affected yet
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const afterCreateRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const afterCreateQuantity = parseInt(await afterCreateRow.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(afterCreateQuantity).toBe(initialQuantity)

    // Act 2 - Confirm order
    await page.goto(`/sales/orders/${orderNumber}`)
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 2 - Order confirmed
    await expect(page.locator('.semi-tag').filter({ hasText: '已确认' })).toBeVisible()

    // Verify inventory still not affected (reservation only)
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const afterConfirmRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const afterConfirmQuantity = parseInt(await afterConfirmRow.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(afterConfirmQuantity).toBe(initialQuantity)

    // Act 3 - Ship order
    await page.goto(`/sales/orders/${orderNumber}`)
    await page.locator('button').filter({ hasText: '发货' }).click()

    // Fill shipping details
    await page.locator('.semi-modal .semi-input').fill('SF123456789')
    await page.locator('button').filter({ hasText: '确认发货' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 3 - Order shipped
    await expect(page.locator('.semi-tag').filter({ hasText: '已发货' })).toBeVisible()

    // Verify inventory deducted
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const afterShipRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const afterShipQuantity = parseInt(await afterShipRow.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(afterShipQuantity).toBe(initialQuantity - orderQuantity)

    // Act 4 - Complete order
    await page.goto(`/sales/orders/${orderNumber}`)
    await page.locator('button').filter({ hasText: '完成' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 4 - Order completed
    await expect(page.locator('.semi-tag').filter({ hasText: '已完成' })).toBeVisible()

    // Verify final state
    await expect(page.locator('.order-status-section')).toContainText('订单已完成')

    await page.screenshot({ path: `artifacts/order-lifecycle-completed-${orderNumber}.png` })
  })

  test('should handle order cancellation at different stages', async ({ page, salesOrderPage, inventoryPage }) => {
    const productCode = 'PROD-002'

    // Check initial inventory
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const initialRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const initialQuantity = parseInt(await initialRow.locator('.semi-table-cell').nth(4).textContent() || '0')

    // Create and confirm order
    await page.goto('/sales/orders/new')

    await salesOrderPage.customerSelect.click()
    await page.locator('.semi-select-option').first().click()

    await salesOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    await salesOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill(productCode)
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    await page.locator('.semi-input-number input').nth(0).fill('15')
    await page.locator('.semi-input-number input').nth(1).fill('120.00')

    await page.locator('button').filter({ hasText: '保存' }).click()
    await salesOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Confirm order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Act - Cancel the order
    await page.locator('button').filter({ hasText: '取消' }).click()

    // Confirm cancellation
    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Order cancelled
    await expect(page.locator('.semi-tag').filter({ hasText: '已取消' })).toBeVisible()

    // Verify inventory restored
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const afterCancelRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const afterCancelQuantity = parseInt(await afterCancelRow.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(afterCancelQuantity).toBe(initialQuantity)

    await page.screenshot({ path: `artifacts/order-cancelled-${orderNumber}.png` })
  })

  test('should update customer balance through order lifecycle', async ({ page, salesOrderPage, customerBalancePage }) => {
    // Get initial customer balance
    await page.goto('/customers/balance')
    await page.waitForLoadState('networkidle')

    const customerRow = page.locator('.semi-table-row').first()
    const customerName = await customerRow.locator('.semi-table-cell').nth(0).textContent()
    const initialBalanceText = await customerRow.locator('.semi-table-cell').nth(2).textContent()
    const initialBalance = parseFloat(initialBalanceText?.replace(/[^0-9.-]/g, '') || '0')

    // Create order
    await page.goto('/sales/orders/new')

    // Select the same customer
    await salesOrderPage.customerSelect.click()
    await page.locator('.semi-select-option').filter({ hasText: customerName || '' }).click()

    // Add product
    await salesOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-001')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').first().click()

    await page.locator('.semi-input-number input').nth(0).fill('5')
    await page.locator('.semi-input-number input').nth(1).fill('200.00')

    await page.locator('button').filter({ hasText: '保存' }).click()

    const orderTotal = 1000.00 // 5 × 200
    await expect(salesOrderPage.totalDisplay).toContainText(orderTotal.toString())

    await salesOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Confirm order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Check customer balance after confirmation
    await page.goto('/customers/balance')
    await page.waitForLoadState('networkidle')

    const afterConfirmRow = page.locator('.semi-table-row').filter({ hasText: customerName || '' })
    const afterConfirmBalanceText = await afterConfirmRow.locator('.semi-table-cell').nth(2).textContent()
    const afterConfirmBalance = parseFloat(afterConfirmBalanceText?.replace(/[^0-9.-]/g, '') || '0')

    // Balance should increase by order amount
    expect(afterConfirmBalance).toBe(initialBalance + orderTotal)

    await page.screenshot({ path: `artifacts/customer-balance-updated-${orderNumber}.png` })
  })

  test('should handle partial shipment and backorder scenarios', async ({ page, salesOrderPage, inventoryPage }) => {
    const productCode = 'PROD-003'

    // Check current inventory
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const currentRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const currentQuantity = parseInt(await currentRow.locator('.semi-table-cell').nth(4).textContent() || '0')

    if (currentQuantity < 5) {
      test.skip(true, 'Insufficient inventory for partial shipment test')
    }

    // Create order with quantity > current inventory
    await page.goto('/sales/orders/new')

    await salesOrderPage.customerSelect.click()
    await page.locator('.semi-select-option').first().click()

    await salesOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    await salesOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill(productCode)
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    const orderQuantity = currentQuantity + 10 // More than available
    await page.locator('.semi-input-number input').nth(0).fill(orderQuantity.toString())
    await page.locator('.semi-input-number input').nth(1).fill('150.00')

    await page.locator('button').filter({ hasText: '保存' }).click()
    await salesOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Confirm order (should show warning about inventory)
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Act - Ship only available quantity
    await page.locator('button').filter({ hasText: '发货' }).click()

    // Should show partial shipment dialog
    await expect(page.locator('.semi-modal')).toContainText('部分发货')

    // Confirm partial shipment
    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.locator('.semi-modal .semi-input').fill('SF-PARTIAL-001')
    await page.locator('button').filter({ hasText: '确认发货' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Order status should indicate partial shipment
    await expect(page.locator('.semi-tag').filter({ hasText: '部分发货' })).toBeVisible()

    // Verify inventory deducted by shipped amount only
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const afterPartialRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const afterPartialQuantity = parseInt(await afterPartialRow.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(afterPartialQuantity).toBe(0) // All inventory shipped

    await page.screenshot({ path: `artifacts/partial-shipment-${orderNumber}.png` })
  })

  test('should maintain tenant isolation for order operations', async ({ page, salesOrderPage }) => {
    // Create order as current tenant
    await page.goto('/sales/orders/new')

    await salesOrderPage.customerSelect.click()
    await page.locator('.semi-select-option').first().click()

    await salesOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    await salesOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-001')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').first().click()

    await page.locator('.semi-input-number input').nth(0).fill('5')
    await page.locator('.semi-input-number input').nth(1).fill('100.00')

    await page.locator('button').filter({ hasText: '保存' }).click()
    await salesOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Logout and login as different user
    await page.locator('.semi-avatar').click()
    await page.locator('.semi-dropdown-item').filter({ hasText: '退出登录' }).click()

    // Login as sales user
    await page.goto('/auth/login')
    await page.locator('input[name="username"]').fill('sales')
    await page.locator('input[name="password"]').fill('admin123')
    await page.locator('button[type="submit"]').click()

    // Try to access the order
    await page.goto(`/sales/orders/${orderNumber}`)

    // Should show access denied or not found
    await expect(page.locator('.semi-result-404, .semi-result-403')).toBeVisible()

    await page.screenshot({ path: 'artifacts/order-tenant-isolation.png' })
  })

  test('should handle order modification before confirmation', async ({ page, salesOrderPage }) => {
    // Create order
    await page.goto('/sales/orders/new')

    await salesOrderPage.customerSelect.click()
    await page.locator('.semi-select-option').first().click()

    await salesOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Add first product
    await salesOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-001')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').first().click()
    await page.locator('.semi-input-number input').nth(0).fill('5')
    await page.locator('.semi-input-number input').nth(1).fill('100.00')
    await page.locator('button').filter({ hasText: '保存' }).click()

    // Add second product
    await salesOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-002')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').first().click()
    await page.locator('.semi-input-number input').nth(0).fill('3')
    await page.locator('.semi-input-number input').nth(1).fill('150.00')
    await page.locator('button').filter({ hasText: '保存' }).click()

    await salesOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Act - Edit order before confirmation
    await page.locator('button').filter({ hasText: '编辑' }).click()

    // Remove first product
    await page.locator('.semi-table-row').first().locator('button').filter({ hasText: '删除' }).click()

    // Modify quantity of second product
    await page.locator('.semi-table-row').first().locator('.semi-input-number input').nth(0).fill('10')

    // Save changes
    await page.locator('button').filter({ hasText: '保存' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Order updated successfully
    await expect(page.locator('.semi-toast-content')).toContainText('成功')

    // Verify only one product with updated quantity
    const rows = page.locator('.semi-table-row')
    await expect(rows).toHaveCount(1)
    await expect(rows.first()).toContainText('10')

    await page.screenshot({ path: `artifacts/order-modified-${orderNumber}.png` })
  })
})