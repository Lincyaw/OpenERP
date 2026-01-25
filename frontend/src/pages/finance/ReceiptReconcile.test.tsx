/**
 * Receipt Reconcile Page Tests (P4-QA-005)
 *
 * These tests verify the frontend components for the Receipt Reconciliation page:
 * - Page layout (voucher details card, reconciliation section)
 * - Voucher information display
 * - Mode switching (FIFO vs MANUAL)
 * - Receivables table display
 * - Allocation selection and amount input
 * - Reconciliation summary display
 * - Reconcile button and result handling
 * - Navigation (back)
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import ReceiptReconcilePage from './ReceiptReconcile'
import * as financeApi from '@/api/finance/finance'
import { Toast } from '@douyinfe/semi-ui'

// Mock the API modules
vi.mock('@/api/finance/finance', () => ({
  getFinanceApi: vi.fn(),
}))

// Mock react-router-dom
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ id: 'rv-001' }),
  }
})

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample receipt voucher data
const mockReceiptVoucher = {
  id: 'rv-001',
  voucher_number: 'RV-2024-0001',
  customer_id: 'cust-001',
  customer_name: '测试客户A',
  amount: 5000.0,
  allocated_amount: 0,
  unallocated_amount: 5000.0,
  payment_method: 'BANK_TRANSFER',
  receipt_date: '2024-01-25',
  status: 'CONFIRMED',
  allocations: [],
}

const mockAllocatedVoucher = {
  ...mockReceiptVoucher,
  status: 'ALLOCATED',
  allocated_amount: 5000.0,
  unallocated_amount: 0,
}

const mockDraftVoucher = {
  ...mockReceiptVoucher,
  status: 'DRAFT',
}

// Sample receivable data
const mockReceivables = [
  {
    id: 'recv-001',
    receivable_number: 'AR-2024-0001',
    customer_id: 'cust-001',
    customer_name: '测试客户A',
    source_number: 'SO-2024-0001',
    source_type: 'SALES_ORDER',
    total_amount: 3000.0,
    paid_amount: 0,
    outstanding_amount: 3000.0,
    status: 'PENDING',
    due_date: '2024-02-15',
  },
  {
    id: 'recv-002',
    receivable_number: 'AR-2024-0002',
    customer_id: 'cust-001',
    customer_name: '测试客户A',
    source_number: 'SO-2024-0002',
    source_type: 'SALES_ORDER',
    total_amount: 2500.0,
    paid_amount: 500.0,
    outstanding_amount: 2000.0,
    status: 'PARTIAL',
    due_date: '2024-02-20',
  },
]

// Mock API response helpers
const createMockVoucherResponse = (voucher = mockReceiptVoucher) => ({
  success: true,
  data: voucher,
})

const createMockReceivablesResponse = (receivables = mockReceivables) => ({
  success: true,
  data: receivables,
  meta: {
    total: receivables.length,
    page: 1,
    page_size: 100,
    total_pages: 1,
  },
})

const createMockReconcileResponse = () => ({
  success: true,
  data: {
    voucher: {
      ...mockReceiptVoucher,
      status: 'ALLOCATED',
      allocated_amount: 5000.0,
      unallocated_amount: 0,
    },
    total_reconciled: 5000.0,
    remaining_unallocated: 0,
    fully_reconciled: true,
    updated_receivables: [
      { ...mockReceivables[0], paid_amount: 3000.0, outstanding_amount: 0, status: 'PAID' },
      { ...mockReceivables[1], paid_amount: 2500.0, outstanding_amount: 0, status: 'PAID' },
    ],
  },
})

describe('ReceiptReconcilePage', () => {
  let mockFinanceApiInstance: {
    getFinanceReceiptsId: ReturnType<typeof vi.fn>
    getFinanceReceivables: ReturnType<typeof vi.fn>
    postFinanceReceiptsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock finance API
    mockFinanceApiInstance = {
      getFinanceReceiptsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinanceReceivables: vi.fn().mockResolvedValue(createMockReceivablesResponse()),
      postFinanceReceiptsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  describe('Page Layout', () => {
    it('should display page title', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('收款核销')).toBeInTheDocument()
      })
    })

    it('should display back button', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('返回')).toBeInTheDocument()
      })
    })

    it('should display refresh button', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('刷新')).toBeInTheDocument()
      })
    })

    it('should display voucher information section', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('收款单信息')).toBeInTheDocument()
      })
    })

    it('should display pending receivables section for confirmed voucher', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('待核销应收账款')).toBeInTheDocument()
      })
    })
  })

  describe('Voucher Details Display', () => {
    it('should display voucher number', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('RV-2024-0001')).toBeInTheDocument()
      })
    })

    it('should display customer name', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('测试客户A')).toBeInTheDocument()
      })
    })

    it('should display voucher status tag', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('已确认')).toBeInTheDocument()
      })
    })

    it('should display payment method', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('银行转账')).toBeInTheDocument()
      })
    })

    it('should display receipt amount formatted', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        // ¥5,000.00 formatted currency appears multiple times
        // (receipt amount, allocated amount, unallocated amount, etc.)
        const amountElements = screen.getAllByText('¥5,000.00')
        expect(amountElements.length).toBeGreaterThanOrEqual(1)
      })
    })

    it('should display unallocated amount', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        // Label for unallocated amount section
        expect(screen.getByText('未核销金额')).toBeInTheDocument()
      })
    })
  })

  describe('Reconciliation Mode Selection', () => {
    it('should display FIFO mode button', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('自动核销 (FIFO)')).toBeInTheDocument()
      })
    })

    it('should display Manual mode button', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('手动核销')).toBeInTheDocument()
      })
    })

    it('should show FIFO mode description when FIFO is selected', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText(/自动核销将按照应收账款到期日期从早到晚的顺序/)).toBeInTheDocument()
      })
    })

    it('should switch to manual mode when manual button is clicked', async () => {
      const { user } = renderWithProviders(<ReceiptReconcilePage />, {
        route: '/finance/receipts/rv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('手动核销')).toBeInTheDocument()
      })

      await user.click(screen.getByText('手动核销'))

      await waitFor(() => {
        expect(screen.getByText(/手动核销允许您选择要核销的应收账款/)).toBeInTheDocument()
      })
    })
  })

  describe('Receivables Table Display', () => {
    it('should display receivables table header', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('应收账款编号')).toBeInTheDocument()
        expect(screen.getByText('来源单据')).toBeInTheDocument()
        expect(screen.getByText('总金额')).toBeInTheDocument()
        expect(screen.getByText('待收金额')).toBeInTheDocument()
        expect(screen.getByText('到期日')).toBeInTheDocument()
      })
    })

    it('should display receivable numbers in the table', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('AR-2024-0001')).toBeInTheDocument()
        expect(screen.getByText('AR-2024-0002')).toBeInTheDocument()
      })
    })

    it('should display source numbers in the table', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
        expect(screen.getByText('SO-2024-0002')).toBeInTheDocument()
      })
    })

    it('should display FIFO allocation preview column in FIFO mode', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('预计核销')).toBeInTheDocument()
      })
    })

    it('should display receivable status tags', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('待收款')).toBeInTheDocument()
        expect(screen.getByText('部分收款')).toBeInTheDocument()
      })
    })
  })

  describe('Reconciliation Summary', () => {
    it('should display available allocation amount', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('可核销金额：')).toBeInTheDocument()
      })
    })

    it('should display expected allocation amount in FIFO mode', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('预计核销金额：')).toBeInTheDocument()
      })
    })

    it('should display remaining amount after reconciliation', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('核销后剩余：')).toBeInTheDocument()
      })
    })
  })

  describe('Reconcile Action', () => {
    it('should display confirm reconcile button', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('确认核销')).toBeInTheDocument()
      })
    })

    it('should display cancel button', async () => {
      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('取消')).toBeInTheDocument()
      })
    })

    it('should call reconcile API when confirm button is clicked', async () => {
      const { user } = renderWithProviders(<ReceiptReconcilePage />, {
        route: '/finance/receipts/rv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('确认核销')).toBeInTheDocument()
      })

      const confirmButton = screen.getByText('确认核销')
      await user.click(confirmButton)

      await waitFor(() => {
        expect(mockFinanceApiInstance.postFinanceReceiptsIdReconcile).toHaveBeenCalled()
      })
    })

    it('should show success toast after successful reconciliation', async () => {
      const { user } = renderWithProviders(<ReceiptReconcilePage />, {
        route: '/finance/receipts/rv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('确认核销')).toBeInTheDocument()
      })

      await user.click(screen.getByText('确认核销'))

      await waitFor(() => {
        expect(Toast.success).toHaveBeenCalledWith('核销成功')
      })
    })
  })

  describe('Navigation', () => {
    it('should navigate back when clicking back button', async () => {
      const { user } = renderWithProviders(<ReceiptReconcilePage />, {
        route: '/finance/receipts/rv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('返回')).toBeInTheDocument()
      })

      await user.click(screen.getByText('返回'))

      expect(mockNavigate).toHaveBeenCalledWith('/finance/receivables')
    })

    it('should navigate back when clicking cancel button', async () => {
      const { user } = renderWithProviders(<ReceiptReconcilePage />, {
        route: '/finance/receipts/rv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('取消')).toBeInTheDocument()
      })

      await user.click(screen.getByText('取消'))

      expect(mockNavigate).toHaveBeenCalledWith('/finance/receivables')
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when voucher load fails', async () => {
      mockFinanceApiInstance.getFinanceReceiptsId.mockResolvedValueOnce({
        success: false,
        error: '加载收款单失败',
      })

      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('加载收款单失败')
      })
    })

    it('should navigate away when voucher not found', async () => {
      mockFinanceApiInstance.getFinanceReceiptsId.mockResolvedValueOnce({
        success: false,
        data: null,
      })

      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/finance/receivables')
      })
    })

    it('should handle reconciliation API failure', async () => {
      mockFinanceApiInstance.postFinanceReceiptsIdReconcile.mockRejectedValueOnce(
        new Error('Network error')
      )

      const { user } = renderWithProviders(<ReceiptReconcilePage />, {
        route: '/finance/receipts/rv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('确认核销')).toBeInTheDocument()
      })

      await user.click(screen.getByText('确认核销'))

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('核销请求失败')
      })
    })

    it('should handle reconciliation response failure', async () => {
      mockFinanceApiInstance.postFinanceReceiptsIdReconcile.mockResolvedValueOnce({
        success: false,
        error: '核销失败',
      })

      const { user } = renderWithProviders(<ReceiptReconcilePage />, {
        route: '/finance/receipts/rv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('确认核销')).toBeInTheDocument()
      })

      await user.click(screen.getByText('确认核销'))

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('核销失败')
      })
    })
  })
})

describe('ReceiptReconcilePage - Voucher Status States', () => {
  let mockFinanceApiInstance: {
    getFinanceReceiptsId: ReturnType<typeof vi.fn>
    getFinanceReceivables: ReturnType<typeof vi.fn>
    postFinanceReceiptsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinanceReceiptsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinanceReceivables: vi.fn().mockResolvedValue(createMockReceivablesResponse()),
      postFinanceReceiptsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  describe('Draft Voucher', () => {
    it('should show warning banner for draft voucher', async () => {
      mockFinanceApiInstance.getFinanceReceiptsId.mockResolvedValueOnce(
        createMockVoucherResponse(mockDraftVoucher)
      )

      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText(/该收款单尚未确认/)).toBeInTheDocument()
      })
    })

    it('should display draft status tag', async () => {
      mockFinanceApiInstance.getFinanceReceiptsId.mockResolvedValueOnce(
        createMockVoucherResponse(mockDraftVoucher)
      )

      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('草稿')).toBeInTheDocument()
      })
    })
  })

  describe('Fully Allocated Voucher', () => {
    it('should show success banner for fully allocated voucher', async () => {
      mockFinanceApiInstance.getFinanceReceiptsId.mockResolvedValueOnce(
        createMockVoucherResponse(mockAllocatedVoucher)
      )

      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText(/该收款单已完全核销/)).toBeInTheDocument()
      })
    })

    it('should display allocated status tag', async () => {
      mockFinanceApiInstance.getFinanceReceiptsId.mockResolvedValueOnce(
        createMockVoucherResponse(mockAllocatedVoucher)
      )

      renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('已核销')).toBeInTheDocument()
      })
    })
  })
})

describe('ReceiptReconcilePage - Empty Receivables', () => {
  let mockFinanceApiInstance: {
    getFinanceReceiptsId: ReturnType<typeof vi.fn>
    getFinanceReceivables: ReturnType<typeof vi.fn>
    postFinanceReceiptsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinanceReceiptsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinanceReceivables: vi.fn().mockResolvedValue(createMockReceivablesResponse([])),
      postFinanceReceiptsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  it('should show empty state when no receivables are available', async () => {
    renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

    await waitFor(() => {
      expect(screen.getByText('该客户暂无待核销的应收账款')).toBeInTheDocument()
    })
  })
})

describe('ReceiptReconcilePage - Reconcile Result Display', () => {
  let mockFinanceApiInstance: {
    getFinanceReceiptsId: ReturnType<typeof vi.fn>
    getFinanceReceivables: ReturnType<typeof vi.fn>
    postFinanceReceiptsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinanceReceiptsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinanceReceivables: vi.fn().mockResolvedValue(createMockReceivablesResponse()),
      postFinanceReceiptsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  it('should display reconciliation result after successful reconcile', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      expect(screen.getByText('核销完成')).toBeInTheDocument()
    })
  })

  it('should show fully reconciled banner when voucher is fully allocated', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      expect(screen.getByText('收款单已完全核销')).toBeInTheDocument()
    })
  })

  it('should display updated receivables table in result', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      expect(screen.getByText('已核销应收账款')).toBeInTheDocument()
    })
  })

  it('should display back to list button in result view', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      expect(screen.getByText('返回列表')).toBeInTheDocument()
    })
  })
})

describe('ReceiptReconcilePage - Manual Mode', () => {
  let mockFinanceApiInstance: {
    getFinanceReceiptsId: ReturnType<typeof vi.fn>
    getFinanceReceivables: ReturnType<typeof vi.fn>
    postFinanceReceiptsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinanceReceiptsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinanceReceivables: vi.fn().mockResolvedValue(createMockReceivablesResponse()),
      postFinanceReceiptsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  it('should show allocation amount column in manual mode', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('手动核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('手动核销'))

    await waitFor(() => {
      expect(screen.getByText('核销金额')).toBeInTheDocument()
    })
  })

  it('should show selected allocation amount label in manual mode', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('手动核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('手动核销'))

    await waitFor(() => {
      expect(screen.getByText('已选核销金额：')).toBeInTheDocument()
    })
  })

  it('should show error when trying to reconcile without selection in manual mode', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('手动核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('手动核销'))

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    // The confirm button should be disabled when no selection is made
    const confirmButton = screen.getByText('确认核销').closest('button')
    expect(confirmButton).toHaveAttribute('disabled')
  })
})

describe('ReceiptReconcilePage - API Integration', () => {
  let mockFinanceApiInstance: {
    getFinanceReceiptsId: ReturnType<typeof vi.fn>
    getFinanceReceivables: ReturnType<typeof vi.fn>
    postFinanceReceiptsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinanceReceiptsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinanceReceivables: vi.fn().mockResolvedValue(createMockReceivablesResponse()),
      postFinanceReceiptsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  it('should call getFinanceReceiptsId on mount', async () => {
    renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

    await waitFor(() => {
      expect(mockFinanceApiInstance.getFinanceReceiptsId).toHaveBeenCalledWith('rv-001')
    })
  })

  it('should call getFinanceReceivables with customer_id filter', async () => {
    renderWithProviders(<ReceiptReconcilePage />, { route: '/finance/receipts/rv-001/reconcile' })

    await waitFor(() => {
      expect(mockFinanceApiInstance.getFinanceReceivables).toHaveBeenCalledWith(
        expect.objectContaining({
          customer_id: 'cust-001',
        })
      )
    })
  })

  it('should send FIFO strategy when in FIFO mode', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      expect(mockFinanceApiInstance.postFinanceReceiptsIdReconcile).toHaveBeenCalledWith(
        'rv-001',
        expect.objectContaining({
          strategy_type: 'FIFO',
        })
      )
    })
  })

  it('should reload data after successful reconciliation', async () => {
    const { user } = renderWithProviders(<ReceiptReconcilePage />, {
      route: '/finance/receipts/rv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    // Clear call counts
    mockFinanceApiInstance.getFinanceReceiptsId.mockClear()

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      // Should reload voucher data after reconciliation
      expect(mockFinanceApiInstance.getFinanceReceiptsId).toHaveBeenCalled()
    })
  })
})
