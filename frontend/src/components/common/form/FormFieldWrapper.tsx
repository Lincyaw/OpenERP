import type { ReactNode } from 'react'
import './FormFieldWrapper.css'

interface FormFieldWrapperProps {
  label?: string
  required?: boolean
  error?: string
  helperText?: string
  labelPosition?: 'top' | 'left' | 'inset'
  hideLabel?: boolean
  className?: string
  children: ReactNode
  htmlFor?: string
}

/**
 * Wrapper component for form fields providing consistent styling
 * for labels, errors, and helper text
 */
export function FormFieldWrapper({
  label,
  required,
  error,
  helperText,
  labelPosition = 'top',
  hideLabel = false,
  className = '',
  children,
  htmlFor,
}: FormFieldWrapperProps) {
  const hasError = !!error
  const showHelperText = !hasError && helperText

  return (
    <div
      className={`form-field-wrapper form-field-wrapper--${labelPosition} ${hasError ? 'form-field-wrapper--error' : ''} ${className}`}
    >
      {label && !hideLabel && (
        <label className="form-field-label" htmlFor={htmlFor}>
          {label}
          {required && <span className="form-field-required">*</span>}
        </label>
      )}
      {label && hideLabel && (
        <label className="form-field-label form-field-label--hidden" htmlFor={htmlFor}>
          {label}
          {required && <span className="form-field-required">*</span>}
        </label>
      )}
      <div className="form-field-control">{children}</div>
      {hasError && <span className="form-field-error">{error}</span>}
      {showHelperText && <span className="form-field-helper">{helperText}</span>}
    </div>
  )
}

export default FormFieldWrapper
