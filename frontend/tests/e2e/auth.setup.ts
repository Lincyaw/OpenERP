import { test as setup } from '@playwright/test'
import { login, saveAuthState } from './utils/auth'

/**
 * Authentication Setup
 *
 * This setup test runs before all other tests to:
 * 1. Log in as the default admin user
 * 2. Save the authentication state for reuse
 *
 * Other tests will use the saved state to skip login
 */
setup('authenticate', async ({ page, context }) => {
  // Login as admin user
  await login(page, 'admin')

  // Save authentication state to file
  await saveAuthState(context, 'tests/e2e/.auth/user.json')
})
