import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { Input } from '@douyinfe/semi-ui-19'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps } from './types'
import type { ComponentProps } from 'react'

type InputProps = ComponentProps<typeof Input>

type TextFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> &
  Omit<InputProps, 'name' | 'value' | 'onChange' | 'onBlur'>

/**
 * Controlled text input field with validation support
 */
export function TextField<
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
  ...inputProps
}: TextFieldProps<TFieldValues, TName>) {
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
          <Input
            {...inputProps}
            id={name}
            name={name}
            placeholder={placeholder}
            disabled={disabled}
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

export default TextField
