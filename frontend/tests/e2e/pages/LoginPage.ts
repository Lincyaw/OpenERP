import { type Page, expect } from '@playwright/test'
import { BasePage } from './BasePage'

/**
 * LoginPage - Page Object for login functionality
 *
 * Handles:
 * - Login form interactions
 * - Authentication verification
 * - Error message validation
 */
export class LoginPage extends BasePage {
  // Selectors
  private readonly usernameInput = 'input[name="username"], input[placeholder*="用户名"], #username'
  private readonly passwordInput =
    'input[name="password"], input[type="password"], input[placeholder*="密码"], #password'
  private readonly loginButton = 'button[type="submit"], .login-button, button:has-text("登录")'
  private readonly errorMessage = '.semi-form-field-error-message, .error-message, .login-error'
  private readonly userMenu = '.user-menu, .semi-avatar, [data-testid="user-menu"]'

  constructor(page: Page) {
    super(page)
  }

  /**
   * Navigate to login page
   */
  async navigate(): Promise<void> {
    await this.goto('/login')
    await this.waitForPageLoad()
  }

  /**
   * Fill username field
   */
  async fillUsername(username: string): Promise<void> {
    await this.page.fill(this.usernameInput, username)
  }

  /**
   * Fill password field
   */
  async fillPassword(password: string): Promise<void> {
    await this.page.fill(this.passwordInput, password)
  }

  /**
   * Click login button
   */
  async clickLogin(): Promise<void> {
    await this.page.click(this.loginButton)
  }

  /**
   * Perform complete login
   */
  async login(username: string, password: string): Promise<void> {
    await this.fillUsername(username)
    await this.fillPassword(password)
    await this.clickLogin()
  }

  /**
   * Login and wait for navigation
   */
  async loginAndWait(username: string, password: string): Promise<void> {
    await this.login(username, password)
    // Wait for either navigation to dashboard or error message
    await Promise.race([
      this.page.waitForURL('**/dashboard**', { timeout: 10000 }),
      this.page.waitForURL('**/', { timeout: 10000 }),
      this.page.locator(this.errorMessage).waitFor({ timeout: 10000 }),
    ]).catch(() => {
      // One of the conditions should have been met
    })
  }

  /**
   * Check if login was successful
   */
  async isLoggedIn(): Promise<boolean> {
    // Check if we're on a protected page (not login)
    const url = this.getUrl()
    if (url.includes('/login')) {
      return false
    }
    // Check for user menu presence
    return this.page
      .locator(this.userMenu)
      .isVisible()
      .catch(() => false)
  }

  /**
   * Get error message text
   */
  async getErrorMessage(): Promise<string | null> {
    const isVisible = await this.page.locator(this.errorMessage).isVisible()
    if (!isVisible) return null
    return this.page.locator(this.errorMessage).first().textContent()
  }

  /**
   * Assert successful login
   */
  async assertLoginSuccess(): Promise<void> {
    await expect(this.page).not.toHaveURL(/.*login.*/)
  }

  /**
   * Assert login failure with error
   */
  async assertLoginError(expectedError?: string): Promise<void> {
    await expect(this.page.locator(this.errorMessage)).toBeVisible()
    if (expectedError) {
      await expect(this.page.locator(this.errorMessage)).toContainText(expectedError)
    }
  }

  /**
   * Assert still on login page
   */
  async assertOnLoginPage(): Promise<void> {
    await expect(this.page).toHaveURL(/.*login.*/)
  }
}
