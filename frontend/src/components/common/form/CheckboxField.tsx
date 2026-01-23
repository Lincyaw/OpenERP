import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { Checkbox, CheckboxGroup } from '@douyinfe/semi-ui'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps, SelectOption } from './types'

type CheckboxFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> & {
  /** Text to display next to the checkbox */
  text?: string
}

/**
 * Controlled single checkbox field with validation support
 */
export function CheckboxField<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
>({
  name,
  control,
  label,
  helperText,
  required,
  disabled,
  className,
  labelPosition,
  hideLabel,
  rules,
  text,
}: CheckboxFieldProps<TFieldValues, TName>) {
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
        >
          <Checkbox
            disabled={disabled}
            checked={field.value}
            onChange={(e) => field.onChange(e.target.checked)}
          >
            {text}
          </Checkbox>
        </FormFieldWrapper>
      )}
    />
  )
}

type CheckboxGroupFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> & {
  /** Checkbox options */
  options: SelectOption[]
  /** Layout direction */
  direction?: 'horizontal' | 'vertical'
}

/**
 * Controlled checkbox group field with validation support
 */
export function CheckboxGroupField<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
>({
  name,
  control,
  label,
  helperText,
  required,
  disabled,
  className,
  labelPosition,
  hideLabel,
  rules,
  options,
  direction = 'horizontal',
}: CheckboxGroupFieldProps<TFieldValues, TName>) {
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
        >
          <CheckboxGroup
            disabled={disabled}
            direction={direction}
            options={options}
            value={field.value ?? []}
            onChange={(values) => field.onChange(values)}
          />
        </FormFieldWrapper>
      )}
    />
  )
}
