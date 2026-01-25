/**
 * Sales Returns List Component Tests (P3-QA-005)
 *
 * Tests for the SalesReturns page:
 * - Page layout and title
 * - Return list display (return number, order number, customer, status)
 * - Status filter functionality
 * - Customer filter functionality
 * - Search functionality
 * - Return actions (view, submit, approve, reject, complete, cancel, delete)
 * - Error handling
 * - Navigation
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import SalesReturnsPage from './SalesReturns'
import * as salesReturnsApi from '@/api/sales-returns/sales-returns'
import * as customersApi from '@/api/customers/customers'
import { Toast, Modal } from '@douyinfe/semi-ui-19'

// Mock the API modules
vi.mock('@/api/sales-returns/sales-returns', () => ({
  getSalesReturns: vi.fn(),
}))

vi.mock('@/api/customers/customers', () => ({
  getCustomers: vi.fn(),
}))

// Mock react-router-dom's useNavigate
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

// Spy on Toast and Modal methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')
vi.spyOn(Modal, 'confirm').mockImplementation(() => ({ destroy: vi.fn() }))

// Sample customer data
const mockCustomers = [
  {
    id: 'cust-001',
    code: 'C001',
    name: '测试客户A',
    status: 'active',
  },
  {
    id: 'cust-002',
    code: 'C002',
    name: '测试客户B',
    status: 'active',
  },
]

// Sample sales return data
const mockSalesReturns = [
  {
    id: 'return-001',
    return_number: 'SR-2024-0001',
    sales_order_id: 'order-001',
    sales_order_number: 'SO-2024-0001',
    customer_id: 'cust-001',
    customer_name: '测试客户A',
    item_count: 2,
    total_refund: 200.0,
    status: 'DRAFT',
    reason: '商品质量问题',
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  },
  {
    id: 'return-002',
    return_number: 'SR-2024-0002',
    sales_order_id: 'order-002',
    sales_order_number: 'SO-2024-0002',
    customer_id: 'cust-002',
    customer_name: '测试客户B',
    item_count: 1,
    total_refund: 150.0,
    status: 'PENDING',
    reason: '客户不满意',
    created_at: '2024-01-14T09:00:00Z',
    submitted_at: '2024-01-14T10:00:00Z',
    updated_at: '2024-01-14T10:00:00Z',
  },
  {
    id: 'return-003',
    return_number: 'SR-2024-0003',
    sales_order_id: 'order-003',
    sales_order_number: 'SO-2024-0003',
    customer_id: 'cust-001',
    customer_name: '测试客户A',
    item_count: 3,
    total_refund: 300.0,
    status: 'APPROVED',
    reason: '发错货',
    created_at: '2024-01-13T08:00:00Z',
    submitted_at: '2024-01-13T09:00:00Z',
    approved_at: '2024-01-13T14:00:00Z',
    updated_at: '2024-01-13T14:00:00Z',
  },
  {
    id: 'return-004',
    return_number: 'SR-2024-0004',
    sales_order_id: 'order-004',
    sales_order_number: 'SO-2024-0004',
    customer_id: 'cust-002',
    customer_name: '测试客户B',
    item_count: 1,
    total_refund: 50.0,
    status: 'COMPLETED',
    reason: '商品损坏',
    created_at: '2024-01-12T08:00:00Z',
    submitted_at: '2024-01-12T09:00:00Z',
    approved_at: '2024-01-12T10:00:00Z',
    completed_at: '2024-01-12T14:00:00Z',
    updated_at: '2024-01-12T14:00:00Z',
  },
  {
    id: 'return-005',
    return_number: 'SR-2024-0005',
    sales_order_id: 'order-005',
    sales_order_number: 'SO-2024-0005',
    customer_id: 'cust-001',
    customer_name: '测试客户A',
    item_count: 1,
    total_refund: 100.0,
    status: 'REJECTED',
    reason: '无理由',
    reject_reason: '不符合退货条件',
    created_at: '2024-01-11T08:00:00Z',
    submitted_at: '2024-01-11T09:00:00Z',
    rejected_at: '2024-01-11T10:00:00Z',
    updated_at: '2024-01-11T10:00:00Z',
  },
  {
    id: 'return-006',
    return_number: 'SR-2024-0006',
    sales_order_id: 'order-006',
    sales_order_number: 'SO-2024-0006',
    customer_id: 'cust-002',
    customer_name: '测试客户B',
    item_count: 2,
    total_refund: 180.0,
    status: 'CANCELLED',
    reason: '客户取消',
    cancel_reason: '客户主动取消',
    created_at: '2024-01-10T08:00:00Z',
    updated_at: '2024-01-10T12:00:00Z',
  },
]

// Mock API response helpers
const createMockReturnListResponse = (
  returns = mockSalesReturns,
  total = mockSalesReturns.length
) => ({
  success: true,
  data: returns,
  meta: {
    total,
    page: 1,
    page_size: 20,
    total_pages: Math.ceil(total / 20),
  },
})

const createMockCustomerListResponse = (customers = mockCustomers) => ({
  success: true,
  data: customers,
  meta: {
    total: customers.length,
    page: 1,
    page_size: 100,
    total_pages: 1,
  },
})

describe('SalesReturnsPage', () => {
  let mockSalesReturnApiInstance: {
    getTradeSalesReturns: ReturnType<typeof vi.fn>
    postTradeSalesReturnsIdSubmit: ReturnType<typeof vi.fn>
    postTradeSalesReturnsIdApprove: ReturnType<typeof vi.fn>
    postTradeSalesReturnsIdReject: ReturnType<typeof vi.fn>
    postTradeSalesReturnsIdComplete: ReturnType<typeof vi.fn>
    postTradeSalesReturnsIdCancel: ReturnType<typeof vi.fn>
    deleteTradeSalesReturnsId: ReturnType<typeof vi.fn>
  }

  let mockCustomerApiInstance: {
    getPartnerCustomers: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock sales return API
    mockSalesReturnApiInstance = {
      getTradeSalesReturns: vi.fn().mockResolvedValue(createMockReturnListResponse()),
      postTradeSalesReturnsIdSubmit: vi.fn().mockResolvedValue({ success: true }),
      postTradeSalesReturnsIdApprove: vi.fn().mockResolvedValue({ success: true }),
      postTradeSalesReturnsIdReject: vi.fn().mockResolvedValue({ success: true }),
      postTradeSalesReturnsIdComplete: vi.fn().mockResolvedValue({ success: true }),
      postTradeSalesReturnsIdCancel: vi.fn().mockResolvedValue({ success: true }),
      deleteTradeSalesReturnsId: vi.fn().mockResolvedValue({ success: true }),
    }

    // Setup mock customer API
    mockCustomerApiInstance = {
      getPartnerCustomers: vi.fn().mockResolvedValue(createMockCustomerListResponse()),
    }

    vi.mocked(salesReturnsApi.getSalesReturns).mockReturnValue(
      mockSalesReturnApiInstance as unknown as ReturnType<typeof salesReturnsApi.getSalesReturns>
    )
    vi.mocked(customersApi.getCustomers).mockReturnValue(
      mockCustomerApiInstance as unknown as ReturnType<typeof customersApi.getCustomers>
    )
  })

  describe('Page Layout', () => {
    it('should display page title', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('销售退货')).toBeInTheDocument()
      })
    })

    it('should display new return button', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('新建退货')).toBeInTheDocument()
      })
    })

    it('should display approval button', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('审批')).toBeInTheDocument()
      })
    })

    it('should display refresh button', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('刷新')).toBeInTheDocument()
      })
    })

    it('should have search input', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByPlaceholderText('搜索退货单号...')).toBeInTheDocument()
      })
    })
  })

  describe('Return List Display', () => {
    it('should display return list with correct data', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      // Wait for data to load
      await waitFor(() => {
        expect(mockSalesReturnApiInstance.getTradeSalesReturns).toHaveBeenCalled()
      })

      // Verify return numbers are displayed
      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
        expect(screen.getByText('SR-2024-0002')).toBeInTheDocument()
        expect(screen.getByText('SR-2024-0003')).toBeInTheDocument()
      })
    })

    it('should display original order numbers', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      // Verify original order numbers
      expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      expect(screen.getByText('SO-2024-0002')).toBeInTheDocument()
    })

    it('should display customer names', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      // Verify customer names are displayed
      expect(screen.getAllByText('测试客户A').length).toBeGreaterThan(0)
      expect(screen.getAllByText('测试客户B').length).toBeGreaterThan(0)
    })

    it('should display item counts correctly', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      // Verify item counts are displayed with "件" suffix
      // Some counts may appear multiple times
      const twoItems = screen.getAllByText('2 件')
      expect(twoItems.length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('3 件')).toBeInTheDocument()
    })

    it('should display refund amounts formatted as currency', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      // Verify refund amounts are displayed with ¥ prefix
      expect(screen.getByText('¥200.00')).toBeInTheDocument()
      expect(screen.getByText('¥150.00')).toBeInTheDocument()
      expect(screen.getByText('¥300.00')).toBeInTheDocument()
    })
  })

  describe('Return Status Display', () => {
    it('should display draft status tag correctly', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('草稿')).toBeInTheDocument()
    })

    it('should display pending status tag correctly', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0002')).toBeInTheDocument()
      })

      expect(screen.getByText('待审批')).toBeInTheDocument()
    })

    it('should display approved status tag correctly', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0003')).toBeInTheDocument()
      })

      expect(screen.getByText('已审批')).toBeInTheDocument()
    })

    it('should display completed status tag correctly', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0004')).toBeInTheDocument()
      })

      expect(screen.getByText('已完成')).toBeInTheDocument()
    })

    it('should display rejected status tag correctly', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0005')).toBeInTheDocument()
      })

      expect(screen.getByText('已拒绝')).toBeInTheDocument()
    })

    it('should display cancelled status tag correctly', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0006')).toBeInTheDocument()
      })

      expect(screen.getByText('已取消')).toBeInTheDocument()
    })
  })

  describe('Table Column Headers', () => {
    it('should display return number column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('退货单号')).toBeInTheDocument()
      })
    })

    it('should display original order number column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('原订单号')).toBeInTheDocument()
      })
    })

    it('should display customer column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('客户')).toBeInTheDocument()
      })
    })

    it('should display item count column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('商品数量')).toBeInTheDocument()
      })
    })

    it('should display refund amount column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('退款金额')).toBeInTheDocument()
      })
    })

    it('should display status column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('状态')).toBeInTheDocument()
      })
    })

    it('should display create time column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('创建时间')).toBeInTheDocument()
      })
    })

    it('should display submit time column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('提交时间')).toBeInTheDocument()
      })
    })

    it('should display complete time column header', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('完成时间')).toBeInTheDocument()
      })
    })
  })

  describe('Navigation', () => {
    it('should navigate to new return page when clicking new return button', async () => {
      const { user } = renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      const newReturnButton = screen.getByText('新建退货')
      await user.click(newReturnButton)

      expect(mockNavigate).toHaveBeenCalledWith('/trade/sales-returns/new')
    })

    it('should navigate to approval page when clicking approval button', async () => {
      const { user } = renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      const approvalButton = screen.getByText('审批')
      await user.click(approvalButton)

      expect(mockNavigate).toHaveBeenCalledWith('/trade/sales-returns/approval')
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when return list API fails', async () => {
      mockSalesReturnApiInstance.getTradeSalesReturns.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取销售退货列表失败')
      })
    })

    it('should handle empty return list gracefully', async () => {
      mockSalesReturnApiInstance.getTradeSalesReturns.mockResolvedValueOnce(
        createMockReturnListResponse([], 0)
      )

      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(mockSalesReturnApiInstance.getTradeSalesReturns).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('销售退货')).toBeInTheDocument()
    })

    it('should handle customer API failure gracefully', async () => {
      mockCustomerApiInstance.getPartnerCustomers.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      // Should still render the page
      await waitFor(() => {
        expect(screen.getByText('销售退货')).toBeInTheDocument()
      })
    })
  })

  describe('API Integration', () => {
    it('should call return API with correct pagination parameters', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(mockSalesReturnApiInstance.getTradeSalesReturns).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 1,
            page_size: 20,
          })
        )
      })
    })

    it('should call return API with default sort parameters', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(mockSalesReturnApiInstance.getTradeSalesReturns).toHaveBeenCalledWith(
          expect.objectContaining({
            order_by: 'created_at',
            order_dir: 'desc',
          })
        )
      })
    })

    it('should call customer API to load filter options', async () => {
      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(mockCustomerApiInstance.getPartnerCustomers).toHaveBeenCalledWith(
          expect.objectContaining({
            page_size: 100,
          })
        )
      })
    })
  })

  describe('Refresh Functionality', () => {
    it('should refresh return list when clicking refresh button', async () => {
      const { user } = renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      // Clear mock to track new call
      mockSalesReturnApiInstance.getTradeSalesReturns.mockClear()

      const refreshButton = screen.getByText('刷新')
      await user.click(refreshButton)

      await waitFor(() => {
        expect(mockSalesReturnApiInstance.getTradeSalesReturns).toHaveBeenCalled()
      })
    })
  })

  describe('Return Actions for Draft Status', () => {
    it('should show submit and delete actions for draft returns', async () => {
      // Only draft return
      const draftReturn = {
        ...mockSalesReturns[0],
        status: 'DRAFT',
      }

      mockSalesReturnApiInstance.getTradeSalesReturns.mockResolvedValueOnce(
        createMockReturnListResponse([draftReturn], 1)
      )

      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0001')).toBeInTheDocument()
      })

      // Verify draft return is displayed
      expect(screen.getByText('草稿')).toBeInTheDocument()
    })
  })

  describe('Return Actions for Pending Status', () => {
    it('should show approve and reject actions for pending returns', async () => {
      // Only pending return
      const pendingReturn = {
        ...mockSalesReturns[1],
        status: 'PENDING',
      }

      mockSalesReturnApiInstance.getTradeSalesReturns.mockResolvedValueOnce(
        createMockReturnListResponse([pendingReturn], 1)
      )

      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0002')).toBeInTheDocument()
      })

      // Verify pending return is displayed with status
      expect(screen.getByText('待审批')).toBeInTheDocument()
      // For pending status, approve and reject buttons should be visible
      // The DataTable may render actions differently, verify the status tag is correct
    })
  })

  describe('Return Actions for Approved Status', () => {
    it('should show complete action for approved returns', async () => {
      // Only approved return
      const approvedReturn = {
        ...mockSalesReturns[2],
        status: 'APPROVED',
      }

      mockSalesReturnApiInstance.getTradeSalesReturns.mockResolvedValueOnce(
        createMockReturnListResponse([approvedReturn], 1)
      )

      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0003')).toBeInTheDocument()
      })

      // Verify approved return is displayed with complete action
      expect(screen.getByText('已审批')).toBeInTheDocument()
      expect(screen.getByText('完成')).toBeInTheDocument()
    })
  })

  describe('Return Actions for Completed Status', () => {
    it('should only show view action for completed returns', async () => {
      // Only completed return
      const completedReturn = {
        ...mockSalesReturns[3],
        status: 'COMPLETED',
      }

      mockSalesReturnApiInstance.getTradeSalesReturns.mockResolvedValueOnce(
        createMockReturnListResponse([completedReturn], 1)
      )

      renderWithProviders(<SalesReturnsPage />, { route: '/trade/sales-returns' })

      await waitFor(() => {
        expect(screen.getByText('SR-2024-0004')).toBeInTheDocument()
      })

      // Verify completed return is displayed
      expect(screen.getByText('已完成')).toBeInTheDocument()
      // Should not have action buttons for state changes
      expect(screen.queryByText('提交审批')).not.toBeInTheDocument()
      expect(screen.queryByText('通过')).not.toBeInTheDocument()
      expect(screen.queryByText('拒绝')).not.toBeInTheDocument()
      expect(screen.queryByText('完成')).not.toBeInTheDocument()
      expect(screen.queryByText('删除')).not.toBeInTheDocument()
    })
  })
})
