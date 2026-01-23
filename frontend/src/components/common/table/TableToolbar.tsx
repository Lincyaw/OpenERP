import type { ReactNode } from 'react'
import { Button, Input, Space } from '@douyinfe/semi-ui'
import { IconSearch, IconPlus } from '@douyinfe/semi-icons'
import type { TableToolbarProps } from './types'
import './TableToolbar.css'

/**
 * TableToolbar component for search, filters, and actions
 *
 * @example
 * ```tsx
 * <TableToolbar
 *   searchValue={keyword}
 *   onSearchChange={setKeyword}
 *   searchPlaceholder="搜索商品名称、SKU..."
 *   primaryAction={{
 *     label: '新增商品',
 *     icon: <IconPlus />,
 *     onClick: () => navigate('/catalog/products/new'),
 *   }}
 *   secondaryActions={[
 *     { key: 'export', label: '导出', onClick: handleExport },
 *     { key: 'import', label: '导入', onClick: handleImport },
 *   ]}
 *   filters={
 *     <Space>
 *       <Select placeholder="分类" options={categoryOptions} onChange={setCategory} />
 *       <Select placeholder="状态" options={statusOptions} onChange={setStatus} />
 *     </Space>
 *   }
 * />
 * ```
 */
export function TableToolbar({
  searchPlaceholder = '搜索...',
  searchValue = '',
  onSearchChange,
  primaryAction,
  secondaryActions = [],
  filters,
  className = '',
  showSearch = true,
  selectedCount = 0,
  bulkActions,
}: TableToolbarProps) {
  const showBulkMode = selectedCount > 0 && bulkActions

  return (
    <div className={`table-toolbar ${className}`}>
      {showBulkMode ? (
        <div className="table-toolbar-bulk">
          <span className="table-toolbar-bulk-info">已选择 {selectedCount} 项</span>
          <div className="table-toolbar-bulk-actions">{bulkActions}</div>
        </div>
      ) : (
        <>
          <div className="table-toolbar-left">
            {showSearch && onSearchChange && (
              <Input
                prefix={<IconSearch />}
                placeholder={searchPlaceholder}
                value={searchValue}
                onChange={onSearchChange}
                showClear
                className="table-toolbar-search"
              />
            )}
            {filters && <div className="table-toolbar-filters">{filters}</div>}
          </div>
          <div className="table-toolbar-right">
            <Space>
              {secondaryActions.map((action) => (
                <Button key={action.key} icon={action.icon} onClick={action.onClick}>
                  {action.label}
                </Button>
              ))}
              {primaryAction && (
                <Button
                  theme="solid"
                  icon={primaryAction.icon || <IconPlus />}
                  onClick={primaryAction.onClick}
                >
                  {primaryAction.label}
                </Button>
              )}
            </Space>
          </div>
        </>
      )}
    </div>
  )
}

interface BulkActionBarProps {
  /** Number of selected items */
  selectedCount: number
  /** Cancel selection handler */
  onCancel: () => void
  /** Action buttons */
  children: ReactNode
  /** Additional class name */
  className?: string
}

/**
 * BulkActionBar component for bulk operations on selected rows
 *
 * @example
 * ```tsx
 * <BulkActionBar
 *   selectedCount={selectedRowKeys.length}
 *   onCancel={() => setSelectedRowKeys([])}
 * >
 *   <Button onClick={handleBulkEnable}>批量启用</Button>
 *   <Button onClick={handleBulkDisable}>批量禁用</Button>
 *   <Button type="danger" onClick={handleBulkDelete}>批量删除</Button>
 * </BulkActionBar>
 * ```
 */
export function BulkActionBar({
  selectedCount,
  onCancel,
  children,
  className = '',
}: BulkActionBarProps) {
  if (selectedCount === 0) {
    return null
  }

  return (
    <div className={`bulk-action-bar ${className}`}>
      <div className="bulk-action-bar-info">
        <span className="bulk-action-bar-count">已选择 {selectedCount} 项</span>
        <Button size="small" theme="borderless" onClick={onCancel}>
          取消选择
        </Button>
      </div>
      <Space className="bulk-action-bar-actions">{children}</Space>
    </div>
  )
}
