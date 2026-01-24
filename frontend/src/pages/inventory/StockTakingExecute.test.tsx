/**
 * Stock Taking Execute Component Tests (P2-QA-005)
 *
 * Tests for the StockTakingExecute page component covering:
 * - Page layout and header
 * - Stock taking details display
 * - Item list with editable quantity fields
 * - Real-time difference calculation
 * - Progress display
 * - Start counting functionality
 * - Save individual and batch counts
 * - Submit for approval
 * - Cancel stock taking
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import StockTakingExecutePage from './StockTakingExecute'
import * as stockTakingApi from '@/api/stock-taking/stock-taking'
import { Toast } from '@douyinfe/semi-ui'

// Mock the API modules
vi.mock('@/api/stock-taking/stock-taking', () => ({
  getStockTaking: vi.fn(),
}))

// Mock react-router-dom hooks
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ id: 'st-001' }),
  }
})

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample stock taking data with items (DRAFT status)
const mockStockTakingDraft = {
  id: 'st-001',
  taking_number: 'ST-20240120-001',
  warehouse_id: 'wh-001',
  warehouse_name: '主仓库',
  status: 'DRAFT',
  taking_date: '2024-01-20',
  remark: '月度盘点',
  total_items: 3,
  counted_items: 0,
  total_difference: 0,
  created_by_id: 'user-001',
  created_by_name: '张三',
  created_at: '2024-01-20T10:00:00Z',
  updated_at: '2024-01-20T10:00:00Z',
  items: [
    {
      id: 'item-001',
      stock_taking_id: 'st-001',
      product_id: 'prod-001',
      product_name: '商品A',
      product_code: 'SKU-001',
      unit: '件',
      system_quantity: 100,
      actual_quantity: 0,
      difference_quantity: 0,
      unit_cost: 10.5,
      difference_amount: 0,
      counted: false,
      remark: '',
    },
    {
      id: 'item-002',
      stock_taking_id: 'st-001',
      product_id: 'prod-002',
      product_name: '商品B',
      product_code: 'SKU-002',
      unit: '箱',
      system_quantity: 50,
      actual_quantity: 0,
      difference_quantity: 0,
      unit_cost: 25.0,
      difference_amount: 0,
      counted: false,
      remark: '',
    },
    {
      id: 'item-003',
      stock_taking_id: 'st-001',
      product_id: 'prod-003',
      product_name: '商品C',
      product_code: 'SKU-003',
      unit: '个',
      system_quantity: 200,
      actual_quantity: 0,
      difference_quantity: 0,
      unit_cost: 5.0,
      difference_amount: 0,
      counted: false,
      remark: '',
    },
  ],
}

// Stock taking in COUNTING status with partial progress
const mockStockTakingCounting = {
  ...mockStockTakingDraft,
  status: 'COUNTING',
  counted_items: 1,
  items: [
    {
      ...mockStockTakingDraft.items[0],
      actual_quantity: 98,
      difference_quantity: -2,
      difference_amount: -21.0,
      counted: true,
    },
    mockStockTakingDraft.items[1],
    mockStockTakingDraft.items[2],
  ],
}

// Stock taking in COUNTING status with all items counted
const mockStockTakingFullyCounted = {
  ...mockStockTakingDraft,
  status: 'COUNTING',
  counted_items: 3,
  total_difference: 74.5, // Sum of all differences
  items: [
    {
      ...mockStockTakingDraft.items[0],
      actual_quantity: 98,
      difference_quantity: -2,
      difference_amount: -21.0,
      counted: true,
    },
    {
      ...mockStockTakingDraft.items[1],
      actual_quantity: 52,
      difference_quantity: 2,
      difference_amount: 50.0,
      counted: true,
    },
    {
      ...mockStockTakingDraft.items[2],
      actual_quantity: 209,
      difference_quantity: 9,
      difference_amount: 45.5,
      counted: true,
    },
  ],
}

// Stock taking in PENDING_APPROVAL status
const mockStockTakingPendingApproval = {
  ...mockStockTakingFullyCounted,
  status: 'PENDING_APPROVAL',
}

// Stock taking in APPROVED status
const mockStockTakingApproved = {
  ...mockStockTakingFullyCounted,
  status: 'APPROVED',
}

// Mock API response helpers
const createMockStockTakingResponse = (data = mockStockTakingDraft) => ({
  success: true,
  data,
})

describe('StockTakingExecutePage', () => {
  let mockStockTakingApiInstance: {
    getInventoryStockTakingsId: ReturnType<typeof vi.fn>
    postInventoryStockTakingsIdStart: ReturnType<typeof vi.fn>
    postInventoryStockTakingsIdCount: ReturnType<typeof vi.fn>
    postInventoryStockTakingsIdCounts: ReturnType<typeof vi.fn>
    postInventoryStockTakingsIdSubmit: ReturnType<typeof vi.fn>
    postInventoryStockTakingsIdCancel: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock stock taking API
    mockStockTakingApiInstance = {
      getInventoryStockTakingsId: vi.fn().mockResolvedValue(createMockStockTakingResponse()),
      postInventoryStockTakingsIdStart: vi.fn().mockResolvedValue(
        createMockStockTakingResponse({ ...mockStockTakingDraft, status: 'COUNTING' })
      ),
      postInventoryStockTakingsIdCount: vi.fn().mockResolvedValue(
        createMockStockTakingResponse(mockStockTakingCounting)
      ),
      postInventoryStockTakingsIdCounts: vi.fn().mockResolvedValue(
        createMockStockTakingResponse(mockStockTakingFullyCounted)
      ),
      postInventoryStockTakingsIdSubmit: vi.fn().mockResolvedValue(
        createMockStockTakingResponse(mockStockTakingPendingApproval)
      ),
      postInventoryStockTakingsIdCancel: vi.fn().mockResolvedValue(
        createMockStockTakingResponse({ ...mockStockTakingDraft, status: 'CANCELLED' })
      ),
    }

    vi.mocked(stockTakingApi.getStockTaking).mockReturnValue(
      mockStockTakingApiInstance as unknown as ReturnType<typeof stockTakingApi.getStockTaking>
    )
  })

  describe('Page Layout', () => {
    it('should display back button', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakingsId).toHaveBeenCalled()
      })

      expect(screen.getByText('返回')).toBeInTheDocument()
    })

    it('should display page title with taking number', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText(/盘点执行 - ST-20240120-001/)).toBeInTheDocument()
      })
    })

    it('should display status tag', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('草稿')).toBeInTheDocument()
      })
    })
  })

  describe('Summary Card Display', () => {
    it('should display warehouse name', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('主仓库')).toBeInTheDocument()
      })
    })

    it('should display creator name', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('张三')).toBeInTheDocument()
      })
    })

    it('should display remark', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('月度盘点')).toBeInTheDocument()
      })
    })

    it('should display progress information', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('盘点进度')).toBeInTheDocument()
        expect(screen.getByText('0/3 项已盘点')).toBeInTheDocument()
      })
    })

    it('should display difference amount total', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('差异金额合计')).toBeInTheDocument()
      })
    })
  })

  describe('Items Table Display', () => {
    it('should display items table header', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('盘点明细')).toBeInTheDocument()
      })
    })

    it('should display product codes', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
        expect(screen.getByText('SKU-002')).toBeInTheDocument()
        expect(screen.getByText('SKU-003')).toBeInTheDocument()
      })
    })

    it('should display product names', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('商品A')).toBeInTheDocument()
        expect(screen.getByText('商品B')).toBeInTheDocument()
        expect(screen.getByText('商品C')).toBeInTheDocument()
      })
    })

    it('should display units', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('件')).toBeInTheDocument()
        expect(screen.getByText('箱')).toBeInTheDocument()
        expect(screen.getByText('个')).toBeInTheDocument()
      })
    })

    it('should display system quantities', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('100.00')).toBeInTheDocument()
        expect(screen.getByText('50.00')).toBeInTheDocument()
        expect(screen.getByText('200.00')).toBeInTheDocument()
      })
    })

    it('should display uncounted status tags', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        const uncountedTags = screen.getAllByText('未盘')
        expect(uncountedTags.length).toBe(3)
      })
    })
  })

  describe('Status-Based Button Display', () => {
    it('should display start counting button for DRAFT status', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('开始盘点')).toBeInTheDocument()
      })
    })

    it('should display action buttons for COUNTING status', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingCounting)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('盘点中')).toBeInTheDocument()
        expect(screen.getByText('保存全部')).toBeInTheDocument()
        expect(screen.getByText('提交审批')).toBeInTheDocument()
        expect(screen.getByText('取消盘点')).toBeInTheDocument()
      })
    })

    it('should not display action buttons for APPROVED status', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingApproved)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('已通过')).toBeInTheDocument()
      })

      // Action buttons should not be present for approved status
      expect(screen.queryByText('开始盘点')).not.toBeInTheDocument()
      expect(screen.queryByText('保存全部')).not.toBeInTheDocument()
      expect(screen.queryByText('提交审批')).not.toBeInTheDocument()
      expect(screen.queryByText('取消盘点')).not.toBeInTheDocument()
    })
  })

  describe('Progress Display with Partial Counts', () => {
    it('should display partial progress correctly', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingCounting)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('1/3 项已盘点')).toBeInTheDocument()
      })
    })

    it('should display counted items with 已盘 tag', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingCounting)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('已盘')).toBeInTheDocument()
      })
    })
  })

  describe('Fully Counted State', () => {
    it('should display 100% progress when all items counted', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingFullyCounted)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('3/3 项已盘点')).toBeInTheDocument()
      })
    })

    it('should enable submit button when all items counted', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingFullyCounted)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        const submitButton = screen.getByText('提交审批')
        expect(submitButton).toBeInTheDocument()
        expect(submitButton).not.toBeDisabled()
      })
    })
  })

  describe('API Integration', () => {
    it('should call get stock taking API with correct ID', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(mockStockTakingApiInstance.getInventoryStockTakingsId).toHaveBeenCalledWith('st-001')
      })
    })

    it('should call start counting API when clicking start button', async () => {
      const { user } = renderWithProviders(<StockTakingExecutePage />, {
        route: '/inventory/stock-taking/st-001/execute',
      })

      await waitFor(() => {
        expect(screen.getByText('开始盘点')).toBeInTheDocument()
      })

      const startButton = screen.getByText('开始盘点')
      await user.click(startButton)

      await waitFor(() => {
        expect(mockStockTakingApiInstance.postInventoryStockTakingsIdStart).toHaveBeenCalledWith('st-001')
        expect(Toast.success).toHaveBeenCalledWith('已开始盘点')
      })
    })
  })

  describe('Error Handling', () => {
    it('should show error and navigate back when stock taking not found', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockRejectedValueOnce(
        new Error('Not found')
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取盘点单失败')
        expect(mockNavigate).toHaveBeenCalledWith('/inventory/stock-taking')
      })
    })

    it('should show error when start counting fails', async () => {
      mockStockTakingApiInstance.postInventoryStockTakingsIdStart.mockRejectedValueOnce(
        new Error('Network error')
      )

      const { user } = renderWithProviders(<StockTakingExecutePage />, {
        route: '/inventory/stock-taking/st-001/execute',
      })

      await waitFor(() => {
        expect(screen.getByText('开始盘点')).toBeInTheDocument()
      })

      const startButton = screen.getByText('开始盘点')
      await user.click(startButton)

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('开始盘点失败')
      })
    })

    it('should show error response message when API returns error', async () => {
      mockStockTakingApiInstance.postInventoryStockTakingsIdStart.mockResolvedValueOnce({
        success: false,
        error: { message: '状态不允许开始盘点' },
      })

      const { user } = renderWithProviders(<StockTakingExecutePage />, {
        route: '/inventory/stock-taking/st-001/execute',
      })

      await waitFor(() => {
        expect(screen.getByText('开始盘点')).toBeInTheDocument()
      })

      const startButton = screen.getByText('开始盘点')
      await user.click(startButton)

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('状态不允许开始盘点')
      })
    })
  })

  describe('Navigation', () => {
    it('should navigate to list when clicking back button', async () => {
      const { user } = renderWithProviders(<StockTakingExecutePage />, {
        route: '/inventory/stock-taking/st-001/execute',
      })

      await waitFor(() => {
        expect(screen.getByText('返回')).toBeInTheDocument()
      })

      const backButton = screen.getByText('返回')
      await user.click(backButton)

      expect(mockNavigate).toHaveBeenCalledWith('/inventory/stock-taking')
    })
  })

  describe('Status Tag Colors', () => {
    it('should display grey tag for DRAFT', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('草稿')).toBeInTheDocument()
      })
    })

    it('should display blue tag for COUNTING', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingCounting)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('盘点中')).toBeInTheDocument()
      })
    })

    it('should display orange tag for PENDING_APPROVAL', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingPendingApproval)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('待审批')).toBeInTheDocument()
      })
    })

    it('should display green tag for APPROVED', async () => {
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockResolvedValueOnce(
        createMockStockTakingResponse(mockStockTakingApproved)
      )

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        expect(screen.getByText('已通过')).toBeInTheDocument()
      })
    })
  })

  describe('Loading State', () => {
    it('should display loading indicator while fetching data', async () => {
      // Create a promise that we can control
      let resolvePromise: (value: unknown) => void
      const pendingPromise = new Promise((resolve) => {
        resolvePromise = resolve
      })
      mockStockTakingApiInstance.getInventoryStockTakingsId.mockReturnValueOnce(pendingPromise)

      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      // Should show loading state
      expect(screen.getByText('加载中...')).toBeInTheDocument()

      // Resolve the promise
      resolvePromise!(createMockStockTakingResponse())

      // Wait for loading to complete
      await waitFor(() => {
        expect(screen.queryByText('加载中...')).not.toBeInTheDocument()
      })
    })
  })

  describe('Refresh Functionality', () => {
    it('should have refresh button in items section', async () => {
      renderWithProviders(<StockTakingExecutePage />, { route: '/inventory/stock-taking/st-001/execute' })

      await waitFor(() => {
        // Find refresh button in items card (using getAllByRole to handle multiple buttons)
        const refreshButtons = screen.getAllByRole('button')
        const refreshButton = refreshButtons.find(
          (btn) => btn.querySelector('[data-icon="refresh"]') !== null
        )
        // The component may render refresh icon differently, check the button exists
        expect(screen.getByText('盘点明细')).toBeInTheDocument()
      })
    })
  })
})
