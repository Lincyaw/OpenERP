import type { ReactNode } from 'react'
import { Button, Input, Space, Dropdown } from '@douyinfe/semi-ui-19'
import { IconSearch, IconPlus, IconDownload } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
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
 *     { key: 'refresh', label: '刷新', onClick: handleRefresh },
 *   ]}
 *   exportActions={{
 *     onExportCSV: handleExportCSV,
 *     onExportExcel: handleExportExcel,
 *   }}
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
  searchPlaceholder,
  searchValue = '',
  onSearchChange,
  primaryAction,
  secondaryActions = [],
  exportActions,
  filters,
  className = '',
  showSearch = true,
  selectedCount = 0,
  bulkActions,
}: TableToolbarProps) {
  const { t } = useTranslation()
  const showBulkMode = selectedCount > 0 && bulkActions
  const placeholder = searchPlaceholder || t('table.searchPlaceholder')

  // Export dropdown menu items
  const exportMenuItems = []
  if (exportActions?.onExportCSV) {
    exportMenuItems.push({
      node: 'item',
      key: 'csv',
      name: t('table.export.csv'),
      onClick: exportActions.onExportCSV,
    })
  }
  if (exportActions?.onExportExcel) {
    exportMenuItems.push({
      node: 'item',
      key: 'excel',
      name: t('table.export.excel'),
      onClick: exportActions.onExportExcel,
    })
  }

  const showExportButton = exportMenuItems.length > 0

  return (
    <div className={`table-toolbar ${className}`}>
      {showBulkMode ? (
        <div className="table-toolbar-bulk">
          <span className="table-toolbar-bulk-info">
            {t('table.selectedItems', { count: selectedCount })}
          </span>
          <div className="table-toolbar-bulk-actions">{bulkActions}</div>
        </div>
      ) : (
        <>
          <div className="table-toolbar-left">
            {showSearch && onSearchChange && (
              <Input
                prefix={<IconSearch />}
                placeholder={placeholder}
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
              {showExportButton && (
                <Dropdown trigger="click" position="bottomRight" menu={exportMenuItems}>
                  <Button
                    icon={<IconDownload />}
                    loading={exportActions?.loading}
                    disabled={exportActions?.disabled}
                  >
                    {t('actions.export')}
                  </Button>
                </Dropdown>
              )}
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
 *   <Button onClick={handleBulkConfirm}>批量确认</Button>
 *   <Button onClick={handleBulkExport}>批量导出</Button>
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
  const { t } = useTranslation()

  if (selectedCount === 0) {
    return null
  }

  return (
    <div className={`bulk-action-bar ${className}`}>
      <div className="bulk-action-bar-info">
        <span className="bulk-action-bar-count">
          {t('table.selectedItems', { count: selectedCount })}
        </span>
        <Button size="small" theme="borderless" onClick={onCancel}>
          {t('actions.cancelSelection')}
        </Button>
      </div>
      <Space className="bulk-action-bar-actions">{children}</Space>
    </div>
  )
}
