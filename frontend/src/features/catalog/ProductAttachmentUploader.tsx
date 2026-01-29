import { useState, useCallback, useRef } from 'react'
import { Card, Toast, Spin, Button, Modal, Progress, Typography, Empty } from '@douyinfe/semi-ui-19'
import { IconUpload, IconDelete, IconImage, IconStar, IconStarStroked } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import {
  useListProductAttachments,
  useInitiateProductAttachmentUpload,
  useConfirmProductAttachmentUpload,
  useDeleteProductAttachment,
  useSetProductAttachmentAsMainImage,
  getListProductAttachmentsQueryKey,
} from '@/api/product-attachments/product-attachments'
import type { CatalogAttachmentListResponse, HandlerInitiateUploadRequest } from '@/api/models'
import { HandlerInitiateUploadRequestType } from '@/api/models/handlerInitiateUploadRequestType'
import './ProductAttachmentUploader.css'

const { Text, Title } = Typography

/** Allowed file types for image attachments */
const ALLOWED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']

/** Allowed file types for documents */
const ALLOWED_DOCUMENT_TYPES = [
  'application/pdf',
  'application/msword',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  'application/vnd.ms-excel',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  'text/plain',
  'text/csv',
]

/** All allowed file types */
const ALLOWED_TYPES = [...ALLOWED_IMAGE_TYPES, ...ALLOWED_DOCUMENT_TYPES]

/** Max file size: 100MB */
const MAX_FILE_SIZE = 100 * 1024 * 1024

interface UploadingFile {
  id: string
  file: File
  progress: number
  status: 'uploading' | 'confirming' | 'done' | 'error'
  error?: string
  attachmentId?: string
}

interface ProductAttachmentUploaderProps {
  /** Product ID to upload attachments for */
  productId: string
  /** Whether the uploader is disabled */
  disabled?: boolean
  /** Callback when upload completes */
  onUploadComplete?: () => void
}

/**
 * Product attachment uploader component
 *
 * Features:
 * - Presigned URL upload to object storage
 * - Image preview with thumbnails
 * - Set main image
 * - Delete attachments
 * - Upload progress
 * - File type and size validation
 */
export function ProductAttachmentUploader({
  productId,
  disabled = false,
  onUploadComplete,
}: ProductAttachmentUploaderProps) {
  const { t } = useTranslation(['catalog', 'common'])
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)

  // State
  const [uploadingFiles, setUploadingFiles] = useState<UploadingFile[]>([])
  const [previewImage, setPreviewImage] = useState<string | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  // Fetch existing attachments
  const { data: attachmentsResponse, isLoading: isLoadingAttachments } = useListProductAttachments(
    productId,
    { status: 'active' },
    {
      query: {
        enabled: !!productId,
      },
    }
  )

  // Mutations
  const initiateUpload = useInitiateProductAttachmentUpload()
  const confirmUpload = useConfirmProductAttachmentUpload()
  const deleteAttachment = useDeleteProductAttachment()
  const setMainImage = useSetProductAttachmentAsMainImage()

  // Extract attachments from response
  const attachments = (
    attachmentsResponse?.status === 200 ? attachmentsResponse.data.data : []
  ) as CatalogAttachmentListResponse[]

  // Find main image
  const mainImage = attachments.find((a) => a.type === 'main_image')

  /**
   * Validate a file before upload
   */
  const validateFile = useCallback(
    (file: File): string | null => {
      if (!ALLOWED_TYPES.includes(file.type)) {
        return t('attachments.errors.invalidType')
      }
      if (file.size > MAX_FILE_SIZE) {
        return t('attachments.errors.fileTooLarge', {
          max: formatFileSize(MAX_FILE_SIZE),
        })
      }
      return null
    },
    [t]
  )

  /**
   * Upload a file using presigned URL
   */
  const uploadFile = useCallback(
    async (file: File) => {
      const fileId = crypto.randomUUID()
      const isImage = ALLOWED_IMAGE_TYPES.includes(file.type)

      // Add to uploading list
      setUploadingFiles((prev) => [
        ...prev,
        {
          id: fileId,
          file,
          progress: 0,
          status: 'uploading',
        },
      ])

      try {
        // Step 1: Initiate upload to get presigned URL
        const uploadRequest: HandlerInitiateUploadRequest = {
          product_id: productId,
          file_name: file.name,
          file_size: file.size,
          content_type: file.type,
          type: isImage
            ? HandlerInitiateUploadRequestType.gallery_image
            : HandlerInitiateUploadRequestType.document,
        }

        const initiateResponse = await initiateUpload.mutateAsync({
          data: uploadRequest,
        })

        if (initiateResponse.status !== 201 || !initiateResponse.data.data) {
          throw new Error(
            (initiateResponse.data as { error?: { message?: string } })?.error?.message ||
              t('attachments.errors.initiateError')
          )
        }

        const { upload_url, attachment_id } = initiateResponse.data.data

        if (!upload_url || !attachment_id) {
          throw new Error(t('attachments.errors.initiateError'))
        }

        // Update with attachment ID
        setUploadingFiles((prev) =>
          prev.map((f) => (f.id === fileId ? { ...f, attachmentId: attachment_id } : f))
        )

        // Step 2: Upload directly to storage using presigned URL
        await uploadToStorage(upload_url, file, (progress) => {
          setUploadingFiles((prev) => prev.map((f) => (f.id === fileId ? { ...f, progress } : f)))
        })

        // Step 3: Confirm upload
        setUploadingFiles((prev) =>
          prev.map((f) => (f.id === fileId ? { ...f, status: 'confirming', progress: 100 } : f))
        )

        const confirmResponse = await confirmUpload.mutateAsync({
          id: attachment_id,
          data: {},
        })

        if (confirmResponse.status !== 200) {
          throw new Error(
            (confirmResponse.data as { error?: { message?: string } })?.error?.message ||
              t('attachments.errors.confirmError')
          )
        }

        // Success
        setUploadingFiles((prev) =>
          prev.map((f) => (f.id === fileId ? { ...f, status: 'done' } : f))
        )

        // Refresh attachments list
        await queryClient.invalidateQueries({
          queryKey: getListProductAttachmentsQueryKey(productId),
        })

        // Remove from uploading list after delay
        setTimeout(() => {
          setUploadingFiles((prev) => prev.filter((f) => f.id !== fileId))
        }, 1500)

        onUploadComplete?.()
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : t('attachments.errors.uploadError')

        setUploadingFiles((prev) =>
          prev.map((f) =>
            f.id === fileId ? { ...f, status: 'error' as const, error: errorMessage } : f
          )
        )

        Toast.error(errorMessage)
      }
    },
    [productId, initiateUpload, confirmUpload, queryClient, t, onUploadComplete]
  )

  /**
   * Handle file selection
   */
  const handleFileSelect = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files || [])

      files.forEach((file) => {
        const error = validateFile(file)
        if (error) {
          Toast.error(`${file.name}: ${error}`)
          return
        }
        uploadFile(file)
      })

      // Reset input
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    },
    [validateFile, uploadFile]
  )

  /**
   * Handle drag and drop
   */
  const handleDrop = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      event.preventDefault()
      event.stopPropagation()

      if (disabled) return

      const files = Array.from(event.dataTransfer.files)
      files.forEach((file) => {
        const error = validateFile(file)
        if (error) {
          Toast.error(`${file.name}: ${error}`)
          return
        }
        uploadFile(file)
      })
    },
    [disabled, validateFile, uploadFile]
  )

  /**
   * Handle delete attachment
   */
  const handleDelete = useCallback(
    async (attachmentId: string) => {
      try {
        const response = await deleteAttachment.mutateAsync({ id: attachmentId })
        if (response.status !== 204) {
          throw new Error(
            (response.data as { error?: { message?: string } })?.error?.message ||
              t('attachments.errors.deleteError')
          )
        }

        Toast.success(t('attachments.messages.deleteSuccess'))
        await queryClient.invalidateQueries({
          queryKey: getListProductAttachmentsQueryKey(productId),
        })
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : t('attachments.errors.deleteError')
        Toast.error(errorMessage)
      } finally {
        setDeleteConfirmId(null)
      }
    },
    [deleteAttachment, productId, queryClient, t]
  )

  /**
   * Handle set as main image
   */
  const handleSetMainImage = useCallback(
    async (attachmentId: string) => {
      try {
        const response = await setMainImage.mutateAsync({
          id: attachmentId,
          data: {},
        })

        if (response.status !== 200) {
          throw new Error(
            (response.data as { error?: { message?: string } })?.error?.message ||
              t('attachments.errors.setMainError')
          )
        }

        Toast.success(t('attachments.messages.setMainSuccess'))
        await queryClient.invalidateQueries({
          queryKey: getListProductAttachmentsQueryKey(productId),
        })
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : t('attachments.errors.setMainError')
        Toast.error(errorMessage)
      }
    },
    [setMainImage, productId, queryClient, t]
  )

  /**
   * Check if an attachment is an image
   */
  const isImageAttachment = (attachment: CatalogAttachmentListResponse): boolean => {
    return (
      attachment.type === 'main_image' ||
      attachment.type === 'gallery_image' ||
      (attachment.content_type?.startsWith('image/') ?? false)
    )
  }

  return (
    <Card className="product-attachment-uploader">
      <div className="attachment-uploader-header">
        <Title heading={5} className="attachment-uploader-title">
          {t('attachments.title')}
        </Title>
        <Text type="secondary" size="small">
          {t('attachments.description')}
        </Text>
      </div>

      {/* Upload zone */}
      <div
        className={`attachment-upload-zone ${disabled ? 'disabled' : ''}`}
        onClick={() => !disabled && fileInputRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault()
          e.stopPropagation()
        }}
        onDrop={handleDrop}
        role="button"
        tabIndex={disabled ? -1 : 0}
        aria-label={t('attachments.uploadArea')}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            fileInputRef.current?.click()
          }
        }}
      >
        <IconUpload size="extra-large" className="upload-icon" />
        <Text className="upload-text">{t('attachments.dragDropHint')}</Text>
        <Text type="secondary" size="small" className="upload-hint">
          {t('attachments.allowedTypes')}
        </Text>
        <Text type="tertiary" size="small" className="upload-size-hint">
          {t('attachments.maxSize', { max: formatFileSize(MAX_FILE_SIZE) })}
        </Text>
        <input
          ref={fileInputRef}
          type="file"
          multiple
          accept={ALLOWED_TYPES.join(',')}
          onChange={handleFileSelect}
          style={{ display: 'none' }}
          disabled={disabled}
          aria-hidden="true"
        />
      </div>

      {/* Uploading files progress */}
      {uploadingFiles.length > 0 && (
        <div className="uploading-files-list">
          {uploadingFiles.map((item) => (
            <div key={item.id} className="uploading-file-item">
              <div className="uploading-file-info">
                <IconImage size="small" />
                <Text size="small" ellipsis className="uploading-file-name">
                  {item.file.name}
                </Text>
                <Text type="secondary" size="small">
                  {formatFileSize(item.file.size)}
                </Text>
              </div>
              <div className="uploading-file-progress">
                {item.status === 'error' ? (
                  <Text type="danger" size="small">
                    {item.error}
                  </Text>
                ) : (
                  <Progress
                    percent={item.progress}
                    size="small"
                    showInfo
                    stroke={item.status === 'done' ? 'var(--semi-color-success)' : undefined}
                  />
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Attachments grid */}
      {isLoadingAttachments ? (
        <div className="attachments-loading">
          <Spin size="large" />
        </div>
      ) : attachments.length === 0 && uploadingFiles.length === 0 ? (
        <Empty
          image={<IconImage size="extra-large" />}
          description={t('attachments.empty')}
          className="attachments-empty"
        />
      ) : (
        <div className="attachments-grid">
          {attachments.map((attachment) => (
            <div
              key={attachment.id}
              className={`attachment-item ${attachment.id === mainImage?.id ? 'main-image' : ''}`}
            >
              {isImageAttachment(attachment) ? (
                <div
                  className="attachment-thumbnail"
                  onClick={() => setPreviewImage(attachment.url || null)}
                  role="button"
                  tabIndex={0}
                  aria-label={t('attachments.previewImage')}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      setPreviewImage(attachment.url || null)
                    }
                  }}
                >
                  <img
                    src={attachment.thumbnail_url || attachment.url}
                    alt={attachment.file_name || 'Attachment'}
                    loading="lazy"
                  />
                  {attachment.id === mainImage?.id && (
                    <div className="main-image-badge">
                      <IconStar size="small" />
                      <span>{t('attachments.mainImage')}</span>
                    </div>
                  )}
                </div>
              ) : (
                <div className="attachment-document">
                  <IconImage size="large" />
                  <Text size="small" ellipsis>
                    {attachment.file_name}
                  </Text>
                </div>
              )}

              <div className="attachment-actions">
                <Text
                  size="small"
                  ellipsis
                  className="attachment-name"
                  title={attachment.file_name}
                >
                  {attachment.file_name}
                </Text>
                <div className="attachment-action-buttons">
                  {isImageAttachment(attachment) && attachment.id !== mainImage?.id && (
                    <Button
                      icon={<IconStarStroked />}
                      size="small"
                      theme="borderless"
                      onClick={() => handleSetMainImage(attachment.id!)}
                      disabled={disabled || setMainImage.isPending}
                      aria-label={t('attachments.setAsMain')}
                      title={t('attachments.setAsMain')}
                    />
                  )}
                  <Button
                    icon={<IconDelete />}
                    size="small"
                    theme="borderless"
                    type="danger"
                    onClick={() => setDeleteConfirmId(attachment.id!)}
                    disabled={disabled || deleteAttachment.isPending}
                    aria-label={t('attachments.delete')}
                    title={t('attachments.delete')}
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Image preview modal */}
      <Modal
        visible={!!previewImage}
        onCancel={() => setPreviewImage(null)}
        footer={null}
        centered
        width="auto"
        className="attachment-preview-modal"
        closeOnEsc
      >
        {previewImage && <img src={previewImage} alt="Preview" className="preview-image" />}
      </Modal>

      {/* Delete confirmation modal */}
      <Modal
        visible={!!deleteConfirmId}
        onCancel={() => setDeleteConfirmId(null)}
        onOk={() => {
          if (deleteConfirmId) {
            handleDelete(deleteConfirmId)
          }
        }}
        title={t('attachments.confirmDelete.title')}
        okText={t('common:actions.delete')}
        cancelText={t('common:actions.cancel')}
        okButtonProps={{ type: 'danger', loading: deleteAttachment.isPending }}
        centered
        closeOnEsc
      >
        <Text>{t('attachments.confirmDelete.content')}</Text>
      </Modal>
    </Card>
  )
}

/**
 * Upload file to storage using presigned URL
 */
async function uploadToStorage(
  presignedUrl: string,
  file: File,
  onProgress: (progress: number) => void
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()

    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) {
        const progress = Math.round((event.loaded / event.total) * 100)
        onProgress(progress)
      }
    })

    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve()
      } else {
        reject(new Error(`Upload failed with status ${xhr.status}`))
      }
    })

    xhr.addEventListener('error', () => {
      reject(new Error('Upload failed'))
    })

    xhr.addEventListener('abort', () => {
      reject(new Error('Upload aborted'))
    })

    xhr.open('PUT', presignedUrl)
    xhr.setRequestHeader('Content-Type', file.type)
    xhr.send(file)
  })
}

/**
 * Format file size in human-readable format
 */
function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

export default ProductAttachmentUploader
