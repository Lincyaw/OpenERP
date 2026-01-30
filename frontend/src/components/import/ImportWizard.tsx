import { useState, useCallback, useMemo, useRef } from 'react'
import { Modal, Steps, Toast } from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import type {
  ImportWizardProps,
  WizardStep,
  ValidationResult,
  ImportResult,
  ConflictMode,
  EntityType,
} from './types'
import { FileUploadStep } from './FileUploadStep'
import { ValidationStep } from './ValidationStep'
import { ImportStep } from './ImportStep'
import { ResultsStep } from './ResultsStep'
import { axiosInstance } from '@/services/axios-instance'
import './ImportWizard.css'

const { Step } = Steps

// API endpoints for different entity types
const API_ENDPOINTS: Record<EntityType, { validate: string; import: string }> = {
  products: {
    validate: '/import/products/validate',
    import: '/import/products',
  },
  customers: {
    validate: '/import/customers/validate',
    import: '/import/customers',
  },
  suppliers: {
    validate: '/import/suppliers/validate',
    import: '/import/suppliers',
  },
  inventory: {
    validate: '/import/inventory/validate',
    import: '/import/inventory',
  },
  categories: {
    validate: '/import/categories/validate',
    import: '/import/categories',
  },
}

// Map wizard step to step index
const STEP_INDEX: Record<WizardStep, number> = {
  upload: 0,
  validate: 1,
  import: 2,
  results: 3,
}

/**
 * ImportWizard component provides a 4-step import workflow:
 * 1. Upload - File selection and preview
 * 2. Validate - CSV validation and error review
 * 3. Import - Configure conflict mode and execute
 * 4. Results - View import results
 */
export function ImportWizard({
  entityType,
  visible,
  onClose,
  onSuccess,
  templateUrl,
}: ImportWizardProps) {
  const { t } = useTranslation('common')
  const abortControllerRef = useRef<AbortController | null>(null)

  // State
  const [currentStep, setCurrentStep] = useState<WizardStep>('upload')
  const [file, setFile] = useState<File | null>(null)
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null)
  const [importResult, setImportResult] = useState<ImportResult | null>(null)
  const [conflictMode, setConflictMode] = useState<ConflictMode>('skip')
  const [validating, setValidating] = useState(false)
  const [importing, setImporting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Get entity display name
  const entityDisplayName = useMemo(() => {
    const names: Record<string, string> = {
      products: t('import.entityTypes.products'),
      customers: t('import.entityTypes.customers'),
      suppliers: t('import.entityTypes.suppliers'),
      inventory: t('import.entityTypes.inventory'),
      categories: t('import.entityTypes.categories'),
    }
    return names[entityType] || entityType
  }, [entityType, t])

  // Reset wizard state
  const resetWizard = useCallback(() => {
    setCurrentStep('upload')
    setFile(null)
    setValidationResult(null)
    setImportResult(null)
    setConflictMode('skip')
    setValidating(false)
    setImporting(false)
    setError(null)
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
      abortControllerRef.current = null
    }
  }, [])

  // Handle file selection and auto-validate
  const handleFileSelect = useCallback(
    async (selectedFile: File | null) => {
      if (!selectedFile) {
        setFile(null)
        return
      }

      setFile(selectedFile)
      setError(null)
      setCurrentStep('validate')
      setValidating(true)

      // Abort any previous validation
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      abortControllerRef.current = new AbortController()

      try {
        const formData = new FormData()
        formData.append('file', selectedFile)

        const endpoints = API_ENDPOINTS[entityType]
        if (!endpoints) {
          throw new Error(t('import.errors.unsupportedEntity'))
        }

        const response = await axiosInstance.post(endpoints.validate, formData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
          signal: abortControllerRef.current.signal,
        })

        const data = response.data
        if (data.success && data.data) {
          const result: ValidationResult = {
            validationId: data.data.validation_id,
            totalRows: data.data.total_rows || 0,
            validRows: data.data.valid_rows || 0,
            errorRows: data.data.error_rows || 0,
            errors: data.data.errors || [],
            preview: data.data.preview || [],
            warnings: data.data.warnings || [],
            isTruncated: data.data.is_truncated || false,
            totalErrors: data.data.total_errors || 0,
          }
          setValidationResult(result)
        } else {
          throw new Error(data.error || t('import.errors.validationFailed'))
        }
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') {
          return // Ignore aborted requests
        }
        const errorMessage =
          err instanceof Error ? err.message : t('import.errors.validationFailed')
        setError(errorMessage)
        Toast.error(errorMessage)
      } finally {
        setValidating(false)
      }
    },
    [entityType, t]
  )

  // Handle retry validation
  const handleRetry = useCallback(() => {
    setError(null)
    setValidationResult(null)
    setCurrentStep('upload')
  }, [])

  // Handle proceed to import step
  const handleProceed = useCallback(() => {
    if (!validationResult || validationResult.validRows === 0) {
      Toast.error(t('import.errors.noValidRows'))
      return
    }
    setCurrentStep('import')
  }, [validationResult, t])

  // Handle import execution
  const handleImport = useCallback(async () => {
    if (!validationResult) {
      Toast.error(t('import.errors.noValidation'))
      return
    }

    setImporting(true)
    setError(null)

    try {
      const endpoints = API_ENDPOINTS[entityType]
      if (!endpoints) {
        throw new Error(t('import.errors.unsupportedEntity'))
      }

      const response = await axiosInstance.post(endpoints.import, {
        validation_id: validationResult.validationId,
        conflict_mode: conflictMode,
      })

      const data = response.data
      if (data.success && data.data) {
        const result: ImportResult = {
          totalRows: data.data.total_rows || 0,
          importedRows: data.data.imported_rows || 0,
          updatedRows: data.data.updated_rows || 0,
          skippedRows: data.data.skipped_rows || 0,
          errorRows: data.data.error_rows || 0,
          errors: data.data.errors || [],
          isTruncated: data.data.is_truncated || false,
          totalErrors: data.data.total_errors || 0,
        }
        setImportResult(result)
        setCurrentStep('results')

        // Notify success if any rows were imported
        if (result.importedRows > 0 || result.updatedRows > 0) {
          Toast.success(
            t('import.messages.importSuccess', {
              imported: result.importedRows,
              updated: result.updatedRows,
            })
          )
          onSuccess?.()
        }
      } else {
        throw new Error(data.error || t('import.errors.importFailed'))
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('import.errors.importFailed')
      setError(errorMessage)
      Toast.error(errorMessage)
    } finally {
      setImporting(false)
    }
  }, [validationResult, conflictMode, entityType, t, onSuccess])

  // Handle import more
  const handleImportMore = useCallback(() => {
    resetWizard()
  }, [resetWizard])

  // Handle modal close
  const handleClose = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }
    onClose()
    // Reset after modal animation completes
    setTimeout(resetWizard, 300)
  }, [onClose, resetWizard])

  // Step titles
  const stepTitles = useMemo(
    () => [
      t('import.steps.upload'),
      t('import.steps.validate'),
      t('import.steps.import'),
      t('import.steps.results'),
    ],
    [t]
  )

  // Render current step content
  const renderStepContent = useCallback(() => {
    switch (currentStep) {
      case 'upload':
        return (
          <FileUploadStep
            file={file}
            onFileSelect={handleFileSelect}
            templateUrl={templateUrl}
            entityType={entityType}
            disabled={validating}
          />
        )
      case 'validate':
        return (
          <ValidationStep
            result={validationResult}
            loading={validating}
            error={error}
            onRetry={handleRetry}
            onProceed={handleProceed}
            hasValidRows={!!validationResult && validationResult.validRows > 0}
          />
        )
      case 'import':
        return (
          <ImportStep
            conflictMode={conflictMode}
            onConflictModeChange={setConflictMode}
            onImport={handleImport}
            validRowCount={validationResult?.validRows || 0}
            loading={importing}
          />
        )
      case 'results':
        return (
          <ResultsStep
            result={importResult}
            onImportMore={handleImportMore}
            onClose={handleClose}
          />
        )
      default:
        return null
    }
  }, [
    currentStep,
    file,
    handleFileSelect,
    templateUrl,
    entityType,
    validating,
    validationResult,
    error,
    handleRetry,
    handleProceed,
    conflictMode,
    handleImport,
    importing,
    importResult,
    handleImportMore,
    handleClose,
  ])

  return (
    <Modal
      visible={visible}
      onCancel={handleClose}
      title={t('import.wizard.title', { entity: entityDisplayName })}
      footer={null}
      centered
      closable={!validating && !importing}
      maskClosable={!validating && !importing}
      className="import-wizard-modal"
      aria-label={t('import.wizard.title', { entity: entityDisplayName })}
    >
      <div className="import-wizard">
        {/* Steps indicator */}
        <Steps current={STEP_INDEX[currentStep]} className="import-wizard-steps">
          {stepTitles.map((title, index) => (
            <Step key={index} title={title} />
          ))}
        </Steps>

        {/* Step content */}
        <div className="import-wizard-content">{renderStepContent()}</div>
      </div>
    </Modal>
  )
}

export default ImportWizard
