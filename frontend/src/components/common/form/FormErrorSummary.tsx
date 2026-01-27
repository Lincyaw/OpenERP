import {
  useEffect,
  useRef,
  useCallback,
  useMemo,
  useId,
  type MouseEvent,
  type KeyboardEvent,
} from 'react'
import { useTranslation } from 'react-i18next'
import { Banner } from '@douyinfe/semi-ui-19'
import { IconAlertTriangle } from '@douyinfe/semi-icons'
import type { FieldErrors, FieldValues } from 'react-hook-form'
import './FormErrorSummary.css'

/**
 * Error item structure for form error summary
 */
export interface FormErrorItem {
  /** Field name/path */
  field: string
  /** Error message */
  message: string
  /** Display label for the field */
  label?: string
}

interface FormErrorSummaryProps<T extends FieldValues = FieldValues> {
  /** React Hook Form errors object */
  errors: FieldErrors<T>
  /** Map of field names to display labels */
  fieldLabels?: Record<string, string>
  /** Whether to auto-focus the first error field when clicking */
  autoFocusOnClick?: boolean
  /** Additional class name */
  className?: string
  /** Whether to show the summary */
  show?: boolean
  /** Maximum number of errors to show before collapsing */
  maxVisible?: number
  /** Callback when an error item is clicked */
  onErrorClick?: (field: string) => void
}

/**
 * Flattens nested field errors into a flat array
 */
function flattenErrors<T extends FieldValues>(
  errors: FieldErrors<T>,
  prefix = ''
): Array<{ field: string; message: string }> {
  const result: Array<{ field: string; message: string }> = []

  for (const [key, value] of Object.entries(errors)) {
    const fieldPath = prefix ? `${prefix}.${key}` : key

    if (value && typeof value === 'object') {
      // Check if it's a FieldError with a message
      if ('message' in value && typeof value.message === 'string') {
        result.push({ field: fieldPath, message: value.message })
      }
      // Check for nested errors (e.g., array fields)
      if ('root' in value || !('message' in value)) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        result.push(...flattenErrors(value as FieldErrors<any>, fieldPath))
      }
    }
  }

  return result
}

/**
 * FormErrorSummary - Displays a summary of form validation errors at the top of a form
 *
 * Features:
 * - Accessible error summary with aria-live for screen readers
 * - Click-to-navigate to error fields
 * - Keyboard navigation support
 * - Collapsible when many errors
 * - i18n support
 *
 * @example
 * ```tsx
 * <FormErrorSummary
 *   errors={form.formState.errors}
 *   fieldLabels={{
 *     name: '产品名称',
 *     price: '价格',
 *   }}
 *   show={form.formState.submitCount > 0}
 * />
 * ```
 */
export function FormErrorSummary<T extends FieldValues = FieldValues>({
  errors,
  fieldLabels = {},
  autoFocusOnClick = true,
  className = '',
  show = true,
  maxVisible = 5,
  onErrorClick,
}: FormErrorSummaryProps<T>) {
  const { t } = useTranslation('validation')
  const titleId = useId()
  const summaryRef = useRef<HTMLDivElement>(null)
  const announceRef = useRef<HTMLDivElement>(null)

  // Flatten and format errors
  const errorItems = useMemo((): FormErrorItem[] => {
    const flattened = flattenErrors(errors)
    return flattened.map(({ field, message }) => ({
      field,
      message,
      label: fieldLabels[field] || t(`fieldLabels.${field}`, { defaultValue: field }),
    }))
  }, [errors, fieldLabels, t])

  const errorCount = errorItems.length
  const hasErrors = errorCount > 0 && show

  // Announce errors to screen readers when they change
  useEffect(() => {
    if (hasErrors && announceRef.current) {
      const announcement =
        errorCount === 1
          ? t('errorSummary.single')
          : t('errorSummary.screenReaderAnnounce', { count: errorCount })

      // Use a setTimeout to ensure the announcement is made after render
      const timer = setTimeout(() => {
        if (announceRef.current) {
          announceRef.current.textContent = announcement
        }
      }, 100)

      return () => clearTimeout(timer)
    }
  }, [hasErrors, errorCount, t])

  // Focus the first error field
  const focusField = useCallback((fieldName: string) => {
    // Try different strategies to find the field
    const selectors = [
      `[name="${fieldName}"]`,
      `#${fieldName}`,
      `[id="${fieldName}"]`,
      `[data-field="${fieldName}"]`,
    ]

    for (const selector of selectors) {
      const element = document.querySelector<HTMLElement>(selector)
      if (element) {
        element.focus()
        element.scrollIntoView({ behavior: 'smooth', block: 'center' })
        return true
      }
    }

    return false
  }, [])

  const handleErrorClick = useCallback(
    (field: string) => (event: MouseEvent | KeyboardEvent) => {
      event.preventDefault()

      if (autoFocusOnClick) {
        focusField(field)
      }

      onErrorClick?.(field)
    },
    [autoFocusOnClick, focusField, onErrorClick]
  )

  const handleKeyDown = useCallback(
    (field: string) => (event: KeyboardEvent) => {
      if (event.key === 'Enter' || event.key === ' ') {
        handleErrorClick(field)(event)
      }
    },
    [handleErrorClick]
  )

  if (!hasErrors) {
    return null
  }

  const visibleErrors = maxVisible > 0 ? errorItems.slice(0, maxVisible) : errorItems
  const hiddenCount = errorItems.length - visibleErrors.length

  return (
    <>
      {/* Screen reader announcement (aria-live region) */}
      <div
        ref={announceRef}
        className="sr-only"
        role="status"
        aria-live="polite"
        aria-atomic="true"
      />

      {/* Visual error summary */}
      <div
        ref={summaryRef}
        className={`form-error-summary ${className}`}
        role="alert"
        aria-labelledby={titleId}
      >
        <Banner
          type="danger"
          icon={<IconAlertTriangle />}
          closeIcon={null}
          fullMode={false}
          title={
            <span id={titleId} className="form-error-summary__title">
              {t('errorSummary.title')}
              <span className="form-error-summary__count">
                {errorCount === 1
                  ? t('errorSummary.single')
                  : t('errorSummary.multiple', { count: errorCount })}
              </span>
            </span>
          }
          description={
            <ul className="form-error-summary__list" role="list">
              {visibleErrors.map(({ field, message, label }) => (
                <li key={field} className="form-error-summary__item">
                  <a
                    href={`#${field}`}
                    className="form-error-summary__link"
                    onClick={handleErrorClick(field)}
                    onKeyDown={handleKeyDown(field)}
                    tabIndex={0}
                  >
                    <strong className="form-error-summary__field">{label}:</strong>
                    <span className="form-error-summary__message">{message}</span>
                  </a>
                </li>
              ))}
              {hiddenCount > 0 && (
                <li className="form-error-summary__more">
                  {t('errorSummary.multiple', { count: hiddenCount })}
                </li>
              )}
            </ul>
          }
        />
      </div>
    </>
  )
}

export default FormErrorSummary
