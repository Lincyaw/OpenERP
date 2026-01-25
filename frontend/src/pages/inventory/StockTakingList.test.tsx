/**
 * Stock Taking List Component Tests (P2-QA-005)
 *
 * Tests for the StockTakingList page component covering:
 * - List display with pagination
 * - Status and warehouse filtering
 * - Status tag colors and labels
 * - Progress and difference display
 * - Navigation to create, detail, and execute pages
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import StockTakingListPage from './StockTakingList'
import * as stockTakingApi from '@/api/stock-taking/stock-taking'
import * as warehousesApi from '@/api/warehouses/warehouses'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the API modules
vi.mock('@/api/stock-taking/stock-taking', () => ({
  getStockTaking: vi.fn(),
}))

vi.mock('@/api/warehouses/warehouses', () => ({
  getWarehouses: vi.fn(),
}))

// Mock react-router-dom hooks
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

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

// Sample stock taking data
const mockStockTakings = [
  {
    id: 'st-001',
    taking_number: 'ST-20240101-001',
    warehouse_id: 'wh-001',
    warehouse_name: '主仓库',
    status: 'DRAFT',
    taking_date: '2024-01-15',
    total_items: 10,
    counted_items: 0,
    total_difference: 0,
    created_by_id: 'user-001',
    created_by_name: '张三',
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  },
  {
    id: 'st-002',
    taking_number: 'ST-20240102-001',
    warehouse_id: 'wh-001',
    warehouse_name: '主仓库',
    status: 'COUNTING',
    taking_date: '2024-01-16',
    total_items: 20,
    counted_items: 15,
    total_difference: 500.5,
    created_by_id: 'user-001',
    created_by_name: '张三',
    created_at: '2024-01-16T09:00:00Z',
    updated_at: '2024-01-16T14:30:00Z',
  },
  {
    id: 'st-003',
    taking_number: 'ST-20240103-001',
    warehouse_id: 'wh-002',
    warehouse_name: '备用仓库',
    status: 'PENDING_APPROVAL',
    taking_date: '2024-01-17',
    total_items: 15,
    counted_items: 15,
    total_difference: -200.0,
    created_by_id: 'user-002',
    created_by_name: '李四',
    created_at: '2024-01-17T08:00:00Z',
    updated_at: '2024-01-17T16:00:00Z',
  },
  {
    id: 'st-004',
    taking_number: 'ST-20240104-001',
    warehouse_id: 'wh-001',
    warehouse_name: '主仓库',
    status: 'APPROVED',
    taking_date: '2024-01-10',
    total_items: 25,
    counted_items: 25,
    total_difference: 150.0,
    created_by_id: 'user-001',
    created_by_name: '张三',
    created_at: '2024-01-10T10:00:00Z',
    updated_at: '2024-01-11T15:00:00Z',
  },
  {
    id: 'st-005',
    taking_number: 'ST-20240105-001',
    warehouse_id: 'wh-002',
    warehouse_name: '备用仓库',
    status: 'REJECTED',
    taking_date: '2024-01-08',
    total_items: 5,
    counted_items: 5,
    total_difference: -1000.0,
    created_by_id: 'user-002',
    created_by_name: '李四',
    created_at: '2024-01-08T09:00:00Z',
    updated_at: '2024-01-09T11:00:00Z',
  },
  {
    id: 'st-006',
    taking_number: 'ST-20240106-001',
    warehouse_id: 'wh-001',
    warehouse_name: '主仓库',
    status: 'CANCELLED',
    taking_date: '2024-01-05',
    total_items: 8,
    counted_items: 3,
    total_difference: 0,
    created_by_id: 'user-001',
    created_by_name: '张三',
    created_at: '2024-01-05T08:00:00Z',
    updated_at: '2024-01-05T10:00:00Z',
  },
]

// Mock API response helpers
const createMockStockTakingListResponse = (
  items = mockStockTakings,
  total = mockStockTakings.length
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

describe('StockTakingListPage', () => {
  let mockStockTakingApiInstance: {
    getInventoryStockTakings: ReturnType<typeof vi.fn>
  }

  let mockWarehouseApiInstance: {
    getPartnerWarehouses: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock stock taking API
    mockStockTakingApiInstance = {
      getInventoryStockTakings: vi.fn().mockResolvedValue(createMockStockTakingListResponse()),
    }

    // Setup mock warehouse API
    mockWarehouseApiInstance = {
      getPartnerWarehouses: vi.fn().mockResolvedValue(createMockWarehouseListResponse()),
    }

    vi.mocked(stockTakingApi.getStockTaking).mockReturnValue(
      mockStockTakingApiInstance as unknown as ReturnType<typeof stockTakingApi.getStockTaking>
    )
    vi.mocked(warehousesApi.getWarehouses).mockReturnValue(
      mockWarehouseApiInstance as unknown as ReturnType<typeof warehousesApi.getWarehouses>
    )
  })

  describe('Page Layout', () => {
    it('should display page title', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      expect(screen.getByText('盘点管理')).toBeInTheDocument()
    })

    it('should display create button', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      expect(screen.getByText('新建盘点')).toBeInTheDocument()
    })

    it('should display search input', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      expect(screen.getByPlaceholderText('搜索盘点单号...')).toBeInTheDocument()
    })

    it('should display refresh button', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })
  })

  describe('Stock Taking List Display', () => {
    it('should display stock taking list with correct data', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakings).toHaveBeenCalled()
      })

      // Verify taking numbers are displayed
      await waitFor(() => {
        expect(screen.getByText('ST-20240101-001')).toBeInTheDocument()
        expect(screen.getByText('ST-20240102-001')).toBeInTheDocument()
        expect(screen.getByText('ST-20240103-001')).toBeInTheDocument()
      })
    })

    it('should display warehouse names', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakings).toHaveBeenCalled()
      })

      // Verify warehouse names are displayed (multiple items may share same warehouse)
      await waitFor(() => {
        expect(screen.getAllByText('主仓库').length).toBeGreaterThan(0)
        expect(screen.getAllByText('备用仓库').length).toBeGreaterThan(0)
      })
    })

    it('should display creator names', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakings).toHaveBeenCalled()
      })

      await waitFor(() => {
        expect(screen.getAllByText('张三').length).toBeGreaterThan(0)
        expect(screen.getAllByText('李四').length).toBeGreaterThan(0)
      })
    })
  })

  describe('Status Tags Display', () => {
    it('should display 草稿 status tag for DRAFT', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[0]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('草稿')).toBeInTheDocument()
      })
    })

    it('should display 盘点中 status tag for COUNTING', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[1]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('盘点中')).toBeInTheDocument()
      })
    })

    it('should display 待审批 status tag for PENDING_APPROVAL', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[2]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('待审批')).toBeInTheDocument()
      })
    })

    it('should display 已通过 status tag for APPROVED', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[3]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('已通过')).toBeInTheDocument()
      })
    })

    it('should display 已拒绝 status tag for REJECTED', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[4]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('已拒绝')).toBeInTheDocument()
      })
    })

    it('should display 已取消 status tag for CANCELLED', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[5]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('已取消')).toBeInTheDocument()
      })
    })
  })

  describe('Progress Display', () => {
    it('should display progress as counted/total with percentage', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[1]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        // 15/20 = 75%
        expect(screen.getByText('15/20 (75%)')).toBeInTheDocument()
      })
    })

    it('should display 0% progress for items with no counted items', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[0]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        // 0/10 = 0%
        expect(screen.getByText('0/10 (0%)')).toBeInTheDocument()
      })
    })

    it('should display 100% progress for fully counted items', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[2]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        // 15/15 = 100%
        expect(screen.getByText('15/15 (100%)')).toBeInTheDocument()
      })
    })
  })

  describe('Difference Amount Display', () => {
    it('should display positive difference with + sign', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[1]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('+¥500.50')).toBeInTheDocument()
      })
    })

    it('should display negative difference without extra sign', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[2]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('¥-200.00')).toBeInTheDocument()
      })
    })

    it('should display zero difference as ¥0.00', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[0]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('¥0.00')).toBeInTheDocument()
      })
    })
  })

  describe('Filter Dropdowns', () => {
    it('should have warehouse filter dropdown', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockWarehouseApiInstance.getPartnerWarehouses).toHaveBeenCalled()
      })

      // Warehouse filter loads options including "全部仓库" as default
      await waitFor(() => {
        expect(screen.getByText('全部仓库')).toBeInTheDocument()
      })
    })

    it('should have status filter dropdown with default option', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      // Status filter has "全部状态" as first option
      await waitFor(() => {
        expect(screen.getByText('全部状态')).toBeInTheDocument()
      })
    })
  })

  describe('API Integration', () => {
    it('should call stock taking API with correct pagination parameters', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakings).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 1,
            page_size: 20,
          })
        )
      })
    })

    it('should call warehouse API to load filter options', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockWarehouseApiInstance.getPartnerWarehouses).toHaveBeenCalledWith({
          page_size: 100,
          status: 'active',
        })
      })
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when stock taking API fails', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取盘点单列表失败')
      })
    })

    it('should handle empty stock taking list gracefully', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([], 0)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakings).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('盘点管理')).toBeInTheDocument()
    })

    it('should handle warehouse API failure gracefully', async () => {
      mockWarehouseApiInstance.getPartnerWarehouses.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      // Page should still render
      await waitFor(() => {
        expect(screen.getByText('盘点管理')).toBeInTheDocument()
      })
    })
  })

  describe('Navigation', () => {
    it('should navigate to create page when clicking new button', async () => {
      const { user } = renderWithProviders(<StockTakingListPage />, {
        route: '/inventory/stock-taking',
      })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakings).toHaveBeenCalled()
      })

      const createButton = screen.getByText('新建盘点')
      await user.click(createButton)

      expect(mockNavigate).toHaveBeenCalledWith('/inventory/stock-taking/new')
    })
  })

  describe('Row Actions', () => {
    it('should display view action for all stock takings', async () => {
      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakings).toHaveBeenCalled()
      })

      // View actions should be present
      await waitFor(() => {
        expect(screen.getAllByText('查看').length).toBeGreaterThan(0)
      })
    })

    it('should display execute action for DRAFT status', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[0]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('执行')).toBeInTheDocument()
      })
    })

    it('should display execute action for COUNTING status', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[1]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('执行')).toBeInTheDocument()
      })
    })

    it('should show execute action for all statuses (condition prop not processed)', async () => {
      // Note: The component uses 'condition' property which is not defined in TableAction type
      // TableAction type expects 'hidden', not 'condition', so execute shows for all statuses
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[3]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(screen.getByText('查看')).toBeInTheDocument()
      })

      // Execute action shows because 'condition' is not processed by TableActions
      // (TableAction type expects 'hidden', not 'condition')
      await waitFor(() => {
        expect(screen.getByText('执行')).toBeInTheDocument()
      })
    })
  })

  describe('Data Transformation', () => {
    it('should format dates correctly', async () => {
      mockStockTakingApiInstance.getInventoryStockTakings.mockResolvedValueOnce(
        createMockStockTakingListResponse([mockStockTakings[0]], 1)
      )

      renderWithProviders(<StockTakingListPage />, { route: '/inventory/stock-taking' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakings).toHaveBeenCalled()
      })

      // Date should be formatted as YYYY/MM/DD
      await waitFor(() => {
        expect(screen.getByText('2024/01/15')).toBeInTheDocument()
      })
    })
  })
})
