/**
 * Payables Page Tests (P4-QA-005)
 *
 * These tests verify the frontend components for the Payables (Account Payable) module:
 * - Payable list display (payable number, supplier, amounts, status)
 * - Filter functionality (status filter, source type, date range, overdue)
 * - Summary cards display
 * - Navigation and actions (view, pay)
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import PayablesPage from './Payables'
import * as financeApi from '@/api/finance/finance'
import { Toast } from '@douyinfe/semi-ui'

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

// Sample payable data
const mockPayables = [
  {
    id: 'pay-001',
    payable_number: 'AP-2024-0001',
    supplier_id: 'supp-001',
    supplier_name: '测试供应商A',
    source_type: 'PURCHASE_ORDER',
    source_id: 'po-001',
    source_number: 'PO-2024-0001',
    total_amount: 2000.0,
    paid_amount: 500.0,
    outstanding_amount: 1500.0,
    status: 'PARTIAL',
    due_date: '2024-02-20',
    created_at: '2024-01-20T10:00:00Z',
    updated_at: '2024-01-20T10:00:00Z',
  },
  {
    id: 'pay-002',
    payable_number: 'AP-2024-0002',
    supplier_id: 'supp-002',
    supplier_name: '测试供应商B',
    source_type: 'PURCHASE_ORDER',
    source_id: 'po-002',
    source_number: 'PO-2024-0002',
    total_amount: 800.0,
    paid_amount: 0,
    outstanding_amount: 800.0,
    status: 'PENDING',
    due_date: '2024-01-05', // Overdue
    created_at: '2024-01-05T09:00:00Z',
    updated_at: '2024-01-05T09:00:00Z',
  },
  {
    id: 'pay-003',
    payable_number: 'AP-2024-0003',
    supplier_id: 'supp-001',
    supplier_name: '测试供应商A',
    source_type: 'PURCHASE_RETURN',
    source_id: 'pr-001',
    source_number: 'PR-2024-0001',
    total_amount: -300.0,
    paid_amount: 0,
    outstanding_amount: -300.0,
    status: 'REVERSED',
    due_date: null,
    created_at: '2024-01-15T08:00:00Z',
    updated_at: '2024-01-15T08:00:00Z',
  },
  {
    id: 'pay-004',
    payable_number: 'AP-2024-0004',
    supplier_id: 'supp-002',
    supplier_name: '测试供应商B',
    source_type: 'MANUAL',
    source_id: null,
    source_number: null,
    total_amount: 450.0,
    paid_amount: 450.0,
    outstanding_amount: 0,
    status: 'PAID',
    due_date: '2024-01-25',
    created_at: '2024-01-10T08:00:00Z',
    updated_at: '2024-01-25T10:00:00Z',
  },
  {
    id: 'pay-005',
    payable_number: 'AP-2024-0005',
    supplier_id: 'supp-001',
    supplier_name: '测试供应商A',
    source_type: 'PURCHASE_ORDER',
    source_id: 'po-003',
    source_number: 'PO-2024-0003',
    total_amount: 200.0,
    paid_amount: 0,
    outstanding_amount: 0,
    status: 'CANCELLED',
    due_date: '2024-01-30',
    created_at: '2024-01-08T08:00:00Z',
    updated_at: '2024-01-09T10:00:00Z',
  },
]

// Sample summary data
const mockSummary = {
  total_outstanding: 2300.0,
  total_overdue: 800.0,
  pending_count: 1,
  partial_count: 1,
  overdue_count: 1,
}

// Mock API response helpers
const createMockPayableListResponse = (payables = mockPayables, total = mockPayables.length) => ({
  success: true,
  data: payables,
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

describe('PayablesPage', () => {
  let mockFinanceApiInstance: {
    getFinancePayables: ReturnType<typeof vi.fn>
    getFinancePayablesSummary: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock finance API
    mockFinanceApiInstance = {
      getFinancePayables: vi.fn().mockResolvedValue(createMockPayableListResponse()),
      getFinancePayablesSummary: vi.fn().mockResolvedValue(createMockSummaryResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  describe('Page Layout', () => {
    it('should display page title', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('应付账款')).toBeInTheDocument()
      })
    })

    it('should have search input', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      const searchInput = screen.getByPlaceholderText('搜索单据编号、供应商名称...')
      expect(searchInput).toBeInTheDocument()
    })

    it('should have refresh button', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('刷新')).toBeInTheDocument()
    })

    it('should have filter dropdowns', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      // Should have multiple filter dropdowns (status, source type, overdue)
      const comboboxes = screen.getAllByRole('combobox')
      expect(comboboxes.length).toBeGreaterThanOrEqual(3)
    })
  })

  describe('Summary Cards Display', () => {
    it('should display total outstanding amount', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayablesSummary).toHaveBeenCalled()
      })

      expect(screen.getByText('待付总额')).toBeInTheDocument()
    })

    it('should display overdue total amount', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayablesSummary).toHaveBeenCalled()
      })

      expect(screen.getByText('逾期总额')).toBeInTheDocument()
    })

    it('should display pending count', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayablesSummary).toHaveBeenCalled()
      })

      expect(screen.getByText('待付款单')).toBeInTheDocument()
    })

    it('should display partial count', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayablesSummary).toHaveBeenCalled()
      })

      // "部分付款" appears in both summary label and status tags
      // Just verify at least one exists
      const partialElements = screen.getAllByText('部分付款')
      expect(partialElements.length).toBeGreaterThanOrEqual(1)
    })

    it('should display overdue count', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayablesSummary).toHaveBeenCalled()
      })

      expect(screen.getByText('逾期单数')).toBeInTheDocument()
    })
  })

  describe('Payable List Display', () => {
    it('should display payable numbers', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayables).toHaveBeenCalled()
      })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
        expect(screen.getByText('AP-2024-0002')).toBeInTheDocument()
        expect(screen.getByText('AP-2024-0003')).toBeInTheDocument()
      })
    })

    it('should display supplier names', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getAllByText('测试供应商A').length).toBeGreaterThan(0)
      expect(screen.getAllByText('测试供应商B').length).toBeGreaterThan(0)
    })

    it('should display source numbers', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('PO-2024-0001')).toBeInTheDocument()
      expect(screen.getByText('PO-2024-0002')).toBeInTheDocument()
    })

    it('should display amounts formatted as currency', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      // Verify total amounts are displayed with ¥ prefix
      expect(screen.getByText('¥2,000.00')).toBeInTheDocument()
      expect(screen.getAllByText('¥800.00').length).toBeGreaterThanOrEqual(1)
    })

    it('should display table column headers', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      // Verify column headers
      expect(screen.getByText('单据编号')).toBeInTheDocument()
      expect(screen.getByText('供应商名称')).toBeInTheDocument()
      expect(screen.getByText('来源单据')).toBeInTheDocument()
      expect(screen.getByText('应付金额')).toBeInTheDocument()
      expect(screen.getByText('已付金额')).toBeInTheDocument()
      expect(screen.getByText('待付金额')).toBeInTheDocument()
      expect(screen.getByText('到期日')).toBeInTheDocument()
      expect(screen.getByText('状态')).toBeInTheDocument()
      expect(screen.getByText('创建时间')).toBeInTheDocument()
    })
  })

  describe('Payable Status Display', () => {
    it('should display PENDING status tag', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0002')).toBeInTheDocument()
      })

      expect(screen.getByText('待付款')).toBeInTheDocument()
    })

    it('should display PARTIAL status tag', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      // Note: "部分付款" appears both in summary and status tag
      expect(screen.getAllByText('部分付款').length).toBeGreaterThanOrEqual(1)
    })

    it('should display PAID status tag', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0004')).toBeInTheDocument()
      })

      expect(screen.getByText('已付款')).toBeInTheDocument()
    })

    it('should display REVERSED status tag', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0003')).toBeInTheDocument()
      })

      expect(screen.getByText('已冲红')).toBeInTheDocument()
    })

    it('should display CANCELLED status tag', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0005')).toBeInTheDocument()
      })

      expect(screen.getByText('已取消')).toBeInTheDocument()
    })
  })

  describe('Source Type Display', () => {
    it('should display purchase order source type label', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getAllByText('采购订单').length).toBeGreaterThanOrEqual(1)
    })

    it('should display purchase return source type label', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0003')).toBeInTheDocument()
      })

      expect(screen.getByText('采购退货')).toBeInTheDocument()
    })

    it('should display manual source type label', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0004')).toBeInTheDocument()
      })

      expect(screen.getByText('手工录入')).toBeInTheDocument()
    })
  })

  describe('Actions', () => {
    it('should show view action for all payables', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      // View action should be available
      expect(screen.getAllByText('查看').length).toBeGreaterThanOrEqual(1)
    })

    it('should show pay action for PENDING payables', async () => {
      // Only pending payable
      const pendingPayable = {
        ...mockPayables[1],
        status: 'PENDING',
      }

      mockFinanceApiInstance.getFinancePayables.mockResolvedValueOnce(
        createMockPayableListResponse([pendingPayable], 1)
      )

      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0002')).toBeInTheDocument()
      })

      expect(screen.getByText('付款')).toBeInTheDocument()
    })

    it('should show pay action for PARTIAL payables', async () => {
      // Only partial payable
      const partialPayable = {
        ...mockPayables[0],
        status: 'PARTIAL',
      }

      mockFinanceApiInstance.getFinancePayables.mockResolvedValueOnce(
        createMockPayableListResponse([partialPayable], 1)
      )

      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('付款')).toBeInTheDocument()
    })

    it('should not show pay action for PAID payables', async () => {
      // Only paid payable
      const paidPayable = {
        ...mockPayables[3],
        status: 'PAID',
      }

      mockFinanceApiInstance.getFinancePayables.mockResolvedValueOnce(
        createMockPayableListResponse([paidPayable], 1)
      )

      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0004')).toBeInTheDocument()
      })

      // Only view action should be visible, no pay
      expect(screen.getByText('查看')).toBeInTheDocument()
      expect(screen.queryByText('付款')).not.toBeInTheDocument()
    })

    it('should not show pay action for CANCELLED payables', async () => {
      // Only cancelled payable
      const cancelledPayable = {
        ...mockPayables[4],
        status: 'CANCELLED',
      }

      mockFinanceApiInstance.getFinancePayables.mockResolvedValueOnce(
        createMockPayableListResponse([cancelledPayable], 1)
      )

      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0005')).toBeInTheDocument()
      })

      expect(screen.queryByText('付款')).not.toBeInTheDocument()
    })
  })

  describe('Navigation', () => {
    it('should navigate to payment voucher creation when clicking pay', async () => {
      // Only pending payable
      const pendingPayable = {
        ...mockPayables[1],
        status: 'PENDING',
      }

      mockFinanceApiInstance.getFinancePayables.mockResolvedValueOnce(
        createMockPayableListResponse([pendingPayable], 1)
      )

      const { user } = renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0002')).toBeInTheDocument()
      })

      const payButton = screen.getByText('付款')
      await user.click(payButton)

      expect(mockNavigate).toHaveBeenCalledWith(
        expect.stringContaining('/finance/payments/new?payable_id=pay-002')
      )
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when payables API fails', async () => {
      mockFinanceApiInstance.getFinancePayables.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取应付账款列表失败')
      })
    })

    it('should handle empty payables list gracefully', async () => {
      mockFinanceApiInstance.getFinancePayables.mockResolvedValueOnce(
        createMockPayableListResponse([], 0)
      )

      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayables).toHaveBeenCalled()
      })

      // Page should render without errors
      expect(screen.getByText('应付账款')).toBeInTheDocument()
    })

    it('should handle summary API failure gracefully', async () => {
      mockFinanceApiInstance.getFinancePayablesSummary.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      // Should still render the page
      await waitFor(() => {
        expect(screen.getByText('应付账款')).toBeInTheDocument()
      })
    })
  })

  describe('API Integration', () => {
    it('should call payables API with correct pagination parameters', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayables).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 1,
            page_size: 20,
          })
        )
      })
    })

    it('should call summary API on mount', async () => {
      renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayablesSummary).toHaveBeenCalled()
      })
    })
  })

  describe('Refresh Functionality', () => {
    it('should refresh payables list when clicking refresh button', async () => {
      const { user } = renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      // Clear mock to track new call
      mockFinanceApiInstance.getFinancePayables.mockClear()
      mockFinanceApiInstance.getFinancePayablesSummary.mockClear()

      const refreshButton = screen.getByText('刷新')
      await user.click(refreshButton)

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayables).toHaveBeenCalled()
        expect(mockFinanceApiInstance.getFinancePayablesSummary).toHaveBeenCalled()
      })
    })
  })

  describe('Search Functionality', () => {
    it('should call API with search parameter when searching', async () => {
      const { user } = renderWithProviders(<PayablesPage />, { route: '/finance/payables' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
      })

      // Clear mock to track new call
      mockFinanceApiInstance.getFinancePayables.mockClear()

      const searchInput = screen.getByPlaceholderText('搜索单据编号、供应商名称...')
      await user.type(searchInput, 'AP-2024-0001')

      await waitFor(() => {
        expect(mockFinanceApiInstance.getFinancePayables).toHaveBeenCalledWith(
          expect.objectContaining({
            search: 'AP-2024-0001',
          })
        )
      })
    })
  })
})
