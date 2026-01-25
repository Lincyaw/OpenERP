import { type Locator, type Page, expect } from '@playwright/test'

/**
 * BasePage - Base class for all Page Object classes
 *
 * Provides common functionality for all page objects:
 * - Navigation
 * - Common element interactions
 * - Wait utilities
 * - Screenshot capture
 */
export class BasePage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  /**
   * Navigate to a specific path
   */
  async goto(path: string): Promise<void> {
    await this.page.goto(path)
  }

  /**
   * Wait for page to be fully loaded
   */
  async waitForPageLoad(): Promise<void> {
    await this.page.waitForLoadState('networkidle')
  }

  /**
   * Get page title
   */
  async getTitle(): Promise<string> {
    return this.page.title()
  }

  /**
   * Get current URL
   */
  getUrl(): string {
    return this.page.url()
  }

  /**
   * Wait for URL to contain a specific string
   */
  async waitForUrlContains(urlPart: string): Promise<void> {
    await this.page.waitForURL(`**/${urlPart}**`)
  }

  /**
   * Wait for an element to be visible
   */
  async waitForElement(selector: string): Promise<Locator> {
    const element = this.page.locator(selector)
    await element.waitFor({ state: 'visible' })
    return element
  }

  /**
   * Click on an element
   */
  async click(selector: string): Promise<void> {
    await this.page.click(selector)
  }

  /**
   * Fill an input field
   */
  async fill(selector: string, value: string): Promise<void> {
    await this.page.fill(selector, value)
  }

  /**
   * Get text content of an element
   */
  async getText(selector: string): Promise<string | null> {
    return this.page.textContent(selector)
  }

  /**
   * Check if an element is visible
   */
  async isVisible(selector: string): Promise<boolean> {
    return this.page.locator(selector).isVisible()
  }

  /**
   * Take a screenshot
   */
  async screenshot(name: string): Promise<void> {
    await this.page.screenshot({ path: `test-results/screenshots/${name}.png` })
  }

  /**
   * Wait for API response
   */
  async waitForApiResponse(urlPattern: string | RegExp): Promise<void> {
    await this.page.waitForResponse(urlPattern)
  }

  /**
   * Check if page has toast/notification with specific text
   */
  async hasToast(text: string): Promise<boolean> {
    // Semi Design Toast selector
    const toast = this.page.locator('.semi-toast-content').filter({ hasText: text })
    return toast.isVisible()
  }

  /**
   * Wait for Semi Design Toast to appear
   */
  async waitForToast(text: string): Promise<void> {
    await this.page.locator('.semi-toast-content').filter({ hasText: text }).waitFor()
  }

  /**
   * Wait for Semi Design Modal to appear
   */
  async waitForModal(): Promise<void> {
    await this.page.locator('.semi-modal').waitFor()
  }

  /**
   * Close Semi Design Modal
   */
  async closeModal(): Promise<void> {
    await this.page.locator('.semi-modal-close').click()
  }

  /**
   * Confirm Semi Design Modal
   */
  async confirmModal(): Promise<void> {
    await this.page.locator('.semi-modal-footer .semi-button-primary').click()
  }

  /**
   * Cancel Semi Design Modal
   */
  async cancelModal(): Promise<void> {
    await this.page.locator('.semi-modal-footer .semi-button:not(.semi-button-primary)').click()
  }

  /**
   * Wait for table to load (Semi Design Table)
   */
  async waitForTableLoad(): Promise<void> {
    // Wait for loading spinner to disappear
    await this.page
      .locator('.semi-spin-spinning')
      .waitFor({ state: 'hidden', timeout: 30000 })
      .catch(() => {
        /* spinner might not appear */
      })
    // Wait for table body to have content
    await this.page.locator('.semi-table-tbody').waitFor()
  }

  /**
   * Get table row count
   */
  async getTableRowCount(): Promise<number> {
    const rows = this.page.locator('.semi-table-tbody .semi-table-row')
    return rows.count()
  }

  /**
   * Click table row by index (0-based)
   */
  async clickTableRow(index: number): Promise<void> {
    await this.page.locator(`.semi-table-tbody .semi-table-row`).nth(index).click()
  }

  /**
   * Assert element contains text
   */
  async assertContainsText(selector: string, text: string): Promise<void> {
    await expect(this.page.locator(selector)).toContainText(text)
  }

  /**
   * Assert element is visible
   */
  async assertVisible(selector: string): Promise<void> {
    await expect(this.page.locator(selector)).toBeVisible()
  }

  /**
   * Assert element is hidden
   */
  async assertHidden(selector: string): Promise<void> {
    await expect(this.page.locator(selector)).toBeHidden()
  }

  /**
   * Assert current URL
   */
  async assertUrl(expectedUrl: string): Promise<void> {
    await expect(this.page).toHaveURL(expectedUrl)
  }

  /**
   * Assert URL contains
   */
  async assertUrlContains(urlPart: string): Promise<void> {
    await expect(this.page).toHaveURL(new RegExp(urlPart))
  }
}
