import { useState } from 'react'
import { Card, Spin, Modal, Typography, Empty } from '@douyinfe/semi-ui-19'
import {
  IconImage,
  IconFile,
  IconStar,
  IconChevronLeft,
  IconChevronRight,
} from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useListProductAttachments } from '@/api/product-attachments/product-attachments'
import type { CatalogAttachmentListResponse } from '@/api/models'
import './ProductAttachmentViewer.css'

const { Text, Title } = Typography

/** Image content types */
const IMAGE_TYPES = [
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  'image/bmp',
  'image/tiff',
]

interface ProductAttachmentViewerProps {
  /** Product ID to display attachments for */
  productId: string
  /** Custom title for the card */
  title?: string
  /** Whether to show the card wrapper */
  showCard?: boolean
}

/**
 * Product attachment viewer component (read-only)
 *
 * Features:
 * - Display product attachments in a grid
 * - Image preview with lightbox
 * - Show main image badge
 * - Document download links
 */
export function ProductAttachmentViewer({
  productId,
  title,
  showCard = true,
}: ProductAttachmentViewerProps) {
  const { t } = useTranslation(['catalog', 'common'])
  const [previewVisible, setPreviewVisible] = useState(false)
  const [previewIndex, setPreviewIndex] = useState(0)

  // Fetch attachments
  const { data: attachmentsResponse, isLoading } = useListProductAttachments(
    productId,
    { status: 'active' },
    {
      query: {
        enabled: !!productId,
      },
    }
  )

  // Extract attachments from response
  const attachments = (
    attachmentsResponse?.status === 200 ? attachmentsResponse.data.data : []
  ) as CatalogAttachmentListResponse[]

  // Separate images and documents
  const images = attachments.filter(
    (a) =>
      a.type === 'main_image' ||
      a.type === 'gallery_image' ||
      IMAGE_TYPES.includes(a.content_type || '')
  )
  const documents = attachments.filter(
    (a) =>
      a.type !== 'main_image' &&
      a.type !== 'gallery_image' &&
      !IMAGE_TYPES.includes(a.content_type || '')
  )

  // Find main image
  const mainImage = attachments.find((a) => a.type === 'main_image')

  // Handle image click for preview
  const handleImageClick = (index: number) => {
    setPreviewIndex(index)
    setPreviewVisible(true)
  }

  // Format file size
  const formatFileSize = (bytes?: number): string => {
    if (!bytes) return '-'
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
  }

  // Render content
  const renderContent = () => {
    if (isLoading) {
      return (
        <div className="attachment-viewer-loading">
          <Spin size="large" />
        </div>
      )
    }

    if (attachments.length === 0) {
      return (
        <Empty
          image={<IconImage size="extra-large" />}
          description={t('attachments.empty')}
          className="attachment-viewer-empty"
        />
      )
    }

    return (
      <div className="attachment-viewer-content">
        {/* Images section */}
        {images.length > 0 && (
          <div className="attachment-viewer-section">
            <Title heading={6} className="attachment-viewer-section-title">
              {t('attachments.images')} ({images.length})
            </Title>
            <div className="attachment-viewer-grid">
              {images.map((attachment, index) => (
                <div
                  key={attachment.id}
                  className={`attachment-viewer-item ${attachment.id === mainImage?.id ? 'main-image' : ''}`}
                  onClick={() => handleImageClick(index)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      handleImageClick(index)
                    }
                  }}
                >
                  <div className="attachment-viewer-thumbnail">
                    <img
                      src={attachment.thumbnail_url || attachment.url}
                      alt={attachment.file_name || 'Image'}
                      loading="lazy"
                    />
                    {attachment.id === mainImage?.id && (
                      <div className="attachment-viewer-main-badge">
                        <IconStar size="small" />
                        <span>{t('attachments.mainImage')}</span>
                      </div>
                    )}
                  </div>
                  <Text size="small" ellipsis className="attachment-viewer-name">
                    {attachment.file_name}
                  </Text>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Documents section */}
        {documents.length > 0 && (
          <div className="attachment-viewer-section">
            <Title heading={6} className="attachment-viewer-section-title">
              {t('attachments.documents')} ({documents.length})
            </Title>
            <div className="attachment-viewer-documents">
              {documents.map((attachment) => (
                <a
                  key={attachment.id}
                  href={attachment.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="attachment-viewer-document"
                >
                  <IconFile size="large" className="attachment-viewer-document-icon" />
                  <div className="attachment-viewer-document-info">
                    <Text ellipsis className="attachment-viewer-document-name">
                      {attachment.file_name}
                    </Text>
                    <Text type="tertiary" size="small">
                      {formatFileSize(attachment.file_size)}
                    </Text>
                  </div>
                </a>
              ))}
            </div>
          </div>
        )}

        {/* Image preview modal */}
        <Modal
          visible={previewVisible}
          onCancel={() => setPreviewVisible(false)}
          footer={null}
          centered
          width="auto"
          className="attachment-preview-modal"
          closeOnEsc
        >
          {images.length > 0 && (
            <div className="attachment-preview-container">
              <img
                src={images[previewIndex]?.url}
                alt={images[previewIndex]?.file_name || 'Preview'}
                className="attachment-preview-image"
              />
              {images.length > 1 && (
                <div className="attachment-preview-nav">
                  <button
                    className="attachment-preview-nav-btn"
                    onClick={() => setPreviewIndex((i) => (i > 0 ? i - 1 : images.length - 1))}
                    aria-label="Previous"
                  >
                    <IconChevronLeft size="large" />
                  </button>
                  <span className="attachment-preview-counter">
                    {previewIndex + 1} / {images.length}
                  </span>
                  <button
                    className="attachment-preview-nav-btn"
                    onClick={() => setPreviewIndex((i) => (i < images.length - 1 ? i + 1 : 0))}
                    aria-label="Next"
                  >
                    <IconChevronRight size="large" />
                  </button>
                </div>
              )}
            </div>
          )}
        </Modal>
      </div>
    )
  }

  if (!showCard) {
    return renderContent()
  }

  return (
    <Card className="attachment-viewer-card" title={title || t('attachments.title')}>
      {renderContent()}
    </Card>
  )
}

export default ProductAttachmentViewer
