/**
 * Supplier Module Integration Tests (P1-INT-002)
 *
 * These tests verify the frontend-backend integration for the Supplier module:
 * - Supplier list data display
 * - Supplier status tags and ratings
 * - Search and filter functionality
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import SuppliersPage from './Suppliers'
import * as suppliersApi from '@/api/suppliers/suppliers'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the suppliers API module
vi.mock('@/api/suppliers/suppliers', () => ({
  getSuppliers: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample supplier data matching backend response
const mockSuppliers = [
  {
    id: '550e8400-e29b-41d4-a716-446655440001',
    code: 'SUPP-001',
    name: '优质供应商有限公司',
    short_name: '优质供应',
    type: 'manufacturer' as const,
    phone: '13800138001',
    email: 'supplier1@example.com',
    province: '浙江省',
    city: '杭州市',
    rating: 4.5,
    payment_term_days: 30,
    status: 'active' as const,
    created_at: '2024-01-15T10:30:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440002',
    code: 'SUPP-002',
    name: '普通经销商',
    type: 'distributor' as const,
    phone: '13800138002',
    rating: 3.0,
    payment_term_days: 15,
    status: 'inactive' as const,
    created_at: '2024-01-14T09:00:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440003',
    code: 'SUPP-003',
    name: '问题供应商集团',
    short_name: '问题供应',
    type: 'retailer' as const,
    phone: '13800138003',
    email: 'blocked@example.com',
    province: '江苏省',
    city: '南京市',
    rating: 1.5,
    payment_term_days: 0,
    status: 'blocked' as const,
    created_at: '2024-01-13T08:00:00Z',
  },
]

// Mock API response helpers
const createMockListResponse = (suppliers = mockSuppliers, total = mockSuppliers.length) => ({
  success: true,
  data: suppliers,
  meta: {
    total,
    page: 1,
    page_size: 20,
    total_pages: Math.ceil(total / 20),
  },
})

describe('SuppliersPage', () => {
  let mockApi: {
    listSuppliers: ReturnType<typeof vi.fn>
    postPartnerSuppliersIdActivate: ReturnType<typeof vi.fn>
    postPartnerSuppliersIdDeactivate: ReturnType<typeof vi.fn>
    postPartnerSuppliersIdBlock: ReturnType<typeof vi.fn>
    deleteSupplier: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()

    // Setup mock API with default implementations
    mockApi = {
      listSuppliers: vi.fn().mockResolvedValue(createMockListResponse()),
      postPartnerSuppliersIdActivate: vi
        .fn()
        .mockResolvedValue({ success: true, data: mockSuppliers[1] }),
      postPartnerSuppliersIdDeactivate: vi
        .fn()
        .mockResolvedValue({ success: true, data: mockSuppliers[0] }),
      postPartnerSuppliersIdBlock: vi
        .fn()
        .mockResolvedValue({ success: true, data: mockSuppliers[0] }),
      deleteSupplier: vi.fn().mockResolvedValue({ success: true }),
    }

    vi.mocked(suppliersApi.getSuppliers).mockReturnValue(
      mockApi as unknown as ReturnType<typeof suppliersApi.getSuppliers>
    )
  })

  describe('Supplier List Display', () => {
    it('should display supplier list with correct data', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      // Wait for data to load
      await waitFor(() => {
        expect(mockApi.listSuppliers).toHaveBeenCalled()
      })

      // Verify supplier codes are displayed
      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
        expect(screen.getByText('SUPP-002')).toBeInTheDocument()
        expect(screen.getByText('SUPP-003')).toBeInTheDocument()
      })

      // Verify supplier names are displayed
      expect(screen.getByText('优质供应商有限公司')).toBeInTheDocument()
      expect(screen.getByText('普通经销商')).toBeInTheDocument()
      expect(screen.getByText('问题供应商集团')).toBeInTheDocument()
    })

    it('should display supplier status tags correctly', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      // Verify status tags
      expect(screen.getByText('启用')).toBeInTheDocument()
      expect(screen.getByText('停用')).toBeInTheDocument()
      expect(screen.getByText('拉黑')).toBeInTheDocument()
    })

    it('should display payment term days', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      // Verify payment term days are displayed
      expect(screen.getByText('30')).toBeInTheDocument()
      expect(screen.getByText('15')).toBeInTheDocument()
    })

    it('should display contact information', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      // Verify phone numbers are displayed
      expect(screen.getByText('13800138001')).toBeInTheDocument()
      expect(screen.getByText('13800138002')).toBeInTheDocument()
    })

    it('should display location information', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      // Verify location is displayed (province + city)
      expect(screen.getByText('浙江省 杭州市')).toBeInTheDocument()
      expect(screen.getByText('江苏省 南京市')).toBeInTheDocument()
    })

    it('should display page title', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      expect(screen.getByText('供应商管理')).toBeInTheDocument()
    })

    it('should call API with correct pagination parameters', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(mockApi.listSuppliers).toHaveBeenCalledWith(
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
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      // Verify search input exists
      const searchInput = screen.getByPlaceholderText('搜索供应商名称、编码、电话、邮箱...')
      expect(searchInput).toBeInTheDocument()
    })

    it('should have status filter dropdown', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      // Verify status filter exists - shows "全部状态" by default
      expect(screen.getByText('全部状态')).toBeInTheDocument()
    })

    it('should have type filter dropdown', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      // Verify type filter exists - shows "全部类型" by default
      expect(screen.getByText('全部类型')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when API fails', async () => {
      mockApi.listSuppliers.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取供应商列表失败')
      })
    })

    it('should handle empty supplier list gracefully', async () => {
      mockApi.listSuppliers.mockResolvedValueOnce(createMockListResponse([], 0))

      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(mockApi.listSuppliers).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('供应商管理')).toBeInTheDocument()
    })
  })

  describe('Supplier Actions', () => {
    it('should have "新增供应商" button', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      expect(screen.getByText('新增供应商')).toBeInTheDocument()
    })

    it('should have refresh button', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('SUPP-001')).toBeInTheDocument()
      })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })
  })

  describe('API Integration Verification', () => {
    it('should transform API response to display format correctly', async () => {
      // Use supplier with all fields to verify transformation
      const detailedSupplier = {
        id: '550e8400-e29b-41d4-a716-446655440099',
        code: 'DETAILED-001',
        name: '完整供应商信息有限公司',
        short_name: '完整供应',
        type: 'manufacturer' as const,
        phone: '13800138099',
        email: 'detailed@example.com',
        province: '广东省',
        city: '广州市',
        district: '天河区',
        address: '天河路100号',
        rating: 5.0,
        payment_term_days: 45,
        status: 'active' as const,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-15T12:00:00Z',
      }

      mockApi.listSuppliers.mockResolvedValueOnce(
        createMockListResponse([detailedSupplier], 1)
      )

      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('DETAILED-001')).toBeInTheDocument()
      })

      // Verify all displayed fields
      expect(screen.getByText('完整供应商信息有限公司')).toBeInTheDocument()
      expect(screen.getByText('13800138099')).toBeInTheDocument()
      expect(screen.getByText('45')).toBeInTheDocument()
      expect(screen.getByText('广东省 广州市')).toBeInTheDocument()
    })

    it('should handle missing optional fields gracefully', async () => {
      // Supplier with minimal required fields only
      const minimalSupplier = {
        id: '550e8400-e29b-41d4-a716-446655440088',
        code: 'MINIMAL-001',
        name: '最小供应商',
        type: 'distributor' as const,
        status: 'active' as const,
        created_at: '2024-06-15T12:00:00Z',
        // No phone, email, province, city, rating, payment_term_days
      }

      mockApi.listSuppliers.mockResolvedValueOnce(
        createMockListResponse([minimalSupplier], 1)
      )

      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(screen.getByText('MINIMAL-001')).toBeInTheDocument()
      })

      // Should display without errors
      expect(screen.getByText('最小供应商')).toBeInTheDocument()
    })

    it('should send correct request parameters', async () => {
      renderWithProviders(<SuppliersPage />, { route: '/partner/suppliers' })

      await waitFor(() => {
        expect(mockApi.listSuppliers).toHaveBeenCalled()
      })

      // Verify API was called with expected parameters
      expect(mockApi.listSuppliers).toHaveBeenCalledWith(
        expect.objectContaining({
          page: 1,
          page_size: 20,
          order_by: 'created_at',
          order_dir: 'desc',
        })
      )
    })
  })
})
