/**
 * usePrint Hook
 *
 * Custom hook for print functionality including:
 * - Loading print preview
 * - Template selection
 * - Browser print integration
 * - PDF generation
 *
 * @example
 * const { preview, print, isLoading, error, templates } = usePrint({
 *   documentType: 'SALES_ORDER',
 *   documentId: orderId,
 *   documentNumber: orderNumber,
 * })
 */

import { useCallback, useEffect, useRef, useState } from 'react'
import * as printApi from '@/api/printing'
import type { PrintPreviewResponse, PrintTemplate, PrintJob } from '@/api/printing/types'

export interface UsePrintOptions {
  /** Document type (e.g., 'SALES_ORDER', 'SALES_DELIVERY') */
  documentType: string
  /** Document UUID */
  documentId: string
  /** Document number for display (e.g., 'SO-2024-001') */
  documentNumber: string
  /** Additional data to pass to template rendering */
  data?: unknown
  /** Auto-load preview on mount */
  autoLoad?: boolean
}

export interface UsePrintReturn {
  /** Current preview data */
  preview: PrintPreviewResponse | null
  /** Whether loading is in progress */
  isLoading: boolean
  /** Error message if any */
  error: string | null
  /** Available templates for this document type */
  templates: PrintTemplate[]
  /** Currently selected template ID */
  selectedTemplateId: string | null
  /** Select a template */
  selectTemplate: (templateId: string | null) => void
  /** Load preview with current or specified template */
  loadPreview: (templateId?: string) => Promise<void>
  /** Trigger browser print */
  print: () => void
  /** Generate PDF and get job ID */
  generatePdf: (copies?: number) => Promise<PrintJob>
  /** Number of copies for printing */
  copies: number
  /** Set number of copies */
  setCopies: (copies: number) => void
  /** Zoom level (percentage) */
  zoom: number
  /** Set zoom level */
  setZoom: (zoom: number) => void
  /** Reference to the iframe element */
  iframeRef: React.RefObject<HTMLIFrameElement | null>
}

const ZOOM_LEVELS = [50, 75, 100, 125, 150, 200]

export function usePrint({
  documentType,
  documentId,
  documentNumber,
  data,
  autoLoad = false,
}: UsePrintOptions): UsePrintReturn {
  const [preview, setPreview] = useState<PrintPreviewResponse | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [templates, setTemplates] = useState<PrintTemplate[]>([])
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(null)
  const [copies, setCopies] = useState(1)
  const [zoom, setZoom] = useState(100)
  const iframeRef = useRef<HTMLIFrameElement | null>(null)

  // Load available templates for document type
  useEffect(() => {
    async function loadTemplates() {
      try {
        const result = await printApi.getTemplatesByDocType(documentType)
        setTemplates(result)

        // Auto-select default template
        const defaultTemplate = result.find((t) => t.isDefault)
        if (defaultTemplate) {
          setSelectedTemplateId(defaultTemplate.id)
        }
      } catch {
        // Templates load silently - not critical for preview
        setTemplates([])
      }
    }

    if (documentType) {
      loadTemplates()
    }
  }, [documentType])

  // Load preview
  const loadPreview = useCallback(
    async (templateId?: string) => {
      setIsLoading(true)
      setError(null)

      try {
        const result = await printApi.previewDocument({
          documentType,
          documentId,
          templateId: templateId ?? selectedTemplateId ?? undefined,
          data,
        })
        setPreview(result)

        // Update selected template if specified
        if (templateId) {
          setSelectedTemplateId(templateId)
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : '加载预览失败'
        setError(message)
        setPreview(null)
      } finally {
        setIsLoading(false)
      }
    },
    [documentType, documentId, selectedTemplateId, data]
  )

  // Auto-load preview if enabled
  useEffect(() => {
    if (autoLoad && documentId) {
      loadPreview()
    }
  }, [autoLoad, documentId, loadPreview])

  // Select template
  const selectTemplate = useCallback(
    (templateId: string | null) => {
      setSelectedTemplateId(templateId)
      // Reload preview with new template
      if (templateId) {
        loadPreview(templateId)
      }
    },
    [loadPreview]
  )

  // Browser print function
  const print = useCallback(() => {
    const iframe = iframeRef.current
    if (!iframe?.contentWindow) {
      return
    }

    try {
      iframe.contentWindow.focus()
      iframe.contentWindow.print()
    } catch {
      // Handle cross-origin issues if any
      window.print()
    }
  }, [])

  // Generate PDF
  const generatePdf = useCallback(
    async (numCopies?: number) => {
      setIsLoading(true)
      setError(null)

      try {
        const result = await printApi.generatePDF({
          documentType,
          documentId,
          documentNumber,
          templateId: selectedTemplateId ?? undefined,
          copies: numCopies ?? copies,
          data,
        })
        return result
      } catch (err) {
        const message = err instanceof Error ? err.message : '生成PDF失败'
        setError(message)
        throw err
      } finally {
        setIsLoading(false)
      }
    },
    [documentType, documentId, documentNumber, selectedTemplateId, copies, data]
  )

  // Validate zoom level
  const handleSetZoom = useCallback((newZoom: number) => {
    const validZoom = Math.max(
      ZOOM_LEVELS[0],
      Math.min(newZoom, ZOOM_LEVELS[ZOOM_LEVELS.length - 1])
    )
    setZoom(validZoom)
  }, [])

  // Validate copies
  const handleSetCopies = useCallback((newCopies: number) => {
    const validCopies = Math.max(1, Math.min(newCopies, 100))
    setCopies(validCopies)
  }, [])

  return {
    preview,
    isLoading,
    error,
    templates,
    selectedTemplateId,
    selectTemplate,
    loadPreview,
    print,
    generatePdf,
    copies,
    setCopies: handleSetCopies,
    zoom,
    setZoom: handleSetZoom,
    iframeRef,
  }
}

export { ZOOM_LEVELS }
