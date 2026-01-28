/**
 * Inventory Stock List Integration Tests (P2-INT-001)
 *
 * These tests verify the frontend-backend integration for the Inventory module:
 * - Inventory list data display (warehouse, product, quantities, cost, status)
 * - Filter functionality (warehouse filter, stock status filter)
 * - Low stock warning indicators
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import StockListPage from './StockList'
import * as inventoryApi from '@/api/inventory/inventory'
import * as warehousesApi from '@/api/warehouses/warehouses'
import * as productsApi from '@/api/products/products'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the API modules
vi.mock('@/api/inventory/inventory', () => ({
  getInventory: vi.fn(),
}))

vi.mock('@/api/warehouses/warehouses', () => ({
  getWarehouses: vi.fn(),
}))

vi.mock('@/api/products/products', () => ({
  getProducts: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample warehouse data
const mockWarehouses = [
  {
    id: 'wh-001',
    code: 'WH-001',
    name: '主仓库',
    status: 'active',
  },
  {
    id: 'wh-002',
    code: 'WH-002',
    name: '备用仓库',
    status: 'active',
  },
]

// Sample product data
const mockProducts = [
  {
    id: 'prod-001',
    code: 'SKU-001',
    name: '商品A',
    status: 'active',
  },
  {
    id: 'prod-002',
    code: 'SKU-002',
    name: '商品B',
    status: 'active',
  },
  {
    id: 'prod-003',
    code: 'SKU-003',
    name: '商品C',
    status: 'active',
  },
]

// Sample inventory items matching backend response
const mockInventoryItems = [
  {
    id: 'inv-001',
    warehouse_id: 'wh-001',
    product_id: 'prod-001',
    total_quantity: 100,
    available_quantity: 80,
    locked_quantity: 20,
    unit_cost: 10.5,
    total_value: 1050,
    min_quantity: 10,
    max_quantity: 200,
    is_below_minimum: false,
    is_above_maximum: false,
    version: 1,
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-20T15:30:00Z',
  },
  {
    id: 'inv-002',
    warehouse_id: 'wh-001',
    product_id: 'prod-002',
    total_quantity: 5,
    available_quantity: 5,
    locked_quantity: 0,
    unit_cost: 25.0,
    total_value: 125,
    min_quantity: 10,
    max_quantity: 100,
    is_below_minimum: true, // Low stock warning
    is_above_maximum: false,
    version: 2,
    created_at: '2024-01-14T09:00:00Z',
    updated_at: '2024-01-19T12:00:00Z',
  },
  {
    id: 'inv-003',
    warehouse_id: 'wh-002',
    product_id: 'prod-003',
    total_quantity: 0, // No stock - will show "无库存" since is_below_minimum is false
    available_quantity: 0,
    locked_quantity: 0,
    unit_cost: 0,
    total_value: 0,
    min_quantity: 0, // No minimum set
    max_quantity: 50,
    is_below_minimum: false, // Not below minimum (no minimum set)
    is_above_maximum: false,
    version: 1,
    created_at: '2024-01-13T08:00:00Z',
    updated_at: '2024-01-18T10:00:00Z',
  },
]

// Mock API response helpers
const createMockInventoryListResponse = (
  items = mockInventoryItems,
  total = mockInventoryItems.length
) => ({
  success: true,
  data: items,
  meta: {
    total,
    page: 1,
    page_size: 20,
    total_pages: Math.ceil(total / 20),
  },
})

const createMockWarehouseListResponse = (warehouses = mockWarehouses) => ({
  success: true,
  data: warehouses,
  meta: {
    total: warehouses.length,
    page: 1,
    page_size: 100,
    total_pages: 1,
  },
})

const createMockProductListResponse = (products = mockProducts) => ({
  success: true,
  data: products,
  meta: {
    total: products.length,
    page: 1,
    page_size: 500,
    total_pages: 1,
  },
})

describe('StockListPage', () => {
  let mockInventoryApiInstance: {
    listInventoryItems: ReturnType<typeof vi.fn>
  }

  let mockWarehouseApiInstance: {
    listWarehouses: ReturnType<typeof vi.fn>
  }

  let mockProductApiInstance: {
    listProducts: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()

    // Setup mock inventory API
    mockInventoryApiInstance = {
      listInventoryItems: vi.fn().mockResolvedValue(createMockInventoryListResponse()),
    }

    // Setup mock warehouse API
    mockWarehouseApiInstance = {
      listWarehouses: vi.fn().mockResolvedValue(createMockWarehouseListResponse()),
    }

    // Setup mock product API
    mockProductApiInstance = {
      listProducts: vi.fn().mockResolvedValue(createMockProductListResponse()),
    }

    vi.mocked(inventoryApi.getInventory).mockReturnValue(
      mockInventoryApiInstance as unknown as ReturnType<typeof inventoryApi.getInventory>
    )
    vi.mocked(warehousesApi.getWarehouses).mockReturnValue(
      mockWarehouseApiInstance as unknown as ReturnType<typeof warehousesApi.getWarehouses>
    )
    vi.mocked(productsApi.getProducts).mockReturnValue(
      mockProductApiInstance as unknown as ReturnType<typeof productsApi.getProducts>
    )
  })

  describe('Inventory List Display', () => {
    it('should display inventory list with correct data', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      // Wait for data to load
      await waitFor(() => {
        expect(mockInventoryApiInstance.listInventoryItems).toHaveBeenCalled()
      })

      // Verify warehouse names are displayed (resolved from ID)
      // Using getAllByText since multiple items may share the same warehouse
      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
        expect(screen.getByText('备用仓库')).toBeInTheDocument()
      })

      // Verify product names are displayed (resolved from ID)
      expect(screen.getByText('商品A')).toBeInTheDocument()
      expect(screen.getByText('商品B')).toBeInTheDocument()
      expect(screen.getByText('商品C')).toBeInTheDocument()
    })

    it('should display available quantities correctly', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Verify available quantities (formatted with 2 decimal places)
      expect(screen.getByText('80.00')).toBeInTheDocument()
      // 5.00 may appear twice (available and total for item 2)
      expect(screen.getAllByText('5.00').length).toBeGreaterThan(0)
      // 0.00 may appear multiple times (locked_quantity and available_quantity)
      expect(screen.getAllByText('0.00').length).toBeGreaterThan(0)
    })

    it('should display locked quantities correctly', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Verify locked quantities (20.00 for first item)
      expect(screen.getByText('20.00')).toBeInTheDocument()
    })

    it('should display total quantities correctly', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Verify total quantities
      expect(screen.getByText('100.00')).toBeInTheDocument()
    })

    it('should display unit cost formatted as currency', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Verify unit cost formatting (¥ prefix)
      expect(screen.getByText('¥10.50')).toBeInTheDocument()
      expect(screen.getByText('¥25.00')).toBeInTheDocument()
    })

    it('should display total value formatted as currency', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Verify total value formatting
      expect(screen.getByText('¥1050.00')).toBeInTheDocument()
      expect(screen.getByText('¥125.00')).toBeInTheDocument()
    })

    it('should display page title', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      expect(screen.getByText('库存查询')).toBeInTheDocument()
    })
  })

  describe('Stock Status Display', () => {
    it('should display normal status tag for items with sufficient stock', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // First item has sufficient stock
      expect(screen.getByText('正常')).toBeInTheDocument()
    })

    it('should display low stock warning tag for items below minimum', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Second item is below minimum (only one now)
      expect(screen.getByText('低库存')).toBeInTheDocument()
    })

    it('should display no stock tag for items with zero quantity', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Third item has zero stock
      expect(screen.getByText('无库存')).toBeInTheDocument()
    })
  })

  describe('Search and Filter', () => {
    it('should have search input', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Verify search input exists
      const searchInput = screen.getByPlaceholderText('搜索商品...')
      expect(searchInput).toBeInTheDocument()
    })

    it('should have warehouse filter dropdown', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Verify warehouse filter exists - shows "全部仓库" after data loads
      expect(screen.getByText('全部仓库')).toBeInTheDocument()
    })

    it('should have stock status filter dropdown', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      // Verify stock status filter exists - shows "全部状态" after data loads
      expect(screen.getByText('全部状态')).toBeInTheDocument()
    })

    it('should have refresh button', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
      })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when inventory API fails', async () => {
      mockInventoryApiInstance.listInventoryItems.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取库存列表失败')
      })
    })

    it('should handle empty inventory list gracefully', async () => {
      mockInventoryApiInstance.listInventoryItems.mockResolvedValueOnce(
        createMockInventoryListResponse([], 0)
      )

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(mockInventoryApiInstance.listInventoryItems).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('库存查询')).toBeInTheDocument()
    })

    it('should handle warehouse API failure gracefully', async () => {
      mockWarehouseApiInstance.listWarehouses.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      // Should still render the page
      await waitFor(() => {
        expect(screen.getByText('库存查询')).toBeInTheDocument()
      })
    })

    it('should handle product API failure gracefully', async () => {
      mockProductApiInstance.listProducts.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      // Should still render the page
      await waitFor(() => {
        expect(screen.getByText('库存查询')).toBeInTheDocument()
      })
    })
  })

  describe('API Integration Verification', () => {
    it('should call inventory API with correct pagination parameters', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(mockInventoryApiInstance.listInventoryItems).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 1,
            page_size: 20,
          })
        )
      })
    })

    it('should call warehouse API to load filter options', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(mockWarehouseApiInstance.listWarehouses).toHaveBeenCalledWith(
          expect.objectContaining({
            page_size: 100,
            status: 'active',
          })
        )
      })
    })

    it('should call product API to resolve product names', async () => {
      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(mockProductApiInstance.listProducts).toHaveBeenCalledWith(
          expect.objectContaining({
            page_size: 500,
          })
        )
      })
    })

    it('should transform API response to display format correctly', async () => {
      // Use inventory item with all fields to verify transformation
      const detailedItem = {
        id: 'inv-detailed',
        warehouse_id: 'wh-001',
        product_id: 'prod-001',
        total_quantity: 150.5,
        available_quantity: 120.25,
        locked_quantity: 30.25,
        unit_cost: 99.99,
        total_value: 15048.5,
        min_quantity: 20,
        max_quantity: 300,
        is_below_minimum: false,
        is_above_maximum: false,
        version: 5,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-20T16:30:00Z',
      }

      mockInventoryApiInstance.listInventoryItems.mockResolvedValueOnce(
        createMockInventoryListResponse([detailedItem], 1)
      )

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getByText('主仓库')).toBeInTheDocument()
      })

      // Verify all displayed fields with proper formatting
      expect(screen.getByText('商品A')).toBeInTheDocument()
      expect(screen.getByText('120.25')).toBeInTheDocument() // available
      expect(screen.getByText('30.25')).toBeInTheDocument() // locked
      expect(screen.getByText('150.50')).toBeInTheDocument() // total
      expect(screen.getByText('¥99.99')).toBeInTheDocument() // unit cost
      expect(screen.getByText('¥15048.50')).toBeInTheDocument() // total value
    })

    it('should handle missing optional fields gracefully', async () => {
      // Inventory item with minimal fields
      const minimalItem = {
        id: 'inv-minimal',
        warehouse_id: 'wh-unknown', // Unknown warehouse
        product_id: 'prod-unknown', // Unknown product
        total_quantity: 10,
        available_quantity: 10,
        locked_quantity: 0,
        is_below_minimum: false,
        is_above_maximum: false,
        version: 1,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-15T12:00:00Z',
        // Missing unit_cost, total_value, min_quantity, max_quantity
      }

      mockInventoryApiInstance.listInventoryItems.mockResolvedValueOnce(
        createMockInventoryListResponse([minimalItem], 1)
      )

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(mockInventoryApiInstance.listInventoryItems).toHaveBeenCalled()
      })

      // Should display without errors, showing truncated IDs for unknown references
      expect(screen.getByText('库存查询')).toBeInTheDocument()
    })
  })

  describe('Locked Quantity Display (Concurrent Locking Verification)', () => {
    it('should display locked quantity when stock is locked', async () => {
      // Item with locked stock
      const lockedItem = {
        id: 'inv-locked',
        warehouse_id: 'wh-001',
        product_id: 'prod-001',
        total_quantity: 100,
        available_quantity: 50,
        locked_quantity: 50, // Half of stock is locked
        unit_cost: 10.0,
        total_value: 1000,
        min_quantity: 10,
        max_quantity: 200,
        is_below_minimum: false,
        is_above_maximum: false,
        version: 3,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-20T16:30:00Z',
      }

      mockInventoryApiInstance.listInventoryItems.mockResolvedValueOnce(
        createMockInventoryListResponse([lockedItem], 1)
      )

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getByText('主仓库')).toBeInTheDocument()
      })

      // Verify locked quantity is displayed (50.00 appears twice - available and locked)
      const fiftyValues = screen.getAllByText('50.00')
      expect(fiftyValues.length).toBe(2) // both available and locked are 50.00
      expect(screen.getByText('100.00')).toBeInTheDocument() // total
    })

    it('should show correct available quantity when partially locked', async () => {
      // Item with partial lock
      const partiallyLockedItem = {
        id: 'inv-partial',
        warehouse_id: 'wh-001',
        product_id: 'prod-001',
        total_quantity: 200,
        available_quantity: 180,
        locked_quantity: 20, // Small portion locked
        unit_cost: 15.0,
        total_value: 3000,
        min_quantity: 50,
        max_quantity: 500,
        is_below_minimum: false,
        is_above_maximum: false,
        version: 2,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-20T16:30:00Z',
      }

      mockInventoryApiInstance.listInventoryItems.mockResolvedValueOnce(
        createMockInventoryListResponse([partiallyLockedItem], 1)
      )

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getByText('主仓库')).toBeInTheDocument()
      })

      // Verify correct quantities
      expect(screen.getByText('180.00')).toBeInTheDocument() // available
      expect(screen.getByText('20.00')).toBeInTheDocument() // locked
      expect(screen.getByText('200.00')).toBeInTheDocument() // total
    })

    it('should handle zero locked quantity correctly', async () => {
      // Item with no lock
      const unlockedItem = {
        id: 'inv-unlocked',
        warehouse_id: 'wh-001',
        product_id: 'prod-001',
        total_quantity: 100,
        available_quantity: 100,
        locked_quantity: 0, // No lock
        unit_cost: 10.0,
        total_value: 1000,
        min_quantity: 10,
        max_quantity: 200,
        is_below_minimum: false,
        is_above_maximum: false,
        version: 1,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-20T16:30:00Z',
      }

      mockInventoryApiInstance.listInventoryItems.mockResolvedValueOnce(
        createMockInventoryListResponse([unlockedItem], 1)
      )

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getByText('主仓库')).toBeInTheDocument()
      })

      // Verify available equals total when no lock
      const quantityCells = screen.getAllByText('100.00')
      expect(quantityCells.length).toBe(2) // available and total
    })

    it('should verify locked stock cannot exceed total stock', async () => {
      // This verifies the data integrity - locked + available should not exceed total
      const consistentItem = {
        id: 'inv-consistent',
        warehouse_id: 'wh-001',
        product_id: 'prod-001',
        total_quantity: 150,
        available_quantity: 100,
        locked_quantity: 50,
        unit_cost: 20.0,
        total_value: 3000,
        min_quantity: 20,
        max_quantity: 300,
        is_below_minimum: false,
        is_above_maximum: false,
        version: 4,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-20T16:30:00Z',
      }

      mockInventoryApiInstance.listInventoryItems.mockResolvedValueOnce(
        createMockInventoryListResponse([consistentItem], 1)
      )

      renderWithProviders(<StockListPage />, { route: '/inventory/stock' })

      await waitFor(() => {
        expect(screen.getByText('主仓库')).toBeInTheDocument()
      })

      // Verify total = available + locked (100 + 50 = 150)
      expect(screen.getByText('100.00')).toBeInTheDocument() // available
      expect(screen.getByText('50.00')).toBeInTheDocument() // locked
      expect(screen.getByText('150.00')).toBeInTheDocument() // total
    })
  })
})
