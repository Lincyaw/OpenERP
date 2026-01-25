import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Modal,
  Spin,
  Form,
  Button,
  Tree,
  Empty,
  Descriptions,
  Banner,
} from '@douyinfe/semi-ui'
import type { FormApi } from '@douyinfe/semi-ui/lib/es/form/interface'
import type { TreeNodeData } from '@douyinfe/semi-ui/lib/es/tree/interface'
import { IconPlus, IconRefresh, IconLock, IconSetting } from '@douyinfe/semi-icons'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getIdentity } from '@/api/identity'
import type { Role, RoleListQuery, CreateRoleRequest, UpdateRoleRequest } from '@/api/identity'
import './Roles.css'

const { Title, Text } = Typography

// Role type with index signature for DataTable compatibility
type RoleRow = Role & Record<string, unknown>

/**
 * Format date for display
 */
function formatDate(dateStr: string | undefined, locale: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Group permissions by resource for tree display
 */
function groupPermissionsByResource(permissions: string[]): Map<string, string[]> {
  const grouped = new Map<string, string[]>()

  for (const permission of permissions) {
    const parts = permission.split(':')
    if (parts.length === 2) {
      const [resource] = parts
      if (!grouped.has(resource)) {
        grouped.set(resource, [])
      }
      grouped.get(resource)!.push(permission)
    }
  }

  return grouped
}

/**
 * Roles management page
 *
 * Features:
 * - Role listing with pagination
 * - Search by code or name
 * - Filter by status and type
 * - Create/edit roles
 * - Enable/disable role actions
 * - Permission configuration
 */
export default function RolesPage() {
  const { t, i18n } = useTranslation('system')
  const api = useMemo(() => getIdentity(), [])

  // Status options for filter (with i18n)
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('roles.allStatus'), value: '' },
      { label: t('roles.enabled'), value: 'true' },
      { label: t('roles.disabled'), value: 'false' },
    ],
    [t]
  )

  // Type options for filter (with i18n)
  const TYPE_OPTIONS = useMemo(
    () => [
      { label: t('roles.allTypes'), value: '' },
      { label: t('roles.systemRole'), value: 'true' },
      { label: t('roles.customRole'), value: 'false' },
    ],
    [t]
  )

  /**
   * Get readable permission label
   */
  const getPermissionLabel = useCallback(
    (permission: string): string => {
      const parts = permission.split(':')
      if (parts.length !== 2) return permission

      const [resource, action] = parts
      const resourceLabel = String(t(`permissions.resources.${resource}`)) || resource
      const actionLabel = String(t(`permissions.actions.${action}`)) || action

      return i18n.language === 'zh-CN'
        ? `${actionLabel}${resourceLabel}`
        : `${actionLabel} ${resourceLabel}`
    },
    [t, i18n.language]
  )

  /**
   * Convert permissions to tree data
   */
  const permissionsToTreeData = useCallback(
    (permissions: string[]): TreeNodeData[] => {
      const grouped = groupPermissionsByResource(permissions)
      const treeData: TreeNodeData[] = []

      grouped.forEach((perms, resource) => {
        const resourceLabel = String(t(`permissions.resources.${resource}`)) || resource
        treeData.push({
          key: resource,
          label: resourceLabel,
          value: resource,
          children: perms.map((perm) => {
            const action = perm.split(':')[1]
            const actionLabel = String(t(`permissions.actions.${action}`)) || action
            return {
              key: perm,
              label: actionLabel,
              value: perm,
            }
          }),
        })
      })

      // Sort by resource name
      treeData.sort((a, b) => {
        const aLabel = a.label as string
        const bLabel = b.label as string
        return aLabel.localeCompare(bLabel, i18n.language)
      })

      return treeData
    },
    [t, i18n.language]
  )

  // State for data
  const [roleList, setRoleList] = useState<RoleRow[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [typeFilter, setTypeFilter] = useState<string>('')

  // Available permissions
  const [allPermissions, setAllPermissions] = useState<string[]>([])
  const [permissionsLoading, setPermissionsLoading] = useState(false)

  // Modal state
  const [modalVisible, setModalVisible] = useState(false)
  const [modalMode, setModalMode] = useState<'create' | 'edit'>('create')
  const [editingRole, setEditingRole] = useState<Role | null>(null)
  const [modalLoading, setModalLoading] = useState(false)

  // Permission config modal state
  const [permissionModalVisible, setPermissionModalVisible] = useState(false)
  const [permissionRole, setPermissionRole] = useState<Role | null>(null)
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([])
  const [permissionSaving, setPermissionSaving] = useState(false)

  // Role detail modal state
  const [detailModalVisible, setDetailModalVisible] = useState(false)
  const [detailRole, setDetailRole] = useState<Role | null>(null)

  // Form ref
  const formApiRef = useRef<FormApi | null>(null)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'sort_order',
    defaultSortOrder: 'asc',
  })

  // Fetch all available permissions
  const fetchPermissions = useCallback(async () => {
    setPermissionsLoading(true)
    try {
      const response = await api.getAllPermissions()
      if (response.success && response.data) {
        setAllPermissions(response.data.permissions || [])
      }
    } catch {
      // Silent fail
    } finally {
      setPermissionsLoading(false)
    }
  }, [api])

  // Fetch roles
  const fetchRoles = useCallback(async () => {
    setLoading(true)
    try {
      const params: RoleListQuery = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        keyword: searchKeyword || undefined,
        is_enabled: statusFilter ? statusFilter === 'true' : undefined,
        is_system_role: typeFilter ? typeFilter === 'true' : undefined,
      }

      const response = await api.listRoles(params)

      if (response.success && response.data) {
        setRoleList(response.data.roles as RoleRow[])
        setTotal(response.data.total)
      }
    } catch {
      Toast.error(t('roles.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    api,
    state.pagination.page,
    state.pagination.pageSize,
    searchKeyword,
    statusFilter,
    typeFilter,
  ])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchPermissions()
  }, [fetchPermissions])

  useEffect(() => {
    fetchRoles()
  }, [fetchRoles])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle status filter change
  const handleStatusChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const statusValue = typeof value === 'string' ? value : ''
      setStatusFilter(statusValue)
      setFilter('is_enabled', statusValue || null)
    },
    [setFilter]
  )

  // Handle type filter change
  const handleTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const typeValue = typeof value === 'string' ? value : ''
      setTypeFilter(typeValue)
      setFilter('is_system_role', typeValue || null)
    },
    [setFilter]
  )

  // Handle create role
  const handleCreate = useCallback(() => {
    setModalMode('create')
    setEditingRole(null)
    setModalVisible(true)
  }, [])

  // Handle edit role
  const handleEdit = useCallback(
    (role: RoleRow) => {
      if (role.is_system_role) {
        Toast.warning(t('roles.messages.cannotEditSystem'))
        return
      }
      setModalMode('edit')
      setEditingRole(role)
      setModalVisible(true)
    },
    [t]
  )

  // Handle view role details
  const handleViewDetail = useCallback((role: RoleRow) => {
    setDetailRole(role)
    setDetailModalVisible(true)
  }, [])

  // Handle modal submit
  const handleModalSubmit = useCallback(async () => {
    if (!formApiRef.current) return

    try {
      await formApiRef.current.validate()
      const values = formApiRef.current.getValues()
      setModalLoading(true)

      if (modalMode === 'create') {
        const request: CreateRoleRequest = {
          code: values.code,
          name: values.name,
          description: values.description || undefined,
          sort_order: values.sort_order ? Number(values.sort_order) : 0,
        }
        const response = await api.createRole(request)
        if (response.success) {
          Toast.success(t('roles.messages.createSuccess'))
          setModalVisible(false)
          fetchRoles()
        } else {
          Toast.error(response.error?.message || t('roles.messages.createError'))
        }
      } else if (editingRole) {
        const request: UpdateRoleRequest = {
          name: values.name,
          description: values.description || undefined,
          sort_order: values.sort_order ? Number(values.sort_order) : undefined,
        }
        const response = await api.updateRole(editingRole.id, request)
        if (response.success) {
          Toast.success(t('roles.messages.updateSuccess'))
          setModalVisible(false)
          fetchRoles()
        } else {
          Toast.error(response.error?.message || t('roles.messages.updateError'))
        }
      }
    } catch {
      // Validation failed or API error
    } finally {
      setModalLoading(false)
    }
  }, [modalMode, editingRole, api, fetchRoles, t])

  // Handle enable role
  const handleEnable = useCallback(
    async (role: RoleRow) => {
      try {
        const response = await api.enableRole(role.id)
        if (response.success) {
          Toast.success(t('roles.messages.enableSuccess', { name: role.name }))
          fetchRoles()
        } else {
          Toast.error(response.error?.message || t('roles.messages.enableError'))
        }
      } catch {
        Toast.error(t('roles.messages.enableError'))
      }
    },
    [api, fetchRoles, t]
  )

  // Handle disable role
  const handleDisable = useCallback(
    async (role: RoleRow) => {
      if (role.is_system_role) {
        Toast.warning(t('roles.messages.cannotDisableSystem'))
        return
      }
      Modal.confirm({
        title: t('roles.confirm.disableTitle'),
        content: t('roles.confirm.disableContent', { name: role.name }),
        okText: t('roles.confirm.disableOk'),
        cancelText: t('common.cancel'),
        okButtonProps: { type: 'warning' },
        onOk: async () => {
          try {
            const response = await api.disableRole(role.id)
            if (response.success) {
              Toast.success(t('roles.messages.disableSuccess', { name: role.name }))
              fetchRoles()
            } else {
              Toast.error(response.error?.message || t('roles.messages.disableError'))
            }
          } catch {
            Toast.error(t('roles.messages.disableError'))
          }
        },
      })
    },
    [api, fetchRoles, t]
  )

  // Handle delete role
  const handleDelete = useCallback(
    async (role: RoleRow) => {
      if (role.is_system_role) {
        Toast.warning(t('roles.messages.cannotDeleteSystem'))
        return
      }
      if (role.user_count && role.user_count > 0) {
        Toast.warning(t('roles.messages.cannotDeleteWithUsers', { count: role.user_count }))
        return
      }
      Modal.confirm({
        title: t('roles.confirm.deleteTitle'),
        content: t('roles.confirm.deleteContent', { name: role.name }),
        okText: t('roles.confirm.deleteOk'),
        cancelText: t('common.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            const response = await api.deleteRole(role.id)
            if (response.success) {
              Toast.success(t('roles.messages.deleteSuccess', { name: role.name }))
              fetchRoles()
            } else {
              Toast.error(response.error?.message || t('roles.messages.deleteError'))
            }
          } catch {
            Toast.error(t('roles.messages.deleteError'))
          }
        },
      })
    },
    [api, fetchRoles, t]
  )

  // Handle configure permissions
  const handleConfigurePermissions = useCallback((role: RoleRow) => {
    setPermissionRole(role)
    setSelectedPermissions(role.permissions || [])
    setPermissionModalVisible(true)
  }, [])

  // Handle save permissions
  const handleSavePermissions = useCallback(async () => {
    if (!permissionRole) return

    setPermissionSaving(true)
    try {
      const response = await api.setRolePermissions(permissionRole.id, {
        permissions: selectedPermissions,
      })
      if (response.success) {
        Toast.success(t('roles.messages.savePermissionsSuccess'))
        setPermissionModalVisible(false)
        fetchRoles()
      } else {
        Toast.error(response.error?.message || t('roles.messages.savePermissionsError'))
      }
    } catch {
      Toast.error(t('roles.messages.savePermissionsError'))
    } finally {
      setPermissionSaving(false)
    }
  }, [api, permissionRole, selectedPermissions, fetchRoles, t])

  // Handle permission tree selection
  const handlePermissionChange = useCallback(
    (value?: string | number | TreeNodeData | (string | number | TreeNodeData)[]) => {
      // Normalize to array of strings (permission codes only, not resource groups)
      let permissions: string[] = []

      if (value === undefined) {
        setSelectedPermissions([])
        return
      }

      if (Array.isArray(value)) {
        permissions = value
          .map((v) => (typeof v === 'object' && v !== null ? v.value : v) as string)
          .filter((v) => v && v.includes(':')) // Only include permission codes (contain ':')
      } else if (typeof value === 'string' && value.includes(':')) {
        permissions = [value]
      }

      setSelectedPermissions(permissions)
    },
    []
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchRoles()
  }, [fetchRoles])

  // Permission tree data
  const permissionTreeData = useMemo(() => {
    return permissionsToTreeData(allPermissions)
  }, [allPermissions])

  // Table columns
  const tableColumns: DataTableColumn<RoleRow>[] = useMemo(
    () => [
      {
        title: t('roles.columns.code'),
        dataIndex: 'code',
        width: 140,
        render: (code: unknown) => <span className="role-code">{(code as string) || '-'}</span>,
      },
      {
        title: t('roles.columns.name'),
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: RoleRow) => (
          <div className="role-name-cell">
            <span className="role-name">{name as string}</span>
            {record.is_system_role && (
              <Tag size="small" color="blue" className="role-system-tag">
                {t('roles.systemTag')}
              </Tag>
            )}
          </div>
        ),
      },
      {
        title: t('roles.columns.description'),
        dataIndex: 'description',
        width: 200,
        ellipsis: true,
        render: (desc: unknown) => <Text type="tertiary">{(desc as string) || '-'}</Text>,
      },
      {
        title: t('roles.columns.permissionCount'),
        dataIndex: 'permissions',
        width: 100,
        align: 'center',
        render: (permissions: unknown) => {
          const perms = permissions as string[] | undefined
          const count = perms?.length || 0
          return (
            <Tag color={count > 0 ? 'cyan' : 'grey'}>{t('roles.permissionCount', { count })}</Tag>
          )
        },
      },
      {
        title: t('roles.columns.userCount'),
        dataIndex: 'user_count',
        width: 90,
        align: 'center',
        render: (count: unknown) => <Text type="secondary">{(count as number) || 0}</Text>,
      },
      {
        title: t('roles.columns.status'),
        dataIndex: 'is_enabled',
        width: 90,
        align: 'center',
        render: (isEnabled: unknown) => {
          const enabled = isEnabled as boolean
          return (
            <Tag color={enabled ? 'green' : 'grey'}>
              {enabled ? t('roles.enabled') : t('roles.disabled')}
            </Tag>
          )
        },
      },
      {
        title: t('roles.columns.sortOrder'),
        dataIndex: 'sort_order',
        width: 80,
        align: 'center',
        sortable: true,
        render: (order: unknown) => (order as number) || 0,
      },
      {
        title: t('roles.columns.updatedAt'),
        dataIndex: 'updated_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, i18n.language),
      },
    ],
    [t, i18n.language]
  )

  // Table row actions
  const tableActions: TableAction<RoleRow>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('roles.actions.view'),
        onClick: handleViewDetail,
      },
      {
        key: 'edit',
        label: t('roles.actions.edit'),
        onClick: handleEdit,
        hidden: (record) => record.is_system_role,
      },
      {
        key: 'permissions',
        label: t('roles.actions.configurePermissions'),
        icon: <IconSetting size="small" />,
        onClick: handleConfigurePermissions,
      },
      {
        key: 'enable',
        label: t('roles.actions.enable'),
        type: 'primary',
        onClick: handleEnable,
        hidden: (record) => record.is_enabled,
      },
      {
        key: 'disable',
        label: t('roles.actions.disable'),
        type: 'warning',
        icon: <IconLock size="small" />,
        onClick: handleDisable,
        hidden: (record) => !record.is_enabled || record.is_system_role,
      },
      {
        key: 'delete',
        label: t('roles.actions.delete'),
        type: 'danger',
        onClick: handleDelete,
        hidden: (record) => record.is_system_role,
      },
    ],
    [
      t,
      handleViewDetail,
      handleEdit,
      handleConfigurePermissions,
      handleEnable,
      handleDisable,
      handleDelete,
    ]
  )

  return (
    <Container size="full" className="roles-page">
      <Card className="roles-card">
        <div className="roles-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('roles.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('roles.searchPlaceholder')}
          primaryAction={{
            label: t('roles.addRole'),
            icon: <IconPlus />,
            onClick: handleCreate,
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('common.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="roles-filter-container">
              <Select
                placeholder={t('roles.statusFilter')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('roles.typeFilter')}
                value={typeFilter}
                onChange={handleTypeChange}
                optionList={TYPE_OPTIONS}
                style={{ width: 120 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<RoleRow>
            data={roleList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={{
              page: state.pagination.page,
              page_size: state.pagination.pageSize,
              total,
              total_pages: Math.ceil(total / state.pagination.pageSize),
            }}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1100 }}
          />
        </Spin>
      </Card>

      {/* Create/Edit Role Modal */}
      <Modal
        title={modalMode === 'create' ? t('roles.addRole') : t('roles.editRole')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleModalSubmit}
        confirmLoading={modalLoading}
        width={500}
        okText={modalMode === 'create' ? t('common.create') : t('common.save')}
        cancelText={t('common.cancel')}
      >
        <Form
          getFormApi={(api) => {
            formApiRef.current = api
          }}
          initValues={
            editingRole
              ? {
                  code: editingRole.code,
                  name: editingRole.name,
                  description: editingRole.description,
                  sort_order: editingRole.sort_order,
                }
              : { sort_order: 0 }
          }
          labelPosition="left"
          labelWidth={80}
        >
          <Form.Input
            field="code"
            label={t('roles.form.code')}
            placeholder={t('roles.form.codePlaceholder')}
            rules={[
              { required: true, message: t('roles.form.code') },
              { min: 2, message: t('roles.form.codeMinError') },
              { max: 50, message: t('roles.form.codeMaxError') },
              {
                pattern: /^[a-z][a-z0-9_]*$/,
                message: t('roles.form.codeRegexError'),
              },
            ]}
            disabled={modalMode === 'edit'}
          />
          <Form.Input
            field="name"
            label={t('roles.form.name')}
            placeholder={t('roles.form.namePlaceholder')}
            rules={[
              { required: true, message: t('roles.form.name') },
              { min: 2, message: t('roles.form.nameMinError') },
              { max: 100, message: t('roles.form.nameMaxError') },
            ]}
          />
          <Form.TextArea
            field="description"
            label={t('roles.form.description')}
            placeholder={t('roles.form.descriptionPlaceholder')}
            rows={3}
            maxLength={500}
          />
          <Form.InputNumber
            field="sort_order"
            label={t('roles.form.sortOrder')}
            placeholder={t('roles.form.sortOrderPlaceholder')}
            min={0}
            max={9999}
            style={{ width: '100%' }}
          />
        </Form>
      </Modal>

      {/* Permission Configuration Modal */}
      <Modal
        title={t('roles.permissionConfig.title', { name: permissionRole?.name || '' })}
        visible={permissionModalVisible}
        onCancel={() => setPermissionModalVisible(false)}
        onOk={handleSavePermissions}
        confirmLoading={permissionSaving}
        width={700}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        bodyStyle={{ maxHeight: '60vh', overflow: 'auto' }}
      >
        {permissionRole?.is_system_role && (
          <Banner
            type="warning"
            description={t('roles.permissionConfig.systemWarning')}
            style={{ marginBottom: 16 }}
          />
        )}

        <div className="permission-config-header">
          <Text>
            {t('roles.permissionConfig.selectedCount', { count: selectedPermissions.length })}
          </Text>
          <Space>
            <Button size="small" onClick={() => setSelectedPermissions(allPermissions)}>
              {t('roles.permissionConfig.selectAll')}
            </Button>
            <Button size="small" onClick={() => setSelectedPermissions([])}>
              {t('roles.permissionConfig.clearAll')}
            </Button>
          </Space>
        </div>

        {permissionsLoading ? (
          <div className="permission-loading">
            <Spin />
          </div>
        ) : permissionTreeData.length > 0 ? (
          <Tree
            treeData={permissionTreeData}
            multiple
            checkRelation="related"
            value={selectedPermissions}
            onChange={handlePermissionChange}
            expandAll
            className="permission-tree"
          />
        ) : (
          <Empty description={t('roles.permissionConfig.noPermissions')} />
        )}
      </Modal>

      {/* Role Detail Modal */}
      <Modal
        title={t('roles.detail.title')}
        visible={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={
          <Button onClick={() => setDetailModalVisible(false)}>{t('roles.detail.close')}</Button>
        }
        width={600}
      >
        {detailRole && (
          <div className="role-detail">
            <Descriptions
              data={[
                { key: t('roles.detail.code'), value: detailRole.code },
                { key: t('roles.detail.name'), value: detailRole.name },
                { key: t('roles.detail.description'), value: detailRole.description || '-' },
                {
                  key: t('roles.detail.status'),
                  value: detailRole.is_enabled ? t('roles.enabled') : t('roles.disabled'),
                },
                {
                  key: t('roles.detail.type'),
                  value: detailRole.is_system_role ? t('roles.systemRole') : t('roles.customRole'),
                },
                { key: t('roles.detail.sortOrder'), value: String(detailRole.sort_order || 0) },
                { key: t('roles.detail.userCount'), value: String(detailRole.user_count || 0) },
                {
                  key: t('roles.detail.createdAt'),
                  value: formatDate(detailRole.created_at, i18n.language),
                },
                {
                  key: t('roles.detail.updatedAt'),
                  value: formatDate(detailRole.updated_at, i18n.language),
                },
              ]}
            />

            <div className="role-permissions-section">
              <Title heading={6}>
                {t('roles.detail.permissions')} ({detailRole.permissions?.length || 0})
              </Title>
              {detailRole.permissions && detailRole.permissions.length > 0 ? (
                <div className="role-permissions-list">
                  {detailRole.permissions.map((perm) => (
                    <Tag key={perm} className="permission-tag">
                      {getPermissionLabel(perm)}
                    </Tag>
                  ))}
                </div>
              ) : (
                <Text type="tertiary">{t('roles.detail.noPermissions')}</Text>
              )}
            </div>
          </div>
        )}
      </Modal>
    </Container>
  )
}
