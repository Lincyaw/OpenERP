import { Controller, type FieldValues, type FieldPath } from 'react-hook-form'
import { TreeSelect } from '@douyinfe/semi-ui-19'
import { FormFieldWrapper } from './FormFieldWrapper'
import type { ControlledFieldProps } from './types'

/**
 * Tree node data structure
 */
export interface TreeNode {
  label: string
  value: string
  key?: string
  disabled?: boolean
  children?: TreeNode[]
}

type TreeSelectFieldProps<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = ControlledFieldProps<TFieldValues, TName> & {
  /** Tree data */
  treeData: TreeNode[]
  /** Allow multiple selection */
  multiple?: boolean
  /** Allow clearing the selection */
  allowClear?: boolean
  /** Show search input */
  showSearch?: boolean
  /** Expand all nodes by default */
  expandAll?: boolean
  /** Empty state text */
  emptyContent?: string
}

/**
 * Controlled tree select field with validation support
 * Useful for hierarchical data like categories
 */
export function TreeSelectField<
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
  treeData,
  multiple,
  allowClear = true,
  showSearch = false,
  expandAll = false,
  emptyContent = '暂无数据',
}: TreeSelectFieldProps<TFieldValues, TName>) {
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
          <TreeSelect
            placeholder={placeholder}
            disabled={disabled}
            multiple={multiple}
            showClear={allowClear}
            filterTreeNode={showSearch}
            expandAll={expandAll}
            emptyContent={emptyContent}
            treeData={treeData}
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

export default TreeSelectField
