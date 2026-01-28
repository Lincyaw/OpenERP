/**
 * Customer Module Integration Tests (P1-INT-002)
 *
 * These tests verify the frontend-backend integration for the Customer module:
 * - Customer list data display
 * - Customer status tags and levels
 * - Search and filter functionality
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import CustomersPage from './Customers'
import * as customersApi from '@/api/customers/customers'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the customers API module
vi.mock('@/api/customers/customers', () => ({
  getCustomers: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample customer data matching backend response
const mockCustomers = [
  {
    id: '550e8400-e29b-41d4-a716-446655440001',
    code: 'CUST-001',
    name: '张三电子商务有限公司',
    short_name: '张三电商',
    type: 'organization' as const,
    phone: '13800138001',
    email: 'zhangsan@example.com',
    province: '广东省',
    city: '深圳市',
    level: 'gold' as const,
    status: 'active' as const,
    created_at: '2024-01-15T10:30:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440002',
    code: 'CUST-002',
    name: '李四',
    type: 'individual' as const,
    phone: '13800138002',
    level: 'normal' as const,
    status: 'inactive' as const,
    created_at: '2024-01-14T09:00:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440003',
    code: 'CUST-003',
    name: '王五科技集团',
    short_name: '王五科技',
    type: 'organization' as const,
    phone: '13800138003',
    email: 'wangwu@tech.com',
    province: '北京市',
    city: '朝阳区',
    level: 'vip' as const,
    status: 'suspended' as const,
    created_at: '2024-01-13T08:00:00Z',
  },
]

// Mock API response helpers
const createMockListResponse = (customers = mockCustomers, total = mockCustomers.length) => ({
  success: true,
  data: customers,
  meta: {
    total,
    page: 1,
    page_size: 20,
    total_pages: Math.ceil(total / 20),
  },
})

describe('CustomersPage', () => {
  let mockApi: {
    listCustomers: ReturnType<typeof vi.fn>
    activateCustomer: ReturnType<typeof vi.fn>
    deactivateCustomer: ReturnType<typeof vi.fn>
    postPartnerCustomersIdSuspend: ReturnType<typeof vi.fn>
    deleteCustomer: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()

    // Setup mock API with default implementations
    mockApi = {
      listCustomers: vi.fn().mockResolvedValue(createMockListResponse()),
      activateCustomer: vi
        .fn()
        .mockResolvedValue({ success: true, data: mockCustomers[1] }),
      deactivateCustomer: vi
        .fn()
        .mockResolvedValue({ success: true, data: mockCustomers[0] }),
      postPartnerCustomersIdSuspend: vi
        .fn()
        .mockResolvedValue({ success: true, data: mockCustomers[0] }),
      deleteCustomer: vi.fn().mockResolvedValue({ success: true }),
    }

    vi.mocked(customersApi.getCustomers).mockReturnValue(
      mockApi as unknown as ReturnType<typeof customersApi.getCustomers>
    )
  })

  describe('Customer List Display', () => {
    it('should display customer list with correct data', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      // Wait for data to load
      await waitFor(() => {
        expect(mockApi.listCustomers).toHaveBeenCalled()
      })

      // Verify customer codes are displayed
      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
        expect(screen.getByText('CUST-002')).toBeInTheDocument()
        expect(screen.getByText('CUST-003')).toBeInTheDocument()
      })

      // Verify customer names are displayed
      expect(screen.getByText('张三电子商务有限公司')).toBeInTheDocument()
      expect(screen.getByText('李四')).toBeInTheDocument()
      expect(screen.getByText('王五科技集团')).toBeInTheDocument()
    })

    it('should display customer status tags correctly', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify status tags
      expect(screen.getByText('启用')).toBeInTheDocument()
      expect(screen.getByText('停用')).toBeInTheDocument()
      expect(screen.getByText('暂停')).toBeInTheDocument()
    })

    it('should display customer level tags correctly', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify level tags
      expect(screen.getByText('黄金')).toBeInTheDocument()
      expect(screen.getByText('普通')).toBeInTheDocument()
      expect(screen.getByText('VIP')).toBeInTheDocument()
    })

    it('should display customer type tags correctly', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify type tags (企业 for organization, 个人 for individual)
      expect(screen.getAllByText('企业').length).toBeGreaterThan(0)
      expect(screen.getByText('个人')).toBeInTheDocument()
    })

    it('should display contact information', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify phone numbers are displayed
      expect(screen.getByText('13800138001')).toBeInTheDocument()
      expect(screen.getByText('13800138002')).toBeInTheDocument()
    })

    it('should display location information', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify location is displayed (province + city)
      expect(screen.getByText('广东省 深圳市')).toBeInTheDocument()
      expect(screen.getByText('北京市 朝阳区')).toBeInTheDocument()
    })

    it('should display page title', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      expect(screen.getByText('客户管理')).toBeInTheDocument()
    })

    it('should call API with correct pagination parameters', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(mockApi.listCustomers).toHaveBeenCalledWith(
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
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify search input exists
      const searchInput = screen.getByPlaceholderText('搜索客户名称、编码、电话、邮箱...')
      expect(searchInput).toBeInTheDocument()
    })

    it('should have status filter dropdown', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify status filter exists - shows "全部状态" by default
      expect(screen.getByText('全部状态')).toBeInTheDocument()
    })

    it('should have type filter dropdown', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify type filter exists - shows "全部类型" by default
      expect(screen.getByText('全部类型')).toBeInTheDocument()
    })

    it('should have level filter dropdown', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      // Verify level filter exists - shows "全部等级" by default
      expect(screen.getByText('全部等级')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when API fails', async () => {
      mockApi.listCustomers.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取客户列表失败')
      })
    })

    it('should handle empty customer list gracefully', async () => {
      mockApi.listCustomers.mockResolvedValueOnce(createMockListResponse([], 0))

      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(mockApi.listCustomers).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('客户管理')).toBeInTheDocument()
    })
  })

  describe('Customer Actions', () => {
    it('should have "新增客户" button', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      expect(screen.getByText('新增客户')).toBeInTheDocument()
    })

    it('should have refresh button', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('CUST-001')).toBeInTheDocument()
      })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })
  })

  describe('API Integration Verification', () => {
    it('should transform API response to display format correctly', async () => {
      // Use customer with all fields to verify transformation
      const detailedCustomer = {
        id: '550e8400-e29b-41d4-a716-446655440099',
        code: 'DETAILED-001',
        name: '完整客户信息有限公司',
        short_name: '完整客户',
        type: 'organization' as const,
        phone: '13800138099',
        email: 'detailed@example.com',
        province: '上海市',
        city: '浦东新区',
        district: '陆家嘴',
        address: '世纪大道100号',
        level: 'platinum' as const,
        status: 'active' as const,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-15T12:00:00Z',
      }

      mockApi.listCustomers.mockResolvedValueOnce(
        createMockListResponse([detailedCustomer], 1)
      )

      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('DETAILED-001')).toBeInTheDocument()
      })

      // Verify all displayed fields
      expect(screen.getByText('完整客户信息有限公司')).toBeInTheDocument()
      expect(screen.getByText('13800138099')).toBeInTheDocument()
      expect(screen.getByText('铂金')).toBeInTheDocument()
      expect(screen.getByText('上海市 浦东新区')).toBeInTheDocument()
    })

    it('should handle missing optional fields gracefully', async () => {
      // Customer with minimal required fields only
      const minimalCustomer = {
        id: '550e8400-e29b-41d4-a716-446655440088',
        code: 'MINIMAL-001',
        name: '最小客户',
        type: 'individual' as const,
        level: 'normal' as const,
        status: 'active' as const,
        created_at: '2024-06-15T12:00:00Z',
        // No phone, email, province, city
      }

      mockApi.listCustomers.mockResolvedValueOnce(
        createMockListResponse([minimalCustomer], 1)
      )

      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(screen.getByText('MINIMAL-001')).toBeInTheDocument()
      })

      // Should display without errors
      expect(screen.getByText('最小客户')).toBeInTheDocument()
    })

    it('should send correct request parameters', async () => {
      renderWithProviders(<CustomersPage />, { route: '/partner/customers' })

      await waitFor(() => {
        expect(mockApi.listCustomers).toHaveBeenCalled()
      })

      // Verify API was called with expected parameters
      expect(mockApi.listCustomers).toHaveBeenCalledWith(
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
