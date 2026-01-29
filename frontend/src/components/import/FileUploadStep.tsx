import { useCallback, useRef, useState } from 'react'
import { Button, Typography, Card, Space, Toast } from '@douyinfe/semi-ui-19'
import { IconUpload, IconFile, IconDelete, IconDownload } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import type { FileUploadStepProps } from './types'
import './FileUploadStep.css'

const { Title, Text } = Typography

// Maximum file size: 10MB
const MAX_FILE_SIZE = 10 * 1024 * 1024
// Accepted file types
const ACCEPTED_TYPES = '.csv,text/csv,application/vnd.ms-excel'

/**
 * FileUploadStep component for file selection
 * Supports drag & drop and click to select
 */
export function FileUploadStep({
  file,
  onFileSelect,
  templateUrl,
  entityType,
  disabled = false,
}: FileUploadStepProps) {
  const { t } = useTranslation('common')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [isDragOver, setIsDragOver] = useState(false)

  // Get entity type display name
  const getEntityDisplayName = useCallback(() => {
    const names: Record<string, string> = {
      products: t('import.entityTypes.products'),
      customers: t('import.entityTypes.customers'),
      suppliers: t('import.entityTypes.suppliers'),
      inventory: t('import.entityTypes.inventory'),
      categories: t('import.entityTypes.categories'),
    }
    return names[entityType] || entityType
  }, [entityType, t])

  // Format file size for display
  const formatFileSize = useCallback((bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }, [])

  // Validate file before accepting
  const validateFile = useCallback(
    (fileToValidate: File): boolean => {
      // Check file size
      if (fileToValidate.size > MAX_FILE_SIZE) {
        Toast.error(t('import.errors.fileTooLarge', { maxSize: '10MB' }))
        return false
      }

      // Check file extension
      const fileName = fileToValidate.name.toLowerCase()
      if (!fileName.endsWith('.csv')) {
        Toast.error(t('import.errors.invalidFileType'))
        return false
      }

      return true
    },
    [t]
  )

  // Handle file selection
  const handleFileChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const files = event.target.files
      if (files && files.length > 0) {
        const selectedFile = files[0]
        if (validateFile(selectedFile)) {
          onFileSelect(selectedFile)
        }
      }
      // Reset input to allow selecting the same file again
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    },
    [onFileSelect, validateFile]
  )

  // Handle drag events
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(true)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragOver(false)
  }, [])

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setIsDragOver(false)

      if (disabled) return

      const files = e.dataTransfer.files
      if (files && files.length > 0) {
        const droppedFile = files[0]
        if (validateFile(droppedFile)) {
          onFileSelect(droppedFile)
        }
      }
    },
    [disabled, onFileSelect, validateFile]
  )

  // Handle click to select file
  const handleClickUpload = useCallback(() => {
    if (!disabled && fileInputRef.current) {
      fileInputRef.current.click()
    }
  }, [disabled])

  // Handle remove file
  const handleRemoveFile = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation()
      // Reset by selecting a placeholder file - parent handles null state
      onFileSelect(null as unknown as File)
    },
    [onFileSelect]
  )

  // Handle template download
  const handleDownloadTemplate = useCallback(() => {
    if (templateUrl) {
      window.open(templateUrl, '_blank')
    }
  }, [templateUrl])

  return (
    <div className="file-upload-step">
      <div className="file-upload-header">
        <Title heading={5}>{t('import.upload.title', { entity: getEntityDisplayName() })}</Title>
        <Text type="secondary">{t('import.upload.description')}</Text>
      </div>

      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        accept={ACCEPTED_TYPES}
        onChange={handleFileChange}
        className="file-upload-input-hidden"
        aria-label={t('import.upload.selectFile')}
      />

      {/* Drop zone */}
      <div
        className={`file-upload-dropzone ${isDragOver ? 'file-upload-dropzone--drag-over' : ''} ${
          disabled ? 'file-upload-dropzone--disabled' : ''
        } ${file ? 'file-upload-dropzone--has-file' : ''}`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={handleClickUpload}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            handleClickUpload()
          }
        }}
        role="button"
        tabIndex={disabled ? -1 : 0}
        aria-label={t('import.upload.dropzone')}
      >
        {file ? (
          <Card className="file-upload-file-card" shadows="hover">
            <div className="file-upload-file-info">
              <IconFile size="extra-large" className="file-upload-file-icon" />
              <div className="file-upload-file-details">
                <Text strong ellipsis={{ showTooltip: true }} className="file-upload-file-name">
                  {file.name}
                </Text>
                <Text type="secondary" size="small">
                  {formatFileSize(file.size)}
                </Text>
              </div>
              <Button
                icon={<IconDelete />}
                type="danger"
                theme="borderless"
                onClick={handleRemoveFile}
                aria-label={t('import.upload.removeFile')}
              />
            </div>
          </Card>
        ) : (
          <div className="file-upload-empty">
            <IconUpload size="extra-large" className="file-upload-empty-icon" />
            <Text>{t('import.upload.dragDropText')}</Text>
            <Text type="tertiary" size="small">
              {t('import.upload.orClickText')}
            </Text>
            <Text type="tertiary" size="small">
              {t('import.upload.fileRequirements', { maxSize: '10MB' })}
            </Text>
          </div>
        )}
      </div>

      {/* Template download */}
      {templateUrl && (
        <div className="file-upload-template">
          <Space>
            <Text type="secondary">{t('import.upload.templateHint')}</Text>
            <Button
              icon={<IconDownload />}
              theme="borderless"
              size="small"
              onClick={handleDownloadTemplate}
            >
              {t('import.upload.downloadTemplate')}
            </Button>
          </Space>
        </div>
      )}
    </div>
  )
}

export default FileUploadStep
