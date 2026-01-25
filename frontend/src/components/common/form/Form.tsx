import type { ReactNode } from 'react'
import { Button, Space, Spin } from '@douyinfe/semi-ui-19'
import './Form.css'

interface FormProps {
  /** Form submit handler */
  onSubmit: (e?: React.BaseSyntheticEvent) => Promise<void>
  /** Whether the form is submitting */
  isSubmitting?: boolean
  /** Form children */
  children: ReactNode
  /** Additional class name */
  className?: string
  /** Form layout */
  layout?: 'horizontal' | 'vertical'
}

/**
 * Form wrapper component with consistent styling
 */
export function Form({
  onSubmit,
  isSubmitting = false,
  children,
  className = '',
  layout = 'vertical',
}: FormProps) {
  return (
    <form onSubmit={onSubmit} className={`form form--${layout} ${className}`} noValidate>
      <Spin spinning={isSubmitting}>{children}</Spin>
    </form>
  )
}

interface FormActionsProps {
  /** Submit button text */
  submitText?: string
  /** Cancel button text */
  cancelText?: string
  /** Whether submit is in progress */
  isSubmitting?: boolean
  /** Whether submit is disabled */
  disabled?: boolean
  /** Cancel handler */
  onCancel?: () => void
  /** Show cancel button */
  showCancel?: boolean
  /** Additional class name */
  className?: string
  /** Alignment of buttons */
  align?: 'left' | 'center' | 'right'
  /** Additional buttons */
  extra?: ReactNode
}

/**
 * Form action buttons (submit, cancel, etc.)
 */
export function FormActions({
  submitText = '提交',
  cancelText = '取消',
  isSubmitting = false,
  disabled = false,
  onCancel,
  showCancel = true,
  className = '',
  align = 'right',
  extra,
}: FormActionsProps) {
  return (
    <div className={`form-actions form-actions--${align} ${className}`}>
      <Space>
        {extra}
        {showCancel && onCancel && (
          <Button onClick={onCancel} disabled={isSubmitting}>
            {cancelText}
          </Button>
        )}
        <Button
          htmlType="submit"
          theme="solid"
          loading={isSubmitting}
          disabled={disabled || isSubmitting}
        >
          {submitText}
        </Button>
      </Space>
    </div>
  )
}

interface FormSectionProps {
  /** Section title */
  title?: string
  /** Section description */
  description?: string
  /** Section content */
  children: ReactNode
  /** Additional class name */
  className?: string
}

/**
 * Form section with title and description
 */
export function FormSection({ title, description, children, className = '' }: FormSectionProps) {
  return (
    <div className={`form-section ${className}`}>
      {(title || description) && (
        <div className="form-section-header">
          {title && <h3 className="form-section-title">{title}</h3>}
          {description && <p className="form-section-description">{description}</p>}
        </div>
      )}
      <div className="form-section-content">{children}</div>
    </div>
  )
}

interface FormRowProps {
  /** Row children (form fields) */
  children: ReactNode
  /** Number of columns */
  cols?: 1 | 2 | 3 | 4
  /** Gap between columns */
  gap?: 'small' | 'medium' | 'large'
  /** Additional class name */
  className?: string
}

/**
 * Form row for horizontal layout of fields
 */
export function FormRow({ children, cols = 2, gap = 'medium', className = '' }: FormRowProps) {
  return (
    <div className={`form-row form-row--cols-${cols} form-row--gap-${gap} ${className}`}>
      {children}
    </div>
  )
}
