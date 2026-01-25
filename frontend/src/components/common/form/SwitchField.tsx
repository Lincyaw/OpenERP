import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { Switch } from '@douyinfe/semi-ui'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps } from './types'

type SwitchFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> & {
  /** Text to display when checked */
  checkedText?: string
  /** Text to display when unchecked */
  uncheckedText?: string
  /** Switch size */
  size?: 'default' | 'small' | 'large'
}

/**
 * Controlled switch field with validation support
 */
export function SwitchField<
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
  checkedText,
  uncheckedText,
  size,
}: SwitchFieldProps<TFieldValues, TName>) {
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
          <Switch
            id={name}
            disabled={disabled}
            size={size}
            checkedText={checkedText}
            uncheckedText={uncheckedText}
            checked={field.value}
            onChange={(checked) => field.onChange(checked)}
          />
        </FormFieldWrapper>
      )}
    />
  )
}

export default SwitchField
