/**
 * Warehouse Module Integration Tests (P1-INT-002)
 *
 * These tests verify the frontend-backend integration for the Warehouse module:
 * - Warehouse list data display
 * - Warehouse status and type tags
 * - Default warehouse indicator
 * - Search and filter functionality
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import WarehousesPage from './Warehouses'
import * as warehousesApi from '@/api/warehouses/warehouses'
import { Toast } from '@douyinfe/semi-ui'

// Mock the warehouses API module
vi.mock('@/api/warehouses/warehouses', () => ({
  getWarehouses: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample warehouse data matching backend response
// Note: Using distinct names to avoid collision with type labels (普通仓库/虚拟仓库/中转仓库)
const mockWarehouses = [
  {
    id: '550e8400-e29b-41d4-a716-446655440001',
    code: 'WH-001',
    name: '主仓库',
    short_name: '主仓',
    type: 'normal' as const,
    province: '广东省',
    city: '深圳市',
    sort_order: 1,
    is_default: true,
    status: 'enabled' as const,
    created_at: '2024-01-15T10:30:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440002',
    code: 'WH-002',
    name: '线上仓',
    type: 'virtual' as const,
    province: '上海市',
    city: '浦东新区',
    sort_order: 2,
    is_default: false,
    status: 'enabled' as const,
    created_at: '2024-01-14T09:00:00Z',
  },
  {
    id: '550e8400-e29b-41d4-a716-446655440003',
    code: 'WH-003',
    name: '周转仓',
    short_name: '周转',
    type: 'transit' as const,
    province: '北京市',
    city: '朝阳区',
    sort_order: 3,
    is_default: false,
    status: 'disabled' as const,
    created_at: '2024-01-13T08:00:00Z',
  },
]

// Mock API response helpers
const createMockListResponse = (warehouses = mockWarehouses, total = mockWarehouses.length) => ({
  success: true,
  data: warehouses,
  meta: {
    total,
    page: 1,
    page_size: 20,
    total_pages: Math.ceil(total / 20),
  },
})

describe('WarehousesPage', () => {
  let mockApi: {
    getPartnerWarehouses: ReturnType<typeof vi.fn>
    postPartnerWarehousesIdEnable: ReturnType<typeof vi.fn>
    postPartnerWarehousesIdDisable: ReturnType<typeof vi.fn>
    postPartnerWarehousesIdSetDefault: ReturnType<typeof vi.fn>
    deletePartnerWarehousesId: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()

    // Setup mock API with default implementations
    mockApi = {
      getPartnerWarehouses: vi.fn().mockResolvedValue(createMockListResponse()),
      postPartnerWarehousesIdEnable: vi.fn().mockResolvedValue({ success: true, data: mockWarehouses[2] }),
      postPartnerWarehousesIdDisable: vi.fn().mockResolvedValue({ success: true, data: mockWarehouses[1] }),
      postPartnerWarehousesIdSetDefault: vi.fn().mockResolvedValue({ success: true, data: mockWarehouses[1] }),
      deletePartnerWarehousesId: vi.fn().mockResolvedValue({ success: true }),
    }

    vi.mocked(warehousesApi.getWarehouses).mockReturnValue(mockApi as unknown as ReturnType<typeof warehousesApi.getWarehouses>)
  })

  describe('Warehouse List Display', () => {
    it('should display warehouse list with correct data', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      // Wait for data to load
      await waitFor(() => {
        expect(mockApi.getPartnerWarehouses).toHaveBeenCalled()
      })

      // Verify warehouse codes are displayed
      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
        expect(screen.getByText('WH-002')).toBeInTheDocument()
        expect(screen.getByText('WH-003')).toBeInTheDocument()
      })

      // Verify warehouse names are displayed
      expect(screen.getByText('主仓库')).toBeInTheDocument()
      expect(screen.getByText('线上仓')).toBeInTheDocument()
      expect(screen.getByText('周转仓')).toBeInTheDocument()
    })

    it('should display warehouse status tags correctly', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      // Verify status tags - enabled shows as "启用", disabled shows as "停用"
      expect(screen.getAllByText('启用').length).toBeGreaterThan(0)
      expect(screen.getByText('停用')).toBeInTheDocument()
    })

    it('should display warehouse type tags correctly', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      // Verify type tags
      expect(screen.getByText('普通仓库')).toBeInTheDocument()
      expect(screen.getByText('虚拟仓库')).toBeInTheDocument()
      expect(screen.getByText('中转仓库')).toBeInTheDocument()
    })

    it('should display default warehouse indicator', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      // Verify default indicator is shown
      expect(screen.getByText('默认')).toBeInTheDocument()
    })

    it('should display sort order', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      // Verify sort order values are displayed (may have multiple "1" for page number)
      // Use querySelectorAll for sort order column
      const sortOrders = screen.getAllByText('1')
      expect(sortOrders.length).toBeGreaterThan(0)
      expect(screen.getByText('2')).toBeInTheDocument()
      expect(screen.getByText('3')).toBeInTheDocument()
    })

    it('should display location information', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      // Verify location is displayed (province + city)
      expect(screen.getByText('广东省 深圳市')).toBeInTheDocument()
      expect(screen.getByText('上海市 浦东新区')).toBeInTheDocument()
      expect(screen.getByText('北京市 朝阳区')).toBeInTheDocument()
    })

    it('should display page title', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      expect(screen.getByText('仓库管理')).toBeInTheDocument()
    })

    it('should call API with correct pagination parameters', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(mockApi.getPartnerWarehouses).toHaveBeenCalledWith(
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
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      // Verify search input exists
      const searchInput = screen.getByPlaceholderText('搜索仓库名称、编码...')
      expect(searchInput).toBeInTheDocument()
    })

    it('should have status filter dropdown', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      // Verify status filter exists - shows "全部状态" by default
      expect(screen.getByText('全部状态')).toBeInTheDocument()
    })

    it('should have type filter dropdown', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      // Verify type filter exists - shows "全部类型" by default
      expect(screen.getByText('全部类型')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when API fails', async () => {
      mockApi.getPartnerWarehouses.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取仓库列表失败')
      })
    })

    it('should handle empty warehouse list gracefully', async () => {
      mockApi.getPartnerWarehouses.mockResolvedValueOnce(createMockListResponse([], 0))

      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(mockApi.getPartnerWarehouses).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('仓库管理')).toBeInTheDocument()
    })
  })

  describe('Warehouse Actions', () => {
    it('should have "新增仓库" button', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      expect(screen.getByText('新增仓库')).toBeInTheDocument()
    })

    it('should have refresh button', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('WH-001')).toBeInTheDocument()
      })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })
  })

  describe('API Integration Verification', () => {
    it('should transform API response to display format correctly', async () => {
      // Use warehouse with all fields to verify transformation
      const detailedWarehouse = {
        id: '550e8400-e29b-41d4-a716-446655440099',
        code: 'DETAILED-001',
        name: '完整仓库信息',
        short_name: '完整仓',
        type: 'normal' as const,
        province: '浙江省',
        city: '杭州市',
        district: '西湖区',
        address: '文三路100号',
        sort_order: 10,
        is_default: false,
        status: 'enabled' as const,
        created_at: '2024-06-15T12:00:00Z',
        updated_at: '2024-06-15T12:00:00Z',
      }

      mockApi.getPartnerWarehouses.mockResolvedValueOnce(createMockListResponse([detailedWarehouse], 1))

      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('DETAILED-001')).toBeInTheDocument()
      })

      // Verify all displayed fields
      expect(screen.getByText('完整仓库信息')).toBeInTheDocument()
      expect(screen.getByText('普通仓库')).toBeInTheDocument()
      expect(screen.getByText('10')).toBeInTheDocument()
      expect(screen.getByText('浙江省 杭州市')).toBeInTheDocument()
    })

    it('should handle missing optional fields gracefully', async () => {
      // Warehouse with minimal required fields only
      const minimalWarehouse = {
        id: '550e8400-e29b-41d4-a716-446655440088',
        code: 'MINIMAL-001',
        name: '最小仓库',
        type: 'normal' as const,
        sort_order: 0,
        is_default: false,
        status: 'enabled' as const,
        created_at: '2024-06-15T12:00:00Z',
        // No province, city, short_name
      }

      mockApi.getPartnerWarehouses.mockResolvedValueOnce(createMockListResponse([minimalWarehouse], 1))

      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(screen.getByText('MINIMAL-001')).toBeInTheDocument()
      })

      // Should display without errors
      expect(screen.getByText('最小仓库')).toBeInTheDocument()
    })

    it('should send correct request parameters', async () => {
      renderWithProviders(<WarehousesPage />, { route: '/partner/warehouses' })

      await waitFor(() => {
        expect(mockApi.getPartnerWarehouses).toHaveBeenCalled()
      })

      // Verify API was called with expected parameters
      // Note: Warehouses default sort by sort_order ascending
      expect(mockApi.getPartnerWarehouses).toHaveBeenCalledWith(
        expect.objectContaining({
          page: 1,
          page_size: 20,
          order_by: 'sort_order',
          order_dir: 'asc',
        })
      )
    })
  })
})
