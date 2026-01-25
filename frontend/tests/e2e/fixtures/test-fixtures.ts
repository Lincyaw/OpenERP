import { test as base } from '@playwright/test'
import { LoginPage, ProductsPage, CustomersPage, SuppliersPage, WarehousesPage } from '../pages'

/**
 * Test Users - Seed data users for testing
 * These credentials match the users in docker/seed-data.sql
 */
export const TEST_USERS = {
  admin: {
    username: 'admin',
    password: 'test123',
    role: 'System Administrator',
  },
  sales: {
    username: 'sales',
    password: 'test123',
    role: 'Sales Manager',
  },
  warehouse: {
    username: 'warehouse',
    password: 'test123',
    role: 'Warehouse Manager',
  },
  finance: {
    username: 'finance',
    password: 'test123',
    role: 'Finance Manager',
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
  suppliersPage: SuppliersPage
  warehousesPage: WarehousesPage
}>({
  /**
   * Login page fixture - provides a fresh LoginPage instance
   */
  loginPage: async ({ page }, use) => {
    const loginPage = new LoginPage(page)
    await use(loginPage)
  },

  /**
   * Authenticated page fixture - logs in as admin before test
   */
  authenticatedPage: async ({ page }, use) => {
    const loginPage = new LoginPage(page)
    await loginPage.navigate()
    await loginPage.loginAndWait(TEST_USERS.admin.username, TEST_USERS.admin.password)
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
})

export { expect } from '@playwright/test'
