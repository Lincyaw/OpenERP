import { useMemo, useCallback, useState } from 'react'
import { Button, Dropdown, Popconfirm, Space, Modal } from '@douyinfe/semi-ui-19'
import { IconMore } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import type { TableAction } from './types'
import './TableActions.css'

interface TableActionsProps<T = unknown> {
  /** Row record */
  record: T
  /** Row index */
  index: number
  /** Action definitions */
  actions: TableAction<T>[]
  /** Max visible actions (rest go to dropdown) */
  maxVisible?: number
  /** Size of action buttons */
  size?: 'small' | 'default'
}

/**
 * TableActions component for rendering row operation buttons
 *
 * @example
 * ```tsx
 * const actions: TableAction<Product>[] = [
 *   {
 *     key: 'edit',
 *     label: '编辑',
 *     icon: <IconEdit />,
 *     onClick: (record) => handleEdit(record),
 *   },
 *   {
 *     key: 'delete',
 *     label: '删除',
 *     type: 'danger',
 *     onClick: (record) => handleDelete(record),
 *     confirm: {
 *       title: '确认删除',
 *       content: '删除后无法恢复，确定要删除吗？',
 *     },
 *   },
 * ]
 *
 * <TableActions record={record} index={index} actions={actions} />
 * ```
 */
export function TableActions<T>({
  record,
  index,
  actions,
  maxVisible = 3,
  size = 'small',
}: TableActionsProps<T>) {
  const { t } = useTranslation()
  const [confirmAction, setConfirmAction] = useState<TableAction<T> | null>(null)

  // Filter out hidden actions
  const visibleActions = useMemo(() => {
    return actions.filter((action) => {
      if (typeof action.hidden === 'function') {
        return !action.hidden(record)
      }
      return !action.hidden
    })
  }, [actions, record])

  // Split actions into visible and dropdown
  const { directActions, dropdownActions } = useMemo(() => {
    if (visibleActions.length <= maxVisible) {
      return { directActions: visibleActions, dropdownActions: [] }
    }
    return {
      directActions: visibleActions.slice(0, maxVisible - 1),
      dropdownActions: visibleActions.slice(maxVisible - 1),
    }
  }, [visibleActions, maxVisible])

  const getButtonTheme = (type?: string) => {
    switch (type) {
      case 'primary':
        return 'solid'
      case 'danger':
      case 'warning':
        return 'light'
      default:
        return 'borderless'
    }
  }

  const getButtonType = (
    type?: string
  ): 'primary' | 'secondary' | 'tertiary' | 'warning' | 'danger' => {
    switch (type) {
      case 'primary':
        return 'primary'
      case 'danger':
        return 'danger'
      case 'warning':
        return 'warning'
      case 'tertiary':
        return 'tertiary'
      default:
        return 'tertiary'
    }
  }

  const handleClick = useCallback(
    (action: TableAction<T>) => {
      action.onClick(record, index)
    },
    [record, index]
  )

  const handleDropdownActionClick = useCallback(
    (action: TableAction<T>) => {
      if (action.confirm) {
        setConfirmAction(action)
      } else {
        handleClick(action)
      }
    },
    [handleClick]
  )

  const renderActionButton = useCallback(
    (action: TableAction<T>) => {
      const isDisabled =
        typeof action.disabled === 'function' ? action.disabled(record) : action.disabled

      const label = typeof action.label === 'function' ? action.label(record) : action.label

      const button = (
        <Button
          key={action.key}
          size={size}
          theme={getButtonTheme(action.type)}
          type={getButtonType(action.type)}
          icon={action.icon}
          disabled={isDisabled}
          onClick={action.confirm ? undefined : () => handleClick(action)}
          className="table-action-button"
        >
          {label}
        </Button>
      )

      if (action.confirm) {
        return (
          <Popconfirm
            key={action.key}
            title={action.confirm.title}
            content={action.confirm.content}
            okText={action.confirm.okText || t('actions.confirm')}
            cancelText={action.confirm.cancelText || t('actions.cancel')}
            onConfirm={() => handleClick(action)}
          >
            <span style={{ display: 'inline-flex' }}>{button}</span>
          </Popconfirm>
        )
      }

      return button
    },
    [record, size, handleClick, t]
  )

  if (visibleActions.length === 0) {
    return null
  }

  const dropdownMenu = dropdownActions.map((action) => {
    const isDisabled =
      typeof action.disabled === 'function' ? action.disabled(record) : action.disabled
    const label = typeof action.label === 'function' ? action.label(record) : action.label

    return {
      node: 'item' as const,
      key: action.key,
      name: label,
      icon: action.icon,
      disabled: isDisabled,
      type: action.type === 'danger' ? ('danger' as const) : undefined,
      onClick: () => handleDropdownActionClick(action),
    }
  })

  return (
    <>
      <Space className="table-actions" spacing={4}>
        {directActions.map(renderActionButton)}
        {dropdownActions.length > 0 && (
          <Dropdown
            trigger="click"
            position="bottomRight"
            clickToHide
            menu={dropdownMenu}
            className="table-actions-dropdown"
            getPopupContainer={() => document.body}
          >
            <span style={{ display: 'inline-flex' }}>
              <Button
                size={size}
                theme="borderless"
                icon={<IconMore />}
                data-testid="table-row-more-actions"
                aria-label={t('actions.moreActions')}
              />
            </span>
          </Dropdown>
        )}
      </Space>

      {/* Confirmation modal for dropdown actions */}
      {confirmAction?.confirm && (
        <Modal
          title={confirmAction.confirm.title}
          visible={!!confirmAction}
          onOk={() => {
            handleClick(confirmAction)
            setConfirmAction(null)
          }}
          onCancel={() => setConfirmAction(null)}
          okText={confirmAction.confirm.okText || t('actions.confirm')}
          cancelText={confirmAction.confirm.cancelText || t('actions.cancel')}
        >
          {confirmAction.confirm.content}
        </Modal>
      )}
    </>
  )
}

/**
 * Helper to create an actions column for DataTable
 */
export function createActionsColumn<T>(
  actions: TableAction<T>[],
  options?: {
    title?: string
    width?: number
    fixed?: 'left' | 'right'
    maxVisible?: number
  }
) {
  const { title = '操作', width = 160, fixed = 'right', maxVisible = 3 } = options || {}

  return {
    title,
    dataIndex: '__actions__',
    width,
    fixed,
    render: (_: unknown, record: T, index: number) => (
      <TableActions record={record} index={index} actions={actions} maxVisible={maxVisible} />
    ),
  }
}
