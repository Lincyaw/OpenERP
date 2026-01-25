import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { TextArea } from '@douyinfe/semi-ui'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps } from './types'
import type { ComponentProps } from 'react'

type TextAreaProps = ComponentProps<typeof TextArea>

type TextAreaFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> &
  Omit<TextAreaProps, 'name' | 'value' | 'onChange' | 'onBlur'>

/**
 * Controlled textarea field with validation support
 */
export function TextAreaField<
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
  rows = 4,
  maxCount,
  autosize,
  ...textAreaProps
}: TextAreaFieldProps<TFieldValues, TName>) {
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
          className={className}
          htmlFor={name}
        >
          <TextArea
            {...textAreaProps}
            id={name}
            name={name}
            placeholder={placeholder}
            disabled={disabled}
            rows={rows}
            maxCount={maxCount}
            autosize={autosize}
            validateStatus={fieldState.error ? 'error' : undefined}
            value={field.value ?? ''}
            onChange={(value) => field.onChange(value)}
            onBlur={field.onBlur}
          />
        </FormFieldWrapper>
      )}
    />
  )
}

export default TextAreaField
