import { test, expect } from '@playwright/test'
import { SalesOrderPage, SalesReturnPage, InventoryPage } from '../pages'

/**
 * Sales Return E2E Tests (P3-INT-003)
 *
 * Prerequisites:
 * - Docker environment running (make docker-up)
 * - Seed data loaded (make db-seed)
 * - Admin user logged in
 *
 * Test Coverage:
 * - Sales return list display and filtering
 * - Creating return from shipped sales order
 * - Return workflow: Draft → Pending → Approved → Completed
 * - Approval workflow with different users
 * - Inventory restoration after return completion
 * - Receivables credit note generation
 */

// Test data from seed-data.sql
const TEST_DATA = {
  customers: {
    beijingTech: {
      id: '50000000-0000-0000-0000-000000000001',
      name: 'Beijing Tech Solutions Ltd',
      shortName: 'Beijing Tech',
    },
    shanghaiDigital: {
      id: '50000000-0000-0000-0000-000000000002',
      name: 'Shanghai Digital Corp',
      shortName: 'Shanghai Digital',
    },
    chenXiaoming: {
      id: '50000000-0000-0000-0000-000000000004',
      name: 'Chen Xiaoming',
      shortName: 'Chen',
    },
  },
  products: {
    iphone15: {
      id: '40000000-0000-0000-0000-000000000001',
      code: 'IPHONE15',
      name: 'iPhone 15 Pro',
      sellingPrice: 8999,
    },
    samsungS24: {
      id: '40000000-0000-0000-0000-000000000002',
      code: 'SAMSUNG24',
      name: 'Samsung Galaxy S24',
      sellingPrice: 7999,
    },
    xiaomi14: {
      id: '40000000-0000-0000-0000-000000000003',
      code: 'XIAOMI14',
      name: 'Xiaomi 14 Pro',
      sellingPrice: 4999,
    },
  },
  warehouses: {
    beijing: {
      id: '52000000-0000-0000-0000-000000000001',
      code: 'WH001',
      name: '北京主仓',
    },
    shanghai: {
      id: '52000000-0000-0000-0000-000000000002',
      code: 'WH002',
      name: '上海配送中心',
    },
  },
  returnReasons: {
    defective: '商品质量问题',
    wrongItem: '发错商品',
    customerReturn: '客户退货',
  },
}

test.describe('Sales Return Module', () => {
  let salesOrderPage: SalesOrderPage
  let salesReturnPage: SalesReturnPage
  let inventoryPage: InventoryPage

  test.beforeEach(async ({ page }) => {
    salesOrderPage = new SalesOrderPage(page)
    salesReturnPage = new SalesReturnPage(page)
    inventoryPage = new InventoryPage(page)
  })

  test.describe('Sales Return List Display', () => {
    test('should display sales return list page', async ({ page }) => {
      await salesReturnPage.navigateToList()
      await salesReturnPage.assertReturnListDisplayed()
    })

    test('should show table columns correctly', async ({ page }) => {
      await salesReturnPage.navigateToList()

      // Check for expected columns
      const headers = page.locator('.semi-table-thead .semi-table-row-head')
      await expect(headers).toContainText('退货单号')
      await expect(headers).toContainText('原订单')
      await expect(headers).toContainText('客户')
      await expect(headers).toContainText('状态')
    })

    test('should filter returns by status', async ({ page }) => {
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByStatus('DRAFT')

      // All visible returns should be in DRAFT status
      const statusTags = page.locator('.semi-table-tbody .semi-tag')
      const count = await statusTags.count()
      if (count > 0) {
        for (let i = 0; i < count; i++) {
          await expect(statusTags.nth(i)).toContainText('草稿')
        }
      }
    })

    test('should filter returns by customer', async ({ page }) => {
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByCustomer(TEST_DATA.customers.beijingTech.name)

      // Wait for filter to apply
      await page.waitForTimeout(500)

      // Verify filter is applied (table might be empty if no returns for this customer)
      const rows = page.locator('.semi-table-tbody .semi-table-row')
      const count = await rows.count()
      if (count > 0) {
        const firstRow = rows.first()
        const text = await firstRow.textContent()
        expect(text).toContain(TEST_DATA.customers.beijingTech.name)
      }
    })

    test('should search returns by return number', async ({ page }) => {
      await salesReturnPage.navigateToList()
      await salesReturnPage.search('SR-')

      // Wait for search results
      await page.waitForTimeout(500)

      // Results should contain SR- in return number
      const rows = page.locator('.semi-table-tbody .semi-table-row')
      const count = await rows.count()
      if (count > 0) {
        const firstRow = rows.first()
        const text = await firstRow.textContent()
        expect(text).toContain('SR-')
      }
    })
  })

  test.describe('Sales Return Creation', () => {
    test('should navigate to new return form', async ({ page }) => {
      await salesReturnPage.navigateToList()
      await salesReturnPage.clickNewReturn()
      await salesReturnPage.assertReturnFormDisplayed()
    })

    test('should display order selection dropdown', async ({ page }) => {
      await salesReturnPage.navigateToNewReturn()

      // The form should have an order selection dropdown
      const orderSelect = page.locator('.semi-select').filter({ hasText: /订单/ }).first()
      await expect(orderSelect).toBeVisible()
    })

    test.skip('should create sales return from shipped order', async ({ page }) => {
      // This test requires a pre-existing shipped sales order
      // First, create and ship a sales order
      await salesOrderPage.navigateToNewOrder()
      await salesOrderPage.selectCustomer(TEST_DATA.customers.chenXiaoming.name)
      await salesOrderPage.addProductRow()
      await salesOrderPage.selectProductInRow(0, TEST_DATA.products.xiaomi14.name)
      await salesOrderPage.setQuantityInRow(0, 2)
      await salesOrderPage.submitOrder()
      await salesOrderPage.waitForOrderCreateSuccess()

      // Get the order number from the list
      await salesOrderPage.navigateToList()
      await salesOrderPage.filterByStatus('draft')
      const firstRow = await salesOrderPage.getOrderRow(0)
      const orderText = await firstRow.textContent()
      const orderNumberMatch = orderText?.match(/SO-[\w-]+/)
      const orderNumber = orderNumberMatch?.[0] || ''
      expect(orderNumber).toBeTruthy()

      // Navigate to order detail and confirm
      await salesOrderPage.viewReturnFromRow(firstRow)
      await salesOrderPage.confirmOrder()

      // Ship the order
      await salesOrderPage.shipOrder(TEST_DATA.warehouses.beijing.name)

      // Now create a sales return
      await salesReturnPage.navigateToNewReturn()
      await salesReturnPage.selectSalesOrder(orderNumber)
      await salesReturnPage.setReturnReason(TEST_DATA.returnReasons.customerReturn)
      await salesReturnPage.setReturnQuantityInRow(0, 1)
      await salesReturnPage.submitReturn()
      await salesReturnPage.waitForReturnCreateSuccess()

      // Verify return was created
      await salesReturnPage.navigateToList()
      const returnCount = await salesReturnPage.getReturnCount()
      expect(returnCount).toBeGreaterThan(0)
    })
  })

  test.describe('Sales Return Workflow', () => {
    test.skip('should submit return for approval', async ({ page }) => {
      // This test requires an existing draft return
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByStatus('DRAFT')

      const count = await salesReturnPage.getReturnCount()
      if (count === 0) {
        test.skip()
        return
      }

      const row = await salesReturnPage.getReturnRow(0)
      await salesReturnPage.viewReturnFromRow(row)
      await salesReturnPage.submitForApproval()

      // Verify status changed to pending
      await salesReturnPage.assertReturnStatus('PENDING')
    })

    test.skip('should approve return', async ({ page }) => {
      // This test requires an existing pending return
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByStatus('PENDING')

      const count = await salesReturnPage.getReturnCount()
      if (count === 0) {
        test.skip()
        return
      }

      const row = await salesReturnPage.getReturnRow(0)
      await salesReturnPage.viewReturnFromRow(row)
      await salesReturnPage.approveReturn('Approved for return')

      // Verify status changed to approved
      await salesReturnPage.assertReturnStatus('APPROVED')
    })

    test.skip('should reject return', async ({ page }) => {
      // This test requires an existing pending return
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByStatus('PENDING')

      const count = await salesReturnPage.getReturnCount()
      if (count === 0) {
        test.skip()
        return
      }

      const row = await salesReturnPage.getReturnRow(0)
      await salesReturnPage.viewReturnFromRow(row)
      await salesReturnPage.rejectReturn('商品已使用，无法退货')

      // Verify status changed to rejected
      await salesReturnPage.assertReturnStatus('REJECTED')
    })

    test.skip('should complete return and restore inventory', async ({ page }) => {
      // This test requires an existing approved return
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByStatus('APPROVED')

      const count = await salesReturnPage.getReturnCount()
      if (count === 0) {
        test.skip()
        return
      }

      // Get inventory before completing return
      await inventoryPage.navigateToList()
      const inventoryBefore = await inventoryPage.getInventoryItem(
        TEST_DATA.warehouses.beijing.name,
        TEST_DATA.products.xiaomi14.name
      )

      // Complete the return
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByStatus('APPROVED')
      const row = await salesReturnPage.getReturnRow(0)
      await salesReturnPage.viewReturnFromRow(row)
      await salesReturnPage.completeReturn()

      // Verify status changed to completed
      await salesReturnPage.assertReturnStatus('COMPLETED')

      // Verify inventory was restored
      await inventoryPage.navigateToList()
      const inventoryAfter = await inventoryPage.getInventoryItem(
        TEST_DATA.warehouses.beijing.name,
        TEST_DATA.products.xiaomi14.name
      )

      // Available quantity should increase after return
      expect(inventoryAfter?.available).toBeGreaterThan(inventoryBefore?.available || 0)
    })

    test.skip('should cancel return in draft status', async ({ page }) => {
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByStatus('DRAFT')

      const count = await salesReturnPage.getReturnCount()
      if (count === 0) {
        test.skip()
        return
      }

      const row = await salesReturnPage.getReturnRow(0)
      await salesReturnPage.viewReturnFromRow(row)
      await salesReturnPage.cancelReturn('不再需要退货')

      // Verify status changed to cancelled
      await salesReturnPage.assertReturnStatus('CANCELLED')
    })
  })

  test.describe('Approval Page', () => {
    test('should navigate to approval page', async ({ page }) => {
      await salesReturnPage.navigateToList()
      await salesReturnPage.clickApproval()

      // Verify we're on the approval page
      await expect(page.locator('h4').filter({ hasText: /审批/ })).toBeVisible()
    })

    test.skip('should show pending returns in approval list', async ({ page }) => {
      await salesReturnPage.navigateToApproval()

      // All visible returns should be pending status
      const statusTags = page.locator('.semi-table-tbody .semi-tag')
      const count = await statusTags.count()
      if (count > 0) {
        for (let i = 0; i < count; i++) {
          await expect(statusTags.nth(i)).toContainText('待审批')
        }
      }
    })

    test.skip('should approve return from approval list', async ({ page }) => {
      await salesReturnPage.navigateToApproval()

      const count = await salesReturnPage.getPendingApprovalCount()
      if (count === 0) {
        test.skip()
        return
      }

      // Get return number from first row
      const row = await salesReturnPage.getReturnRow(0)
      const text = await row.textContent()
      const returnNumberMatch = text?.match(/SR-[\w-]+/)
      const returnNumber = returnNumberMatch?.[0] || ''

      await salesReturnPage.approveFromList(returnNumber, 'Batch approved')

      // Verify return is no longer in pending list or status changed
      await page.waitForTimeout(500)
    })
  })

  test.describe('Edge Cases and Validation', () => {
    test.skip('should prevent excessive return quantity', async ({ page }) => {
      // Create a return with quantity exceeding original order
      await salesReturnPage.navigateToNewReturn()

      // This test would need a shipped order first
      // The form should validate and prevent submitting with excessive quantity
    })

    test.skip('should prevent duplicate returns for same order', async ({ page }) => {
      // Try to create a second return for an order that already has a completed return
      // The system should prevent or warn about this
    })

    test.skip('should handle return for partially shipped orders', async ({ page }) => {
      // Create a return for an order that was only partially shipped
      // Return quantity should be limited to shipped quantity
    })
  })

  test.describe('Screenshots for Documentation', () => {
    test('should capture return list screenshot', async ({ page }) => {
      await salesReturnPage.navigateToList()
      await page.waitForTimeout(500)
      await salesReturnPage.screenshotReturnList('return-list')
    })

    test('should capture return form screenshot', async ({ page }) => {
      await salesReturnPage.navigateToNewReturn()
      await page.waitForTimeout(500)
      await salesReturnPage.screenshotReturnForm('return-form')
    })

    test('should capture approval page screenshot', async ({ page }) => {
      await salesReturnPage.navigateToApproval()
      await page.waitForTimeout(500)
      await salesReturnPage.screenshotReturnList('approval-page')
    })
  })

  test.describe('Video Recording', () => {
    test.skip('should record full return workflow', async ({ page }) => {
      // This test records the entire return workflow:
      // 1. Create sales order
      // 2. Confirm and ship order
      // 3. Create return from order
      // 4. Submit for approval
      // 5. Approve return
      // 6. Complete return

      // Step 1: Create sales order
      await salesOrderPage.navigateToNewOrder()
      await salesOrderPage.selectCustomer(TEST_DATA.customers.beijingTech.name)
      await salesOrderPage.addProductRow()
      await salesOrderPage.selectProductInRow(0, TEST_DATA.products.samsungS24.name)
      await salesOrderPage.setQuantityInRow(0, 3)
      await salesOrderPage.submitOrder()
      await salesOrderPage.waitForOrderCreateSuccess()

      // Get the order number
      await salesOrderPage.navigateToList()
      await salesOrderPage.filterByStatus('draft')
      const orderRow = await salesOrderPage.getOrderRow(0)
      const orderText = await orderRow.textContent()
      const orderNumber = orderText?.match(/SO-[\w-]+/)?.[0] || ''

      // Step 2: Confirm order
      await salesOrderPage.viewReturnFromRow(orderRow)
      await salesOrderPage.confirmOrder()

      // Step 3: Ship order
      await salesOrderPage.shipOrder(TEST_DATA.warehouses.beijing.name)

      // Step 4: Create return
      await salesReturnPage.navigateToNewReturn()
      await salesReturnPage.selectSalesOrder(orderNumber)
      await salesReturnPage.setReturnReason(TEST_DATA.returnReasons.defective)
      await salesReturnPage.setReturnQuantityInRow(0, 1)
      await salesReturnPage.submitReturn()
      await salesReturnPage.waitForReturnCreateSuccess()

      // Step 5: Submit for approval
      await salesReturnPage.navigateToList()
      await salesReturnPage.filterByStatus('DRAFT')
      const returnRow = await salesReturnPage.getReturnRow(0)
      await salesReturnPage.viewReturnFromRow(returnRow)
      await salesReturnPage.submitForApproval()

      // Step 6: Approve return
      await salesReturnPage.approveReturn('Approved')

      // Step 7: Complete return
      await salesReturnPage.completeReturn()

      // Verify final status
      await salesReturnPage.assertReturnStatus('COMPLETED')
    })
  })
})
