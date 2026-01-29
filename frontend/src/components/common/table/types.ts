import type { ReactNode } from 'react'
import type { PaginationMeta, PaginationParams } from '@/types/api'

/**
 * Sort direction
 */
export type SortOrder = 'asc' | 'desc' | undefined

/**
 * Sort state
 */
export interface SortState {
  field: string | undefined
  order: SortOrder
}

/**
 * Filter value type
 */
export type FilterValue = string | number | boolean | string[] | number[] | null

/**
 * Filter state
 */
export interface FilterState {
  [key: string]: FilterValue
}

/**
 * Table state containing pagination, sorting, and filtering
 */
export interface TableState {
  pagination: {
    page: number
    pageSize: number
  }
  sort: SortState
  filters: FilterState
}

/**
 * Table state change handler
 */
export interface TableStateChange {
  pagination?: { page: number; pageSize: number }
  sort?: SortState
  filters?: FilterState
}

/**
 * Extended column props for DataTable
 */
export interface DataTableColumn<T = Record<string, unknown>> {
  /** Column key (required for sorting/filtering) */
  dataIndex: string
  /** Column title */
  title: string | ReactNode
  /** Enable sorting for this column */
  sortable?: boolean
  /** Enable filtering for this column */
  filterable?: boolean
  /** Custom filter options */
  filterOptions?: FilterOption[]
  /** Filter type */
  filterType?: 'select' | 'search' | 'dateRange' | 'numberRange'
  /** Custom render function */
  render?: (text: unknown, record: T, index: number) => ReactNode
  /** Column width */
  width?: number | string
  /** Fixed column position */
  fixed?: 'left' | 'right' | boolean
  /** Column alignment */
  align?: 'left' | 'center' | 'right'
  /** Whether column is hidden */
  hidden?: boolean
  /** Ellipsis overflow */
  ellipsis?: boolean | { showTitle?: boolean }
  /** Additional class name for the column */
  className?: string
}

/**
 * Filter option for select-type filters
 */
export interface FilterOption {
  label: string
  value: string | number
}

/**
 * Row selection configuration
 */
export interface RowSelection<T = unknown> {
  /** Selected row keys */
  selectedRowKeys: string[]
  /** Selection change handler */
  onChange: (selectedRowKeys: string[], selectedRows: T[]) => void
  /** Row selection type */
  type?: 'checkbox' | 'radio'
  /** Disable selection for specific rows */
  getCheckboxProps?: (record: T) => { disabled?: boolean; name?: string }
  /** Fixed selection column */
  fixed?: boolean
}

/**
 * Table action button configuration
 */
export interface TableAction<T = unknown> {
  /** Action key for identification */
  key: string
  /** Action label */
  label: string | ((record: T) => string)
  /** Action icon */
  icon?: ReactNode
  /** Action handler */
  onClick: (record: T, index: number) => void
  /** Whether action is disabled */
  disabled?: boolean | ((record: T) => boolean)
  /** Whether action is hidden */
  hidden?: boolean | ((record: T) => boolean)
  /** Action type for styling */
  type?: 'primary' | 'secondary' | 'tertiary' | 'warning' | 'danger'
  /** Show confirmation before action */
  confirm?: {
    title: string
    content?: string
    okText?: string
    cancelText?: string
  }
}

/**
 * Bulk action for selected rows
 */
export interface BulkAction<T = unknown> {
  /** Action key */
  key: string
  /** Action label */
  label: string
  /** Action icon */
  icon?: ReactNode
  /** Action handler */
  onClick: (selectedRows: T[], selectedRowKeys: string[]) => void
  /** Whether action is disabled */
  disabled?: boolean
  /** Action type */
  type?: 'primary' | 'secondary' | 'warning' | 'danger'
  /** Show confirmation */
  confirm?: {
    title: string
    content?: string
  }
}

/**
 * DataTable props
 */
export interface DataTableProps<T = unknown> {
  /** Table data */
  data: T[]
  /** Column definitions */
  columns: DataTableColumn<T>[]
  /** Row key field or getter function */
  rowKey: string | ((record: T) => string)
  /** Whether data is loading */
  loading?: boolean
  /** Pagination metadata from API */
  pagination?: PaginationMeta | false
  /** Row selection configuration */
  rowSelection?: RowSelection<T>
  /** Row actions */
  actions?: TableAction<T>[]
  /** Bulk actions for selected rows */
  bulkActions?: BulkAction<T>[]
  /** Table state change handler */
  onStateChange?: (change: TableStateChange) => void
  /** Current sort state (controlled) */
  sortState?: SortState
  /** Current filter state (controlled) */
  filterState?: FilterState
  /** Page size options */
  pageSizeOptions?: number[]
  /** Table size */
  size?: 'default' | 'small' | 'middle'
  /** Whether to show header */
  showHeader?: boolean
  /** Whether to show border */
  bordered?: boolean
  /** Empty state content */
  empty?: ReactNode
  /** Table title */
  title?: ReactNode | (() => ReactNode)
  /** Table footer */
  footer?: ReactNode | (() => ReactNode)
  /** Row click handler */
  onRow?: (
    record: T,
    index: number
  ) => {
    onClick?: (event: React.MouseEvent) => void
    onDoubleClick?: (event: React.MouseEvent) => void
  }
  /** Scroll configuration */
  scroll?: { x?: number | string; y?: number | string }
  /** Additional class name */
  className?: string
  /** Sticky header */
  sticky?: boolean
  /** Resizable columns */
  resizable?: boolean
  /** Expandable row configuration */
  expandable?: {
    expandedRowRender?: (record: T, index: number) => ReactNode
    rowExpandable?: (record: T) => boolean
    expandRowByClick?: boolean
  }
  /**
   * Columns to show as primary in mobile card view
   * Primary columns are displayed prominently at the top of each card.
   * If not specified, the first 2 columns are used as primary.
   */
  mobileCardPrimaryColumns?: string[]
  /**
   * Disable mobile card view and always show table
   * @default false
   */
  disableMobileCard?: boolean
}

/**
 * useTableState hook options
 */
export interface UseTableStateOptions {
  /** Default page size */
  defaultPageSize?: number
  /** Default sort field */
  defaultSortField?: string
  /** Default sort order */
  defaultSortOrder?: SortOrder
  /** Default filters */
  defaultFilters?: FilterState
  /** Sync with URL query params */
  syncWithUrl?: boolean
}

/**
 * useTableState hook return type
 */
export interface UseTableStateReturn {
  /** Current table state */
  state: TableState
  /** Set pagination */
  setPagination: (page: number, pageSize?: number) => void
  /** Set sort */
  setSort: (field: string | undefined, order?: SortOrder) => void
  /** Set filters */
  setFilters: (filters: FilterState) => void
  /** Set single filter */
  setFilter: (key: string, value: FilterValue) => void
  /** Clear filters */
  clearFilters: () => void
  /** Reset to initial state */
  reset: () => void
  /** Handle state change from DataTable */
  handleStateChange: (change: TableStateChange) => void
  /** Convert state to API params */
  toApiParams: () => PaginationParams & FilterState
}

/**
 * Export action configuration
 */
export interface ExportAction {
  /** Handler for CSV export */
  onExportCSV?: () => void
  /** Handler for Excel export */
  onExportExcel?: () => void
  /** Whether export is loading */
  loading?: boolean
  /** Disable export (e.g., when no data) */
  disabled?: boolean
}

/**
 * TableToolbar props
 */
export interface TableToolbarProps {
  /** Search placeholder */
  searchPlaceholder?: string
  /** Search value */
  searchValue?: string
  /** Search change handler */
  onSearchChange?: (value: string) => void
  /** Primary action button */
  primaryAction?: {
    label: string
    icon?: ReactNode
    onClick: () => void
  }
  /** Secondary actions */
  secondaryActions?: Array<{
    key: string
    label: string
    icon?: ReactNode
    onClick: () => void
  }>
  /** Export actions configuration */
  exportActions?: ExportAction
  /** Filter elements */
  filters?: ReactNode
  /** Additional class name */
  className?: string
  /** Show search */
  showSearch?: boolean
  /** Selected count for bulk actions */
  selectedCount?: number
  /** Bulk actions */
  bulkActions?: ReactNode
}
