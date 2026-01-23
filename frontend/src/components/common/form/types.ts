import type { ReactNode } from 'react'
import type {
  FieldPath,
  FieldValues,
  UseFormReturn,
  RegisterOptions,
  ControllerRenderProps,
  FieldError,
} from 'react-hook-form'

/**
 * Base props for all form field components
 */
export interface BaseFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> {
  /** Field name (must match form schema) */
  name: TName
  /** Field label */
  label?: string
  /** Helper text shown below the field */
  helperText?: string
  /** Whether the field is required */
  required?: boolean
  /** Whether the field is disabled */
  disabled?: boolean
  /** Placeholder text */
  placeholder?: string
  /** Additional class name */
  className?: string
  /** Label position */
  labelPosition?: 'top' | 'left' | 'inset'
  /** Hide label visually (still accessible) */
  hideLabel?: boolean
}

/**
 * Props for controlled form field components
 */
export interface ControlledFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> extends BaseFieldProps<TFieldValues, TName> {
  /** React Hook Form control */
  control: UseFormReturn<TFieldValues>['control']
  /** Validation rules */
  rules?: RegisterOptions<TFieldValues, TName>
}

/**
 * Props passed to the render function of controlled fields
 */
export interface FieldRenderProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> {
  field: ControllerRenderProps<TFieldValues, TName>
  error?: FieldError
  hasError: boolean
  errorMessage?: string
}

/**
 * Form field wrapper props
 */
export interface FormFieldWrapperProps {
  /** Field label */
  label?: string
  /** Whether the field is required */
  required?: boolean
  /** Error message to display */
  error?: string
  /** Helper text shown below the field */
  helperText?: string
  /** Label position */
  labelPosition?: 'top' | 'left' | 'inset'
  /** Hide label visually (still accessible) */
  hideLabel?: boolean
  /** Additional class name */
  className?: string
  /** Children elements */
  children: ReactNode
}

/**
 * Select option type
 */
export interface SelectOption<T = string> {
  label: string
  value: T
  disabled?: boolean
}

/**
 * Form submission state
 */
export interface FormSubmitState {
  isSubmitting: boolean
  isSubmitSuccessful: boolean
  submitCount: number
}

/**
 * Form configuration options
 */
export interface FormConfig {
  /** Show validation errors inline */
  showInlineErrors?: boolean
  /** Validate on blur */
  validateOnBlur?: boolean
  /** Validate on change */
  validateOnChange?: boolean
  /** Reset form on successful submit */
  resetOnSuccess?: boolean
}
