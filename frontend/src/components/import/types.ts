import type { CsvimportEntityType, CsvimportRowError } from '@/api/models'

/**
 * Supported entity types for import
 */
export type EntityType = CsvimportEntityType

/**
 * Conflict resolution modes
 */
export type ConflictMode = 'skip' | 'update' | 'fail'

/**
 * Wizard step identifiers
 */
export type WizardStep = 'upload' | 'validate' | 'import' | 'results'

/**
 * Validation result from backend
 */
export interface ValidationResult {
  validationId: string
  totalRows: number
  validRows: number
  errorRows: number
  errors: CsvimportRowError[]
  preview: Array<Record<string, unknown>>
  warnings: string[]
  isTruncated: boolean
  totalErrors: number
}

/**
 * Import result from backend
 */
export interface ImportResult {
  totalRows: number
  importedRows: number
  updatedRows: number
  skippedRows: number
  errorRows: number
  errors: CsvimportRowError[]
  isTruncated: boolean
  totalErrors: number
}

/**
 * Props for the ImportWizard component
 */
export interface ImportWizardProps {
  /** The entity type to import */
  entityType: EntityType
  /** Whether the wizard modal is visible */
  visible: boolean
  /** Callback when modal is closed */
  onClose: () => void
  /** Callback when import is successful */
  onSuccess?: () => void
  /** Template download URL */
  templateUrl?: string
}

/**
 * Props for FileUploadStep
 */
export interface FileUploadStepProps {
  /** Selected file */
  file: File | null
  /** File selection callback */
  onFileSelect: (file: File) => void
  /** Template download URL */
  templateUrl?: string
  /** Entity type for display */
  entityType: EntityType
  /** Whether upload is disabled */
  disabled?: boolean
}

/**
 * Props for ValidationStep
 */
export interface ValidationStepProps {
  /** Validation result */
  result: ValidationResult | null
  /** Whether validation is in progress */
  loading: boolean
  /** Error message if validation failed */
  error: string | null
  /** Callback to retry validation */
  onRetry: () => void
  /** Callback to proceed with import */
  onProceed: () => void
  /** Whether there are valid rows to import */
  hasValidRows: boolean
}

/**
 * Props for ImportStep
 */
export interface ImportStepProps {
  /** Selected conflict mode */
  conflictMode: ConflictMode
  /** Callback when conflict mode changes */
  onConflictModeChange: (mode: ConflictMode) => void
  /** Callback to start import */
  onImport: () => void
  /** Number of valid rows to import */
  validRowCount: number
  /** Whether import is in progress */
  loading: boolean
}

/**
 * Props for ResultsStep
 */
export interface ResultsStepProps {
  /** Import result */
  result: ImportResult | null
  /** Callback to import more files */
  onImportMore: () => void
  /** Callback to close wizard */
  onClose: () => void
}

/**
 * Props for ErrorTable
 */
export interface ErrorTableProps {
  /** List of row errors */
  errors: CsvimportRowError[]
  /** Whether errors are truncated */
  isTruncated?: boolean
  /** Total error count (if truncated) */
  totalErrors?: number
  /** Whether to show export button */
  showExport?: boolean
  /** Export callback */
  onExport?: () => void
  /** Maximum rows to display */
  maxRows?: number
}
