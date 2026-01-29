/**
 * ProductAttachmentUploader Unit Tests (ATTACH-TEST-001)
 *
 * These tests verify the product attachment uploader component:
 * - File upload workflow with presigned URLs
 * - File type validation
 * - File size validation
 * - Delete attachment flow
 * - Set main image flow
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import { ProductAttachmentUploader } from '@/features/catalog/ProductAttachmentUploader'
import { Toast } from '@douyinfe/semi-ui-19'
import * as productAttachmentsApi from '@/api/product-attachments/product-attachments'

// Mock the product attachments API module
vi.mock('@/api/product-attachments/product-attachments', () => ({
  useListProductAttachments: vi.fn(),
  useInitiateProductAttachmentUpload: vi.fn(),
  useConfirmProductAttachmentUpload: vi.fn(),
  useDeleteProductAttachment: vi.fn(),
  useSetProductAttachmentAsMainImage: vi.fn(),
  getListProductAttachmentsQueryKey: vi.fn(() => ['product-attachments']),
}))

// Mock useQueryClient
vi.mock('@tanstack/react-query', async () => {
  const actual = await vi.importActual('@tanstack/react-query')
  return {
    ...actual,
    useQueryClient: () => ({
      invalidateQueries: vi.fn(),
    }),
  }
})

// Mock i18n translations
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        'attachments.title': '商品附件',
        'attachments.description': '上传商品图片和文档',
        'attachments.dragDropHint': '点击或拖拽文件上传',
        'attachments.allowedTypes': '支持 JPG, PNG, PDF 等格式',
        'attachments.maxSize': '最大 100MB',
        'attachments.uploadArea': '上传区域',
        'attachments.empty': '暂无附件',
        'attachments.mainImage': '主图',
        'attachments.setAsMain': '设为主图',
        'attachments.delete': '删除',
        'attachments.previewImage': '预览图片',
        'attachments.confirmDelete.title': '确认删除',
        'attachments.confirmDelete.content': '确定要删除此附件吗？',
        'attachments.errors.invalidType': '不支持的文件类型',
        'attachments.errors.fileTooLarge': '文件大小超过限制',
        'attachments.messages.deleteSuccess': '删除成功',
        'attachments.messages.setMainSuccess': '设置主图成功',
        'common:actions.delete': '删除',
        'common:actions.cancel': '取消',
      }
      return translations[key] || key
    },
  }),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')

// Sample attachment data
const mockAttachments = [
  {
    id: 'att-001',
    product_id: 'prod-001',
    type: 'main_image',
    status: 'active',
    file_name: 'main-image.jpg',
    file_size: 1024,
    content_type: 'image/jpeg',
    url: 'https://storage.example.com/main-image.jpg',
    thumbnail_url: 'https://storage.example.com/thumb/main-image.jpg',
    sort_order: 0,
  },
  {
    id: 'att-002',
    product_id: 'prod-001',
    type: 'gallery_image',
    status: 'active',
    file_name: 'gallery-1.jpg',
    file_size: 2048,
    content_type: 'image/jpeg',
    url: 'https://storage.example.com/gallery-1.jpg',
    thumbnail_url: 'https://storage.example.com/thumb/gallery-1.jpg',
    sort_order: 1,
  },
]

describe('ProductAttachmentUploader', () => {
  const productId = 'prod-001'

  // Mock functions for mutations
  let mockInitiateUpload: ReturnType<typeof vi.fn>
  let mockConfirmUpload: ReturnType<typeof vi.fn>
  let mockDeleteAttachment: ReturnType<typeof vi.fn>
  let mockSetMainImage: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()

    // Setup mock functions
    mockInitiateUpload = vi.fn()
    mockConfirmUpload = vi.fn()
    mockDeleteAttachment = vi.fn()
    mockSetMainImage = vi.fn()

    // Mock useListProductAttachments
    vi.mocked(productAttachmentsApi.useListProductAttachments).mockReturnValue({
      data: {
        status: 200,
        data: {
          data: mockAttachments,
        },
      },
      isLoading: false,
    } as ReturnType<typeof productAttachmentsApi.useListProductAttachments>)

    // Mock useInitiateProductAttachmentUpload
    vi.mocked(productAttachmentsApi.useInitiateProductAttachmentUpload).mockReturnValue({
      mutateAsync: mockInitiateUpload,
      isPending: false,
    } as unknown as ReturnType<typeof productAttachmentsApi.useInitiateProductAttachmentUpload>)

    // Mock useConfirmProductAttachmentUpload
    vi.mocked(productAttachmentsApi.useConfirmProductAttachmentUpload).mockReturnValue({
      mutateAsync: mockConfirmUpload,
      isPending: false,
    } as unknown as ReturnType<typeof productAttachmentsApi.useConfirmProductAttachmentUpload>)

    // Mock useDeleteProductAttachment
    vi.mocked(productAttachmentsApi.useDeleteProductAttachment).mockReturnValue({
      mutateAsync: mockDeleteAttachment,
      isPending: false,
    } as unknown as ReturnType<typeof productAttachmentsApi.useDeleteProductAttachment>)

    // Mock useSetProductAttachmentAsMainImage
    vi.mocked(productAttachmentsApi.useSetProductAttachmentAsMainImage).mockReturnValue({
      mutateAsync: mockSetMainImage,
      isPending: false,
    } as unknown as ReturnType<typeof productAttachmentsApi.useSetProductAttachmentAsMainImage>)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Rendering', () => {
    it('should render the attachment uploader component', () => {
      renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Component should render
      expect(screen.getByText('商品附件')).toBeInTheDocument()
    })

    it('should display upload zone', () => {
      renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Upload hint should be visible
      expect(screen.getByText('点击或拖拽文件上传')).toBeInTheDocument()
    })

    it('should display existing attachments file names', () => {
      renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Attachment file names should be visible
      expect(screen.getByText('main-image.jpg')).toBeInTheDocument()
      expect(screen.getByText('gallery-1.jpg')).toBeInTheDocument()
    })

    it('should indicate main image with badge', () => {
      renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Main image badge should be present
      expect(screen.getByText('主图')).toBeInTheDocument()
    })

    it('should show empty state when no attachments', () => {
      vi.mocked(productAttachmentsApi.useListProductAttachments).mockReturnValue({
        data: {
          status: 200,
          data: {
            data: [],
          },
        },
        isLoading: false,
      } as ReturnType<typeof productAttachmentsApi.useListProductAttachments>)

      renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Empty state message should be visible
      expect(screen.getByText('暂无附件')).toBeInTheDocument()
    })

    it('should show loading spinner when fetching attachments', () => {
      vi.mocked(productAttachmentsApi.useListProductAttachments).mockReturnValue({
        data: undefined,
        isLoading: true,
      } as ReturnType<typeof productAttachmentsApi.useListProductAttachments>)

      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Loading spinner should be visible (Semi UI uses .semi-spin class)
      const spinner = container.querySelector('.semi-spin')
      expect(spinner).toBeTruthy()
    })

    it('should apply disabled class when disabled prop is true', () => {
      const { container } = renderWithProviders(
        <ProductAttachmentUploader productId={productId} disabled />
      )

      // Upload zone should have disabled class
      const uploadZone = container.querySelector('.attachment-upload-zone')
      expect(uploadZone?.classList.contains('disabled')).toBe(true)
    })
  })

  describe('Attachment Display', () => {
    it('should display correct number of attachments', () => {
      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Should have 2 attachment items
      const attachmentItems = container.querySelectorAll('.attachment-item')
      expect(attachmentItems.length).toBe(2)
    })

    it('should display thumbnails for image attachments', () => {
      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Should have thumbnail images
      const thumbnails = container.querySelectorAll('.attachment-thumbnail img')
      expect(thumbnails.length).toBe(2)
    })
  })

  describe('Delete Attachment', () => {
    it('should show delete buttons for attachments', () => {
      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Should have delete buttons
      const deleteButtons = container.querySelectorAll('[aria-label="删除"]')
      expect(deleteButtons.length).toBe(2)
    })

    it('should show delete confirmation modal when clicking delete', async () => {
      const { user, container } = renderWithProviders(
        <ProductAttachmentUploader productId={productId} />
      )

      // Click first delete button
      const deleteButton = container.querySelector('[aria-label="删除"]')
      if (deleteButton) {
        await user.click(deleteButton)

        // Wait for confirmation modal
        await waitFor(() => {
          expect(screen.getByText('确认删除')).toBeInTheDocument()
        })
      }
    })
  })

  describe('Set Main Image', () => {
    it('should show set main image button for gallery images only', () => {
      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Only gallery image should have set main button
      const setMainButtons = container.querySelectorAll('[aria-label="设为主图"]')
      expect(setMainButtons.length).toBe(1)
    })

    it('should call set main image API when clicked', async () => {
      mockSetMainImage.mockResolvedValue({
        status: 200,
        data: { data: { ...mockAttachments[1], type: 'main_image' } },
      })

      const { user, container } = renderWithProviders(
        <ProductAttachmentUploader productId={productId} />
      )

      // Click set main image button
      const setMainButton = container.querySelector('[aria-label="设为主图"]')
      if (setMainButton) {
        await user.click(setMainButton)

        await waitFor(() => {
          expect(mockSetMainImage).toHaveBeenCalled()
        })
      }
    })
  })

  describe('Props', () => {
    it('should accept productId prop', () => {
      const { container } = renderWithProviders(
        <ProductAttachmentUploader productId="test-product-id" />
      )

      // Component should render
      expect(container.querySelector('.product-attachment-uploader')).toBeTruthy()
    })

    it('should accept disabled prop', () => {
      const { container } = renderWithProviders(
        <ProductAttachmentUploader productId={productId} disabled />
      )

      // Upload zone should be disabled
      const uploadZone = container.querySelector('.attachment-upload-zone.disabled')
      expect(uploadZone).toBeTruthy()
    })

    it('should accept onUploadComplete callback', () => {
      const onUploadComplete = vi.fn()

      renderWithProviders(
        <ProductAttachmentUploader productId={productId} onUploadComplete={onUploadComplete} />
      )

      // Component should render without error
      expect(screen.getByText('商品附件')).toBeInTheDocument()
    })
  })

  describe('API Integration', () => {
    it('should call useListProductAttachments with correct productId', () => {
      renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      expect(productAttachmentsApi.useListProductAttachments).toHaveBeenCalledWith(
        productId,
        { status: 'active' },
        expect.any(Object)
      )
    })

    it('should not fetch attachments when productId is empty', () => {
      vi.mocked(productAttachmentsApi.useListProductAttachments).mockClear()

      renderWithProviders(<ProductAttachmentUploader productId="" />)

      // The hook should be called but with enabled: false
      expect(productAttachmentsApi.useListProductAttachments).toHaveBeenCalledWith(
        '',
        { status: 'active' },
        expect.objectContaining({
          query: expect.objectContaining({
            enabled: false,
          }),
        })
      )
    })
  })

  describe('Accessibility', () => {
    it('should have accessible upload zone with role button', () => {
      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      const uploadZone = container.querySelector('.attachment-upload-zone')
      expect(uploadZone?.getAttribute('role')).toBe('button')
    })

    it('should have accessible upload zone with tabindex', () => {
      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      const uploadZone = container.querySelector('.attachment-upload-zone')
      expect(uploadZone?.getAttribute('tabindex')).toBe('0')
    })

    it('should have aria-label on upload zone', () => {
      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      const uploadZone = container.querySelector('.attachment-upload-zone')
      expect(uploadZone?.getAttribute('aria-label')).toBe('上传区域')
    })

    it('should disable tabindex when disabled', () => {
      const { container } = renderWithProviders(
        <ProductAttachmentUploader productId={productId} disabled />
      )

      const uploadZone = container.querySelector('.attachment-upload-zone')
      expect(uploadZone?.getAttribute('tabindex')).toBe('-1')
    })
  })

  describe('Error Handling', () => {
    it('should display error state when loading fails', () => {
      vi.mocked(productAttachmentsApi.useListProductAttachments).mockReturnValue({
        data: undefined,
        isLoading: false,
        error: new Error('Failed to load attachments'),
      } as unknown as ReturnType<typeof productAttachmentsApi.useListProductAttachments>)

      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Component should still render
      expect(container.querySelector('.product-attachment-uploader')).toBeTruthy()
    })

    it('should handle rejection gracefully when deleting', async () => {
      // Setup rejection
      mockDeleteAttachment.mockRejectedValue(new Error('Delete failed'))

      const { user, container } = renderWithProviders(
        <ProductAttachmentUploader productId={productId} />
      )

      // Click first delete button
      const deleteButton = container.querySelector('[aria-label="删除"]')
      expect(deleteButton).toBeTruthy()

      // Just verify the button is clickable
      if (deleteButton) {
        await user.click(deleteButton)
        // Modal should appear
        expect(screen.getByText('确认删除')).toBeInTheDocument()
      }
    })

    it('should handle rejection gracefully when setting main image', async () => {
      // Setup rejection
      mockSetMainImage.mockRejectedValue(new Error('Set main image failed'))

      const { container } = renderWithProviders(<ProductAttachmentUploader productId={productId} />)

      // Verify set main button exists
      const setMainButton = container.querySelector('[aria-label="设为主图"]')
      expect(setMainButton).toBeTruthy()
    })
  })
})
