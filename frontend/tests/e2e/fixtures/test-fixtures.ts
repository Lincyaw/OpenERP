/* eslint-disable react-hooks/rules-of-hooks */
import { test as base } from '@playwright/test'
import {
  LoginPage,
  ProductsPage,
  CustomersPage,
  SuppliersPage,
  WarehousesPage,
  InventoryPage,
  SalesOrderPage,
  PurchaseOrderPage,
  FinancePage,
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
  suppliersPage: SuppliersPage
  warehousesPage: WarehousesPage
  inventoryPage: InventoryPage
  salesOrderPage: SalesOrderPage
  purchaseOrderPage: PurchaseOrderPage
  financePage: FinancePage
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
})

export { expect } from '@playwright/test'
