import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { Select } from '@douyinfe/semi-ui'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps, SelectOption } from './types'

type SelectFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> & {
  /** Select options */
  options: SelectOption[]
  /** Allow multiple selection */
  multiple?: boolean
  /** Allow clearing the selection */
  allowClear?: boolean
  /** Show search input */
  showSearch?: boolean
  /** Empty state text */
  emptyContent?: string
}

/**
 * Controlled select field with validation support
 */
export function SelectField<
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
  options,
  multiple,
  allowClear = true,
  showSearch = false,
  emptyContent = '暂无数据',
}: SelectFieldProps<TFieldValues, TName>) {
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
          <Select
            id={name}
            placeholder={placeholder}
            disabled={disabled}
            multiple={multiple}
            showClear={allowClear}
            filter={showSearch}
            emptyContent={emptyContent}
            optionList={options}
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

export default SelectField
