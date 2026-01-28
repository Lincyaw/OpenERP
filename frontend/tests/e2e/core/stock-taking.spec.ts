import { test, expect } from '../fixtures/test-fixtures'

/**
 * Stock Taking (Inventory Audit) Process E2E Tests
 *
 * Tests the complete stock taking workflow including:
 * - Stock taking creation and planning
 * - Count sheet generation and assignment
 * - Count entry and variance calculation
 * - Approval workflow for adjustments
 * - Inventory adjustment posting
 * - Audit trail maintenance
 * - Multi-warehouse stock taking
 * - Cycle counting processes
 */
test.describe('Stock Taking Process', () => {
  test.beforeEach(async ({ page, authenticatedPage }) => {
    // Navigate to stock taking page
    await page.goto('/inventory/stock-taking')
    await expect(page).toHaveURL(/.*\/inventory\/stock-taking/)
  })

  test('should complete full stock taking workflow', async ({ page, inventoryPage }) => {
    // Act 1 - Create stock taking plan
    await page.locator('button').filter({ hasText: '新建盘点' }).click()
    await page.waitForLoadState('networkidle')

    // Fill stock taking details
    const stockTakingNumber = `ST-${Date.now()}`
    await page.locator('input[placeholder="请输入盘点单号"]').fill(stockTakingNumber)

    // Select warehouse
    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.locator('.semi-select-option').first().click()

    // Select stock taking type
    await page.locator('.semi-select').filter({ hasText: '请选择盘点类型' }).click()
    await page.locator('.semi-select-option').filter({ hasText: '全盘' }).click()

    // Set planned date
    await page.locator('.semi-datepicker').click()
    await page.locator('.semi-datepicker-day-today').click()

    // Add notes
    await page.locator('textarea').fill('Monthly full stock taking for Q1 2024')

    // Save draft
    await page.locator('button').filter({ hasText: '保存' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 1 - Stock taking created
    await expect(page.locator('.semi-toast-content')).toContainText('保存成功')
    await expect(page.locator('.stock-taking-status')).toContainText('草稿')

    // Act 2 - Generate count sheets
    await page.locator('button').filter({ hasText: '生成盘点表' }).click()
    await page.waitForLoadState('networkidle')

    // Confirm generation
    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 2 - Count sheets generated
    await expect(page.locator('.semi-toast-content')).toContainText('盘点表生成成功')

    // Verify count items loaded
    const countItems = page.locator('.count-item-row')
    await expect(countItems.first()).toBeVisible()

    // Act 3 - Start counting (lock inventory)
    await page.locator('button').filter({ hasText: '开始盘点' }).click()
    await page.waitForLoadState('networkidle')

    // Confirm start
    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 3 - Inventory locked and counting started
    await expect(page.locator('.stock-taking-status')).toContainText('盘点中')

    // Act 4 - Enter count results
    const firstItem = countItems.first()
    const systemQuantityText = await firstItem.locator('.system-quantity').textContent()
    const systemQuantity = parseInt(systemQuantityText?.replace(/[^0-9]/g, '') || '0')

    // Enter actual count (simulate variance)
    const actualCount = systemQuantity - 2 // 2 items missing
    await firstItem.locator('.actual-quantity-input').fill(actualCount.toString())

    // Add count notes
    await firstItem.locator('.count-notes').fill('Found damaged packaging')

    // Act 5 - Complete counting
    await page.locator('button').filter({ hasText: '完成盘点' }).click()
    await page.waitForLoadState('networkidle')

    // Confirm completion
    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 5 - Counting completed
    await expect(page.locator('.stock-taking-status')).toContainText('待审核')

    // Verify variance calculated
    const variance = systemQuantity - actualCount
    await expect(firstItem.locator('.variance-quantity')).toContainText(`-${variance}`)

    // Act 6 - Review and approve adjustments
    await page.locator('button').filter({ hasText: '审核' }).click()
    await page.waitForLoadState('networkidle')

    // Approve adjustments
    await page.locator('.semi-radio').filter({ hasText: '通过' }).click()
    await page.locator('textarea').last().fill('Approved - investigate damaged items')

    await page.locator('button').filter({ hasText: '提交' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 6 - Stock taking approved
    await expect(page.locator('.stock-taking-status')).toContainText('已完成')

    // Act 7 - Post inventory adjustments
    await page.locator('button').filter({ hasText: '生成调整单' }).click()
    await page.waitForLoadState('networkidle')

    // Confirm adjustment posting
    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Assert 7 - Adjustments posted
    await expect(page.locator('.semi-toast-content')).toContainText('调整单生成成功')

    // Verify inventory adjusted
    await page.goto('/inventory/stock')
    await page.waitForLoadState('networkidle')

    const productCode = await firstItem.locator('.product-code').textContent()
    await inventoryPage.searchInput.fill(productCode || '')
    await page.waitForLoadState('networkidle')

    const inventoryRow = inventoryPage.tableRows.filter({ hasText: productCode || '' })
    const adjustedQuantity = parseInt(await inventoryRow.locator('.semi-table-cell').nth(4).textContent() || '0')

    // Inventory should be adjusted to actual count
    expect(adjustedQuantity).toBe(actualCount)

    await page.screenshot({ path: `artifacts/stock-taking-completed-${stockTakingNumber}.png` })
  })

  test('should handle cycle counting process', async ({ page, inventoryPage }) => {
    // Create cycle count for specific product category
    await page.locator('button').filter({ hasText: '新建盘点' }).click()
    await page.waitForLoadState('networkidle')

    const cycleCountNumber = `CC-${Date.now()}`
    await page.locator('input[placeholder="请输入盘点单号"]').fill(cycleCountNumber)

    // Select warehouse
    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.locator('.semi-select-option').first().click()

    // Select cycle count type
    await page.locator('.semi-select').filter({ hasText: '请选择盘点类型' }).click()
    await page.locator('.semi-select-option').filter({ hasText: '循环盘点' }).click()

    // Select ABC category (fast-moving items)
    await page.locator('.semi-select').filter({ hasText: '请选择商品范围' }).click()
    await page.locator('.semi-select-option').filter({ hasText: 'A类商品' }).click()

    await page.locator('textarea').fill('Weekly cycle count for A-class items')

    // Save and generate count sheet
    await page.locator('button').filter({ hasText: '保存' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '生成盘点表' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Verify only A-class items included
    const countItems = page.locator('.count-item-row')
    const itemCount = await countItems.count()

    // Should have fewer items than full count
    expect(itemCount).toBeLessThan(50) // Assuming A-class is < 50 items

    // Start counting
    await page.locator('button').filter({ hasText: '开始盘点' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Complete counts for all items
    for (let i = 0; i < itemCount; i++) {
      const item = countItems.nth(i)
      const systemQty = parseInt(await item.locator('.system-quantity').textContent()?.replace(/[^0-9]/g, '') || '0')

      // Simulate small variances or exact counts
      const actualQty = systemQty + (Math.random() > 0.8 ? -1 : 0) // 20% chance of 1 item variance
      await item.locator('.actual-quantity-input').fill(actualQty.toString())
    }

    // Complete cycle count
    await page.locator('button').filter({ hasText: '完成盘点' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Verify cycle count completed
    await expect(page.locator('.stock-taking-status')).toContainText('待审核')

    await page.screenshot({ path: `artifacts/cycle-count-${cycleCountNumber}.png` })
  })

  test('should handle blind counting process', async ({ page, inventoryPage }) => {
    // Create blind count (quantities hidden)
    await page.locator('button').filter({ hasText: '新建盘点' }).click()
    await page.waitForLoadState('networkidle')

    const blindCountNumber = `BLIND-${Date.now()}`
    await page.locator('input[placeholder="请输入盘点单号"]').fill(blindCountNumber)

    // Select warehouse
    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.locator('.semi-select-option').nth(1).click() // Different warehouse

    // Select blind count type
    await page.locator('.semi-select').filter({ hasText: '请选择盘点类型' }).click()
    await page.locator('.semi-select-option').filter({ hasText: '盲盘' }).click()

    await page.locator('textarea').fill('Blind count for accuracy verification')

    // Save and generate
    await page.locator('button').filter({ hasText: '保存' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '生成盘点表' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - System quantities should be hidden
    const countItems = page.locator('.count-item-row')
    await expect(countItems.first()).toBeVisible()

    // System quantity column should be empty or show "***"
    await expect(page.locator('.system-quantity').first()).toContainText('***')

    // Start blind counting
    await page.locator('button').filter({ hasText: '开始盘点' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Enter counts without seeing system quantities
    const firstItem = countItems.first()
    await firstItem.locator('.actual-quantity-input').fill('25') // Enter actual count

    // Complete blind count
    await page.locator('button').filter({ hasText: '完成盘点' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // After completion, variances should be visible
    await expect(page.locator('.variance-quantity').first()).toBeVisible()

    await page.screenshot({ path: `artifacts/blind-count-${blindCountNumber}.png` })
  })

  test('should handle stock taking approval workflow', async ({ page, inventoryPage }) => {
    // Create and complete a stock taking
    await page.locator('button').filter({ hasText: '新建盘点' }).click()
    await page.waitForLoadState('networkidle')

    const approvalNumber = `APP-${Date.now()}`
    await page.locator('input[placeholder="请输入盘点单号"]').fill(approvalNumber)

    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.locator('.semi-select-option').first().click()

    await page.locator('.semi-select').filter({ hasText: '请选择盘点类型' }).click()
    await page.locator('.semi-select-option').filter({ hasText: '全盘' }).click()

    await page.locator('textarea').fill('Stock taking for approval workflow test')

    // Save, generate, start, and complete counting
    await page.locator('button').filter({ hasText: '保存' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '生成盘点表' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '开始盘点' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Enter counts with significant variances
    const countItems = page.locator('.count-item-row')
    const firstItem = countItems.first()

    const systemQuantity = parseInt(await firstItem.locator('.system-quantity').textContent()?.replace(/[^0-9]/g, '') || '0')
    const actualCount = systemQuantity - 10 // 10 items missing (significant variance)
    await firstItem.locator('.actual-quantity-input').fill(actualCount.toString())

    await page.locator('button').filter({ hasText: '完成盘点' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Act - Test approval rejection
    await page.locator('button').filter({ hasText: '审核' }).click()
    await page.waitForLoadState('networkidle')

    // Reject with reason
    await page.locator('.semi-radio').filter({ hasText: '驳回' }).click()
    await page.locator('textarea').last().fill('Variance too high - please recount')

    await page.locator('button').filter({ hasText: '提交' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Stock taking rejected
    await expect(page.locator('.stock-taking-status')).toContainText('已驳回')

    // Should be able to recount
    await expect(page.locator('button').filter({ hasText: '重新盘点' })).toBeVisible()

    await page.screenshot({ path: `artifacts/stock-taking-rejected-${approvalNumber}.png` })
  })

  test('should maintain audit trail for stock taking', async ({ page, inventoryPage }) => {
    // Create stock taking
    await page.locator('button').filter({ hasText: '新建盘点' }).click()
    await page.waitForLoadState('networkidle')

    const auditNumber = `AUDIT-${Date.now()}`
    await page.locator('input[placeholder="请输入盘点单号"]').fill(auditNumber)

    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.locator('.semi-select-option').first().click()

    await page.locator('.semi-select').filter({ hasText: '请选择盘点类型' }).click()
    await page.locator('.semi-select-option').filter({ hasText: '全盘' }).click()

    await page.locator('textarea').fill('Stock taking with audit trail')

    // Complete the full workflow
    await page.locator('button').filter({ hasText: '保存' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '生成盘点表' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '开始盘点' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Enter count
    const firstItem = page.locator('.count-item-row').first()
    await firstItem.locator('.actual-quantity-input').fill('30')

    await page.locator('button').filter({ hasText: '完成盘点' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '审核' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-radio').filter({ hasText: '通过' }).click()
    await page.locator('button').filter({ hasText: '提交' }).click()
    await page.waitForLoadState('networkidle')

    // Act - Check audit trail
    await page.locator('button').filter({ hasText: '操作日志' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Audit trail recorded
    const auditLogs = page.locator('.audit-log-entry')
    await expect(auditLogs.first()).toBeVisible()

    // Verify all stages are logged
    await expect(page.locator('.audit-log-entry')).toContainText('创建盘点单')
    await expect(page.locator('.audit-log-entry')).toContainText('生成盘点表')
    await expect(page.locator('.audit-log-entry')).toContainText('开始盘点')
    await expect(page.locator('.audit-log-entry')).toContainText('完成盘点')
    await expect(page.locator('.audit-log-entry')).toContainText('审核通过')

    // Verify user and timestamp
    await expect(page.locator('.audit-log-entry').first()).toContainText('admin')
    await expect(page.locator('.audit-log-entry').first()).toContainText(/\d{4}-\d{2}-\d{2}/)

    await page.screenshot({ path: `artifacts/stock-taking-audit-${auditNumber}.png` })
  })

  test('should handle multi-warehouse stock taking', async ({ page, inventoryPage }) => {
    // Create stock taking for multiple warehouses
    await page.locator('button').filter({ hasText: '新建盘点' }).click()
    await page.waitForLoadState('networkidle')

    const multiWarehouseNumber = `MULTI-${Date.now()}`
    await page.locator('input[placeholder="请输入盘点单号"]').fill(multiWarehouseNumber)

    // Select multiple warehouses
    await page.locator('.semi-select').filter({ hasText: '请选择仓库' }).click()
    await page.keyboard.down('Control')
    await page.locator('.semi-select-option').nth(0).click()
    await page.locator('.semi-select-option').nth(1).click()
    await page.keyboard.up('Control')
    await page.locator('body').click() // Close dropdown

    await page.locator('.semi-select').filter({ hasText: '请选择盘点类型' }).click()
    await page.locator('.semi-select-option').filter({ hasText: '全盘' }).click()

    await page.locator('textarea').fill('Multi-warehouse stock taking')

    // Generate count sheets
    await page.locator('button').filter({ hasText: '保存' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('button').filter({ hasText: '生成盘点表' }).click()
    await page.waitForLoadState('networkidle')

    await page.locator('.semi-modal .semi-button').filter({ hasText: '确定' }).click()
    await page.waitForLoadState('networkidle')

    // Assert - Products from multiple warehouses included
    const countItems = page.locator('.count-item-row')
    const totalItems = await countItems.count()

    // Should have items from both warehouses
    expect(totalItems).toBeGreaterThan(10)

    // Verify warehouse column shows different warehouses
    const warehouseCells = page.locator('.warehouse-column')
    const warehouseNames = await warehouseCells.allTextContents()
    const uniqueWarehouses = new Set(warehouseNames)

    expect(uniqueWarehouses.size).toBeGreaterThanOrEqual(2)

    await page.screenshot({ path: `artifacts/multi-warehouse-${multiWarehouseNumber}.png` })
  })
})