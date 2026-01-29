import type { ReactNode } from 'react'
import { Button, Dropdown, Pagination } from '@douyinfe/semi-ui-19'
import { IconMore } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import type { DataTableColumn, TableAction, RowSelection } from './types'
import type { PaginationMeta } from '@/types/api'
import './MobileCardList.css'

interface MobileCardListProps<T> {
  /** Table data */
  data: T[]
  /** Column definitions (used for labels and render functions) */
  columns: DataTableColumn<T>[]
  /** Row key field or getter function */
  rowKey: string | ((record: T) => string)
  /** Row actions */
  actions?: TableAction<T>[]
  /** Whether data is loading */
  loading?: boolean
  /** Pagination metadata */
  pagination?: PaginationMeta | false
  /** Page change handler */
  onPageChange?: (page: number, pageSize: number) => void
  /** Row selection configuration */
  rowSelection?: RowSelection<T>
  /** Columns to show as primary (shown prominently at top of card) */
  primaryColumns?: string[]
  /** Empty state content */
  empty?: ReactNode
  /** Additional class name */
  className?: string
}

/**
 * Mobile card list view for DataTable
 *
 * Renders data as cards instead of table rows on mobile devices.
 * Each card shows all column data with labels.
 */
export function MobileCardList<T extends Record<string, unknown>>({
  data,
  columns,
  rowKey,
  actions,
  loading,
  pagination,
  onPageChange,
  rowSelection,
  primaryColumns = [],
  empty,
  className = '',
}: MobileCardListProps<T>) {
  const { t } = useTranslation()

  // Get row key value
  const getRowKeyValue = (record: T): string => {
    if (typeof rowKey === 'function') {
      return rowKey(record)
    }
    return record[rowKey] as string
  }

  // Get visible columns (not hidden)
  const visibleColumns = columns.filter((col) => !col.hidden)

  // Separate primary and secondary columns
  const primaryCols =
    primaryColumns.length > 0
      ? visibleColumns.filter((col) => primaryColumns.includes(col.dataIndex))
      : visibleColumns.slice(0, 2) // Default: first 2 columns are primary

  const secondaryCols =
    primaryColumns.length > 0
      ? visibleColumns.filter((col) => !primaryColumns.includes(col.dataIndex))
      : visibleColumns.slice(2)

  // Render cell value
  const renderCellValue = (col: DataTableColumn<T>, record: T, index: number): ReactNode => {
    const value = record[col.dataIndex]
    if (col.render) {
      return col.render(value, record, index)
    }
    if (value === null || value === undefined) {
      return '-'
    }
    return String(value)
  }

  // Get visible actions for a record
  const getVisibleActions = (record: T): TableAction<T>[] => {
    if (!actions) return []
    return actions.filter((action) => {
      if (typeof action.hidden === 'function') {
        return !action.hidden(record)
      }
      return !action.hidden
    })
  }

  // Handle action click
  const handleActionClick = (action: TableAction<T>, record: T, index: number) => {
    if (action.disabled) {
      const isDisabled =
        typeof action.disabled === 'function' ? action.disabled(record) : action.disabled
      if (isDisabled) return
    }
    action.onClick(record, index)
  }

  // Render actions
  const renderActions = (record: T, index: number): ReactNode => {
    const visibleActions = getVisibleActions(record)
    if (visibleActions.length === 0) return null

    // If 3 or fewer actions, show as buttons
    if (visibleActions.length <= 3) {
      return (
        <div className="mobile-card-actions">
          {visibleActions.map((action) => {
            const label = typeof action.label === 'function' ? action.label(record) : action.label
            const isDisabled =
              typeof action.disabled === 'function' ? action.disabled(record) : action.disabled

            return (
              <Button
                key={action.key}
                size="small"
                type={action.type === 'danger' ? 'danger' : 'tertiary'}
                theme={action.type === 'primary' ? 'solid' : 'borderless'}
                disabled={isDisabled}
                onClick={() => handleActionClick(action, record, index)}
              >
                {label}
              </Button>
            )
          })}
        </div>
      )
    }

    // If more than 3 actions, show first 2 + dropdown
    const primaryActions = visibleActions.slice(0, 2)
    const moreActions = visibleActions.slice(2)

    return (
      <div className="mobile-card-actions">
        {primaryActions.map((action) => {
          const label = typeof action.label === 'function' ? action.label(record) : action.label
          const isDisabled =
            typeof action.disabled === 'function' ? action.disabled(record) : action.disabled

          return (
            <Button
              key={action.key}
              size="small"
              type={action.type === 'danger' ? 'danger' : 'tertiary'}
              theme={action.type === 'primary' ? 'solid' : 'borderless'}
              disabled={isDisabled}
              onClick={() => handleActionClick(action, record, index)}
            >
              {label}
            </Button>
          )
        })}
        <Dropdown
          trigger="click"
          position="bottomRight"
          render={
            <Dropdown.Menu>
              {moreActions.map((action) => {
                const label =
                  typeof action.label === 'function' ? action.label(record) : action.label
                const isDisabled =
                  typeof action.disabled === 'function' ? action.disabled(record) : action.disabled

                return (
                  <Dropdown.Item
                    key={action.key}
                    disabled={isDisabled}
                    type={action.type === 'danger' ? 'danger' : undefined}
                    onClick={() => handleActionClick(action, record, index)}
                  >
                    {label}
                  </Dropdown.Item>
                )
              })}
            </Dropdown.Menu>
          }
        >
          <span>
            <Button size="small" type="tertiary" icon={<IconMore />} />
          </span>
        </Dropdown>
      </div>
    )
  }

  // Check if row is selected
  const isRowSelected = (record: T): boolean => {
    if (!rowSelection) return false
    const key = getRowKeyValue(record)
    return rowSelection.selectedRowKeys.includes(key)
  }

  // Handle row selection toggle
  const handleSelectionToggle = (record: T) => {
    if (!rowSelection) return
    const key = getRowKeyValue(record)
    const isSelected = rowSelection.selectedRowKeys.includes(key)

    let newSelectedKeys: string[]
    let newSelectedRows: T[]

    if (isSelected) {
      newSelectedKeys = rowSelection.selectedRowKeys.filter((k) => k !== key)
      newSelectedRows = data.filter((r) => newSelectedKeys.includes(getRowKeyValue(r)))
    } else {
      newSelectedKeys = [...rowSelection.selectedRowKeys, key]
      newSelectedRows = data.filter((r) => newSelectedKeys.includes(getRowKeyValue(r)))
    }

    rowSelection.onChange(newSelectedKeys, newSelectedRows)
  }

  // Empty state
  if (!loading && data.length === 0) {
    return (
      <div className={`mobile-card-list-empty ${className}`}>
        {empty || (
          <div className="mobile-card-list-empty-content">
            <p>{t('table.noData')}</p>
            <p className="mobile-card-list-empty-description">{t('table.noDataDescription')}</p>
          </div>
        )}
      </div>
    )
  }

  return (
    <div className={`mobile-card-list ${className}`}>
      <div className="mobile-card-list-content">
        {data.map((record, index) => {
          const key = getRowKeyValue(record)
          const selected = isRowSelected(record)

          return (
            <div
              key={key}
              className={`mobile-card ${selected ? 'mobile-card-selected' : ''}`}
              onClick={() => rowSelection && handleSelectionToggle(record)}
            >
              {/* Primary info (top of card) */}
              <div className="mobile-card-primary">
                {primaryCols.map((col) => (
                  <div key={col.dataIndex} className="mobile-card-primary-item">
                    <span className="mobile-card-primary-value">
                      {renderCellValue(col, record, index)}
                    </span>
                  </div>
                ))}
              </div>

              {/* Secondary info (details grid) */}
              {secondaryCols.length > 0 && (
                <div className="mobile-card-secondary">
                  {secondaryCols.map((col) => (
                    <div key={col.dataIndex} className="mobile-card-field">
                      <span className="mobile-card-label">{col.title as string}</span>
                      <span className="mobile-card-value">
                        {renderCellValue(col, record, index)}
                      </span>
                    </div>
                  ))}
                </div>
              )}

              {/* Actions */}
              {renderActions(record, index)}
            </div>
          )
        })}
      </div>

      {/* Pagination */}
      {pagination && (
        <div className="mobile-card-list-pagination">
          <div className="mobile-card-list-pagination-info">
            {t('table.totalRecords', { total: pagination.total })}
          </div>
          <Pagination
            total={pagination.total}
            currentPage={pagination.page}
            pageSize={pagination.page_size}
            showSizeChanger={false}
            showQuickJumper={false}
            onChange={(page) => onPageChange?.(page, pagination.page_size)}
          />
        </div>
      )}
    </div>
  )
}
