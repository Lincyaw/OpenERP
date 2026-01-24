/**
 * Payment Reconcile Page Tests (P4-QA-005)
 *
 * These tests verify the frontend components for the Payment Reconciliation page:
 * - Page layout (voucher details card, reconciliation section)
 * - Voucher information display
 * - Mode switching (FIFO vs MANUAL)
 * - Payables table display
 * - Allocation selection and amount input
 * - Reconciliation summary display
 * - Reconcile button and result handling
 * - Navigation (back)
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import PaymentReconcilePage from './PaymentReconcile'
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
    useParams: () => ({ id: 'pv-001' }),
  }
})

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample payment voucher data
const mockPaymentVoucher = {
  id: 'pv-001',
  voucher_number: 'PV-2024-0001',
  supplier_id: 'supp-001',
  supplier_name: '测试供应商A',
  amount: 8000.0,
  allocated_amount: 0,
  unallocated_amount: 8000.0,
  payment_method: 'BANK_TRANSFER',
  payment_date: '2024-01-25',
  status: 'CONFIRMED',
  allocations: [],
}

const mockAllocatedVoucher = {
  ...mockPaymentVoucher,
  status: 'ALLOCATED',
  allocated_amount: 8000.0,
  unallocated_amount: 0,
}

const mockDraftVoucher = {
  ...mockPaymentVoucher,
  status: 'DRAFT',
}

// Sample payable data
const mockPayables = [
  {
    id: 'pay-001',
    payable_number: 'AP-2024-0001',
    supplier_id: 'supp-001',
    supplier_name: '测试供应商A',
    source_number: 'PO-2024-0001',
    source_type: 'PURCHASE_ORDER',
    total_amount: 5000.0,
    paid_amount: 0,
    outstanding_amount: 5000.0,
    status: 'PENDING',
    due_date: '2024-02-15',
  },
  {
    id: 'pay-002',
    payable_number: 'AP-2024-0002',
    supplier_id: 'supp-001',
    supplier_name: '测试供应商A',
    source_number: 'PO-2024-0002',
    source_type: 'PURCHASE_ORDER',
    total_amount: 4000.0,
    paid_amount: 1000.0,
    outstanding_amount: 3000.0,
    status: 'PARTIAL',
    due_date: '2024-02-20',
  },
]

// Mock API response helpers
const createMockVoucherResponse = (voucher = mockPaymentVoucher) => ({
  success: true,
  data: voucher,
})

const createMockPayablesResponse = (payables = mockPayables) => ({
  success: true,
  data: payables,
  meta: {
    total: payables.length,
    page: 1,
    page_size: 100,
    total_pages: 1,
  },
})

const createMockReconcileResponse = () => ({
  success: true,
  data: {
    voucher: {
      ...mockPaymentVoucher,
      status: 'ALLOCATED',
      allocated_amount: 8000.0,
      unallocated_amount: 0,
    },
    total_reconciled: 8000.0,
    remaining_unallocated: 0,
    fully_reconciled: true,
    updated_payables: [
      { ...mockPayables[0], paid_amount: 5000.0, outstanding_amount: 0, status: 'PAID' },
      { ...mockPayables[1], paid_amount: 4000.0, outstanding_amount: 0, status: 'PAID' },
    ],
  },
})

describe('PaymentReconcilePage', () => {
  let mockFinanceApiInstance: {
    getFinancePaymentsId: ReturnType<typeof vi.fn>
    getFinancePayables: ReturnType<typeof vi.fn>
    postFinancePaymentsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock finance API
    mockFinanceApiInstance = {
      getFinancePaymentsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse()),
      postFinancePaymentsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  describe('Page Layout', () => {
    it('should display page title', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('付款核销')).toBeInTheDocument()
      })
    })

    it('should display back button', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('返回')).toBeInTheDocument()
      })
    })

    it('should display refresh button', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('刷新')).toBeInTheDocument()
      })
    })

    it('should display voucher information section', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('付款单信息')).toBeInTheDocument()
      })
    })

    it('should display pending payables section for confirmed voucher', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('待核销应付账款')).toBeInTheDocument()
      })
    })
  })

  describe('Voucher Details Display', () => {
    it('should display voucher number', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('PV-2024-0001')).toBeInTheDocument()
      })
    })

    it('should display supplier name', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('测试供应商A')).toBeInTheDocument()
      })
    })

    it('should display voucher status tag', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('已确认')).toBeInTheDocument()
      })
    })

    it('should display payment method', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('银行转账')).toBeInTheDocument()
      })
    })

    it('should display payment amount formatted', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        // ¥8,000.00 formatted currency appears multiple times
        // (payment amount, allocated amount, unallocated amount, etc.)
        const amountElements = screen.getAllByText('¥8,000.00')
        expect(amountElements.length).toBeGreaterThanOrEqual(1)
      })
    })

    it('should display unallocated amount', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        // Label for unallocated amount section
        expect(screen.getByText('未核销金额')).toBeInTheDocument()
      })
    })
  })

  describe('Reconciliation Mode Selection', () => {
    it('should display FIFO mode button', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('自动核销 (FIFO)')).toBeInTheDocument()
      })
    })

    it('should display Manual mode button', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('手动核销')).toBeInTheDocument()
      })
    })

    it('should show FIFO mode description when FIFO is selected', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(
          screen.getByText(/自动核销将按照应付账款到期日期从早到晚的顺序/)
        ).toBeInTheDocument()
      })
    })

    it('should switch to manual mode when manual button is clicked', async () => {
      const { user } = renderWithProviders(<PaymentReconcilePage />, {
        route: '/finance/payments/pv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('手动核销')).toBeInTheDocument()
      })

      await user.click(screen.getByText('手动核销'))

      await waitFor(() => {
        expect(screen.getByText(/手动核销允许您选择要核销的应付账款/)).toBeInTheDocument()
      })
    })
  })

  describe('Payables Table Display', () => {
    it('should display payables table header', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('应付账款编号')).toBeInTheDocument()
        expect(screen.getByText('来源单据')).toBeInTheDocument()
        expect(screen.getByText('总金额')).toBeInTheDocument()
        expect(screen.getByText('待付金额')).toBeInTheDocument()
        expect(screen.getByText('到期日')).toBeInTheDocument()
      })
    })

    it('should display payable numbers in the table', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('AP-2024-0001')).toBeInTheDocument()
        expect(screen.getByText('AP-2024-0002')).toBeInTheDocument()
      })
    })

    it('should display source numbers in the table', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('PO-2024-0001')).toBeInTheDocument()
        expect(screen.getByText('PO-2024-0002')).toBeInTheDocument()
      })
    })

    it('should display FIFO allocation preview column in FIFO mode', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('预计核销')).toBeInTheDocument()
      })
    })

    it('should display payable status tags', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('待付款')).toBeInTheDocument()
        expect(screen.getByText('部分付款')).toBeInTheDocument()
      })
    })
  })

  describe('Reconciliation Summary', () => {
    it('should display available allocation amount', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('可核销金额：')).toBeInTheDocument()
      })
    })

    it('should display expected allocation amount in FIFO mode', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('预计核销金额：')).toBeInTheDocument()
      })
    })

    it('should display remaining amount after reconciliation', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('核销后剩余：')).toBeInTheDocument()
      })
    })
  })

  describe('Reconcile Action', () => {
    it('should display confirm reconcile button', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('确认核销')).toBeInTheDocument()
      })
    })

    it('should display cancel button', async () => {
      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('取消')).toBeInTheDocument()
      })
    })

    it('should call reconcile API when confirm button is clicked', async () => {
      const { user } = renderWithProviders(<PaymentReconcilePage />, {
        route: '/finance/payments/pv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('确认核销')).toBeInTheDocument()
      })

      const confirmButton = screen.getByText('确认核销')
      await user.click(confirmButton)

      await waitFor(() => {
        expect(mockFinanceApiInstance.postFinancePaymentsIdReconcile).toHaveBeenCalled()
      })
    })

    it('should show success toast after successful reconciliation', async () => {
      const { user } = renderWithProviders(<PaymentReconcilePage />, {
        route: '/finance/payments/pv-001/reconcile',
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
      const { user } = renderWithProviders(<PaymentReconcilePage />, {
        route: '/finance/payments/pv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('返回')).toBeInTheDocument()
      })

      await user.click(screen.getByText('返回'))

      expect(mockNavigate).toHaveBeenCalledWith('/finance/payables')
    })

    it('should navigate back when clicking cancel button', async () => {
      const { user } = renderWithProviders(<PaymentReconcilePage />, {
        route: '/finance/payments/pv-001/reconcile',
      })

      await waitFor(() => {
        expect(screen.getByText('取消')).toBeInTheDocument()
      })

      await user.click(screen.getByText('取消'))

      expect(mockNavigate).toHaveBeenCalledWith('/finance/payables')
    })
  })

  describe('Error Handling', () => {
    it('should show error toast when voucher load fails', async () => {
      mockFinanceApiInstance.getFinancePaymentsId.mockResolvedValueOnce({
        success: false,
        error: '加载付款单失败',
      })

      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('加载付款单失败')
      })
    })

    it('should navigate away when voucher not found', async () => {
      mockFinanceApiInstance.getFinancePaymentsId.mockResolvedValueOnce({
        success: false,
        data: null,
      })

      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/finance/payables')
      })
    })

    it('should handle reconciliation API failure', async () => {
      mockFinanceApiInstance.postFinancePaymentsIdReconcile.mockRejectedValueOnce(
        new Error('Network error')
      )

      const { user } = renderWithProviders(<PaymentReconcilePage />, {
        route: '/finance/payments/pv-001/reconcile',
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
      mockFinanceApiInstance.postFinancePaymentsIdReconcile.mockResolvedValueOnce({
        success: false,
        error: '核销失败',
      })

      const { user } = renderWithProviders(<PaymentReconcilePage />, {
        route: '/finance/payments/pv-001/reconcile',
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

describe('PaymentReconcilePage - Voucher Status States', () => {
  let mockFinanceApiInstance: {
    getFinancePaymentsId: ReturnType<typeof vi.fn>
    getFinancePayables: ReturnType<typeof vi.fn>
    postFinancePaymentsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinancePaymentsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse()),
      postFinancePaymentsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  describe('Draft Voucher', () => {
    it('should show warning banner for draft voucher', async () => {
      mockFinanceApiInstance.getFinancePaymentsId.mockResolvedValueOnce(
        createMockVoucherResponse(mockDraftVoucher)
      )

      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText(/该付款单尚未确认/)).toBeInTheDocument()
      })
    })

    it('should display draft status tag', async () => {
      mockFinanceApiInstance.getFinancePaymentsId.mockResolvedValueOnce(
        createMockVoucherResponse(mockDraftVoucher)
      )

      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('草稿')).toBeInTheDocument()
      })
    })
  })

  describe('Fully Allocated Voucher', () => {
    it('should show success banner for fully allocated voucher', async () => {
      mockFinanceApiInstance.getFinancePaymentsId.mockResolvedValueOnce(
        createMockVoucherResponse(mockAllocatedVoucher)
      )

      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText(/该付款单已完全核销/)).toBeInTheDocument()
      })
    })

    it('should display allocated status tag', async () => {
      mockFinanceApiInstance.getFinancePaymentsId.mockResolvedValueOnce(
        createMockVoucherResponse(mockAllocatedVoucher)
      )

      renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

      await waitFor(() => {
        expect(screen.getByText('已核销')).toBeInTheDocument()
      })
    })
  })
})

describe('PaymentReconcilePage - Empty Payables', () => {
  let mockFinanceApiInstance: {
    getFinancePaymentsId: ReturnType<typeof vi.fn>
    getFinancePayables: ReturnType<typeof vi.fn>
    postFinancePaymentsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinancePaymentsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse([])),
      postFinancePaymentsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  it('should show empty state when no payables are available', async () => {
    renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

    await waitFor(() => {
      expect(screen.getByText('该供应商暂无待核销的应付账款')).toBeInTheDocument()
    })
  })
})

describe('PaymentReconcilePage - Reconcile Result Display', () => {
  let mockFinanceApiInstance: {
    getFinancePaymentsId: ReturnType<typeof vi.fn>
    getFinancePayables: ReturnType<typeof vi.fn>
    postFinancePaymentsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinancePaymentsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse()),
      postFinancePaymentsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  it('should display reconciliation result after successful reconcile', async () => {
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
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
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      expect(screen.getByText('付款单已完全核销')).toBeInTheDocument()
    })
  })

  it('should display updated payables table in result', async () => {
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      expect(screen.getByText('已核销应付账款')).toBeInTheDocument()
    })
  })

  it('should display back to list button in result view', async () => {
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
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

describe('PaymentReconcilePage - Manual Mode', () => {
  let mockFinanceApiInstance: {
    getFinancePaymentsId: ReturnType<typeof vi.fn>
    getFinancePayables: ReturnType<typeof vi.fn>
    postFinancePaymentsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinancePaymentsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse()),
      postFinancePaymentsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  it('should show allocation amount column in manual mode', async () => {
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
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
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
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
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
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

describe('PaymentReconcilePage - API Integration', () => {
  let mockFinanceApiInstance: {
    getFinancePaymentsId: ReturnType<typeof vi.fn>
    getFinancePayables: ReturnType<typeof vi.fn>
    postFinancePaymentsIdReconcile: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      getFinancePaymentsId: vi.fn().mockResolvedValue(createMockVoucherResponse()),
      getFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse()),
      postFinancePaymentsIdReconcile: vi.fn().mockResolvedValue(createMockReconcileResponse()),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
  })

  it('should call getFinancePaymentsId on mount', async () => {
    renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

    await waitFor(() => {
      expect(mockFinanceApiInstance.getFinancePaymentsId).toHaveBeenCalledWith('pv-001')
    })
  })

  it('should call getFinancePayables with supplier_id filter', async () => {
    renderWithProviders(<PaymentReconcilePage />, { route: '/finance/payments/pv-001/reconcile' })

    await waitFor(() => {
      expect(mockFinanceApiInstance.getFinancePayables).toHaveBeenCalledWith(
        expect.objectContaining({
          supplier_id: 'supp-001',
        })
      )
    })
  })

  it('should send FIFO strategy when in FIFO mode', async () => {
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      expect(mockFinanceApiInstance.postFinancePaymentsIdReconcile).toHaveBeenCalledWith(
        'pv-001',
        expect.objectContaining({
          strategy_type: 'FIFO',
        })
      )
    })
  })

  it('should reload data after successful reconciliation', async () => {
    const { user } = renderWithProviders(<PaymentReconcilePage />, {
      route: '/finance/payments/pv-001/reconcile',
    })

    await waitFor(() => {
      expect(screen.getByText('确认核销')).toBeInTheDocument()
    })

    // Clear call counts
    mockFinanceApiInstance.getFinancePaymentsId.mockClear()

    await user.click(screen.getByText('确认核销'))

    await waitFor(() => {
      // Should reload voucher data after reconciliation
      expect(mockFinanceApiInstance.getFinancePaymentsId).toHaveBeenCalled()
    })
  })
})
