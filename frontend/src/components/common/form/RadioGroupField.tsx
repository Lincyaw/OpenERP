import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { Radio, RadioGroup } from '@douyinfe/semi-ui'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps, SelectOption } from './types'

type RadioGroupFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> & {
  /** Radio options */
  options: SelectOption[]
  /** Layout direction */
  direction?: 'horizontal' | 'vertical'
  /** Display type */
  type?: 'default' | 'button' | 'card' | 'pureCard'
  /** Button style (only when type='button') */
  buttonSize?: 'small' | 'middle' | 'large'
}

/**
 * Controlled radio group field with validation support
 */
export function RadioGroupField<
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
  type,
  buttonSize,
}: RadioGroupFieldProps<TFieldValues, TName>) {
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
          <RadioGroup
            id={name}
            name={name}
            disabled={disabled}
            direction={direction}
            type={type}
            buttonSize={buttonSize}
            value={field.value}
            onChange={(e) => field.onChange(e.target.value)}
          >
            {options.map((option) => (
              <Radio key={option.value} value={option.value} disabled={option.disabled}>
                {option.label}
              </Radio>
            ))}
          </RadioGroup>
        </FormFieldWrapper>
      )}
    />
  )
}

export default RadioGroupField
