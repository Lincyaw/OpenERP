import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
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

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '已启用', value: 'true' },
  { label: '已禁用', value: 'false' },
]

// Type options for filter
const TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: '系统角色', value: 'true' },
  { label: '自定义角色', value: 'false' },
]

/**
 * Format date for display
 */
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
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
 * Resource name labels (Chinese)
 */
const RESOURCE_LABELS: Record<string, string> = {
  product: '商品',
  category: '分类',
  customer: '客户',
  supplier: '供应商',
  warehouse: '仓库',
  inventory: '库存',
  sales_order: '销售订单',
  purchase_order: '采购订单',
  sales_return: '销售退货',
  purchase_return: '采购退货',
  account_receivable: '应收账款',
  account_payable: '应付账款',
  receipt: '收款单',
  payment: '付款单',
  expense: '费用',
  income: '其他收入',
  report: '报表',
  user: '用户',
  role: '角色',
  tenant: '租户',
}

/**
 * Action name labels (Chinese)
 */
const ACTION_LABELS: Record<string, string> = {
  create: '创建',
  read: '查看',
  update: '修改',
  delete: '删除',
  enable: '启用',
  disable: '停用',
  confirm: '确认',
  cancel: '取消',
  ship: '发货',
  receive: '收货',
  approve: '审批',
  reject: '拒绝',
  adjust: '调整',
  lock: '锁定',
  unlock: '解锁',
  reconcile: '核销',
  export: '导出',
  import: '导入',
  assign_role: '分配角色',
  view_all: '查看全部',
}

/**
 * Get readable permission label
 */
function getPermissionLabel(permission: string): string {
  const parts = permission.split(':')
  if (parts.length !== 2) return permission

  const [resource, action] = parts
  const resourceLabel = RESOURCE_LABELS[resource] || resource
  const actionLabel = ACTION_LABELS[action] || action

  return `${actionLabel}${resourceLabel}`
}

/**
 * Convert permissions to tree data
 */
function permissionsToTreeData(permissions: string[]): TreeNodeData[] {
  const grouped = groupPermissionsByResource(permissions)
  const treeData: TreeNodeData[] = []

  grouped.forEach((perms, resource) => {
    const resourceLabel = RESOURCE_LABELS[resource] || resource
    treeData.push({
      key: resource,
      label: resourceLabel,
      value: resource,
      children: perms.map((perm) => {
        const action = perm.split(':')[1]
        const actionLabel = ACTION_LABELS[action] || action
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
    return aLabel.localeCompare(bLabel, 'zh-CN')
  })

  return treeData
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
  const api = useMemo(() => getIdentity(), [])

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
      Toast.error('获取角色列表失败')
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
  const handleEdit = useCallback((role: RoleRow) => {
    if (role.is_system_role) {
      Toast.warning('系统角色不能编辑')
      return
    }
    setModalMode('edit')
    setEditingRole(role)
    setModalVisible(true)
  }, [])

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
          Toast.success('角色创建成功')
          setModalVisible(false)
          fetchRoles()
        } else {
          Toast.error(response.error?.message || '创建角色失败')
        }
      } else if (editingRole) {
        const request: UpdateRoleRequest = {
          name: values.name,
          description: values.description || undefined,
          sort_order: values.sort_order ? Number(values.sort_order) : undefined,
        }
        const response = await api.updateRole(editingRole.id, request)
        if (response.success) {
          Toast.success('角色更新成功')
          setModalVisible(false)
          fetchRoles()
        } else {
          Toast.error(response.error?.message || '更新角色失败')
        }
      }
    } catch {
      // Validation failed or API error
    } finally {
      setModalLoading(false)
    }
  }, [modalMode, editingRole, api, fetchRoles])

  // Handle enable role
  const handleEnable = useCallback(
    async (role: RoleRow) => {
      try {
        const response = await api.enableRole(role.id)
        if (response.success) {
          Toast.success(`角色 "${role.name}" 已启用`)
          fetchRoles()
        } else {
          Toast.error(response.error?.message || '启用角色失败')
        }
      } catch {
        Toast.error('启用角色失败')
      }
    },
    [api, fetchRoles]
  )

  // Handle disable role
  const handleDisable = useCallback(
    async (role: RoleRow) => {
      if (role.is_system_role) {
        Toast.warning('系统角色不能禁用')
        return
      }
      Modal.confirm({
        title: '确认禁用',
        content: `确定要禁用角色 "${role.name}" 吗？禁用后拥有该角色的用户将失去相关权限。`,
        okText: '确认禁用',
        cancelText: '取消',
        okButtonProps: { type: 'warning' },
        onOk: async () => {
          try {
            const response = await api.disableRole(role.id)
            if (response.success) {
              Toast.success(`角色 "${role.name}" 已禁用`)
              fetchRoles()
            } else {
              Toast.error(response.error?.message || '禁用角色失败')
            }
          } catch {
            Toast.error('禁用角色失败')
          }
        },
      })
    },
    [api, fetchRoles]
  )

  // Handle delete role
  const handleDelete = useCallback(
    async (role: RoleRow) => {
      if (role.is_system_role) {
        Toast.warning('系统角色不能删除')
        return
      }
      if (role.user_count && role.user_count > 0) {
        Toast.warning(`该角色还有 ${role.user_count} 个用户使用，请先移除用户的该角色`)
        return
      }
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除角色 "${role.name}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            const response = await api.deleteRole(role.id)
            if (response.success) {
              Toast.success(`角色 "${role.name}" 已删除`)
              fetchRoles()
            } else {
              Toast.error(response.error?.message || '删除角色失败')
            }
          } catch {
            Toast.error('删除角色失败')
          }
        },
      })
    },
    [api, fetchRoles]
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
        Toast.success('权限配置已保存')
        setPermissionModalVisible(false)
        fetchRoles()
      } else {
        Toast.error(response.error?.message || '保存权限失败')
      }
    } catch {
      Toast.error('保存权限失败')
    } finally {
      setPermissionSaving(false)
    }
  }, [api, permissionRole, selectedPermissions, fetchRoles])

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
        title: '角色编码',
        dataIndex: 'code',
        width: 140,
        render: (code: unknown) => <span className="role-code">{(code as string) || '-'}</span>,
      },
      {
        title: '角色名称',
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: RoleRow) => (
          <div className="role-name-cell">
            <span className="role-name">{name as string}</span>
            {record.is_system_role && (
              <Tag size="small" color="blue" className="role-system-tag">
                系统
              </Tag>
            )}
          </div>
        ),
      },
      {
        title: '描述',
        dataIndex: 'description',
        width: 200,
        ellipsis: true,
        render: (desc: unknown) => <Text type="tertiary">{(desc as string) || '-'}</Text>,
      },
      {
        title: '权限数',
        dataIndex: 'permissions',
        width: 100,
        align: 'center',
        render: (permissions: unknown) => {
          const perms = permissions as string[] | undefined
          const count = perms?.length || 0
          return <Tag color={count > 0 ? 'cyan' : 'grey'}>{count} 项</Tag>
        },
      },
      {
        title: '用户数',
        dataIndex: 'user_count',
        width: 90,
        align: 'center',
        render: (count: unknown) => <Text type="secondary">{(count as number) || 0}</Text>,
      },
      {
        title: '状态',
        dataIndex: 'is_enabled',
        width: 90,
        align: 'center',
        render: (isEnabled: unknown) => {
          const enabled = isEnabled as boolean
          return <Tag color={enabled ? 'green' : 'grey'}>{enabled ? '已启用' : '已禁用'}</Tag>
        },
      },
      {
        title: '排序',
        dataIndex: 'sort_order',
        width: 80,
        align: 'center',
        sortable: true,
        render: (order: unknown) => (order as number) || 0,
      },
      {
        title: '更新时间',
        dataIndex: 'updated_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<RoleRow>[] = useMemo(
    () => [
      {
        key: 'view',
        label: '查看',
        onClick: handleViewDetail,
      },
      {
        key: 'edit',
        label: '编辑',
        onClick: handleEdit,
        hidden: (record) => record.is_system_role,
      },
      {
        key: 'permissions',
        label: '配置权限',
        icon: <IconSetting size="small" />,
        onClick: handleConfigurePermissions,
      },
      {
        key: 'enable',
        label: '启用',
        type: 'primary',
        onClick: handleEnable,
        hidden: (record) => record.is_enabled,
      },
      {
        key: 'disable',
        label: '禁用',
        type: 'warning',
        icon: <IconLock size="small" />,
        onClick: handleDisable,
        hidden: (record) => !record.is_enabled || record.is_system_role,
      },
      {
        key: 'delete',
        label: '删除',
        type: 'danger',
        onClick: handleDelete,
        hidden: (record) => record.is_system_role,
      },
    ],
    [
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
            角色管理
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索角色编码、名称..."
          primaryAction={{
            label: '新增角色',
            icon: <IconPlus />,
            onClick: handleCreate,
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="roles-filter-container">
              <Select
                placeholder="状态筛选"
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="类型筛选"
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
        title={modalMode === 'create' ? '新增角色' : '编辑角色'}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleModalSubmit}
        confirmLoading={modalLoading}
        width={500}
        okText={modalMode === 'create' ? '创建' : '保存'}
        cancelText="取消"
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
            label="角色编码"
            placeholder="请输入角色编码，如 admin、operator"
            rules={[
              { required: true, message: '请输入角色编码' },
              { min: 2, message: '角色编码至少2个字符' },
              { max: 50, message: '角色编码最多50个字符' },
              {
                pattern: /^[a-z][a-z0-9_]*$/,
                message: '角色编码必须以小写字母开头，只能包含小写字母、数字和下划线',
              },
            ]}
            disabled={modalMode === 'edit'}
          />
          <Form.Input
            field="name"
            label="角色名称"
            placeholder="请输入角色名称"
            rules={[
              { required: true, message: '请输入角色名称' },
              { min: 2, message: '角色名称至少2个字符' },
              { max: 100, message: '角色名称最多100个字符' },
            ]}
          />
          <Form.TextArea
            field="description"
            label="描述"
            placeholder="请输入角色描述"
            rows={3}
            maxLength={500}
          />
          <Form.InputNumber
            field="sort_order"
            label="排序值"
            placeholder="数值越小越靠前"
            min={0}
            max={9999}
            style={{ width: '100%' }}
          />
        </Form>
      </Modal>

      {/* Permission Configuration Modal */}
      <Modal
        title={`配置权限 - ${permissionRole?.name || ''}`}
        visible={permissionModalVisible}
        onCancel={() => setPermissionModalVisible(false)}
        onOk={handleSavePermissions}
        confirmLoading={permissionSaving}
        width={700}
        okText="保存"
        cancelText="取消"
        bodyStyle={{ maxHeight: '60vh', overflow: 'auto' }}
      >
        {permissionRole?.is_system_role && (
          <Banner
            type="warning"
            description="系统角色的权限修改可能影响系统核心功能，请谨慎操作。"
            style={{ marginBottom: 16 }}
          />
        )}

        <div className="permission-config-header">
          <Text>
            已选择 <Text strong>{selectedPermissions.length}</Text> 项权限
          </Text>
          <Space>
            <Button size="small" onClick={() => setSelectedPermissions(allPermissions)}>
              全选
            </Button>
            <Button size="small" onClick={() => setSelectedPermissions([])}>
              清空
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
          <Empty description="暂无可用权限" />
        )}
      </Modal>

      {/* Role Detail Modal */}
      <Modal
        title="角色详情"
        visible={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={<Button onClick={() => setDetailModalVisible(false)}>关闭</Button>}
        width={600}
      >
        {detailRole && (
          <div className="role-detail">
            <Descriptions
              data={[
                { key: '角色编码', value: detailRole.code },
                { key: '角色名称', value: detailRole.name },
                { key: '描述', value: detailRole.description || '-' },
                { key: '状态', value: detailRole.is_enabled ? '已启用' : '已禁用' },
                { key: '类型', value: detailRole.is_system_role ? '系统角色' : '自定义角色' },
                { key: '排序值', value: String(detailRole.sort_order || 0) },
                { key: '用户数', value: String(detailRole.user_count || 0) },
                { key: '创建时间', value: formatDate(detailRole.created_at) },
                { key: '更新时间', value: formatDate(detailRole.updated_at) },
              ]}
            />

            <div className="role-permissions-section">
              <Title heading={6}>权限列表 ({detailRole.permissions?.length || 0})</Title>
              {detailRole.permissions && detailRole.permissions.length > 0 ? (
                <div className="role-permissions-list">
                  {detailRole.permissions.map((perm) => (
                    <Tag key={perm} className="permission-tag">
                      {getPermissionLabel(perm)}
                    </Tag>
                  ))}
                </div>
              ) : (
                <Text type="tertiary">该角色暂无权限</Text>
              )}
            </div>
          </div>
        )}
      </Modal>
    </Container>
  )
}
