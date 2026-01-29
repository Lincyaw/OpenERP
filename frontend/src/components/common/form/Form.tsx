import React from 'react'
import type { ReactNode } from 'react'
import { useState, useId } from 'react'
import { Button, Space, Spin, Collapsible } from '@douyinfe/semi-ui-19'
import { IconChevronDown, IconChevronRight } from '@douyinfe/semi-icons'
import type { IconSize, IconProps } from '@douyinfe/semi-icons'
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

/**
 * Icon type for FormSection - accepts either a ReactNode or a Semi icon component
 */
export type FormSectionIcon = ReactNode | React.ComponentType<{ size?: IconSize } & IconProps>

export interface FormSectionProps {
  /** Section title */
  title?: string
  /** Section subtitle/description */
  subtitle?: string
  /**
   * @deprecated Use `subtitle` instead
   * Section description (alias for subtitle for backward compatibility)
   */
  description?: string
  /** Optional icon to display before title */
  icon?: FormSectionIcon
  /** Section content */
  children: ReactNode
  /** Additional class name */
  className?: string
  /** Whether the section is collapsible */
  collapsible?: boolean
  /** Whether the section is expanded by default (only when collapsible=true) */
  defaultExpanded?: boolean
  /**
   * @deprecated Use `defaultExpanded` instead (inverted logic)
   * Whether the section is collapsed by default (only when collapsible=true)
   */
  defaultCollapsed?: boolean
  /** Whether this section contains required fields */
  required?: boolean
  /** Test ID for testing */
  'data-testid'?: string
}

/**
 * Form section with title, subtitle, optional icon, and collapse support.
 *
 * Provides a card-style grouping for form fields with:
 * - Optional icon in the header
 * - Title and subtitle text
 * - Required indicator (asterisk)
 * - Collapsible content with smooth animation
 *
 * @example
 * ```tsx
 * // Basic section
 * <FormSection title="Basic Info" subtitle="Enter your basic information">
 *   <TextField name="name" label="Name" />
 * </FormSection>
 *
 * // With icon and required indicator
 * <FormSection
 *   title="Contact Details"
 *   icon={IconMail}
 *   required
 *   collapsible
 *   defaultExpanded
 * >
 *   <TextField name="email" label="Email" required />
 * </FormSection>
 * ```
 */
export function FormSection({
  title,
  subtitle,
  description,
  icon,
  children,
  className = '',
  collapsible = false,
  defaultExpanded = true,
  defaultCollapsed,
  required = false,
  'data-testid': testId,
}: FormSectionProps) {
  // Handle backward compatibility: defaultCollapsed has priority if explicitly provided
  const initialExpanded = defaultCollapsed !== undefined ? !defaultCollapsed : defaultExpanded
  const [isExpanded, setIsExpanded] = useState(initialExpanded)

  // Use subtitle if provided, fall back to description for backward compatibility
  const displaySubtitle = subtitle ?? description

  const sectionId = useId()
  const headerId = `${sectionId}-header`
  const contentId = `${sectionId}-content`

  const toggleExpanded = () => {
    if (collapsible) {
      setIsExpanded((prev) => !prev)
    }
  }

  const renderIcon = () => {
    if (!icon) return null

    // Check if icon is a valid React element (already instantiated)
    if (React.isValidElement(icon)) {
      return <span className="form-section-icon">{icon}</span>
    }

    // Check if icon is a component (function or ForwardRef object)
    // Semi Design icons are ForwardRef components with $$typeof property
    const isComponent =
      typeof icon === 'function' ||
      (typeof icon === 'object' && icon !== null && '$$typeof' in icon)

    if (isComponent) {
      const IconComponent = icon as React.ComponentType<{ size?: IconSize }>
      return (
        <span className="form-section-icon">
          <IconComponent size="default" />
        </span>
      )
    }

    // Fallback: render as ReactNode
    return <span className="form-section-icon">{icon}</span>
  }

  const renderHeader = () => {
    if (!title && !displaySubtitle && !icon) return null

    const headerContent = (
      <div className="form-section-header-content">
        <div className="form-section-title-row">
          {renderIcon()}
          {title && (
            <h3 className="form-section-title">
              {title}
              {required && <span className="form-section-required">*</span>}
            </h3>
          )}
        </div>
        {displaySubtitle && <p className="form-section-subtitle">{displaySubtitle}</p>}
      </div>
    )

    if (collapsible) {
      return (
        <div
          id={headerId}
          className="form-section-header form-section-header--collapsible"
          onClick={toggleExpanded}
          role="button"
          tabIndex={0}
          aria-expanded={isExpanded}
          aria-controls={contentId}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              toggleExpanded()
            }
          }}
        >
          {headerContent}
          <span className="form-section-chevron">
            {isExpanded ? <IconChevronDown /> : <IconChevronRight />}
          </span>
        </div>
      )
    }

    return (
      <div id={headerId} className="form-section-header">
        {headerContent}
      </div>
    )
  }

  const sectionClasses = [
    'form-section',
    collapsible && 'form-section--collapsible',
    !isExpanded && 'form-section--collapsed',
    className,
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <div
      className={sectionClasses}
      data-testid={testId}
      role="group"
      aria-labelledby={title ? headerId : undefined}
    >
      {renderHeader()}
      {collapsible ? (
        <Collapsible isOpen={isExpanded}>
          <div id={contentId} className="form-section-content" aria-hidden={!isExpanded}>
            {children}
          </div>
        </Collapsible>
      ) : (
        <div id={contentId} className="form-section-content">
          {children}
        </div>
      )}
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
