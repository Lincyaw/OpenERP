import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { DatePicker } from '@douyinfe/semi-ui'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps } from './types'

type DateFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> & {
  /** Output format for the value (default: ISO string) */
  valueFormat?: 'date' | 'string' | 'timestamp'
  /** Display format */
  format?: string
  /** Picker type */
  type?: 'date' | 'dateTime' | 'dateRange' | 'dateTimeRange' | 'month' | 'year'
}

/**
 * Controlled date picker field with validation support
 */
export function DateField<
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
  valueFormat = 'string',
  format = 'yyyy-MM-dd',
  type = 'date',
}: DateFieldProps<TFieldValues, TName>) {
  const formatValue = (date: Date | null): string | number | Date | null => {
    if (!date) return null
    switch (valueFormat) {
      case 'timestamp':
        return date.getTime()
      case 'date':
        return date
      case 'string':
      default:
        return date.toISOString()
    }
  }

  const parseValue = (value: unknown): Date | undefined => {
    if (!value) return undefined
    if (value instanceof Date) return value
    if (typeof value === 'number') return new Date(value)
    if (typeof value === 'string') return new Date(value)
    return undefined
  }

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
          <DatePicker
            placeholder={placeholder}
            disabled={disabled}
            format={format}
            type={type}
            validateStatus={fieldState.error ? 'error' : undefined}
            value={parseValue(field.value)}
            onChange={(date) => {
              const value = date ? formatValue(date as Date) : null
              field.onChange(value)
            }}
            onBlur={field.onBlur}
            style={{ width: '100%' }}
          />
        </FormFieldWrapper>
      )}
    />
  )
}

export default DateField
