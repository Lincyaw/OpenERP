import { type Page } from '@playwright/test'

/**
 * Wait for Semi Design Table to finish loading
 */
export async function waitForTableLoad(page: Page): Promise<void> {
  // Wait for loading spinner to disappear
  await page
    .locator('.semi-spin-spinning')
    .waitFor({ state: 'hidden', timeout: 30000 })
    .catch(() => {
      /* spinner might not appear */
    })

  // Wait for table body
  await page.locator('.semi-table-tbody').waitFor({ timeout: 30000 })
}

/**
 * Get table row count
 */
export async function getTableRowCount(page: Page): Promise<number> {
  await waitForTableLoad(page)
  return page.locator('.semi-table-tbody .semi-table-row').count()
}

/**
 * Click a specific table row
 */
export async function clickTableRow(page: Page, index: number): Promise<void> {
  await waitForTableLoad(page)
  await page.locator('.semi-table-tbody .semi-table-row').nth(index).click()
}

/**
 * Get table cell text by row and column index
 */
export async function getTableCellText(
  page: Page,
  rowIndex: number,
  colIndex: number
): Promise<string | null> {
  await waitForTableLoad(page)
  return page
    .locator('.semi-table-tbody .semi-table-row')
    .nth(rowIndex)
    .locator('.semi-table-row-cell')
    .nth(colIndex)
    .textContent()
}

/**
 * Click table action button (e.g., edit, delete)
 */
export async function clickTableAction(
  page: Page,
  rowIndex: number,
  actionText: string
): Promise<void> {
  await waitForTableLoad(page)
  await page
    .locator('.semi-table-tbody .semi-table-row')
    .nth(rowIndex)
    .locator(`button:has-text("${actionText}"), a:has-text("${actionText}")`)
    .click()
}

/**
 * Search in table
 */
export async function searchInTable(page: Page, searchText: string): Promise<void> {
  // Find search input (Semi Design Input)
  const searchInput = page
    .locator('input[placeholder*="搜索"], input[placeholder*="search"], .semi-input-wrapper input')
    .first()
  await searchInput.fill(searchText)

  // Trigger search (press Enter or wait for debounce)
  await searchInput.press('Enter')

  // Wait for table to reload
  await waitForTableLoad(page)
}

/**
 * Change table page
 */
export async function goToTablePage(page: Page, pageNumber: number): Promise<void> {
  await page.locator(`.semi-page-item:has-text("${pageNumber}")`).click()
  await waitForTableLoad(page)
}

/**
 * Change table page size
 */
export async function changeTablePageSize(page: Page, size: number): Promise<void> {
  // Click page size selector
  await page.locator('.semi-select:has(.semi-page-switch-select)').click()

  // Select size
  await page.locator(`.semi-select-option:has-text("${size}")`).click()

  await waitForTableLoad(page)
}
