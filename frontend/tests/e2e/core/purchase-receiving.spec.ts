import { test, expect } from '../fixtures/test-fixtures'

/**
 * Purchase and Receiving Process E2E Tests
 *
 * Tests the complete purchase and receiving workflow including:
 * - Purchase order creation and approval
 * - Partial and full receiving processes
 * - Quality control checkpoints
 * - Inventory updates on receiving
 * - Supplier balance updates
 * - Accounts payable tracking
 * - Return to supplier process
 * - Multi-warehouse receiving
 */
test.describe('Purchase and Receiving Process', () => {
  test.beforeEach(async ({ page, authenticatedPage }) => {
    // Navigate to purchase orders page
    await page.goto('/purchase/orders')
    await expect(page).toHaveURL(/.*\/purchase\/orders/)
  })

  test('should complete full purchase and receiving workflow', async ({ page, purchaseOrderPage, inventoryPage }) => {
    // Arrange - Get supplier and product
    const productCode = 'PROD-001'

    // Check initial inventory
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const initialRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const initialQuantityText = await initialRow.locator('.semi-table-cell').nth(4).textContent()
    const initialQuantity = parseInt(initialQuantityText || '0')

    // Act 1 - Create purchase order
    await page.goto('/purchase/orders/new')

    // Select supplier
    await purchaseOrderPage.supplierSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Select warehouse
    await purchaseOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Add product
    await purchaseOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill(productCode)
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: productCode }).click()

    // Set quantity and price
    const purchaseQuantity = 50
    const purchasePrice = 75.00
    await page.locator('.semi-input-number input').nth(0).fill(purchaseQuantity.toString())
    await page.locator('.semi-input-number input').nth(1).fill(purchasePrice.toString())

    // Save item
    await page.locator('button').filter({ hasText: '保存' }).click()

    // Verify totals
    await expect(purchaseOrderPage.itemCountDisplay).toContainText('1')
    await expect(purchaseOrderPage.subtotalDisplay).toContainText((purchaseQuantity * purchasePrice).toString())

    // Submit order
    await purchaseOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Get order number
    const orderNumber = await page.locator('.order-number').textContent()

    // Assert 1 - Order created in draft state
    await expect(page.locator('.semi-tag').filter({ hasText: '草稿' })).toBeVisible()

    // Act 2 - Confirm purchase order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 2 - Order confirmed
    await expect(page.locator('.semi-tag').filter({ hasText: '已确认' })).toBeVisible()

    // Act 3 - Receive goods
    await page.locator('button').filter({ hasText: '收货' }).click()
    await page.waitForLoadState('networkidle')

    // Verify receiving page
    await expect(page).toHaveURL(/.*\/purchase\/receiving/)
    await expect(page.locator('h4')).toContainText('采购收货')

    // Check that all items are ready to receive
    const receivingRows = page.locator('.semi-table-tbody .semi-table-row')
    await expect(receivingRows).toHaveCount(1)

    // Set receiving quantity (full receive)
    await page.locator('.semi-input-number input').fill(purchaseQuantity.toString())

    // Add quality check notes
    await page.locator('textarea').fill('Quality check passed - all items in good condition')

    // Submit receiving
    await page.locator('button').filter({ hasText: '确认收货' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 3 - Receiving completed
    await expect(page.locator('.semi-toast-content')).toContainText('收货成功')

    // Verify inventory updated
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill(productCode)
    await page.waitForLoadState('networkidle')

    const afterReceiveRow = inventoryPage.tableRows.filter({ hasText: productCode })
    const afterReceiveQuantity = parseInt(await afterReceiveRow.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(afterReceiveQuantity).toBe(initialQuantity + purchaseQuantity)

    // Verify order status updated
    await page.goto(`/purchase/orders/${orderNumber}`)
    await expect(page.locator('.semi-tag').filter({ hasText: '已完成' })).toBeVisible()

    await page.screenshot({ path: `artifacts/purchase-receiving-completed-${orderNumber}.png` })
  })

  test('should handle partial receiving process', async ({ page, purchaseOrderPage, inventoryPage }) => {
    // Create purchase order
    await page.goto('/purchase/orders/new')

    await purchaseOrderPage.supplierSelect.click()
    await page.locator('.semi-select-option').first().click()

    await purchaseOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Add multiple products
    const products = [
      { code: 'PROD-002', quantity: 30, price: 60.00 },
      { code: 'PROD-003', quantity: 40, price: 80.00 }
    ]

    for (const product of products) {
      await purchaseOrderPage.addProductButton.click()
      await page.locator('.semi-input').fill(product.code)
      await page.waitForTimeout(500)
      await page.locator('.semi-select-option').filter({ hasText: product.code }).click()

      await page.locator('.semi-input-number input').nth(0).fill(product.quantity.toString())
      await page.locator('.semi-input-number input').nth(1).fill(product.price.toString())

      await page.locator('button').filter({ hasText: '保存' }).click()
    }

    await purchaseOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Confirm order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Act - Partial receive first batch
    await page.locator('button').filter({ hasText: '收货' }).click()
    await page.waitForLoadState('networkidle')

    // Receive only first product partially
    const receivingRows = page.locator('.semi-table-tbody .semi-table-row')
    await expect(receivingRows).toHaveCount(2)

    // Receive 20 out of 30 for first product
    await receivingRows.nth(0).locator('.semi-input-number input').fill('20')
    // Receive 0 for second product (not receiving yet)
    await receivingRows.nth(1).locator('.semi-input-number input').fill('0')

    await page.locator('button').filter({ hasText: '确认收货' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Partial receiving completed
    await expect(page.locator('.semi-toast-content')).toContainText('收货成功')

    // Verify order status is "Partially Received"
    await page.goto(`/purchase/orders/${orderNumber}`)
    await expect(page.locator('.semi-tag').filter({ hasText: '部分收货' })).toBeVisible()

    // Verify inventory updated only for received items
    await page.goto('/inventory/stock')

    for (const product of products) {
      await inventoryPage.searchInput.fill(product.code)
      await page.waitForLoadState('networkidle')

      const row = inventoryPage.tableRows.filter({ hasText: product.code })
      const quantity = parseInt(await row.locator('.semi-table-cell').nth(4).textContent() || '0')

      if (product.code === 'PROD-002') {
        expect(quantity).toBeGreaterThanOrEqual(20) // Received 20
      }
    }

    // Act - Receive remaining items
    await page.goto(`/purchase/orders/${orderNumber}`)
    await page.locator('button').filter({ hasText: '收货' }).click()
    await page.waitForLoadState('networkidle')

    // Receive remaining quantities
    await receivingRows.nth(0).locator('.semi-input-number input').fill('10') // Remaining for first product
    await receivingRows.nth(1).locator('.semi-input-number input').fill('40') // Full quantity for second

    await page.locator('button').filter({ hasText: '确认收货' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - All items received
    await expect(page.locator('.semi-toast-content')).toContainText('收货成功')

    // Verify order status is now "Completed"
    await page.goto(`/purchase/orders/${orderNumber}`)
    await expect(page.locator('.semi-tag').filter({ hasText: '已完成' })).toBeVisible()

    await page.screenshot({ path: `artifacts/partial-receiving-${orderNumber}.png` })
  })

  test('should handle quality control rejection during receiving', async ({ page, purchaseOrderPage, inventoryPage }) => {
    // Create purchase order
    await page.goto('/purchase/orders/new')

    await purchaseOrderPage.supplierSelect.click()
    await page.locator('.semi-select-option').first().click()

    await purchaseOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Add product
    await purchaseOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-004')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: 'PROD-004' }).click()

    await page.locator('.semi-input-number input').nth(0).fill('25')
    await page.locator('.semi-input-number input').nth(1).fill('90.00')

    await page.locator('button').filter({ hasText: '保存' }).click()
    await purchaseOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Confirm order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Act - Receive with quality issues
    await page.locator('button').filter({ hasText: '收货' }).click()
    await page.waitForLoadState('networkidle')

    // Receive only acceptable quantity
    await page.locator('.semi-input-number input').fill('20') // 5 items rejected

    // Mark quality check as failed
    await page.locator('.semi-select').filter({ hasText: '合格' }).click()
    await page.locator('.semi-select-option').filter({ hasText: '不合格' }).click()

    // Add quality notes
    await page.locator('textarea').fill('Quality check failed: 5 items damaged during transport')

    // Submit receiving
    await page.locator('button').filter({ hasText: '确认收货' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Partial receiving with quality rejection
    await expect(page.locator('.semi-toast-content')).toContainText('收货成功')

    // Verify order shows quality issues
    await page.goto(`/purchase/orders/${orderNumber}`)
    await expect(page.locator('.quality-status')).toContainText('质量问题')

    // Verify inventory updated only for accepted items
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill('PROD-004')
    await page.waitForLoadState('networkidle')

    const row = inventoryPage.tableRows.filter({ hasText: 'PROD-004' })
    const quantity = parseInt(await row.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(quantity).toBeGreaterThanOrEqual(20) // Only accepted quantity

    await page.screenshot({ path: `artifacts/quality-rejection-${orderNumber}.png` })
  })

  test('should handle return to supplier process', async ({ page, purchaseOrderPage, inventoryPage }) => {
    // First, complete a purchase order
    await page.goto('/purchase/orders/new')

    await purchaseOrderPage.supplierSelect.click()
    await page.locator('.semi-select-option').first().click()

    await purchaseOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    await purchaseOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-005')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: 'PROD-005' }).click()

    await page.locator('.semi-input-number input').nth(0).fill('100')
    await page.locator('.semi-input-number input').nth(1).fill('55.00')

    await page.locator('button').filter({ hasText: '保存' }).click()
    await purchaseOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Confirm and receive
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '收货' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-input-number input').fill('100')
    await page.locator('button').filter({ hasText: '确认收货' }).click()
    await page.waitForLoadState('networkidle')

    // Act - Create return to supplier
    await page.goto('/purchase/returns/new')
    await page.waitForLoadState('networkidle')

    // Select supplier
    await page.locator('.semi-select').filter({ hasText: '请选择供应商' }).click()
    await page.locator('.semi-select-option').first().click()

    // Select the original purchase order
    await page.locator('.semi-select').filter({ hasText: '请选择采购订单' }).click()
    await page.locator('.semi-select-option').filter({ hasText: orderNumber || '' }).click()

    // Add product to return
    await page.locator('button').filter({ hasText: '添加商品' }).click()
    await page.locator('.semi-input').fill('PROD-005')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: 'PROD-005' }).click()

    // Set return quantity (partial return)
    const returnQuantity = 20
    await page.locator('.semi-input-number input').nth(0).fill(returnQuantity.toString())
    await page.locator('.semi-input-number input').nth(1).fill('55.00') // Same price

    // Add return reason
    await page.locator('textarea').fill('Defective items found during inspection')

    // Save item
    await page.locator('button').filter({ hasText: '保存' }).click()

    // Submit return
    await page.locator('button').filter({ hasText: '提交' }).click()
    await page.waitForLoadState('networkidle')

    const returnNumber = await page.locator('.return-number').textContent()

    // Assert - Return created
    await expect(page.locator('.semi-tag').filter({ hasText: '草稿' })).toBeVisible()

    // Confirm return
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Verify inventory reduced after return
    await page.goto('/inventory/stock')
    await inventoryPage.searchInput.fill('PROD-005')
    await page.waitForLoadState('networkidle')

    const afterReturnRow = inventoryPage.tableRows.filter({ hasText: 'PROD-005' })
    const afterReturnQuantity = parseInt(await afterReturnRow.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(afterReturnQuantity).toBe(80) // 100 - 20 returned

    await page.screenshot({ path: `artifacts/supplier-return-${returnNumber}.png` })
  })

  test('should track supplier balance through purchase lifecycle', async ({ page, purchaseOrderPage }) => {
    // Get initial supplier balance
    await page.goto('/suppliers')
    await page.waitForLoadState('networkidle')

    const supplierRow = page.locator('.semi-table-row').first()
    const supplierName = await supplierRow.locator('.semi-table-cell').nth(0).textContent()
    const initialBalanceText = await supplierRow.locator('.semi-table-cell').nth(3).textContent()
    const initialBalance = parseFloat(initialBalanceText?.replace(/[^0-9.-]/g, '') || '0')

    // Create purchase order
    await page.goto('/purchase/orders/new')

    await purchaseOrderPage.supplierSelect.click()
    await page.locator('.semi-select-option').filter({ hasText: supplierName || '' }).click()

    await purchaseOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Add product
    await purchaseOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-006')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: 'PROD-006' }).click()

    const orderAmount = 3000.00 // 50 × 60.00
    await page.locator('.semi-input-number input').nth(0).fill('50')
    await page.locator('.semi-input-number input').nth(1).fill('60.00')

    await page.locator('button').filter({ hasText: '保存' }).click()
    await purchaseOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Confirm order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Check supplier balance after confirmation
    await page.goto('/suppliers')
    await page.waitForLoadState('networkidle')

    const afterConfirmRow = page.locator('.semi-table-row').filter({ hasText: supplierName || '' })
    const afterConfirmBalanceText = await afterConfirmRow.locator('.semi-table-cell').nth(3).textContent()
    const afterConfirmBalance = parseFloat(afterConfirmBalanceText?.replace(/[^0-9.-]/g, '') || '0')

    // Balance should increase by order amount (accounts payable)
    expect(afterConfirmBalance).toBe(initialBalance + orderAmount)

    await page.screenshot({ path: 'artifacts/supplier-balance-updated.png' })
  })

  test('should handle multi-warehouse receiving', async ({ page, purchaseOrderPage, inventoryPage }) => {
    // Create purchase order for multiple warehouses
    await page.goto('/purchase/orders/new')

    await purchaseOrderPage.supplierSelect.click()
    await page.locator('.semi-select-option').first().click()

    // Add product
    await purchaseOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-007')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: 'PROD-007' }).click()

    await page.locator('.semi-input-number input').nth(0).fill('200')
    await page.locator('.semi-input-number input').nth(1).fill('45.00')

    await page.locator('button').filter({ hasText: '保存' }).click()
    await purchaseOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    const orderNumber = await page.locator('.order-number').textContent()

    // Confirm order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Act - Receive to different warehouses
    await page.locator('button').filter({ hasText: '收货' }).click()
    await page.waitForLoadState('networkidle')

    // Split receiving - 100 to warehouse 1, 100 to warehouse 2
    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.locator('.semi-select-option').nth(0).click() // First warehouse

    await page.locator('.semi-input-number input').fill('100')
    await page.locator('button').filter({ hasText: '确认收货' }).click()
    await page.waitForLoadState('networkidle')

    // Receive remaining to second warehouse
    await page.goto(`/purchase/orders/${orderNumber}`)
    await page.locator('button').filter({ hasText: '收货' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.locator('.semi-select-option').nth(1).click() // Second warehouse

    await page.locator('.semi-input-number input').fill('100')
    await page.locator('button').filter({ hasText: '确认收货' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Verify inventory in both warehouses
    await page.goto('/inventory/stock')

    // Check first warehouse
    await inventoryPage.warehouseSelect.click()
    await page.locator('.semi-select-option').nth(0).click()
    await page.waitForLoadState('networkidle')

    await inventoryPage.searchInput.fill('PROD-007')
    await page.waitForLoadState('networkidle')

    const warehouse1Row = inventoryPage.tableRows.filter({ hasText: 'PROD-007' })
    const warehouse1Quantity = parseInt(await warehouse1Row.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(warehouse1Quantity).toBeGreaterThanOrEqual(100)

    // Check second warehouse
    await inventoryPage.warehouseSelect.click()
    await page.locator('.semi-select-option').nth(1).click()
    await page.waitForLoadState('networkidle')

    const warehouse2Row = inventoryPage.tableRows.filter({ hasText: 'PROD-007' })
    const warehouse2Quantity = parseInt(await warehouse2Row.locator('.semi-table-cell').nth(4).textContent() || '0')
    expect(warehouse2Quantity).toBeGreaterThanOrEqual(100)

    await page.screenshot({ path: `artifacts/multi-warehouse-receiving-${orderNumber}.png` })
  })

  test('should validate purchase receiving rules', async ({ page, purchaseOrderPage }) => {
    // Create purchase order
    await page.goto('/purchase/orders/new')

    await purchaseOrderPage.supplierSelect.click()
    await page.locator('.semi-select-option').first().click()

    await purchaseOrderPage.warehouseSelect.click()
    await page.locator('.semi-select-option').first().click()

    await purchaseOrderPage.addProductButton.click()
    await page.locator('.semi-input').fill('PROD-008')
    await page.waitForTimeout(500)
    await page.locator('.semi-select-option').filter({ hasText: 'PROD-008' }).click()

    await page.locator('.semi-input-number input').nth(0).fill('50')
    await page.locator('.semi-input-number input').nth(1).fill('70.00')

    await page.locator('button').filter({ hasText: '保存' }).click()
    await purchaseOrderPage.submitButton.click()
    await page.waitForLoadState('networkidle')

    // Confirm order
    await page.locator('button').filter({ hasText: '确认' }).click()
    await page.waitForLoadState('networkidle')

    // Try to receive more than ordered
    await page.locator('button').filter({ hasText: '收货' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-input-number input').fill('60') // More than ordered (50)

    // Should show validation error
    await expect(page.locator('.semi-form-field-error')).toContainText('收货数量不能大于订单数量')

    // Try to receive negative quantity
    await page.locator('.semi-input-number input').fill('-10')

    // Should show validation error
    await expect(page.locator('.semi-form-field-error')).toContainText('收货数量必须大于0')

    await page.screenshot({ path: 'artifacts/receiving-validation-errors.png' })
  })
})