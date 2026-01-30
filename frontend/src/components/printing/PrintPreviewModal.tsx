/**
 * PrintPreviewModal Component
 *
 * Modal dialog for print preview with:
 * - HTML preview in iframe
 * - Zoom controls
 * - Template selection
 * - Copies setting
 * - Browser print button
 *
 * @example
 * <PrintPreviewModal
 *   visible={isOpen}
 *   onClose={handleClose}
 *   documentType="SALES_ORDER"
 *   documentId={orderId}
 *   documentNumber={orderNumber}
 * />
 */

import { useEffect, useCallback, useMemo } from 'react'
import {
  Modal,
  Button,
  Select,
  InputNumber,
  Spin,
  Typography,
  Space,
  Toast,
} from '@douyinfe/semi-ui-19'
import { IconPrint, IconMinus, IconPlus, IconDownload, IconRefresh } from '@douyinfe/semi-icons'
import { usePrint, ZOOM_LEVELS } from '@/hooks/usePrint'
import { Row, Spacer } from '@/components/common/layout/Flex'
import type { CSSProperties } from 'react'
import './PrintPreviewModal.css'

const { Text } = Typography

export interface PrintPreviewModalProps {
  /** Whether the modal is visible */
  visible: boolean
  /** Callback when modal is closed */
  onClose: () => void
  /** Document type (e.g., 'SALES_ORDER', 'SALES_DELIVERY') */
  documentType: string
  /** Document UUID */
  documentId: string
  /** Document number for display */
  documentNumber: string
  /** Additional data for template rendering */
  data?: unknown
  /** Custom title */
  title?: string
}

// Base paper sizes in mm (portrait orientation)
const PAPER_SIZES: Record<string, { width: number; height: number; name: string }> = {
  A4: { width: 210, height: 297, name: 'A4' },
  A5: { width: 148, height: 210, name: 'A5' },
  RECEIPT_58MM: { width: 58, height: 200, name: '58mm 热敏纸' },
  RECEIPT_80MM: { width: 80, height: 200, name: '80mm 热敏纸' },
  CONTINUOUS_241: { width: 241, height: 280, name: '241mm 连续纸' },
}

// Get paper size label with orientation-aware dimensions
const getPaperSizeLabel = (paperSize: string, orientation: string): string => {
  const size = PAPER_SIZES[paperSize] || PAPER_SIZES.A4
  const isLandscape = orientation === 'LANDSCAPE'

  // For receipt and continuous paper, orientation doesn't change displayed dimensions
  if (paperSize.startsWith('RECEIPT_') || paperSize.startsWith('CONTINUOUS_')) {
    return size.name
  }

  const width = isLandscape ? size.height : size.width
  const height = isLandscape ? size.width : size.height

  return `${size.name} (${width}×${height}mm)`
}

// Get paper dimensions for preview scaling
const getPaperDimensions = (
  paperSize: string,
  orientation: string
): { width: number; height: number } => {
  const size = PAPER_SIZES[paperSize] || PAPER_SIZES.A4

  // Swap dimensions for landscape
  if (orientation === 'LANDSCAPE') {
    return { width: size.height, height: size.width }
  }

  return { width: size.width, height: size.height }
}

export function PrintPreviewModal({
  visible,
  onClose,
  documentType,
  documentId,
  documentNumber,
  data,
  title,
}: PrintPreviewModalProps) {
  const {
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
    setCopies,
    zoom,
    setZoom,
    iframeRef,
  } = usePrint({
    documentType,
    documentId,
    documentNumber,
    data,
    autoLoad: false,
  })

  // Load preview when modal becomes visible
  useEffect(() => {
    if (visible && documentId) {
      loadPreview()
    }
  }, [visible, documentId, loadPreview])

  // Handle zoom controls
  const handleZoomIn = useCallback(() => {
    const currentIndex = ZOOM_LEVELS.indexOf(zoom)
    if (currentIndex < ZOOM_LEVELS.length - 1) {
      setZoom(ZOOM_LEVELS[currentIndex + 1])
    }
  }, [zoom, setZoom])

  const handleZoomOut = useCallback(() => {
    const currentIndex = ZOOM_LEVELS.indexOf(zoom)
    if (currentIndex > 0) {
      setZoom(ZOOM_LEVELS[currentIndex - 1])
    }
  }, [zoom, setZoom])

  // Handle print
  const handlePrint = useCallback(() => {
    print()
  }, [print])

  // Handle PDF generation
  const handleDownloadPdf = useCallback(async () => {
    try {
      const job = await generatePdf(copies)
      if (job.status === 'COMPLETED' && job.pdf_url) {
        // Open PDF in new tab
        window.open(job.pdf_url, '_blank')
        Toast.success('PDF 生成成功')
      } else if (job.status === 'PROCESSING') {
        Toast.info('PDF 正在生成中，请稍候...')
      } else if (job.status === 'FAILED') {
        Toast.error(job.error_message || 'PDF 生成失败')
      }
    } catch {
      Toast.error('PDF 生成失败')
    }
  }, [generatePdf, copies])

  // Handle template change
  const handleTemplateChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      if (typeof value === 'string') {
        selectTemplate(value)
      }
    },
    [selectTemplate]
  )

  // Template options for select
  const templateOptions = useMemo(
    () =>
      templates.map((t) => ({
        value: t.id,
        label: `${t.name}${t.is_default ? ' (默认)' : ''} - ${getPaperSizeLabel(t.paper_size || 'A4', t.orientation || 'PORTRAIT')}`,
      })),
    [templates]
  )

  // Calculate iframe dimensions based on paper size and zoom
  const iframeDimensions = useMemo(() => {
    if (!preview) return { width: 210, height: 297 }
    const dims = getPaperDimensions(preview.paper_size || 'A4', preview.orientation || 'PORTRAIT')
    return {
      width: Math.round((dims.width * zoom) / 100),
      height: Math.round((dims.height * zoom) / 100),
    }
  }, [preview, zoom])

  // Generate iframe srcDoc from preview HTML
  const iframeSrcDoc = useMemo(() => {
    if (!preview?.html) return ''

    // Wrap HTML with additional print styles
    return `
      <!DOCTYPE html>
      <html>
        <head>
          <meta charset="utf-8">
          <style>
            body {
              margin: 0;
              padding: 0;
              background: white;
            }
            @media print {
              body {
                -webkit-print-color-adjust: exact;
                print-color-adjust: exact;
              }
            }
          </style>
        </head>
        <body>
          ${preview.html}
        </body>
      </html>
    `
  }, [preview])

  // Modal title
  const modalTitle = title || `打印预览 - ${documentNumber}`

  // Iframe style (dynamic based on paper size and zoom)
  const iframeStyle: CSSProperties = {
    width: `${iframeDimensions.width}mm`,
    height: `${iframeDimensions.height}mm`,
  }

  return (
    <Modal
      visible={visible}
      onCancel={onClose}
      width="auto"
      style={{ top: 20, maxWidth: '95vw', minWidth: 320 }}
      title={modalTitle}
      footer={null}
      closeOnEsc
      className="print-preview-modal"
    >
      {/* Toolbar */}
      <div className="print-preview-toolbar">
        <Row align="center" gap="md" wrap="wrap">
          {/* Template selector */}
          {templates.length > 0 && (
            <Select
              value={selectedTemplateId || undefined}
              onChange={handleTemplateChange}
              placeholder="选择模板"
              optionList={templateOptions}
            />
          )}

          {/* Zoom controls */}
          <Space className="toolbar-actions">
            <Button
              icon={<IconMinus />}
              type="tertiary"
              size="small"
              onClick={handleZoomOut}
              disabled={zoom <= ZOOM_LEVELS[0]}
              aria-label="缩小"
            />
            <Text style={{ minWidth: 50, textAlign: 'center' }}>{zoom}%</Text>
            <Button
              icon={<IconPlus />}
              type="tertiary"
              size="small"
              onClick={handleZoomIn}
              disabled={zoom >= ZOOM_LEVELS[ZOOM_LEVELS.length - 1]}
              aria-label="放大"
            />
          </Space>

          <Spacer />

          {/* Refresh button */}
          <Button
            icon={<IconRefresh />}
            type="tertiary"
            size="small"
            onClick={() => loadPreview()}
            disabled={isLoading}
            aria-label="刷新预览"
          />
        </Row>
      </div>

      {/* Preview area */}
      <div className="print-preview-container">
        {isLoading ? (
          <div className="print-preview-loading">
            <Spin size="large" tip="加载预览中..." />
          </div>
        ) : error ? (
          <div className="print-preview-error">
            <div>
              <Text type="danger">{error}</Text>
              <br />
              <Button
                type="primary"
                onClick={() => loadPreview()}
                style={{ marginTop: 'var(--spacing-4)' }}
              >
                重试
              </Button>
            </div>
          </div>
        ) : preview ? (
          <div className="print-preview-iframe-wrapper">
            <iframe
              ref={iframeRef}
              srcDoc={iframeSrcDoc}
              style={iframeStyle}
              title="打印预览"
              sandbox="allow-same-origin allow-scripts"
            />
          </div>
        ) : (
          <div className="print-preview-empty">
            <Text type="tertiary">暂无预览内容</Text>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="print-preview-footer">
        <Row align="center" gap="md" wrap="wrap">
          {/* Copies input */}
          <Space className="copies-input">
            <Text>打印份数:</Text>
            <InputNumber
              value={copies}
              onChange={(value) => setCopies(value as number)}
              min={1}
              max={100}
              style={{ width: 80 }}
            />
          </Space>

          <Spacer />

          {/* Action buttons */}
          <Space className="footer-actions">
            <Button onClick={onClose}>取消</Button>
            <Button
              icon={<IconDownload />}
              onClick={handleDownloadPdf}
              disabled={isLoading || !preview}
            >
              下载 PDF
            </Button>
            <Button
              icon={<IconPrint />}
              type="primary"
              onClick={handlePrint}
              disabled={isLoading || !preview}
            >
              打印
            </Button>
          </Space>
        </Row>
      </div>
    </Modal>
  )
}
