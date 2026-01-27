import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Tooltip } from '@douyinfe/semi-ui-19'
import { IconHelpCircle } from '@douyinfe/semi-icons'
import './FormFieldHelp.css'

interface FormFieldHelpProps {
  /** Help text content (can be string or JSX) */
  content: ReactNode
  /** Optional title for the tooltip */
  title?: string
  /** Position of the tooltip */
  position?:
    | 'top'
    | 'topLeft'
    | 'topRight'
    | 'left'
    | 'leftTop'
    | 'leftBottom'
    | 'right'
    | 'rightTop'
    | 'rightBottom'
    | 'bottom'
    | 'bottomLeft'
    | 'bottomRight'
  /** Size of the help icon */
  size?: 'small' | 'default' | 'large'
  /** Additional class name */
  className?: string
  /** Whether to show as inline text instead of icon */
  inline?: boolean
  /** Icon to use (defaults to help circle) */
  icon?: ReactNode
  /** Accessible label for the help button */
  ariaLabel?: string
}

/**
 * FormFieldHelp - Provides contextual help for form fields
 *
 * Features:
 * - Tooltip-based help text display
 * - Keyboard accessible (focusable, shows on focus)
 * - Can display as icon or inline text
 * - i18n ready content
 *
 * @example
 * ```tsx
 * <FormFieldHelp content="SKU must be unique and can contain letters, numbers, and dashes" />
 *
 * // With title and rich content
 * <FormFieldHelp
 *   title="Password Requirements"
 *   content={
 *     <ul>
 *       <li>At least 8 characters</li>
 *       <li>Uppercase and lowercase letters</li>
 *       <li>At least one number</li>
 *     </ul>
 *   }
 * />
 *
 * // Inline mode
 * <FormFieldHelp content="This field is optional" inline />
 * ```
 */
export function FormFieldHelp({
  content,
  title,
  position = 'top',
  size = 'default',
  className = '',
  inline = false,
  icon,
  ariaLabel,
}: FormFieldHelpProps) {
  const { t } = useTranslation('validation')
  const resolvedAriaLabel = ariaLabel || t('help.ariaLabel', { defaultValue: 'Help information' })
  const sizeClass = `form-field-help--${size}`

  if (inline) {
    return <span className={`form-field-help form-field-help--inline ${className}`}>{content}</span>
  }

  const tooltipContent = title ? (
    <div className="form-field-help__tooltip-content">
      <div className="form-field-help__tooltip-title">{title}</div>
      <div className="form-field-help__tooltip-body">{content}</div>
    </div>
  ) : (
    content
  )

  return (
    <Tooltip content={tooltipContent} position={position} trigger="hover">
      <button
        type="button"
        className={`form-field-help ${sizeClass} ${className}`}
        aria-label={resolvedAriaLabel}
        tabIndex={0}
      >
        {icon || <IconHelpCircle />}
      </button>
    </Tooltip>
  )
}

interface FormFieldHintProps {
  /** Hint text to display */
  children: ReactNode
  /** Type of hint (affects styling) */
  type?: 'default' | 'info' | 'warning'
  /** Additional class name */
  className?: string
}

/**
 * FormFieldHint - Displays persistent hint text below a form field
 *
 * Different from helperText in FormFieldWrapper, this is meant for
 * more prominent hints that should always be visible.
 *
 * @example
 * ```tsx
 * <FormFieldHint>Format: YYYY-MM-DD</FormFieldHint>
 * <FormFieldHint type="warning">This action cannot be undone</FormFieldHint>
 * ```
 */
export function FormFieldHint({ children, type = 'default', className = '' }: FormFieldHintProps) {
  return (
    <div className={`form-field-hint form-field-hint--${type} ${className}`} role="note">
      {children}
    </div>
  )
}

interface FormFieldExampleProps {
  /** Example value(s) to display */
  examples: string | string[]
  /** Prefix text before examples */
  prefix?: string
  /** Additional class name */
  className?: string
}

/**
 * FormFieldExample - Shows example values for a form field
 *
 * @example
 * ```tsx
 * <FormFieldExample examples="name@example.com" />
 * <FormFieldExample examples={['PROD-001', 'SKU-ABC-123']} />
 * ```
 */
export function FormFieldExample({ examples, prefix, className = '' }: FormFieldExampleProps) {
  const { t } = useTranslation('validation')
  const resolvedPrefix = prefix ?? t('help.example', { defaultValue: 'Example: ' })
  const exampleList = Array.isArray(examples) ? examples : [examples]

  return (
    <span className={`form-field-example ${className}`}>
      <span className="form-field-example__prefix">{resolvedPrefix}</span>
      {exampleList.map((example, index) => (
        <span key={example}>
          <code className="form-field-example__value">{example}</code>
          {index < exampleList.length - 1 && (
            <span className="form-field-example__separator">, </span>
          )}
        </span>
      ))}
    </span>
  )
}

export default FormFieldHelp
