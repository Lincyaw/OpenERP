import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { TagInput } from '@douyinfe/semi-ui-19'
import { FormFieldWrapper } from '@/components/common/form/FormFieldWrapper'
import type { ControlledFieldProps } from '@/components/common/form/types'

type TagInputFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> & {
  /** Maximum number of tags to show before collapsing */
  maxTagCount?: number
  /** Whether to show rest tags count in a popover */
  showRestTagsPopover?: boolean
  /** Separator for splitting input (default: comma and Enter) */
  separator?: string | string[]
}

/**
 * Controlled tag input field with validation support
 */
export function TagInputField<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
>({
  name,
  control,
  label,
  helperText,
  required,
  disabled,
  placeholder,
  className,
  labelPosition,
  hideLabel,
  rules,
  maxTagCount = 5,
  showRestTagsPopover = true,
  separator = [',', 'Enter'],
}: TagInputFieldProps<TFieldValues, TName>) {
  return (
    <Controller
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FormFieldWrapper
          label={label}
          required={required}
          error={fieldState.error?.message}
          helperText={helperText}
          labelPosition={labelPosition}
          hideLabel={hideLabel}
          className={`tag-input-field ${className || ''}`}
          htmlFor={name}
        >
          <TagInput
            id={name}
            placeholder={placeholder}
            disabled={disabled}
            maxTagCount={maxTagCount}
            showRestTagsPopover={showRestTagsPopover}
            separator={separator}
            value={field.value || []}
            onChange={(value) => field.onChange(value)}
            onBlur={field.onBlur}
            validateStatus={fieldState.error ? 'error' : undefined}
            style={{ width: '100%' }}
          />
        </FormFieldWrapper>
      )}
    />
  )
}

export default TagInputField
