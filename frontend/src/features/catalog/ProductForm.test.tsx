/**
 * ProductForm Integration Tests (P1-INT-001)
 *
 * These tests verify the frontend-backend integration for product creation and editing:
 * - Form validation
 * - Create product workflow
 * - Edit product workflow
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import { ProductForm } from '@/features/catalog/ProductForm'
import * as productsApi from '@/api/products/products'
import { Toast } from '@douyinfe/semi-ui'

// Mock the products API module
vi.mock('@/api/products/products', () => ({
  getProducts: vi.fn(),
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

// Sample product data
const mockProduct = {
  id: '550e8400-e29b-41d4-a716-446655440001',
  code: 'SKU-001',
  name: '测试商品',
  unit: '个',
  barcode: '6901234567890',
  description: '商品描述内容',
  purchase_price: 50.0,
  selling_price: 100.0,
  min_stock: 10,
  sort_order: 1,
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

describe('ProductForm', () => {
  let mockApi: {
    postCatalogProducts: ReturnType<typeof vi.fn>
    putCatalogProductsId: ReturnType<typeof vi.fn>
    getCatalogProductsId: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockApi = {
      postCatalogProducts: vi.fn().mockResolvedValue(createSuccessResponse(mockProduct)),
      putCatalogProductsId: vi.fn().mockResolvedValue(createSuccessResponse(mockProduct)),
      getCatalogProductsId: vi.fn().mockResolvedValue(createSuccessResponse(mockProduct)),
    }

    vi.mocked(productsApi.getProducts).mockReturnValue(mockApi as unknown as ReturnType<typeof productsApi.getProducts>)
  })

  describe('Create Mode', () => {
    it('should display create mode title', () => {
      renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      expect(screen.getByText('新增商品')).toBeInTheDocument()
    })

    it('should display all form sections', () => {
      renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      expect(screen.getByText('基本信息')).toBeInTheDocument()
      expect(screen.getByText('价格信息')).toBeInTheDocument()
      expect(screen.getByText('库存设置')).toBeInTheDocument()
    })

    it('should display required field labels', () => {
      renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      expect(screen.getByText('商品编码')).toBeInTheDocument()
      expect(screen.getByText('商品名称')).toBeInTheDocument()
      expect(screen.getByText('单位')).toBeInTheDocument()
    })

    it('should have editable code field in create mode', () => {
      renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      const codeInput = screen.getByPlaceholderText('请输入商品编码 (SKU)')
      expect(codeInput).not.toBeDisabled()
    })

    it('should have default unit value', () => {
      renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      const unitInput = screen.getByPlaceholderText('请输入计量单位')
      expect(unitInput).toHaveValue('个')
    })

    it('should display create button', () => {
      renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      expect(screen.getByRole('button', { name: /创建/i })).toBeInTheDocument()
    })

    it('should display cancel button', () => {
      renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      expect(screen.getByRole('button', { name: /取消/i })).toBeInTheDocument()
    })

    it('should call create API on form submit', async () => {
      const { user } = renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      // Fill required fields
      await user.clear(screen.getByPlaceholderText('请输入商品编码 (SKU)'))
      await user.type(screen.getByPlaceholderText('请输入商品编码 (SKU)'), 'NEW-SKU-001')
      await user.type(screen.getByPlaceholderText('请输入商品名称'), '新商品')

      // Submit form
      await user.click(screen.getByRole('button', { name: /创建/i }))

      // Wait for API call
      await waitFor(() => {
        expect(mockApi.postCatalogProducts).toHaveBeenCalledWith(
          expect.objectContaining({
            code: 'NEW-SKU-001',
            name: '新商品',
            unit: '个',
          })
        )
      })
    })

    it('should navigate to products list after successful create', async () => {
      const { user } = renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      // Fill required fields
      await user.clear(screen.getByPlaceholderText('请输入商品编码 (SKU)'))
      await user.type(screen.getByPlaceholderText('请输入商品编码 (SKU)'), 'NEW-SKU-001')
      await user.type(screen.getByPlaceholderText('请输入商品名称'), '新商品')

      // Submit form
      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/catalog/products')
      })
    })

    it('should navigate back on cancel', async () => {
      const { user } = renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      await user.click(screen.getByRole('button', { name: /取消/i }))

      expect(mockNavigate).toHaveBeenCalledWith('/catalog/products')
    })
  })

  describe('Edit Mode', () => {
    it('should display edit mode title', () => {
      renderWithProviders(
        <ProductForm productId={mockProduct.id} initialData={mockProduct} />,
        { route: `/catalog/products/${mockProduct.id}/edit` }
      )

      expect(screen.getByText('编辑商品')).toBeInTheDocument()
    })

    it('should populate form with initial data', () => {
      renderWithProviders(
        <ProductForm productId={mockProduct.id} initialData={mockProduct} />,
        { route: `/catalog/products/${mockProduct.id}/edit` }
      )

      expect(screen.getByPlaceholderText('请输入商品编码 (SKU)')).toHaveValue('SKU-001')
      expect(screen.getByPlaceholderText('请输入商品名称')).toHaveValue('测试商品')
      expect(screen.getByPlaceholderText('请输入计量单位')).toHaveValue('个')
      expect(screen.getByPlaceholderText('请输入条形码 (可选)')).toHaveValue('6901234567890')
    })

    it('should have disabled code field in edit mode', () => {
      renderWithProviders(
        <ProductForm productId={mockProduct.id} initialData={mockProduct} />,
        { route: `/catalog/products/${mockProduct.id}/edit` }
      )

      const codeInput = screen.getByPlaceholderText('请输入商品编码 (SKU)')
      expect(codeInput).toBeDisabled()
    })

    it('should have disabled unit field in edit mode', () => {
      renderWithProviders(
        <ProductForm productId={mockProduct.id} initialData={mockProduct} />,
        { route: `/catalog/products/${mockProduct.id}/edit` }
      )

      const unitInput = screen.getByPlaceholderText('请输入计量单位')
      expect(unitInput).toBeDisabled()
    })

    it('should display save button in edit mode', () => {
      renderWithProviders(
        <ProductForm productId={mockProduct.id} initialData={mockProduct} />,
        { route: `/catalog/products/${mockProduct.id}/edit` }
      )

      expect(screen.getByRole('button', { name: /保存/i })).toBeInTheDocument()
    })

    it('should call update API on form submit', async () => {
      const { user } = renderWithProviders(
        <ProductForm productId={mockProduct.id} initialData={mockProduct} />,
        { route: `/catalog/products/${mockProduct.id}/edit` }
      )

      // Modify name
      const nameInput = screen.getByPlaceholderText('请输入商品名称')
      await user.clear(nameInput)
      await user.type(nameInput, '更新的商品名称')

      // Submit form
      await user.click(screen.getByRole('button', { name: /保存/i }))

      await waitFor(() => {
        expect(mockApi.putCatalogProductsId).toHaveBeenCalledWith(
          mockProduct.id,
          expect.objectContaining({
            name: '更新的商品名称',
          })
        )
      })
    })
  })

  describe('Form Validation', () => {
    it('should show validation error for empty required fields', async () => {
      const { user } = renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      // Clear default values and submit
      const codeInput = screen.getByPlaceholderText('请输入商品编码 (SKU)')
      const unitInput = screen.getByPlaceholderText('请输入计量单位')

      await user.clear(codeInput)
      await user.clear(unitInput)

      // Try to submit
      await user.click(screen.getByRole('button', { name: /创建/i }))

      // API should not be called with invalid data
      expect(mockApi.postCatalogProducts).not.toHaveBeenCalled()
    })

    it('should validate code format (alphanumeric, underscore, hyphen)', async () => {
      const { user } = renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      // Enter invalid code with special characters
      const codeInput = screen.getByPlaceholderText('请输入商品编码 (SKU)')
      await user.clear(codeInput)
      await user.type(codeInput, 'INVALID@CODE!')
      await user.type(screen.getByPlaceholderText('请输入商品名称'), '测试')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      // API should not be called
      expect(mockApi.postCatalogProducts).not.toHaveBeenCalled()
    })
  })

  describe('Error Handling', () => {
    it('should show error message when create API fails', async () => {
      mockApi.postCatalogProducts.mockResolvedValueOnce(
        createErrorResponse('商品编码已存在', 'ERR_DUPLICATE')
      )

      const { user } = renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      await user.clear(screen.getByPlaceholderText('请输入商品编码 (SKU)'))
      await user.type(screen.getByPlaceholderText('请输入商品编码 (SKU)'), 'EXISTING-SKU')
      await user.type(screen.getByPlaceholderText('请输入商品名称'), '新商品')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalled()
      })
    })

    it('should show error message when update API fails', async () => {
      mockApi.putCatalogProductsId.mockResolvedValueOnce(
        createErrorResponse('更新失败', 'ERR_UPDATE')
      )

      const { user } = renderWithProviders(
        <ProductForm productId={mockProduct.id} initialData={mockProduct} />,
        { route: `/catalog/products/${mockProduct.id}/edit` }
      )

      const nameInput = screen.getByPlaceholderText('请输入商品名称')
      await user.clear(nameInput)
      await user.type(nameInput, '新名称')

      await user.click(screen.getByRole('button', { name: /保存/i }))

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalled()
      })
    })
  })

  describe('Price Fields', () => {
    it('should accept decimal prices', async () => {
      const { user } = renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      await user.clear(screen.getByPlaceholderText('请输入商品编码 (SKU)'))
      await user.type(screen.getByPlaceholderText('请输入商品编码 (SKU)'), 'PRICE-TEST')
      await user.type(screen.getByPlaceholderText('请输入商品名称'), '价格测试')
      await user.type(screen.getByPlaceholderText('请输入进货价'), '99.99')
      await user.type(screen.getByPlaceholderText('请输入销售价'), '199.99')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(mockApi.postCatalogProducts).toHaveBeenCalledWith(
          expect.objectContaining({
            purchase_price: 99.99,
            selling_price: 199.99,
          })
        )
      })
    })
  })

  describe('API Request Payload', () => {
    it('should send correct payload structure for create', async () => {
      const { user } = renderWithProviders(<ProductForm />, { route: '/catalog/products/new' })

      await user.clear(screen.getByPlaceholderText('请输入商品编码 (SKU)'))
      await user.type(screen.getByPlaceholderText('请输入商品编码 (SKU)'), 'API-TEST-001')
      await user.type(screen.getByPlaceholderText('请输入商品名称'), 'API测试商品')
      await user.type(screen.getByPlaceholderText('请输入条形码 (可选)'), '9999999999999')

      await user.click(screen.getByRole('button', { name: /创建/i }))

      await waitFor(() => {
        expect(mockApi.postCatalogProducts).toHaveBeenCalledWith({
          code: 'API-TEST-001',
          name: 'API测试商品',
          unit: '个',
          barcode: '9999999999999',
          description: undefined,
          purchase_price: undefined,
          selling_price: undefined,
          min_stock: undefined,
          sort_order: undefined,
        })
      })
    })
  })
})
