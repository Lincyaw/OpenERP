import { test, expect } from '../fixtures/test-fixtures'
import { PurchaseOrderPage } from '../pages/PurchaseOrderPage'
import { InventoryPage } from '../pages/InventoryPage'

/**
 * P3-INT-002: Purchase Order E2E Tests
 *
 * Tests cover:
 * - Purchase order list display
 * - Order creation with supplier and products
 * - Order confirmation and status change
 * - Full receiving operation with inventory increase
 * - Partial receiving operation with progress display
 * - Accounts payable auto-generation verification
 * - Screenshot assertions for documentation
 */

// Test data from seed-data.sql
const TEST_DATA = {
  suppliers: {
    apple: 'Apple China Distribution',
    samsung: 'Samsung Electronics China',
    xiaomi: 'Xiaomi Technology Ltd',
  },
  products: {
    iPhone: 'iPhone 15 Pro',
    samsung: 'Samsung Galaxy S24',
    xiaomi: 'Xiaomi 14 Pro',
    macbook: 'MacBook Pro 14',
  },
  warehouses: {
    beijing: '北京主仓',
    shanghai: '上海配送中心',
  },
}

test.describe('Purchase Order List Display', () => {
  test.beforeEach(async ({ page }) => {
    // Use stored authentication state
    await page.goto('/')
  })

  test('should display purchase order list page with correct title', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToList()

    await purchaseOrderPage.assertOrderListDisplayed()
    await expect(page.locator('h4').filter({ hasText: '采购订单' })).toBeVisible()
  })

  test('should display empty list or seed orders correctly', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToList()

    // Verify table structure is present
    await expect(page.locator('.semi-table')).toBeVisible()

    // Check that the new order button is available
    await expect(purchaseOrderPage.newOrderButton).toBeVisible()
  })

  test('should have status filter with correct options', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToList()

    // Click the status filter to open dropdown
    const statusSelect = page.locator('.semi-select').first()
    await statusSelect.click()
    await page.waitForTimeout(200)

    // Verify all status options are present
    const options = page.locator('.semi-select-option')
    await expect(options.filter({ hasText: '全部状态' })).toBeVisible()
    await expect(options.filter({ hasText: '草稿' })).toBeVisible()
    await expect(options.filter({ hasText: '已确认' })).toBeVisible()
    await expect(options.filter({ hasText: '部分收货' })).toBeVisible()
    await expect(options.filter({ hasText: '已完成' })).toBeVisible()
    await expect(options.filter({ hasText: '已取消' })).toBeVisible()

    // Close dropdown
    await page.keyboard.press('Escape')
  })
})

test.describe('Purchase Order Creation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('should navigate to new order form', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToList()

    await purchaseOrderPage.clickNewOrder()
    await expect(page).toHaveURL(/\/trade\/purchase\/new/)
  })

  test('should display supplier selection dropdown', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToNewOrder()

    // Verify supplier selection is available
    const supplierSelect = page.locator('.semi-select').filter({ hasText: /供应商/ })
    await expect(supplierSelect).toBeVisible()
  })

  test('should create purchase order with supplier and single product', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToNewOrder()

    // Select supplier
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.apple)
    await page.waitForTimeout(300)

    // Add product row and select product
    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.iPhone)

    // Set quantity and cost
    await purchaseOrderPage.setQuantityInRow(0, 10)
    await purchaseOrderPage.setUnitCostInRow(0, 7000)
    await page.waitForTimeout(300)

    // Submit order
    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()

    // Verify we're back on the list page
    await expect(page).toHaveURL(/\/trade\/purchase/)
  })

  // Using detail page actions instead of dropdown menus
  test('should create order with multiple products', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToNewOrder()

    // Select supplier
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.samsung)
    await page.waitForTimeout(300)

    // Add first product
    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.samsung)
    await purchaseOrderPage.setQuantityInRow(0, 5)
    await purchaseOrderPage.setUnitCostInRow(0, 6000)

    // Add second product
    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(1, TEST_DATA.products.xiaomi)
    await purchaseOrderPage.setQuantityInRow(1, 15)
    await purchaseOrderPage.setUnitCostInRow(1, 3500)

    await page.waitForTimeout(300)

    // Submit order
    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()
  })
})

test.describe('Order Amount Calculation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  // Using detail page for verification
  test('should calculate item amount correctly (unit cost × quantity)', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToNewOrder()

    // Select supplier
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.apple)
    await page.waitForTimeout(300)

    // Add product with specific values
    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.iPhone)
    await purchaseOrderPage.setQuantityInRow(0, 10)
    await purchaseOrderPage.setUnitCostInRow(0, 7000)
    await page.waitForTimeout(500)

    // Check if amount is calculated (should be 70000)
    const amountDisplay = page.locator('.summary-item, .total-amount')
    const amountText = await amountDisplay.textContent()
    expect(amountText).toContain('70,000') // May be formatted with comma
  })
})

test.describe('Order Confirm with Status Change', () => {
  test.describe.configure({ mode: 'serial' })

  let createdOrderNumber: string

  // Using detail page actions instead of dropdown menus for reliable headless Chrome testing
  test('should create and confirm order, verifying status change', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)

    // Create a new order first
    await purchaseOrderPage.navigateToNewOrder()
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.xiaomi)
    await page.waitForTimeout(300)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.xiaomi)
    await purchaseOrderPage.setQuantityInRow(0, 20)
    await purchaseOrderPage.setUnitCostInRow(0, 3500)
    await page.waitForTimeout(300)

    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()

    // Navigate to order list and find the newly created order
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(500)

    // Get the first order row (most recent)
    const firstRow = await purchaseOrderPage.getOrderRow(0)
    const orderText = await firstRow.textContent()
    const orderNumberMatch = orderText?.match(/PO-[\w-]+/)
    createdOrderNumber = orderNumberMatch?.[0] || ''

    // Verify it's in draft status
    await expect(firstRow.locator('.semi-tag')).toContainText('草稿')

    // Navigate to detail page and confirm from there
    await purchaseOrderPage.viewOrderFromRow(firstRow)
    await purchaseOrderPage.assertDetailPageStatus('draft')

    // Confirm the order from detail page
    await purchaseOrderPage.confirmOrderFromDetail()

    // Verify status changed to confirmed
    await purchaseOrderPage.assertDetailPageStatus('confirmed')

    // Go back to list and verify there too
    await purchaseOrderPage.goBackToList()
    const confirmedRow = await purchaseOrderPage.getOrderRowByNumber(createdOrderNumber)
    if (confirmedRow) {
      await expect(confirmedRow.locator('.semi-tag')).toContainText('已确认')
    }
  })
})

test.describe('Full Receiving Operation', () => {
  test.describe.configure({ mode: 'serial' })

  let _testOrderNumber: string

  // Using detail page actions for reliable headless Chrome testing
  test('should receive all items and verify inventory increase', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    const inventoryPage = new InventoryPage(page)

    // First, get initial inventory for iPhone
    await inventoryPage.navigateToStockList()
    await inventoryPage.filterByWarehouse(TEST_DATA.warehouses.beijing)
    await page.waitForTimeout(300)

    const iPhoneRow = await inventoryPage.getInventoryRowByProductName(TEST_DATA.products.iPhone)
    let initialAvailable = 0
    if (iPhoneRow) {
      const quantities = await inventoryPage.getQuantitiesFromRow(iPhoneRow)
      initialAvailable = quantities.available
    }

    // Create and confirm a new purchase order
    await purchaseOrderPage.navigateToNewOrder()
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.apple)
    await page.waitForTimeout(300)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.iPhone)
    await purchaseOrderPage.setQuantityInRow(0, 5) // Order 5 iPhones
    await purchaseOrderPage.setUnitCostInRow(0, 7000)
    await page.waitForTimeout(300)

    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()

    // Get the order number
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(500)

    const firstRow = await purchaseOrderPage.getOrderRow(0)
    const orderText = await firstRow.textContent()
    const orderNumberMatch = orderText?.match(/PO-[\w-]+/)
    _testOrderNumber = orderNumberMatch?.[0] || ''

    // Navigate to detail page and confirm
    await purchaseOrderPage.viewOrderFromRow(firstRow)
    await purchaseOrderPage.confirmOrderFromDetail()
    await purchaseOrderPage.assertDetailPageStatus('confirmed')

    // Go to receive page from detail page
    await purchaseOrderPage.goToReceiveFromDetail()

    // Verify receive page is displayed
    await purchaseOrderPage.assertReceivePageDisplayed()

    // Select warehouse
    await purchaseOrderPage.selectReceiveWarehouse(TEST_DATA.warehouses.beijing)
    await page.waitForTimeout(300)

    // Click receive all
    await purchaseOrderPage.clickReceiveAll()
    await page.waitForTimeout(300)

    // Submit receive
    await purchaseOrderPage.submitReceive()
    await purchaseOrderPage.waitForReceiveSuccess()

    // Verify inventory increased
    await inventoryPage.navigateToStockList()
    await inventoryPage.filterByWarehouse(TEST_DATA.warehouses.beijing)
    await page.waitForTimeout(500)

    const updatedIPhoneRow = await inventoryPage.getInventoryRowByProductName(
      TEST_DATA.products.iPhone
    )
    if (updatedIPhoneRow) {
      const updatedQuantities = await inventoryPage.getQuantitiesFromRow(updatedIPhoneRow)
      // Available should have increased by 5
      expect(updatedQuantities.available).toBeGreaterThanOrEqual(initialAvailable + 5)
    }
  })
})

test.describe('Partial Receiving Operation', () => {
  test.describe.configure({ mode: 'serial' })

  let partialOrderNumber: string

  // Using detail page actions for reliable headless Chrome testing
  test('should perform partial receiving and verify progress display', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)

    // Create and confirm a new purchase order with larger quantity
    await purchaseOrderPage.navigateToNewOrder()
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.samsung)
    await page.waitForTimeout(300)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.samsung)
    await purchaseOrderPage.setQuantityInRow(0, 20) // Order 20 units
    await purchaseOrderPage.setUnitCostInRow(0, 6000)
    await page.waitForTimeout(300)

    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()

    // Get the order number
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(500)

    const firstRow = await purchaseOrderPage.getOrderRow(0)
    const orderText = await firstRow.textContent()
    const orderNumberMatch = orderText?.match(/PO-[\w-]+/)
    partialOrderNumber = orderNumberMatch?.[0] || ''

    // Navigate to detail page and confirm
    await purchaseOrderPage.viewOrderFromRow(firstRow)
    await purchaseOrderPage.confirmOrderFromDetail()
    await purchaseOrderPage.assertDetailPageStatus('confirmed')

    // Go to receive page from detail page
    await purchaseOrderPage.goToReceiveFromDetail()

    // Set partial receive quantity (only 10 out of 20)
    await purchaseOrderPage.selectReceiveWarehouse(TEST_DATA.warehouses.beijing)
    await purchaseOrderPage.setReceiveQuantity(0, 10)
    await page.waitForTimeout(300)

    // Submit partial receive
    await purchaseOrderPage.submitReceive()
    await purchaseOrderPage.waitForReceiveSuccess()

    // Verify order shows partial received status with progress
    await purchaseOrderPage.navigateToList()
    const partialRow = await purchaseOrderPage.getOrderRowByNumber(partialOrderNumber)
    if (partialRow) {
      // Should show "部分收货" status
      await expect(partialRow.locator('.semi-tag')).toContainText('部分收货')

      // Should show progress bar with ~50%
      const progressBar = partialRow.locator('.semi-progress')
      await expect(progressBar).toBeVisible()
    }
  })

  // Using detail page actions
  test('should complete remaining receiving and show completed status', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)

    // Find the partial order and complete receiving
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(300)

    // Filter to find partial_received orders
    await purchaseOrderPage.filterByStatus('partial_received')
    await page.waitForTimeout(500)

    const partialRow = await purchaseOrderPage.getOrderRow(0)
    if (partialRow) {
      // Navigate to detail page, then to receive page
      await purchaseOrderPage.viewOrderFromRow(partialRow)
      await purchaseOrderPage.goToReceiveFromDetail()

      // Receive all remaining
      await purchaseOrderPage.selectReceiveWarehouse(TEST_DATA.warehouses.beijing)
      await purchaseOrderPage.clickReceiveAll()
      await page.waitForTimeout(300)

      await purchaseOrderPage.submitReceive()
      await purchaseOrderPage.waitForReceiveSuccess()
    }

    // Verify order is now completed
    await purchaseOrderPage.navigateToList()
    await purchaseOrderPage.filterByStatus('completed')
    await page.waitForTimeout(500)

    // Should have at least one completed order
    const orderCount = await purchaseOrderPage.getOrderCount()
    expect(orderCount).toBeGreaterThan(0)
  })
})

test.describe('Accounts Payable Auto-Generation', () => {
  // TODO: Skip - requires dropdown interaction for receive action
  // Using detail page actions for reliable headless Chrome testing
  test('should verify accounts payable is generated after receiving', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)

    // Create, confirm, and receive a purchase order
    await purchaseOrderPage.navigateToNewOrder()
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.xiaomi)
    await page.waitForTimeout(300)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.xiaomi)
    await purchaseOrderPage.setQuantityInRow(0, 3)
    await purchaseOrderPage.setUnitCostInRow(0, 3500)
    await page.waitForTimeout(300)

    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()

    // Get order number
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(500)

    const firstRow = await purchaseOrderPage.getOrderRow(0)
    const orderText = await firstRow.textContent()
    const orderNumberMatch = orderText?.match(/PO-[\w-]+/)
    const orderNumber = orderNumberMatch?.[0] || ''

    // Navigate to detail page, confirm, and receive using detail page actions
    await purchaseOrderPage.viewOrderFromRow(firstRow)
    await purchaseOrderPage.confirmOrderFromDetail()
    await purchaseOrderPage.goToReceiveFromDetail()

    await purchaseOrderPage.selectReceiveWarehouse(TEST_DATA.warehouses.beijing)
    await purchaseOrderPage.clickReceiveAll()
    await purchaseOrderPage.submitReceive()
    await purchaseOrderPage.waitForReceiveSuccess()

    // Navigate to accounts payable list to verify AP was generated
    await page.goto('/finance/payable')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(500)

    // Verify the payable list contains an entry related to our order
    const apTable = page.locator('.semi-table')
    await expect(apTable).toBeVisible()

    // Search for the order number in AP list
    const searchInput = page.locator('input[placeholder*="搜索"]')
    if (await searchInput.isVisible()) {
      await searchInput.fill(orderNumber)
      await page.waitForTimeout(500)
    }

    // Check if AP entry exists (may have different format)
    const tableBody = page.locator('.semi-table-tbody')
    const apText = await tableBody.textContent()

    // The AP should reference the purchase order or supplier
    expect(apText).toContain(TEST_DATA.suppliers.xiaomi)
  })
})

test.describe('Order Cancellation', () => {
  // Using detail page actions for reliable headless Chrome testing
  test('should cancel draft order', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)

    // Create a new order
    await purchaseOrderPage.navigateToNewOrder()
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.apple)
    await page.waitForTimeout(300)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.macbook)
    await purchaseOrderPage.setQuantityInRow(0, 2)
    await purchaseOrderPage.setUnitCostInRow(0, 12000)
    await page.waitForTimeout(300)

    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()

    // Navigate to list and then to detail page
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(500)

    // Get first row and navigate to detail page
    const firstRow = await purchaseOrderPage.getOrderRow(0)
    await purchaseOrderPage.viewOrderFromRow(firstRow)

    // Cancel from detail page
    await purchaseOrderPage.cancelOrderFromDetail()

    // Verify status changed to cancelled
    await purchaseOrderPage.assertDetailPageStatus('cancelled')

    // Go back to list and verify
    await purchaseOrderPage.goBackToList()
    await purchaseOrderPage.filterByStatus('cancelled')
    await page.waitForTimeout(500)

    const cancelledCount = await purchaseOrderPage.getOrderCount()
    expect(cancelledCount).toBeGreaterThan(0)
  })

  test('should cancel confirmed order before receiving', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)

    // Create and confirm a new order
    await purchaseOrderPage.navigateToNewOrder()
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.samsung)
    await page.waitForTimeout(300)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.samsung)
    await purchaseOrderPage.setQuantityInRow(0, 3)
    await purchaseOrderPage.setUnitCostInRow(0, 6000)
    await page.waitForTimeout(300)

    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()

    // Navigate to list and then to detail page to confirm
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(500)

    const firstRow = await purchaseOrderPage.getOrderRow(0)
    await purchaseOrderPage.viewOrderFromRow(firstRow)

    // Confirm then cancel from detail page
    await purchaseOrderPage.confirmOrderFromDetail()
    await purchaseOrderPage.assertDetailPageStatus('confirmed')

    await purchaseOrderPage.cancelOrderFromDetail()
    await purchaseOrderPage.assertDetailPageStatus('cancelled')

    // Verify cancelled in list
    await purchaseOrderPage.goBackToList()
    await purchaseOrderPage.filterByStatus('cancelled')
    const cancelledCount = await purchaseOrderPage.getOrderCount()
    expect(cancelledCount).toBeGreaterThan(0)
  })
})

test.describe('Screenshot Documentation', () => {
  test('should capture purchase order list page screenshot', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(500)

    await purchaseOrderPage.screenshotOrderList('purchase-order-list')
  })

  test('should capture purchase order creation form screenshot', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await purchaseOrderPage.navigateToNewOrder()

    // Fill in some data for a more complete screenshot
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.apple)
    await page.waitForTimeout(300)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.iPhone)
    await purchaseOrderPage.setQuantityInRow(0, 10)
    await purchaseOrderPage.setUnitCostInRow(0, 7000)
    await page.waitForTimeout(300)

    await purchaseOrderPage.screenshotOrderForm('purchase-order-form')
  })

  // Using detail page actions for reliable headless Chrome testing
  test('should capture receiving page screenshot', async ({ page }) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)

    // First create and confirm an order
    await purchaseOrderPage.navigateToNewOrder()
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.xiaomi)
    await page.waitForTimeout(300)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.xiaomi)
    await purchaseOrderPage.setQuantityInRow(0, 10)
    await purchaseOrderPage.setUnitCostInRow(0, 3500)
    await page.waitForTimeout(300)

    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()

    // Navigate to list, then detail page, then confirm
    await purchaseOrderPage.navigateToList()
    const firstRow = await purchaseOrderPage.getOrderRow(0)
    await purchaseOrderPage.viewOrderFromRow(firstRow)
    await purchaseOrderPage.confirmOrderFromDetail()

    // Navigate to receive page from detail page
    await purchaseOrderPage.goToReceiveFromDetail()

    await purchaseOrderPage.selectReceiveWarehouse(TEST_DATA.warehouses.beijing)
    await purchaseOrderPage.clickReceiveAll()
    await page.waitForTimeout(300)

    await purchaseOrderPage.screenshotReceivePage('purchase-order-receive')
  })
})

test.describe('Video Recording - Complete Purchase Flow', () => {
  // Using detail page actions for reliable headless Chrome testing
  test('should record complete purchase order lifecycle video', async ({ page }) => {
    // This test is designed to be run with --video=on flag
    // npx playwright test --grep "complete purchase order lifecycle" --video=on

    const purchaseOrderPage = new PurchaseOrderPage(page)
    const inventoryPage = new InventoryPage(page)

    // Step 1: Navigate to purchase order list
    await purchaseOrderPage.navigateToList()
    await page.waitForTimeout(1000)

    // Step 2: Create new order
    await purchaseOrderPage.clickNewOrder()
    await page.waitForTimeout(500)

    // Step 3: Fill order details
    await purchaseOrderPage.selectSupplier(TEST_DATA.suppliers.apple)
    await page.waitForTimeout(500)

    await purchaseOrderPage.addProductRow()
    await purchaseOrderPage.selectProductInRow(0, TEST_DATA.products.iPhone)
    await purchaseOrderPage.setQuantityInRow(0, 8)
    await purchaseOrderPage.setUnitCostInRow(0, 7000)
    await page.waitForTimeout(500)

    // Step 4: Submit order
    await purchaseOrderPage.submitOrder()
    await purchaseOrderPage.waitForOrderCreateSuccess()
    await page.waitForTimeout(1000)

    // Step 5: Navigate to detail page and confirm order
    await purchaseOrderPage.navigateToList()
    const draftRow = await purchaseOrderPage.getOrderRow(0)
    await purchaseOrderPage.viewOrderFromRow(draftRow)
    await purchaseOrderPage.confirmOrderFromDetail()
    await page.waitForTimeout(1000)

    // Step 6: Receive order from detail page
    await purchaseOrderPage.goToReceiveFromDetail()
    await page.waitForTimeout(500)

    await purchaseOrderPage.selectReceiveWarehouse(TEST_DATA.warehouses.beijing)
    await purchaseOrderPage.clickReceiveAll()
    await page.waitForTimeout(500)

    await purchaseOrderPage.submitReceive()
    await purchaseOrderPage.waitForReceiveSuccess()
    await page.waitForTimeout(1000)

    // Step 7: Verify inventory
    await inventoryPage.navigateToStockList()
    await inventoryPage.filterByWarehouse(TEST_DATA.warehouses.beijing)
    await page.waitForTimeout(1000)

    // Step 8: Verify accounts payable
    await page.goto('/finance/payable')
    await page.waitForLoadState('networkidle')
    await page.waitForTimeout(1000)

    // Done - video will capture the entire flow
  })
})
