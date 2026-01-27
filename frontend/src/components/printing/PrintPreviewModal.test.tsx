/**
 * PrintPreviewModal Component Tests
 *
 * Tests for the print preview modal functionality.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderWithProviders, screen, fireEvent, waitFor } from '@/tests/utils'
import { PrintPreviewModal } from './PrintPreviewModal'
import * as printApi from '@/api/printing'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the print API
vi.mock('@/api/printing', () => ({
  previewDocument: vi.fn(),
  generatePDF: vi.fn(),
  getTemplatesByDocType: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')

describe('PrintPreviewModal', () => {
  const mockPreviewResponse = {
    html: '<html><body><h1>Test Document</h1></body></html>',
    templateId: 'template-1',
    paperSize: 'A4',
    orientation: 'PORTRAIT',
    margins: { top: 10, right: 10, bottom: 10, left: 10 },
  }

  const mockTemplates = [
    {
      id: 'template-1',
      tenantId: 'tenant-1',
      documentType: 'SALES_ORDER',
      name: 'A4 Template',
      description: '',
      paperSize: 'A4',
      orientation: 'PORTRAIT',
      margins: { top: 10, right: 10, bottom: 10, left: 10 },
      isDefault: true,
      status: 'ACTIVE',
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    },
    {
      id: 'template-2',
      tenantId: 'tenant-1',
      documentType: 'SALES_ORDER',
      name: 'Thermal Template',
      description: '',
      paperSize: '80MM',
      orientation: 'PORTRAIT',
      margins: { top: 5, right: 5, bottom: 5, left: 5 },
      isDefault: false,
      status: 'ACTIVE',
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    },
  ]

  const mockPrintJob = {
    id: 'job-1',
    tenantId: 'tenant-1',
    templateId: 'template-1',
    documentType: 'SALES_ORDER',
    documentId: 'doc-1',
    documentNumber: 'SO-2024-001',
    status: 'COMPLETED' as const,
    copies: 1,
    pdfUrl: '/api/v1/print/jobs/job-1/download',
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  const defaultProps = {
    visible: true,
    onClose: vi.fn(),
    documentType: 'SALES_ORDER',
    documentId: 'doc-1',
    documentNumber: 'SO-2024-001',
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(printApi.getTemplatesByDocType).mockResolvedValue(mockTemplates)
    vi.mocked(printApi.previewDocument).mockResolvedValue(mockPreviewResponse)
    vi.mocked(printApi.generatePDF).mockResolvedValue(mockPrintJob)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should not render when not visible', () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} visible={false} />)

    expect(screen.queryByText('打印预览 - SO-2024-001')).not.toBeInTheDocument()
  })

  it('should render modal with title when visible', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText('打印预览 - SO-2024-001')).toBeInTheDocument()
    })
  })

  it('should show loading state initially', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    expect(screen.getByText(/加载预览中/i)).toBeInTheDocument()
  })

  it('should load preview when modal becomes visible', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(printApi.previewDocument).toHaveBeenCalledWith({
        documentType: 'SALES_ORDER',
        documentId: 'doc-1',
        templateId: undefined,
        data: undefined,
      })
    })
  })

  it('should display preview content in iframe', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      const iframe = screen.getByTitle('打印预览')
      expect(iframe).toBeInTheDocument()
      expect(iframe).toHaveAttribute('srcDoc')
      expect(iframe.getAttribute('srcDoc')).toContain('Test Document')
    })
  })

  it('should display error message on preview failure', async () => {
    vi.mocked(printApi.previewDocument).mockRejectedValue(new Error('加载失败'))

    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText('加载失败')).toBeInTheDocument()
    })
  })

  it('should show retry button on error', async () => {
    vi.mocked(printApi.previewDocument).mockRejectedValue(new Error('加载失败'))

    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /重试/i })).toBeInTheDocument()
    })
  })

  it('should load template selector when templates are available', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      // Check for template options in the select
      const select = screen.getByText(/A4 Template.*默认/i)
      expect(select).toBeInTheDocument()
    })
  })

  it('should have zoom controls', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByLabelText(/缩小/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/放大/i)).toBeInTheDocument()
      expect(screen.getByText('100%')).toBeInTheDocument()
    })
  })

  it('should change zoom level on zoom in/out', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText('100%')).toBeInTheDocument()
    })

    // Click zoom in
    fireEvent.click(screen.getByLabelText(/放大/i))

    await waitFor(() => {
      expect(screen.getByText('125%')).toBeInTheDocument()
    })

    // Click zoom out twice to go back and below
    fireEvent.click(screen.getByLabelText(/缩小/i))
    fireEvent.click(screen.getByLabelText(/缩小/i))

    await waitFor(() => {
      expect(screen.getByText('75%')).toBeInTheDocument()
    })
  })

  it('should have copies input', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText(/打印份数/i)).toBeInTheDocument()
    })
  })

  it('should have print button', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      // Find the print button in the footer (not the one in the toolbar header)
      const buttons = screen.getAllByRole('button')
      const printButton = buttons.find(
        (btn) => btn.textContent?.includes('打印') && !btn.textContent?.includes('打印份数')
      )
      expect(printButton).toBeInTheDocument()
    })
  })

  it('should have download PDF button', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /下载 PDF/i })).toBeInTheDocument()
    })
  })

  it('should call onClose when cancel button is clicked', async () => {
    const onClose = vi.fn()
    renderWithProviders(<PrintPreviewModal {...defaultProps} onClose={onClose} />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /取消/i })).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole('button', { name: /取消/i }))

    expect(onClose).toHaveBeenCalled()
  })

  it('should have refresh button', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByLabelText(/刷新预览/i)).toBeInTheDocument()
    })
  })

  it('should reload preview when refresh button is clicked', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} />)

    await waitFor(() => {
      expect(printApi.previewDocument).toHaveBeenCalledTimes(1)
    })

    fireEvent.click(screen.getByLabelText(/刷新预览/i))

    await waitFor(() => {
      expect(printApi.previewDocument).toHaveBeenCalledTimes(2)
    })
  })

  it('should support custom title', async () => {
    renderWithProviders(<PrintPreviewModal {...defaultProps} title="Custom Title" />)

    await waitFor(() => {
      expect(screen.getByText('Custom Title')).toBeInTheDocument()
    })
  })
})
