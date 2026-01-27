/**
 * usePrint Hook Tests
 *
 * Tests for the print preview hook functionality.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { usePrint } from './usePrint'
import * as printApi from '@/api/printing'

// Mock the print API
vi.mock('@/api/printing', () => ({
  previewDocument: vi.fn(),
  generatePDF: vi.fn(),
  getTemplatesByDocType: vi.fn(),
}))

describe('usePrint', () => {
  const mockPreviewResponse = {
    html: '<html><body>Test Preview</body></html>',
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
      name: 'Default Template',
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

  const defaultOptions = {
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

  it('should initialize with default values', () => {
    const { result } = renderHook(() => usePrint(defaultOptions))

    expect(result.current.preview).toBeNull()
    expect(result.current.isLoading).toBe(false)
    expect(result.current.error).toBeNull()
    expect(result.current.copies).toBe(1)
    expect(result.current.zoom).toBe(100)
  })

  it('should load templates for document type on mount', async () => {
    const { result } = renderHook(() => usePrint(defaultOptions))

    await waitFor(() => {
      expect(printApi.getTemplatesByDocType).toHaveBeenCalledWith('SALES_ORDER')
    })

    await waitFor(() => {
      expect(result.current.templates).toHaveLength(2)
      // Should auto-select default template
      expect(result.current.selectedTemplateId).toBe('template-1')
    })
  })

  it('should load preview when loadPreview is called', async () => {
    const { result } = renderHook(() => usePrint(defaultOptions))

    await act(async () => {
      await result.current.loadPreview()
    })

    expect(printApi.previewDocument).toHaveBeenCalledWith({
      documentType: 'SALES_ORDER',
      documentId: 'doc-1',
      templateId: undefined,
      data: undefined,
    })

    expect(result.current.preview).toEqual(mockPreviewResponse)
    expect(result.current.isLoading).toBe(false)
  })

  it('should auto-load preview when autoLoad is true', async () => {
    renderHook(() =>
      usePrint({
        ...defaultOptions,
        autoLoad: true,
      })
    )

    await waitFor(() => {
      expect(printApi.previewDocument).toHaveBeenCalled()
    })
  })

  it('should set error on preview failure', async () => {
    const errorMessage = 'Preview failed'
    vi.mocked(printApi.previewDocument).mockRejectedValue(new Error(errorMessage))

    const { result } = renderHook(() => usePrint(defaultOptions))

    await act(async () => {
      await result.current.loadPreview()
    })

    expect(result.current.error).toBe(errorMessage)
    expect(result.current.preview).toBeNull()
  })

  it('should change template and reload preview', async () => {
    const { result } = renderHook(() => usePrint(defaultOptions))

    // Wait for templates to load
    await waitFor(() => {
      expect(result.current.templates).toHaveLength(2)
    })

    // Select a different template
    await act(async () => {
      result.current.selectTemplate('template-2')
    })

    expect(result.current.selectedTemplateId).toBe('template-2')

    await waitFor(() => {
      expect(printApi.previewDocument).toHaveBeenCalledWith(
        expect.objectContaining({
          templateId: 'template-2',
        })
      )
    })
  })

  it('should generate PDF successfully', async () => {
    const { result } = renderHook(() => usePrint(defaultOptions))

    let job
    await act(async () => {
      job = await result.current.generatePdf(2)
    })

    expect(printApi.generatePDF).toHaveBeenCalledWith({
      documentType: 'SALES_ORDER',
      documentId: 'doc-1',
      documentNumber: 'SO-2024-001',
      templateId: undefined,
      copies: 2,
      data: undefined,
    })

    expect(job).toEqual(mockPrintJob)
  })

  it('should set copies within valid range', () => {
    const { result } = renderHook(() => usePrint(defaultOptions))

    // Set valid copies
    act(() => {
      result.current.setCopies(5)
    })
    expect(result.current.copies).toBe(5)

    // Set below minimum
    act(() => {
      result.current.setCopies(0)
    })
    expect(result.current.copies).toBe(1)

    // Set above maximum
    act(() => {
      result.current.setCopies(150)
    })
    expect(result.current.copies).toBe(100)
  })

  it('should set zoom within valid range', () => {
    const { result } = renderHook(() => usePrint(defaultOptions))

    // Set valid zoom
    act(() => {
      result.current.setZoom(150)
    })
    expect(result.current.zoom).toBe(150)

    // Set below minimum
    act(() => {
      result.current.setZoom(10)
    })
    expect(result.current.zoom).toBe(50)

    // Set above maximum
    act(() => {
      result.current.setZoom(300)
    })
    expect(result.current.zoom).toBe(200)
  })

  it('should provide iframe ref', () => {
    const { result } = renderHook(() => usePrint(defaultOptions))

    expect(result.current.iframeRef).toBeDefined()
    expect(result.current.iframeRef.current).toBeNull()
  })
})
