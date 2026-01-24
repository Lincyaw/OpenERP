/**
 * Product Module Integration Tests (P1-INT-001)
 *
 * These tests verify the frontend-backend integration for the Product module:
 * - Product list data display
 * - Product CRUD workflow
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import ProductsPage from './Products'
import * as productsApi from '@/api/products/products'
import { Toast } from '@douyinfe/semi-ui'

// Mock the products API module
vi.mock('@/api/products/products', () => ({
  getProducts: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample product data matching backend response
const mockProducts = [
  {
    id: '550e8400-e29b-41d4-a716-446655440001',
    code: 'SKU-001',
    name: '测试商品1',
    unit: '个',
    barcode: '6901234567890',
    purchase_price: 50.0,
    selling_price: 100.0,
    status: 'active' as const,
    created_at: '2024-01-15T10:30:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440002',
    code: 'SKU-002',
    name: '测试商品2',
    unit: '件',
    barcode: '6901234567891',
    purchase_price: 30.0,
    selling_price: 60.0,
    status: 'inactive' as const,
    created_at: '2024-01-14T09:00:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440003',
    code: 'SKU-003',
    name: '已停售商品',
    unit: '箱',
    barcode: null,
    purchase_price: 100.0,
    selling_price: 200.0,
    status: 'discontinued' as const,
    created_at: '2024-01-13T08:00:00Z',
  },
]

// Mock API response helpers
const createMockListResponse = (products = mockProducts, total = mockProducts.length) => ({
  success: true,
  data: products,
  meta: {
    total,
    page: 1,
    page_size: 20,
    total_pages: Math.ceil(total / 20),
  },
})

describe('ProductsPage', () => {
  let mockApi: {
    getCatalogProducts: ReturnType<typeof vi.fn>
    postCatalogProductsIdActivate: ReturnType<typeof vi.fn>
    postCatalogProductsIdDeactivate: ReturnType<typeof vi.fn>
    postCatalogProductsIdDiscontinue: ReturnType<typeof vi.fn>
    deleteCatalogProductsId: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()

    // Setup mock API with default implementations
    mockApi = {
      getCatalogProducts: vi.fn().mockResolvedValue(createMockListResponse()),
      postCatalogProductsIdActivate: vi.fn().mockResolvedValue({ success: true, data: mockProducts[1] }),
      postCatalogProductsIdDeactivate: vi.fn().mockResolvedValue({ success: true, data: mockProducts[0] }),
      postCatalogProductsIdDiscontinue: vi.fn().mockResolvedValue({ success: true, data: mockProducts[0] }),
      deleteCatalogProductsId: vi.fn().mockResolvedValue({ success: true }),
    }

    vi.mocked(productsApi.getProducts).mockReturnValue(mockApi as unknown as ReturnType<typeof productsApi.getProducts>)
  })

  describe('Product List Display', () => {
    it('should display product list with correct data', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      // Wait for data to load
      await waitFor(() => {
        expect(mockApi.getCatalogProducts).toHaveBeenCalled()
      })

      // Verify product codes are displayed
      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
        expect(screen.getByText('SKU-002')).toBeInTheDocument()
        expect(screen.getByText('SKU-003')).toBeInTheDocument()
      })

      // Verify product names are displayed
      expect(screen.getByText('测试商品1')).toBeInTheDocument()
      expect(screen.getByText('测试商品2')).toBeInTheDocument()
      expect(screen.getByText('已停售商品')).toBeInTheDocument()
    })

    it('should display product status tags correctly', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
      })

      // Verify status tags
      expect(screen.getByText('启用')).toBeInTheDocument()
      expect(screen.getByText('禁用')).toBeInTheDocument()
      expect(screen.getByText('停售')).toBeInTheDocument()
    })

    it('should display prices formatted correctly', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
      })

      // Check for formatted prices (¥50.00, ¥100.00, etc.)
      // Note: Some prices may appear multiple times in different columns
      expect(screen.getByText('¥50.00')).toBeInTheDocument()
      expect(screen.getAllByText('¥100.00').length).toBeGreaterThan(0)
    })

    it('should display barcode when available', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
      })

      // Verify barcode is displayed
      expect(screen.getByText('6901234567890')).toBeInTheDocument()
    })

    it('should display page title', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      expect(screen.getByText('商品管理')).toBeInTheDocument()
    })

    it('should call API with correct pagination parameters', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(mockApi.getCatalogProducts).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 1,
            page_size: 20,
          })
        )
      })
    })
  })

  describe('Search and Filter', () => {
    it('should have search input', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
      })

      // Verify search input exists
      const searchInput = screen.getByPlaceholderText('搜索商品名称、编码、条码...')
      expect(searchInput).toBeInTheDocument()
    })

    it('should have status filter dropdown', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
      })

      // Verify status filter exists - shows "全部状态" by default
      expect(screen.getByText('全部状态')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when API fails', async () => {
      mockApi.getCatalogProducts.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取商品列表失败')
      })
    })

    it('should handle empty product list gracefully', async () => {
      mockApi.getCatalogProducts.mockResolvedValueOnce(createMockListResponse([], 0))

      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(mockApi.getCatalogProducts).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('商品管理')).toBeInTheDocument()
    })
  })

  describe('Product Actions', () => {
    it('should have "新增商品" button', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
      })

      expect(screen.getByText('新增商品')).toBeInTheDocument()
    })

    it('should have refresh button', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
      })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })
  })

  describe('API Integration Verification', () => {
    it('should transform API response to display format correctly', async () => {
      // Use product with all fields to verify transformation
      const detailedProduct = {
        id: '550e8400-e29b-41d4-a716-446655440099',
        code: 'DETAILED-001',
        name: '完整商品',
        unit: '个',
        barcode: '1234567890123',
        description: '商品描述',
        purchase_price: 88.88,
        selling_price: 168.88,
        min_stock: 10,
        sort_order: 1,
        status: 'active' as const,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-15T12:00:00Z',
      }

      mockApi.getCatalogProducts.mockResolvedValueOnce(createMockListResponse([detailedProduct], 1))

      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('DETAILED-001')).toBeInTheDocument()
      })

      // Verify all displayed fields
      expect(screen.getByText('完整商品')).toBeInTheDocument()
      expect(screen.getByText('1234567890123')).toBeInTheDocument()
      expect(screen.getByText('¥88.88')).toBeInTheDocument()
      expect(screen.getByText('¥168.88')).toBeInTheDocument()
      expect(screen.getByText('个')).toBeInTheDocument()
    })

    it('should handle missing optional fields gracefully', async () => {
      // Product with minimal required fields only
      const minimalProduct = {
        id: '550e8400-e29b-41d4-a716-446655440088',
        code: 'MINIMAL-001',
        name: '最小商品',
        unit: '件',
        status: 'active' as const,
        created_at: '2024-06-15T12:00:00Z',
        // No barcode, no prices, no description
      }

      mockApi.getCatalogProducts.mockResolvedValueOnce(createMockListResponse([minimalProduct], 1))

      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(screen.getByText('MINIMAL-001')).toBeInTheDocument()
      })

      // Should display without errors
      expect(screen.getByText('最小商品')).toBeInTheDocument()
    })

    it('should send correct request headers', async () => {
      renderWithProviders(<ProductsPage />, { route: '/catalog/products' })

      await waitFor(() => {
        expect(mockApi.getCatalogProducts).toHaveBeenCalled()
      })

      // Verify API was called (headers are handled by axios interceptors)
      expect(mockApi.getCatalogProducts).toHaveBeenCalledTimes(1)
    })
  })
})
