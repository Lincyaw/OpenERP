/**
 * CustomerForm Integration Tests (P1-INT-002)
 *
 * These tests verify the frontend-backend integration for customer creation and editing:
 * - Form validation
 * - Create customer workflow
 * - Edit customer workflow
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import { CustomerForm } from '@/features/partner/CustomerForm'
import * as customersApi from '@/api/customers/customers'
import { Toast } from '@douyinfe/semi-ui'

// Mock the customers API module
vi.mock('@/api/customers/customers', () => ({
  getCustomers: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Mock useNavigate
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

// Sample customer data
const mockCustomer = {
  id: '550e8400-e29b-41d4-a716-446655440001',
  code: 'CUST-001',
  name: '测试客户有限公司',
  short_name: '测试客户',
  type: 'organization' as const,
  level: 'gold' as const,
  contact_name: '张三',
  phone: '13800138001',
  email: 'test@example.com',
  tax_id: '91440300MA5D0PKT4J',
  country: '中国',
  province: '广东省',
  city: '深圳市',
  postal_code: '518000',
  address: '南山区科技园',
  credit_limit: 50000,
  sort_order: 1,
  notes: '测试备注',
  status: 'active' as const,
  created_at: '2024-01-15T10:30:00Z',
}

// Mock API response helpers
const createSuccessResponse = (data: unknown) => ({
  success: true,
  data,
})

const createErrorResponse = (message: string, code = 'ERR_VALIDATION') => ({
  success: false,
  error: {
    code,
    message,
    request_id: 'test-req-id',
    timestamp: new Date().toISOString(),
  },
})

describe('CustomerForm', () => {
  let mockApi: {
    postPartnerCustomers: ReturnType<typeof vi.fn>
    putPartnerCustomersId: ReturnType<typeof vi.fn>
    getPartnerCustomersId: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockApi = {
      postPartnerCustomers: vi.fn().mockResolvedValue(createSuccessResponse(mockCustomer)),
      putPartnerCustomersId: vi.fn().mockResolvedValue(createSuccessResponse(mockCustomer)),
      getPartnerCustomersId: vi.fn().mockResolvedValue(createSuccessResponse(mockCustomer)),
    }

    vi.mocked(customersApi.getCustomers).mockReturnValue(mockApi as unknown as ReturnType<typeof customersApi.getCustomers>)
  })

  describe('Create Mode', () => {
    it('should display create mode title', () => {
      renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      expect(screen.getByText('新增客户')).toBeInTheDocument()
    })

    it('should display all form sections', () => {
      renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      expect(screen.getByText('基本信息')).toBeInTheDocument()
      expect(screen.getByText('联系信息')).toBeInTheDocument()
      expect(screen.getByText('地址信息')).toBeInTheDocument()
      expect(screen.getByText('其他设置')).toBeInTheDocument()
    })

    it('should display required field labels', () => {
      renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      expect(screen.getByText('客户编码')).toBeInTheDocument()
      expect(screen.getByText('客户名称')).toBeInTheDocument()
      expect(screen.getByText('客户类型')).toBeInTheDocument()
    })

    it('should have editable code field in create mode', () => {
      renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      const codeInput = screen.getByPlaceholderText('请输入客户编码')
      expect(codeInput).not.toBeDisabled()
    })

    it('should have default country value', () => {
      renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      const countryInput = screen.getByPlaceholderText('请输入国家')
      expect(countryInput).toHaveValue('中国')
    })

    it('should display create button', () => {
      renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      expect(screen.getByRole('button', { name: /创建/i })).toBeInTheDocument()
    })

    it('should display cancel button', () => {
      renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      expect(screen.getByRole('button', { name: /取消/i })).toBeInTheDocument()
    })

    it('should call create API on form submit', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      // Fill required fields
      await user.type(screen.getByPlaceholderText('请输入客户编码'), 'NEW-CUST-001')
      await user.type(screen.getByPlaceholderText('请输入客户全称'), '新客户有限公司')

      // Submit form
      await user.click(screen.getByRole('button', { name: /创建/i }))

      // Wait for API call
      await waitFor(() => {
        expect(mockApi.postPartnerCustomers).toHaveBeenCalledWith(
          expect.objectContaining({
            code: 'NEW-CUST-001',
            name: '新客户有限公司',
            type: 'individual',
          })
        )
      })
    })

    it('should navigate to customers list after successful create', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      // Fill required fields
      await user.type(screen.getByPlaceholderText('请输入客户编码'), 'NEW-CUST-001')
      await user.type(screen.getByPlaceholderText('请输入客户全称'), '新客户有限公司')

      // Submit form
      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/partner/customers')
      })
    })

    it('should navigate back on cancel', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      await user.click(screen.getByRole('button', { name: /取消/i }))

      expect(mockNavigate).toHaveBeenCalledWith('/partner/customers')
    })
  })

  describe('Edit Mode', () => {
    it('should display edit mode title', () => {
      renderWithProviders(
        <CustomerForm customerId={mockCustomer.id} initialData={mockCustomer} />,
        { route: `/partner/customers/${mockCustomer.id}/edit` }
      )

      expect(screen.getByText('编辑客户')).toBeInTheDocument()
    })

    it('should populate form with initial data', () => {
      renderWithProviders(
        <CustomerForm customerId={mockCustomer.id} initialData={mockCustomer} />,
        { route: `/partner/customers/${mockCustomer.id}/edit` }
      )

      expect(screen.getByPlaceholderText('请输入客户编码')).toHaveValue('CUST-001')
      expect(screen.getByPlaceholderText('请输入客户全称')).toHaveValue('测试客户有限公司')
      expect(screen.getByPlaceholderText('请输入客户简称 (可选)')).toHaveValue('测试客户')
      expect(screen.getByPlaceholderText('请输入联系人姓名')).toHaveValue('张三')
      expect(screen.getByPlaceholderText('请输入联系电话')).toHaveValue('13800138001')
    })

    it('should have disabled code field in edit mode', () => {
      renderWithProviders(
        <CustomerForm customerId={mockCustomer.id} initialData={mockCustomer} />,
        { route: `/partner/customers/${mockCustomer.id}/edit` }
      )

      const codeInput = screen.getByPlaceholderText('请输入客户编码')
      expect(codeInput).toBeDisabled()
    })

    it('should display save button in edit mode', () => {
      renderWithProviders(
        <CustomerForm customerId={mockCustomer.id} initialData={mockCustomer} />,
        { route: `/partner/customers/${mockCustomer.id}/edit` }
      )

      expect(screen.getByRole('button', { name: /保存/i })).toBeInTheDocument()
    })

    it('should show customer level field in edit mode', () => {
      renderWithProviders(
        <CustomerForm customerId={mockCustomer.id} initialData={mockCustomer} />,
        { route: `/partner/customers/${mockCustomer.id}/edit` }
      )

      expect(screen.getByText('客户等级')).toBeInTheDocument()
    })

    it('should call update API on form submit', async () => {
      const { user } = renderWithProviders(
        <CustomerForm customerId={mockCustomer.id} initialData={mockCustomer} />,
        { route: `/partner/customers/${mockCustomer.id}/edit` }
      )

      // Modify name
      const nameInput = screen.getByPlaceholderText('请输入客户全称')
      await user.clear(nameInput)
      await user.type(nameInput, '更新的客户名称')

      // Submit form
      await user.click(screen.getByRole('button', { name: /保存/i }))

      await waitFor(() => {
        expect(mockApi.putPartnerCustomersId).toHaveBeenCalledWith(
          mockCustomer.id,
          expect.objectContaining({
            name: '更新的客户名称',
          })
        )
      })
    })
  })

  describe('Form Validation', () => {
    it('should show validation error for empty required fields', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      // Clear default values and try to submit
      await user.click(screen.getByRole('button', { name: /创建/i }))

      // API should not be called with invalid data
      expect(mockApi.postPartnerCustomers).not.toHaveBeenCalled()
    })

    it('should validate code format (alphanumeric, underscore, hyphen)', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      // Enter invalid code with special characters
      const codeInput = screen.getByPlaceholderText('请输入客户编码')
      await user.type(codeInput, 'INVALID@CODE!')
      await user.type(screen.getByPlaceholderText('请输入客户全称'), '测试')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      // API should not be called
      expect(mockApi.postPartnerCustomers).not.toHaveBeenCalled()
    })

    it('should validate email format', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      // Fill required fields
      await user.type(screen.getByPlaceholderText('请输入客户编码'), 'TEST-001')
      await user.type(screen.getByPlaceholderText('请输入客户全称'), '测试客户')

      // Enter invalid email
      await user.type(screen.getByPlaceholderText('请输入电子邮箱'), 'invalid-email')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      // API should not be called
      expect(mockApi.postPartnerCustomers).not.toHaveBeenCalled()
    })
  })

  describe('Error Handling', () => {
    it('should show error message when create API fails', async () => {
      mockApi.postPartnerCustomers.mockResolvedValueOnce(
        createErrorResponse('客户编码已存在', 'ERR_DUPLICATE')
      )

      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      await user.type(screen.getByPlaceholderText('请输入客户编码'), 'EXISTING-CODE')
      await user.type(screen.getByPlaceholderText('请输入客户全称'), '新客户')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalled()
      })
    })

    it('should show error message when update API fails', async () => {
      mockApi.putPartnerCustomersId.mockResolvedValueOnce(
        createErrorResponse('更新失败', 'ERR_UPDATE')
      )

      const { user } = renderWithProviders(
        <CustomerForm customerId={mockCustomer.id} initialData={mockCustomer} />,
        { route: `/partner/customers/${mockCustomer.id}/edit` }
      )

      const nameInput = screen.getByPlaceholderText('请输入客户全称')
      await user.clear(nameInput)
      await user.type(nameInput, '新名称')

      await user.click(screen.getByRole('button', { name: /保存/i }))

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalled()
      })
    })
  })

  describe('Contact Information', () => {
    it('should accept contact details', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      await user.type(screen.getByPlaceholderText('请输入客户编码'), 'CONTACT-TEST')
      await user.type(screen.getByPlaceholderText('请输入客户全称'), '联系测试')
      await user.type(screen.getByPlaceholderText('请输入联系人姓名'), '李四')
      await user.type(screen.getByPlaceholderText('请输入联系电话'), '13900139000')
      await user.type(screen.getByPlaceholderText('请输入电子邮箱'), 'lisi@example.com')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(mockApi.postPartnerCustomers).toHaveBeenCalledWith(
          expect.objectContaining({
            contact_name: '李四',
            phone: '13900139000',
            email: 'lisi@example.com',
          })
        )
      })
    })
  })

  describe('Address Information', () => {
    it('should accept address details', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      await user.type(screen.getByPlaceholderText('请输入客户编码'), 'ADDR-TEST')
      await user.type(screen.getByPlaceholderText('请输入客户全称'), '地址测试')
      await user.type(screen.getByPlaceholderText('请输入省份'), '浙江省')
      await user.type(screen.getByPlaceholderText('请输入城市'), '杭州市')
      await user.type(screen.getByPlaceholderText('请输入详细地址'), '西湖区文三路100号')
      await user.type(screen.getByPlaceholderText('请输入邮政编码'), '310000')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(mockApi.postPartnerCustomers).toHaveBeenCalledWith(
          expect.objectContaining({
            province: '浙江省',
            city: '杭州市',
            address: '西湖区文三路100号',
            postal_code: '310000',
          })
        )
      })
    })
  })

  describe('API Request Payload', () => {
    it('should send correct payload structure for create', async () => {
      const { user } = renderWithProviders(<CustomerForm />, { route: '/partner/customers/new' })

      await user.type(screen.getByPlaceholderText('请输入客户编码'), 'API-TEST-001')
      await user.type(screen.getByPlaceholderText('请输入客户全称'), 'API测试客户')
      await user.type(screen.getByPlaceholderText('请输入客户简称 (可选)'), 'API测试')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(mockApi.postPartnerCustomers).toHaveBeenCalledWith(
          expect.objectContaining({
            code: 'API-TEST-001',
            name: 'API测试客户',
            short_name: 'API测试',
            type: 'individual',
          })
        )
      })
    })
  })
})
