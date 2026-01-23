import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { InputNumber } from '@douyinfe/semi-ui'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps } from './types'
import type { ComponentProps } from 'react'

type InputNumberProps = ComponentProps<typeof InputNumber>

type NumberFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> &
  Omit<InputNumberProps, 'name' | 'value' | 'onChange' | 'onBlur'>

/**
 * Controlled number input field with validation support
 */
export function NumberField<
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
  min,
  max,
  step = 1,
  precision,
  ...inputProps
}: NumberFieldProps<TFieldValues, TName>) {
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
          <InputNumber
            {...inputProps}
            placeholder={placeholder}
            disabled={disabled}
            min={min}
            max={max}
            step={step}
            precision={precision}
            validateStatus={fieldState.error ? 'error' : undefined}
            value={field.value}
            onChange={(value) => field.onChange(value)}
            onBlur={field.onBlur}
            style={{ width: '100%' }}
          />
        </FormFieldWrapper>
      )}
    />
  )
}

export default NumberField
