import { useMemo, useCallback } from 'react'
import { Table, Pagination, Empty } from '@douyinfe/semi-ui'
import { IconInbox } from '@douyinfe/semi-icons'
import type { ColumnProps, ChangeInfo } from '@douyinfe/semi-ui/lib/es/table'
import type { DataTableProps, SortState } from './types'
import { TableActions } from './TableActions'
import './DataTable.css'

const DEFAULT_PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

/**
 * DataTable component - A feature-rich table based on Semi Design Table
 *
 * Features:
 * - Server-side pagination, sorting, and filtering
 * - Row selection with checkbox/radio
 * - Row actions with dropdown for overflow
 * - Responsive design
 * - Loading state
 * - Empty state
 * - Sticky header
 * - Resizable columns
 *
 * @example
 * ```tsx
 * const { state, handleStateChange, toApiParams } = useTableState()
 *
 * const columns: DataTableColumn<Product>[] = [
 *   { title: '商品名称', dataIndex: 'name', sortable: true },
 *   { title: 'SKU', dataIndex: 'sku', width: 120 },
 *   { title: '价格', dataIndex: 'price', align: 'right', sortable: true },
 *   { title: '状态', dataIndex: 'status', render: (status) => <StatusTag status={status} /> },
 * ]
 *
 * const actions: TableAction<Product>[] = [
 *   { key: 'edit', label: '编辑', onClick: (r) => handleEdit(r) },
 *   { key: 'delete', label: '删除', type: 'danger', confirm: { title: '确认删除' }, onClick: handleDelete },
 * ]
 *
 * <DataTable
 *   data={products}
 *   columns={columns}
 *   rowKey="id"
 *   loading={isLoading}
 *   pagination={paginationMeta}
 *   actions={actions}
 *   onStateChange={handleStateChange}
 *   sortState={state.sort}
 * />
 * ```
 */
export function DataTable<T extends Record<string, unknown>>({
  data,
  columns,
  rowKey,
  loading = false,
  pagination = false,
  rowSelection,
  actions,
  onStateChange,
  sortState,
  pageSizeOptions = DEFAULT_PAGE_SIZE_OPTIONS,
  size = 'default',
  showHeader = true,
  bordered = false,
  empty,
  title,
  footer,
  onRow,
  scroll,
  className = '',
  sticky = false,
  resizable = false,
  expandable,
}: DataTableProps<T>) {
  // Convert our column definition to Semi Table column format
  const tableColumns = useMemo<ColumnProps<T>[]>(() => {
    const cols: ColumnProps<T>[] = columns
      .filter((col) => !col.hidden)
      .map((col) => {
        const semiCol: ColumnProps<T> = {
          title: col.title,
          dataIndex: col.dataIndex,
          width: col.width,
          fixed: col.fixed,
          align: col.align,
          ellipsis: col.ellipsis === true ? true : undefined,
          render: col.render as ColumnProps<T>['render'],
          className: col.className,
        }

        // Add sorting
        if (col.sortable) {
          semiCol.sorter = true
          if (sortState?.field === col.dataIndex) {
            semiCol.sortOrder =
              sortState.order === 'asc' ? 'ascend' : sortState.order === 'desc' ? 'descend' : false
          }
        }

        return semiCol
      })

    // Add actions column if actions are provided
    if (actions && actions.length > 0) {
      cols.push({
        title: '操作',
        dataIndex: '__actions__',
        width: calculateActionsWidth(actions.length),
        fixed: 'right',
        render: (_: unknown, record: T, index: number) => (
          <TableActions record={record} index={index} actions={actions} />
        ),
      })
    }

    return cols
  }, [columns, actions, sortState])

  // Handle sort change
  const handleSortChange = useCallback(
    (sortInfo: { dataIndex?: string; sortOrder?: 'ascend' | 'descend' | false }) => {
      if (!onStateChange) return

      let newSort: SortState = { field: undefined, order: undefined }

      if (sortInfo.sortOrder && sortInfo.dataIndex) {
        newSort = {
          field: sortInfo.dataIndex,
          order: sortInfo.sortOrder === 'ascend' ? 'asc' : 'desc',
        }
      }

      onStateChange({ sort: newSort })
    },
    [onStateChange]
  )

  // Handle table change (sort, filter)
  const handleChange = useCallback(
    (changeInfo: ChangeInfo<T>) => {
      // Semi Table returns sorter info in changeInfo
      if (changeInfo?.sorter) {
        const { dataIndex, sortOrder } = changeInfo.sorter as {
          dataIndex?: string
          sortOrder?: 'ascend' | 'descend' | false
        }
        handleSortChange({ dataIndex, sortOrder })
      }
    },
    [handleSortChange]
  )

  // Handle pagination change
  const handlePageChange = useCallback(
    (page: number, pageSize: number) => {
      if (!onStateChange) return
      onStateChange({ pagination: { page, pageSize } })
    },
    [onStateChange]
  )

  // Row selection config for Semi Table
  const rowSelectionConfig = useMemo(() => {
    if (!rowSelection) return undefined

    return {
      selectedRowKeys: rowSelection.selectedRowKeys as (string | number)[],
      onChange: (
        selectedRowKeys: (string | number)[] | undefined,
        selectedRows: T[] | undefined
      ) => {
        rowSelection.onChange((selectedRowKeys || []) as string[], (selectedRows || []) as T[])
      },
      getCheckboxProps: rowSelection.getCheckboxProps,
      fixed: rowSelection.fixed,
    }
  }, [rowSelection])

  // Empty state
  const emptyContent = useMemo(() => {
    if (empty) return empty
    return (
      <Empty
        image={<IconInbox size="extra-large" style={{ color: 'var(--semi-color-text-2)' }} />}
        title="暂无数据"
        description="当前列表为空"
      />
    )
  }, [empty])

  // Get row key
  const getRowKey = useCallback(
    (record: T | undefined): string => {
      if (!record) return ''
      if (typeof rowKey === 'function') {
        return rowKey(record)
      }
      return record[rowKey] as string
    },
    [rowKey]
  )

  // Row handler
  const rowHandler = useMemo(() => {
    if (!onRow) return undefined
    return (record: T | undefined, index: number | undefined) => {
      if (!record) return {}
      return onRow(record, index ?? 0)
    }
  }, [onRow])

  // Expandable handlers
  const expandedRowRenderHandler = useMemo(() => {
    if (!expandable?.expandedRowRender) return undefined
    return (record: T | undefined, index: number | undefined) => {
      if (!record) return null
      return expandable.expandedRowRender!(record, index ?? 0)
    }
  }, [expandable])

  const rowExpandableHandler = useMemo(() => {
    if (!expandable?.rowExpandable) return undefined
    return (record: T | undefined) => {
      if (!record) return false
      return expandable.rowExpandable!(record)
    }
  }, [expandable])

  return (
    <div className={`data-table-container ${className}`}>
      <Table<T>
        columns={tableColumns}
        dataSource={data}
        rowKey={getRowKey}
        loading={loading}
        rowSelection={rowSelectionConfig}
        showHeader={showHeader}
        bordered={bordered}
        size={size}
        empty={emptyContent}
        title={title}
        footer={footer}
        onRow={rowHandler}
        scroll={scroll}
        sticky={sticky}
        resizable={resizable}
        expandedRowRender={expandedRowRenderHandler}
        rowExpandable={rowExpandableHandler}
        expandRowByClick={expandable?.expandRowByClick}
        onChange={handleChange}
        pagination={false}
        className="data-table"
      />

      {pagination && (
        <div className="data-table-pagination">
          <div className="data-table-pagination-info">共 {pagination.total} 条记录</div>
          <Pagination
            total={pagination.total}
            currentPage={pagination.page}
            pageSize={pagination.page_size}
            pageSizeOpts={pageSizeOptions}
            showSizeChanger
            showQuickJumper
            onChange={handlePageChange}
            onPageSizeChange={(pageSize) => handlePageChange(1, pageSize)}
          />
        </div>
      )}
    </div>
  )
}

/**
 * Calculate actions column width based on number of actions
 */
function calculateActionsWidth(actionCount: number): number {
  // Each button is approximately 60px, plus spacing
  const buttonWidth = 60
  const spacing = 4
  const maxVisibleButtons = Math.min(actionCount, 3)

  if (actionCount <= 3) {
    return maxVisibleButtons * buttonWidth + (maxVisibleButtons - 1) * spacing + 16
  }

  // If more than 3 actions, show 2 buttons + more dropdown
  return 2 * buttonWidth + spacing + 32 + 16
}
