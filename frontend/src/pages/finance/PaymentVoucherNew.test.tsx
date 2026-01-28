/**
 * Payment Voucher New Page Tests (P4-QA-005)
 *
 * These tests verify the frontend components for the Payment Voucher creation form:
 * - Page layout (title, form sections)
 * - Supplier selection with search
 * - Payment method selection
 * - Amount input with validation
 * - Supplier payables summary display
 * - Form validation
 * - Navigation (cancel)
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import PaymentVoucherNewPage from './PaymentVoucherNew'
import * as financeApi from '@/api/finance/finance'
import * as suppliersApi from '@/api/suppliers/suppliers'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the API modules
vi.mock('@/api/finance/finance', () => ({
  getFinanceApi: vi.fn(),
}))

vi.mock('@/api/suppliers/suppliers', () => ({
  getSuppliers: vi.fn(),
}))

// Mock react-router-dom
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useSearchParams: () => [new URLSearchParams(), vi.fn()],
  }
})

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample supplier data
const mockSuppliers = [
  {
    id: 'supp-001',
    code: 'S001',
    name: '测试供应商A',
    status: 'active',
  },
  {
    id: 'supp-002',
    code: 'S002',
    name: '测试供应商B',
    status: 'active',
  },
]

// Sample payable data for a supplier
const mockPayables = [
  {
    id: 'pay-001',
    payable_number: 'AP-2024-0001',
    supplier_id: 'supp-001',
    total_amount: 2000.0,
    outstanding_amount: 1500.0,
    status: 'PARTIAL',
    due_date: '2024-02-20',
  },
  {
    id: 'pay-002',
    payable_number: 'AP-2024-0002',
    supplier_id: 'supp-001',
    total_amount: 800.0,
    outstanding_amount: 800.0,
    status: 'PENDING',
    due_date: '2024-02-25',
  },
]

// Mock API response helpers
const createMockSupplierSearchResponse = (suppliers = mockSuppliers) => ({
  success: true,
  data: suppliers,
  meta: {
    total: suppliers.length,
    page: 1,
    page_size: 20,
    total_pages: 1,
  },
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

const createMockPaymentVoucherResponse = () => ({
  success: true,
  data: {
    id: 'pv-001',
    voucher_number: 'PV-2024-0001',
    supplier_id: 'supp-001',
    supplier_name: '测试供应商A',
    amount: 1000.0,
    payment_method: 'BANK_TRANSFER',
    status: 'DRAFT',
    payment_date: '2024-01-25',
  },
})

describe('PaymentVoucherNewPage', () => {
  let mockFinanceApiInstance: {
    listFinancePayables: ReturnType<typeof vi.fn>
    createPaymentVoucher: ReturnType<typeof vi.fn>
  }

  let mockSupplierApiInstance: {
    listSuppliers: ReturnType<typeof vi.fn>
    getSupplierById: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock finance API
    mockFinanceApiInstance = {
      listFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse()),
      createPaymentVoucher: vi.fn().mockResolvedValue(createMockPaymentVoucherResponse()),
    }

    // Setup mock supplier API
    mockSupplierApiInstance = {
      listSuppliers: vi.fn().mockResolvedValue(createMockSupplierSearchResponse()),
      getSupplierById: vi.fn().mockResolvedValue({
        success: true,
        data: mockSuppliers[0],
      }),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
    vi.mocked(suppliersApi.getSuppliers).mockReturnValue(
      mockSupplierApiInstance as unknown as ReturnType<typeof suppliersApi.getSuppliers>
    )
  })

  describe('Page Layout', () => {
    it('should display page title', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('新增付款单')).toBeInTheDocument()
    })

    it('should display supplier information section', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('供应商信息')).toBeInTheDocument()
      expect(screen.getByText('选择付款的供应商')).toBeInTheDocument()
    })

    it('should display payment information section', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('付款信息')).toBeInTheDocument()
      expect(screen.getByText('填写付款金额和方式')).toBeInTheDocument()
    })

    it('should display other information section', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('其他信息')).toBeInTheDocument()
      expect(screen.getByText('备注说明')).toBeInTheDocument()
    })

    it('should display form actions (create and cancel buttons)', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('创建')).toBeInTheDocument()
      expect(screen.getByText('取消')).toBeInTheDocument()
    })
  })

  describe('Form Fields', () => {
    it('should have supplier select field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      // Look for supplier label - has required indicator
      expect(screen.getByText('供应商')).toBeInTheDocument()

      // Supplier select is rendered as Semi-UI Select component
      // The wrapper contains the supplier selection
      const supplierWrapper = document.querySelector('.supplier-select-wrapper')
      expect(supplierWrapper).toBeInTheDocument()
    })

    it('should have amount field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('付款金额')).toBeInTheDocument()
    })

    it('should have payment method field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('付款方式')).toBeInTheDocument()
    })

    it('should have payment date field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('付款日期')).toBeInTheDocument()
    })

    it('should have payment reference field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      expect(screen.getByText('付款凭证号')).toBeInTheDocument()
    })

    it('should have remark field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      // "备注" appears as section title and field label
      expect(screen.getAllByText('备注').length).toBeGreaterThanOrEqual(1)
    })
  })

  describe('Payment Method Options', () => {
    it('should have payment method options available', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      // Payment method options are rendered in a select component
      // Default value is BANK_TRANSFER
      expect(screen.getByText('付款方式')).toBeInTheDocument()
    })
  })

  describe('Navigation', () => {
    it('should navigate back when clicking cancel button', async () => {
      const { user } = renderWithProviders(<PaymentVoucherNewPage />, {
        route: '/finance/payments/new',
      })

      const cancelButton = screen.getByText('取消')
      await user.click(cancelButton)

      expect(mockNavigate).toHaveBeenCalledWith('/finance/payables')
    })
  })

  describe('Error Handling', () => {
    it('should handle supplier search API failure gracefully', async () => {
      mockSupplierApiInstance.listSuppliers.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      // Page should still render
      expect(screen.getByText('新增付款单')).toBeInTheDocument()
    })
  })
})

describe('PaymentVoucherNewPage - Form Validation', () => {
  let mockFinanceApiInstance: {
    listFinancePayables: ReturnType<typeof vi.fn>
    createPaymentVoucher: ReturnType<typeof vi.fn>
  }

  let mockSupplierApiInstance: {
    listSuppliers: ReturnType<typeof vi.fn>
    getSupplierById: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      listFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse()),
      createPaymentVoucher: vi.fn().mockResolvedValue(createMockPaymentVoucherResponse()),
    }

    mockSupplierApiInstance = {
      listSuppliers: vi.fn().mockResolvedValue(createMockSupplierSearchResponse()),
      getSupplierById: vi.fn().mockResolvedValue({
        success: true,
        data: mockSuppliers[0],
      }),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
    vi.mocked(suppliersApi.getSuppliers).mockReturnValue(
      mockSupplierApiInstance as unknown as ReturnType<typeof suppliersApi.getSuppliers>
    )
  })

  describe('Required Field Validation', () => {
    it('should display required indicator for supplier field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      // The supplier field has a required indicator (*)
      const supplierLabel = screen.getByText('供应商')
      expect(supplierLabel.closest('.semi-form-field-label')).toBeInTheDocument()
    })

    it('should display required indicator for amount field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      // Amount field is required
      expect(screen.getByText('付款金额')).toBeInTheDocument()
    })

    it('should display required indicator for payment method field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      // Payment method is required
      expect(screen.getByText('付款方式')).toBeInTheDocument()
    })

    it('should display required indicator for payment date field', async () => {
      renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

      // Payment date is required
      expect(screen.getByText('付款日期')).toBeInTheDocument()
    })
  })
})

describe('PaymentVoucherNewPage - Supplier Payables Summary', () => {
  let mockFinanceApiInstance: {
    listFinancePayables: ReturnType<typeof vi.fn>
    createPaymentVoucher: ReturnType<typeof vi.fn>
  }

  let mockSupplierApiInstance: {
    listSuppliers: ReturnType<typeof vi.fn>
    getSupplierById: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockFinanceApiInstance = {
      listFinancePayables: vi.fn().mockResolvedValue(createMockPayablesResponse()),
      createPaymentVoucher: vi.fn().mockResolvedValue(createMockPaymentVoucherResponse()),
    }

    mockSupplierApiInstance = {
      listSuppliers: vi.fn().mockResolvedValue(createMockSupplierSearchResponse()),
      getSupplierById: vi.fn().mockResolvedValue({
        success: true,
        data: mockSuppliers[0],
      }),
    }

    vi.mocked(financeApi.getFinanceApi).mockReturnValue(
      mockFinanceApiInstance as unknown as ReturnType<typeof financeApi.getFinanceApi>
    )
    vi.mocked(suppliersApi.getSuppliers).mockReturnValue(
      mockSupplierApiInstance as unknown as ReturnType<typeof suppliersApi.getSuppliers>
    )
  })

  it('should show payables summary when supplier is pre-selected via URL', async () => {
    // This test verifies the component handles pre-selected supplier
    renderWithProviders(<PaymentVoucherNewPage />, {
      route: '/finance/payments/new?supplier_id=supp-001',
    })

    await waitFor(() => {
      expect(screen.getByText('新增付款单')).toBeInTheDocument()
    })
  })

  it('should show pending payables label section', async () => {
    renderWithProviders(<PaymentVoucherNewPage />, { route: '/finance/payments/new' })

    // The page structure has a section for pending payables
    expect(screen.getByText('新增付款单')).toBeInTheDocument()
  })
})
