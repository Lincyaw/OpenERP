/**
 * Receivables Page Tests (P4-QA-005)
 *
 * These tests verify the frontend components for the Receivables (Account Receivable) module:
 * - Receivable list display (receivable number, customer, amounts, status)
 * - Filter functionality (status filter, source type, date range, overdue)
 * - Summary cards display
 * - Navigation and actions (view, collect)
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import ReceivablesPage from './Receivables'
import * as financeApi from '@/api/finance/finance'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the API modules
vi.mock('@/api/finance/finance', () => ({
  getFinanceApi: vi.fn(),
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

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample receivable data
const mockReceivables = [
  {
    id: 'recv-001',
    receivable_number: 'AR-2024-0001',
    customer_id: 'cust-001',
    customer_name: '测试客户A',
    source_type: 'SALES_ORDER',
    source_id: 'order-001',
    source_number: 'SO-2024-0001',
    total_amount: 1000.0,
    paid_amount: 200.0,
    outstanding_amount: 800.0,
    status: 'PARTIAL',
    due_date: '2024-02-15',
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  },
  {
    id: 'recv-002',
    receivable_number: 'AR-2024-0002',
    customer_id: 'cust-002',
    customer_name: '测试客户B',
    source_type: 'SALES_ORDER',
    source_id: 'order-002',
    source_number: 'SO-2024-0002',
    total_amount: 500.0,
    paid_amount: 0,
    outstanding_amount: 500.0,
    status: 'PENDING',
    due_date: '2024-01-10', // Overdue
    created_at: '2024-01-10T09:00:00Z',
    updated_at: '2024-01-10T09:00:00Z',
  },
  {
    id: 'recv-003',
    receivable_number: 'AR-2024-0003',
    customer_id: 'cust-001',
    customer_name: '测试客户A',
    source_type: 'SALES_RETURN',
    source_id: 'return-001',
    source_number: 'SR-2024-0001',
    total_amount: -200.0,
    paid_amount: 0,
    outstanding_amount: -200.0,
    status: 'REVERSED',
    due_date: null,
    created_at: '2024-01-12T08:00:00Z',
    updated_at: '2024-01-12T08:00:00Z',
  },
  {
    id: 'recv-004',
    receivable_number: 'AR-2024-0004',
    customer_id: 'cust-002',
    customer_name: '测试客户B',
    source_type: 'MANUAL',
    source_id: null,
    source_number: null,
    total_amount: 300.0,
    paid_amount: 300.0,
    outstanding_amount: 0,
    status: 'PAID',
    due_date: '2024-01-20',
    created_at: '2024-01-08T08:00:00Z',
    updated_at: '2024-01-20T10:00:00Z',
  },
  {
    id: 'recv-005',
    receivable_number: 'AR-2024-0005',
    customer_id: 'cust-001',
    customer_name: '测试客户A',
    source_type: 'SALES_ORDER',
    source_id: 'order-003',
    source_number: 'SO-2024-0003',
    total_amount: 150.0,
    paid_amount: 0,
    outstanding_amount: 0,
    status: 'CANCELLED',
    due_date: '2024-01-25',
    created_at: '2024-01-05T08:00:00Z',
    updated_at: '2024-01-06T10:00:00Z',
  },
]

// Sample summary data
const mockSummary = {
  total_outstanding: 1300.0,
  total_overdue: 500.0,
  pending_count: 1,
  partial_count: 1,
  overdue_count: 1,
}

// Mock API response helpers
const createMockReceivableListResponse = (
  receivables = mockReceivables,
  total = mockReceivables.length
) => ({
  success: true,
  data: receivables,
  meta: {
    total,
    page: 1,
    page_size: 20,
    total_pages: Math.ceil(total / 20),
  },
})

const createMockSummaryResponse = (summary = mockSummary) => ({
  success: true,
  data: summary,
})

describe('ReceivablesPage', () => {
  let mockFinanceApiInstance: {
    listFinanceReceivables: ReturnType<typeof vi.fn>
    getFinanceReceivableReceivableSummary: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock finance API
    mockFinanceApiInstance = {
      listFinanceReceivables: vi.fn().mockResolvedValue(createMockReceivableListResponse()),
      getFinanceReceivableReceivableSummary: vi.fn().mockResolvedValue(createMockSummaryResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  describe('Page Layout', () => {
    it('should display page title', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('应收账款')).toBeInTheDocument()
      })
    })

    it('should have search input', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      const searchInput = screen.getByPlaceholderText('搜索单据编号、客户名称...')
      expect(searchInput).toBeInTheDocument()
    })

    it('should have refresh button', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })

    it('should have filter dropdowns', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      // Should have multiple filter dropdowns (status, source type, overdue)
      const comboboxes = screen.getAllByRole('combobox')
      expect(comboboxes.length).toBeGreaterThanOrEqual(3)
    })
  })

  describe('Summary Cards Display', () => {
    it('should display total outstanding amount', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinanceReceivableReceivableSummary).toHaveBeenCalled()
      })

      expect(screen.getByText('待收总额')).toBeInTheDocument()
    })

    it('should display overdue total amount', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinanceReceivableReceivableSummary).toHaveBeenCalled()
      })

      expect(screen.getByText('逾期总额')).toBeInTheDocument()
    })

    it('should display pending count', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinanceReceivableReceivableSummary).toHaveBeenCalled()
      })

      expect(screen.getByText('待收款单')).toBeInTheDocument()
    })

    it('should display partial count', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinanceReceivableReceivableSummary).toHaveBeenCalled()
      })

      // "部分收款" appears in both summary label and status tags
      // Just verify at least one exists
      const partialElements = screen.getAllByText('部分收款')
      expect(partialElements.length).toBeGreaterThanOrEqual(1)
    })

    it('should display overdue count', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinanceReceivableReceivableSummary).toHaveBeenCalled()
      })

      expect(screen.getByText('逾期单数')).toBeInTheDocument()
    })
  })

  describe('Receivable List Display', () => {
    it('should display receivable numbers', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.listFinanceReceivables).toHaveBeenCalled()
      })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
        expect(screen.getByText('AR-2024-0002')).toBeInTheDocument()
        expect(screen.getByText('AR-2024-0003')).toBeInTheDocument()
      })
    })

    it('should display customer names', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getAllByText('测试客户A').length).toBeGreaterThan(0)
      expect(screen.getAllByText('测试客户B').length).toBeGreaterThan(0)
    })

    it('should display source numbers', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      expect(screen.getByText('SO-2024-0002')).toBeInTheDocument()
    })

    it('should display amounts formatted as currency', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      // Verify total amounts are displayed with ¥ prefix
      expect(screen.getByText('¥1,000.00')).toBeInTheDocument()
      expect(screen.getAllByText('¥500.00').length).toBeGreaterThanOrEqual(1)
    })

    it('should display table column headers', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      // Verify column headers
      expect(screen.getByText('单据编号')).toBeInTheDocument()
      expect(screen.getByText('客户名称')).toBeInTheDocument()
      expect(screen.getByText('来源单据')).toBeInTheDocument()
      expect(screen.getByText('应收金额')).toBeInTheDocument()
      expect(screen.getByText('已收金额')).toBeInTheDocument()
      expect(screen.getByText('待收金额')).toBeInTheDocument()
      expect(screen.getByText('到期日')).toBeInTheDocument()
      expect(screen.getByText('状态')).toBeInTheDocument()
      expect(screen.getByText('创建时间')).toBeInTheDocument()
    })
  })

  describe('Receivable Status Display', () => {
    it('should display PENDING status tag', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0002')).toBeInTheDocument()
      })

      expect(screen.getByText('待收款')).toBeInTheDocument()
    })

    it('should display PARTIAL status tag', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      // Note: "部分收款" appears both in summary and status tag
      expect(screen.getAllByText('部分收款').length).toBeGreaterThanOrEqual(1)
    })

    it('should display PAID status tag', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0004')).toBeInTheDocument()
      })

      expect(screen.getByText('已收款')).toBeInTheDocument()
    })

    it('should display REVERSED status tag', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0003')).toBeInTheDocument()
      })

      expect(screen.getByText('已冲红')).toBeInTheDocument()
    })

    it('should display CANCELLED status tag', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0005')).toBeInTheDocument()
      })

      expect(screen.getByText('已取消')).toBeInTheDocument()
    })
  })

  describe('Source Type Display', () => {
    it('should display sales order source type label', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getAllByText('销售订单').length).toBeGreaterThanOrEqual(1)
    })

    it('should display sales return source type label', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0003')).toBeInTheDocument()
      })

      expect(screen.getByText('销售退货')).toBeInTheDocument()
    })

    it('should display manual source type label', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0004')).toBeInTheDocument()
      })

      expect(screen.getByText('手工录入')).toBeInTheDocument()
    })
  })

  describe('Actions', () => {
    it('should show view action for all receivables', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      // View action should be available
      expect(screen.getAllByText('查看').length).toBeGreaterThanOrEqual(1)
    })

    it('should show collect action for PENDING receivables', async () => {
      // Only pending receivable
      const pendingReceivable = {
        ...mockReceivables[1],
        status: 'PENDING',
      }

      mockFinanceApiInstance.listFinanceReceivables.mockResolvedValueOnce(
        createMockReceivableListResponse([pendingReceivable], 1)
      )

      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0002')).toBeInTheDocument()
      })

      expect(screen.getByText('收款')).toBeInTheDocument()
    })

    it('should show collect action for PARTIAL receivables', async () => {
      // Only partial receivable
      const partialReceivable = {
        ...mockReceivables[0],
        status: 'PARTIAL',
      }

      mockFinanceApiInstance.listFinanceReceivables.mockResolvedValueOnce(
        createMockReceivableListResponse([partialReceivable], 1)
      )

      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('收款')).toBeInTheDocument()
    })

    it('should not show collect action for PAID receivables', async () => {
      // Only paid receivable
      const paidReceivable = {
        ...mockReceivables[3],
        status: 'PAID',
      }

      mockFinanceApiInstance.listFinanceReceivables.mockResolvedValueOnce(
        createMockReceivableListResponse([paidReceivable], 1)
      )

      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0004')).toBeInTheDocument()
      })

      // Only view action should be visible, no collect
      expect(screen.getByText('查看')).toBeInTheDocument()
      expect(screen.queryByText('收款')).not.toBeInTheDocument()
    })

    it('should not show collect action for CANCELLED receivables', async () => {
      // Only cancelled receivable
      const cancelledReceivable = {
        ...mockReceivables[4],
        status: 'CANCELLED',
      }

      mockFinanceApiInstance.listFinanceReceivables.mockResolvedValueOnce(
        createMockReceivableListResponse([cancelledReceivable], 1)
      )

      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0005')).toBeInTheDocument()
      })

      expect(screen.queryByText('收款')).not.toBeInTheDocument()
    })
  })

  describe('Navigation', () => {
    it('should navigate to receipt voucher creation when clicking collect', async () => {
      // Only pending receivable
      const pendingReceivable = {
        ...mockReceivables[1],
        status: 'PENDING',
      }

      mockFinanceApiInstance.listFinanceReceivables.mockResolvedValueOnce(
        createMockReceivableListResponse([pendingReceivable], 1)
      )

      const { user } = renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0002')).toBeInTheDocument()
      })

      const collectButton = screen.getByText('收款')
      await user.click(collectButton)

      expect(mockNavigate).toHaveBeenCalledWith(
        expect.stringContaining('/finance/receipts/new?receivable_id=recv-002')
      )
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when receivables API fails', async () => {
      mockFinanceApiInstance.listFinanceReceivables.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取应收账款列表失败')
      })
    })

    it('should handle empty receivables list gracefully', async () => {
      mockFinanceApiInstance.listFinanceReceivables.mockResolvedValueOnce(
        createMockReceivableListResponse([], 0)
      )

      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.listFinanceReceivables).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('应收账款')).toBeInTheDocument()
    })

    it('should handle summary API failure gracefully', async () => {
      mockFinanceApiInstance.getFinanceReceivableReceivableSummary.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      // Should still render the page
      await waitFor(() => {
        expect(screen.getByText('应收账款')).toBeInTheDocument()
      })
    })
  })

  describe('API Integration', () => {
    it('should call receivables API with correct pagination parameters', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.listFinanceReceivables).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 1,
            page_size: 20,
          })
        )
      })
    })

    it('should call summary API on mount', async () => {
      renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinanceReceivableReceivableSummary).toHaveBeenCalled()
      })
    })
  })

  describe('Refresh Functionality', () => {
    it('should refresh receivables list when clicking refresh button', async () => {
      const { user } = renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      // Clear mock to track new call
      mockFinanceApiInstance.listFinanceReceivables.mockClear()
      mockFinanceApiInstance.getFinanceReceivableReceivableSummary.mockClear()

      const refreshButton = screen.getByText('刷新')
      await user.click(refreshButton)

      await waitFor(() => {
        expect(mockFinanceApiInstance.listFinanceReceivables).toHaveBeenCalled()
        expect(mockFinanceApiInstance.getFinanceReceivableReceivableSummary).toHaveBeenCalled()
      })
    })
  })

  describe('Search Functionality', () => {
    it('should call API with search parameter when searching', async () => {
      const { user } = renderWithProviders(<ReceivablesPage />, { route: '/finance/receivables' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
      })

      // Clear mock to track new call
      mockFinanceApiInstance.listFinanceReceivables.mockClear()

      const searchInput = screen.getByPlaceholderText('搜索单据编号、客户名称...')
      await user.type(searchInput, 'AR-2024-0001')

      await waitFor(() => {
        expect(mockFinanceApiInstance.listFinanceReceivables).toHaveBeenCalledWith(
          expect.objectContaining({
            search: 'AR-2024-0001',
          })
        )
      })
    })
  })
})
