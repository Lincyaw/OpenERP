/* eslint-disable react-hooks/rules-of-hooks */
import { test as base } from '@playwright/test'
import {
  LoginPage,
  ProductsPage,
  CustomersPage,
  CustomerBalancePage,
  SuppliersPage,
  WarehousesPage,
  InventoryPage,
  SalesOrderPage,
  PurchaseOrderPage,
  FinancePage,
  SalesReturnPage,
} from '../pages'

/**
 * Test Users - Seed data users for testing
 * These credentials match the users in docker/seed-data.sql
 * Password: admin123 (bcrypt hash in migration/seed)
 */
export const TEST_USERS = {
  admin: {
    username: 'admin',
    password: 'admin123',
    role: 'System Administrator',
    // Has all permissions (ADMIN role)
  },
  sales: {
    username: 'sales',
    password: 'admin123',
    role: 'Sales Manager',
    // Has sales, customer, product:read, inventory:read permissions
  },
  warehouse: {
    username: 'warehouse',
    password: 'admin123',
    role: 'Warehouse Manager',
    // Has inventory, warehouse, product:read permissions
  },
  finance: {
    username: 'finance',
    password: 'admin123',
    role: 'Finance Manager',
    // Has finance (receivable, payable, expense, income) permissions
  },
} as const

export type TestUserType = keyof typeof TEST_USERS

/**
 * Extended test fixtures
 */
export const test = base.extend<{
  loginPage: LoginPage
  authenticatedPage: LoginPage
  productsPage: ProductsPage
  customersPage: CustomersPage
  customerBalancePage: CustomerBalancePage
  suppliersPage: SuppliersPage
  warehousesPage: WarehousesPage
  inventoryPage: InventoryPage
  salesOrderPage: SalesOrderPage
  purchaseOrderPage: PurchaseOrderPage
  financePage: FinancePage
  salesReturnPage: SalesReturnPage
}>({
  /**
   * Login page fixture - provides a fresh LoginPage instance
   */
  loginPage: async ({ page }, use) => {
    const loginPage = new LoginPage(page)
    await use(loginPage)
  },

  /**
   * Authenticated page fixture - ensures user is logged in before test
   * If storage state is already authenticated, skips login
   */
  authenticatedPage: async ({ page }, use) => {
    const loginPage = new LoginPage(page)

    // Check if already authenticated via storage state
    await page.goto('/')
    await page.waitForLoadState('domcontentloaded')

    // Give time for any redirects to complete
    await page.waitForTimeout(500)

    const currentUrl = page.url()
    const isOnLogin = currentUrl.includes('/login')

    if (isOnLogin) {
      // Not authenticated, need to login
      await loginPage.login(TEST_USERS.admin.username, TEST_USERS.admin.password)

      // Wait for navigation away from login page
      await page
        .waitForFunction(() => !window.location.pathname.includes('/login'), { timeout: 15000 })
        .catch(() => {
          // Navigation might have failed - continue to check auth state
        })

      // Wait for auth state to be persisted
      await page.waitForFunction(
        () => {
          const user = window.localStorage.getItem('user')
          const erpAuth = window.localStorage.getItem('erp-auth')
          if (!user || !erpAuth) return false
          try {
            const parsed = JSON.parse(erpAuth)
            return parsed?.state?.user !== null && parsed?.state?.user !== undefined
          } catch {
            return false
          }
        },
        { timeout: 10000 }
      )
    }

    await use(loginPage)
  },

  /**
   * Products page fixture - provides a ProductsPage instance
   */
  productsPage: async ({ page }, use) => {
    const productsPage = new ProductsPage(page)
    await use(productsPage)
  },

  /**
   * Customers page fixture - provides a CustomersPage instance
   */
  customersPage: async ({ page }, use) => {
    const customersPage = new CustomersPage(page)
    await use(customersPage)
  },

  /**
   * Customer Balance page fixture - provides a CustomerBalancePage instance
   */
  customerBalancePage: async ({ page }, use) => {
    const customerBalancePage = new CustomerBalancePage(page)
    await use(customerBalancePage)
  },

  /**
   * Suppliers page fixture - provides a SuppliersPage instance
   */
  suppliersPage: async ({ page }, use) => {
    const suppliersPage = new SuppliersPage(page)
    await use(suppliersPage)
  },

  /**
   * Warehouses page fixture - provides a WarehousesPage instance
   */
  warehousesPage: async ({ page }, use) => {
    const warehousesPage = new WarehousesPage(page)
    await use(warehousesPage)
  },

  /**
   * Inventory page fixture - provides an InventoryPage instance
   */
  inventoryPage: async ({ page }, use) => {
    const inventoryPage = new InventoryPage(page)
    await use(inventoryPage)
  },

  /**
   * Sales order page fixture - provides a SalesOrderPage instance
   */
  salesOrderPage: async ({ page }, use) => {
    const salesOrderPage = new SalesOrderPage(page)
    await use(salesOrderPage)
  },

  /**
   * Purchase order page fixture - provides a PurchaseOrderPage instance
   */
  purchaseOrderPage: async ({ page }, use) => {
    const purchaseOrderPage = new PurchaseOrderPage(page)
    await use(purchaseOrderPage)
  },

  /**
   * Finance page fixture - provides a FinancePage instance
   */
  financePage: async ({ page }, use) => {
    const financePage = new FinancePage(page)
    await use(financePage)
  },

  /**
   * Sales return page fixture - provides a SalesReturnPage instance
   */
  salesReturnPage: async ({ page }, use) => {
    const salesReturnPage = new SalesReturnPage(page)
    await use(salesReturnPage)
  },
})

export { expect } from '@playwright/test'
