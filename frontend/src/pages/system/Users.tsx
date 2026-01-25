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
  Input,
  Button,
  Checkbox,
  TagGroup,
} from '@douyinfe/semi-ui'
import type { FormApi } from '@douyinfe/semi-ui/lib/es/form/interface'
import { IconPlus, IconRefresh, IconKey, IconUnlock, IconLock } from '@douyinfe/semi-icons'
import {
  DataTable,
  TableToolbar,
  BulkActionBar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getIdentity } from '@/api/identity'
import type {
  User,
  UserStatus,
  UserListQuery,
  Role,
  CreateUserRequest,
  UpdateUserRequest,
} from '@/api/identity'
import './Users.css'

const { Title, Text } = Typography

// User type with index signature for DataTable compatibility
type UserRow = User & Record<string, unknown>

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '待激活', value: 'pending' },
  { label: '正常', value: 'active' },
  { label: '已锁定', value: 'locked' },
  { label: '已停用', value: 'deactivated' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<UserStatus, 'white' | 'green' | 'red' | 'grey'> = {
  pending: 'white',
  active: 'green',
  locked: 'red',
  deactivated: 'grey',
}

// Status labels
const STATUS_LABELS: Record<UserStatus, string> = {
  pending: '待激活',
  active: '正常',
  locked: '已锁定',
  deactivated: '已停用',
}

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
 * Generate a random password
 */
function generatePassword(): string {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%'
  let password = ''
  for (let i = 0; i < 12; i++) {
    password += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return password
}

/**
 * Users management page
 *
 * Features:
 * - User listing with pagination
 * - Search by username, email, phone, display name
 * - Filter by status and role
 * - Create/edit users
 * - Activate/deactivate/lock/unlock user actions
 * - Role assignment
 * - Password reset
 */
export default function UsersPage() {
  const api = useMemo(() => getIdentity(), [])

  // State for data
  const [userList, setUserList] = useState<UserRow[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [roleFilter, setRoleFilter] = useState<string>('')

  // Roles for filter and assignment
  const [roles, setRoles] = useState<Role[]>([])

  // Modal state
  const [modalVisible, setModalVisible] = useState(false)
  const [modalMode, setModalMode] = useState<'create' | 'edit'>('create')
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [modalLoading, setModalLoading] = useState(false)

  // Password reset modal state
  const [resetPasswordVisible, setResetPasswordVisible] = useState(false)
  const [resetPasswordUser, setResetPasswordUser] = useState<User | null>(null)
  const [newPassword, setNewPassword] = useState('')

  // Role assignment modal state
  const [roleModalVisible, setRoleModalVisible] = useState(false)
  const [roleAssignUser, setRoleAssignUser] = useState<User | null>(null)
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>([])

  // Form ref
  const formApiRef = useRef<FormApi | null>(null)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch roles for filter and assignment
  const fetchRoles = useCallback(async () => {
    try {
      const response = await api.listRoles({ page_size: 100 })
      if (response.success && response.data) {
        setRoles(response.data.roles)
      }
    } catch {
      // Silent fail for roles fetch
    }
  }, [api])

  // Fetch users
  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const params: UserListQuery = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        keyword: searchKeyword || undefined,
        status: (statusFilter || undefined) as UserStatus | undefined,
        role_id: roleFilter || undefined,
        sort_by: (state.sort.field as UserListQuery['sort_by']) || 'created_at',
        sort_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await api.listUsers(params)

      if (response.success && response.data) {
        setUserList(response.data.users as UserRow[])
        setTotal(response.data.total)
      }
    } catch {
      Toast.error('获取用户列表失败')
    } finally {
      setLoading(false)
    }
  }, [
    api,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    statusFilter,
    roleFilter,
  ])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchRoles()
  }, [fetchRoles])

  useEffect(() => {
    fetchUsers()
  }, [fetchUsers])

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
      setFilter('status', statusValue || null)
    },
    [setFilter]
  )

  // Handle role filter change
  const handleRoleChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const roleValue = typeof value === 'string' ? value : ''
      setRoleFilter(roleValue)
      setFilter('role_id', roleValue || null)
    },
    [setFilter]
  )

  // Handle create user
  const handleCreate = useCallback(() => {
    setModalMode('create')
    setEditingUser(null)
    setModalVisible(true)
  }, [])

  // Handle edit user
  const handleEdit = useCallback((user: UserRow) => {
    setModalMode('edit')
    setEditingUser(user)
    setModalVisible(true)
  }, [])

  // Handle modal submit
  const handleModalSubmit = useCallback(async () => {
    if (!formApiRef.current) return

    try {
      await formApiRef.current.validate()
      const values = formApiRef.current.getValues()
      setModalLoading(true)

      if (modalMode === 'create') {
        const request: CreateUserRequest = {
          username: values.username,
          password: values.password,
          email: values.email || undefined,
          phone: values.phone || undefined,
          display_name: values.display_name || undefined,
          notes: values.notes || undefined,
          role_ids: values.role_ids || [],
        }
        const response = await api.createUser(request)
        if (response.success) {
          Toast.success('用户创建成功')
          setModalVisible(false)
          fetchUsers()
        } else {
          Toast.error(response.error?.message || '创建用户失败')
        }
      } else if (editingUser) {
        const request: UpdateUserRequest = {
          email: values.email || undefined,
          phone: values.phone || undefined,
          display_name: values.display_name || undefined,
          notes: values.notes || undefined,
        }
        const response = await api.updateUser(editingUser.id, request)
        if (response.success) {
          Toast.success('用户更新成功')
          setModalVisible(false)
          fetchUsers()
        } else {
          Toast.error(response.error?.message || '更新用户失败')
        }
      }
    } catch {
      // Validation failed or API error
    } finally {
      setModalLoading(false)
    }
  }, [modalMode, editingUser, api, fetchUsers])

  // Handle activate user
  const handleActivate = useCallback(
    async (user: UserRow) => {
      try {
        const response = await api.activateUser(user.id)
        if (response.success) {
          Toast.success(`用户 "${user.display_name || user.username}" 已激活`)
          fetchUsers()
        } else {
          Toast.error(response.error?.message || '激活用户失败')
        }
      } catch {
        Toast.error('激活用户失败')
      }
    },
    [api, fetchUsers]
  )

  // Handle deactivate user
  const handleDeactivate = useCallback(
    async (user: UserRow) => {
      Modal.confirm({
        title: '确认停用',
        content: `确定要停用用户 "${user.display_name || user.username}" 吗？停用后该用户将无法登录。`,
        okText: '确认停用',
        cancelText: '取消',
        okButtonProps: { type: 'warning' },
        onOk: async () => {
          try {
            const response = await api.deactivateUser(user.id)
            if (response.success) {
              Toast.success(`用户 "${user.display_name || user.username}" 已停用`)
              fetchUsers()
            } else {
              Toast.error(response.error?.message || '停用用户失败')
            }
          } catch {
            Toast.error('停用用户失败')
          }
        },
      })
    },
    [api, fetchUsers]
  )

  // Handle lock user
  const handleLock = useCallback(
    async (user: UserRow) => {
      Modal.confirm({
        title: '确认锁定',
        content: `确定要锁定用户 "${user.display_name || user.username}" 吗？锁定后该用户将无法登录。`,
        okText: '确认锁定',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            const response = await api.lockUser(user.id)
            if (response.success) {
              Toast.success(`用户 "${user.display_name || user.username}" 已锁定`)
              fetchUsers()
            } else {
              Toast.error(response.error?.message || '锁定用户失败')
            }
          } catch {
            Toast.error('锁定用户失败')
          }
        },
      })
    },
    [api, fetchUsers]
  )

  // Handle unlock user
  const handleUnlock = useCallback(
    async (user: UserRow) => {
      try {
        const response = await api.unlockUser(user.id)
        if (response.success) {
          Toast.success(`用户 "${user.display_name || user.username}" 已解锁`)
          fetchUsers()
        } else {
          Toast.error(response.error?.message || '解锁用户失败')
        }
      } catch {
        Toast.error('解锁用户失败')
      }
    },
    [api, fetchUsers]
  )

  // Handle delete user
  const handleDelete = useCallback(
    async (user: UserRow) => {
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除用户 "${user.display_name || user.username}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            const response = await api.deleteUser(user.id)
            if (response.success) {
              Toast.success(`用户 "${user.display_name || user.username}" 已删除`)
              fetchUsers()
            } else {
              Toast.error(response.error?.message || '删除用户失败')
            }
          } catch {
            Toast.error('删除用户失败')
          }
        },
      })
    },
    [api, fetchUsers]
  )

  // Handle reset password
  const handleResetPassword = useCallback((user: UserRow) => {
    setResetPasswordUser(user)
    setNewPassword(generatePassword())
    setResetPasswordVisible(true)
  }, [])

  // Handle submit reset password
  const handleSubmitResetPassword = useCallback(async () => {
    if (!resetPasswordUser || !newPassword) return

    try {
      const response = await api.resetPassword(resetPasswordUser.id, { new_password: newPassword })
      if (response.success) {
        Toast.success('密码重置成功')
        setResetPasswordVisible(false)
      } else {
        Toast.error(response.error?.message || '密码重置失败')
      }
    } catch {
      Toast.error('密码重置失败')
    }
  }, [api, resetPasswordUser, newPassword])

  // Handle assign roles
  const handleAssignRoles = useCallback((user: UserRow) => {
    setRoleAssignUser(user)
    setSelectedRoleIds(user.role_ids || [])
    setRoleModalVisible(true)
  }, [])

  // Handle submit role assignment
  const handleSubmitRoles = useCallback(async () => {
    if (!roleAssignUser) return

    try {
      const response = await api.assignRoles(roleAssignUser.id, { role_ids: selectedRoleIds })
      if (response.success) {
        Toast.success('角色分配成功')
        setRoleModalVisible(false)
        fetchUsers()
      } else {
        Toast.error(response.error?.message || '角色分配失败')
      }
    } catch {
      Toast.error('角色分配失败')
    }
  }, [api, roleAssignUser, selectedRoleIds, fetchUsers])

  // Handle bulk activate
  const handleBulkActivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.activateUser(id)))
      Toast.success(`已激活 ${selectedRowKeys.length} 个用户`)
      setSelectedRowKeys([])
      fetchUsers()
    } catch {
      Toast.error('批量激活失败')
    }
  }, [api, selectedRowKeys, fetchUsers])

  // Handle bulk deactivate
  const handleBulkDeactivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.deactivateUser(id)))
      Toast.success(`已停用 ${selectedRowKeys.length} 个用户`)
      setSelectedRowKeys([])
      fetchUsers()
    } catch {
      Toast.error('批量停用失败')
    }
  }, [api, selectedRowKeys, fetchUsers])

  // Get role name by ID
  const getRoleName = useCallback(
    (roleId: string) => {
      const role = roles.find((r) => r.id === roleId)
      return role?.name || roleId
    },
    [roles]
  )

  // Table columns
  const tableColumns: DataTableColumn<UserRow>[] = useMemo(
    () => [
      {
        title: '用户名',
        dataIndex: 'username',
        width: 140,
        sortable: true,
        render: (username: unknown) => (
          <span className="user-username">{(username as string) || '-'}</span>
        ),
      },
      {
        title: '显示名称',
        dataIndex: 'display_name',
        sortable: true,
        ellipsis: true,
        render: (displayName: unknown, record: UserRow) => (
          <div className="user-display-name-cell">
            <span className="user-display-name">{(displayName as string) || record.username}</span>
          </div>
        ),
      },
      {
        title: '联系方式',
        dataIndex: 'email',
        width: 200,
        render: (_email: unknown, record: UserRow) => (
          <div className="user-contact-cell">
            {record.email && <span className="user-email">{record.email}</span>}
            {record.phone && <span className="user-phone">{record.phone}</span>}
            {!record.email && !record.phone && '-'}
          </div>
        ),
      },
      {
        title: '角色',
        dataIndex: 'role_ids',
        width: 200,
        render: (roleIds: unknown) => {
          const ids = roleIds as string[] | undefined
          if (!ids || ids.length === 0) {
            return <Text type="tertiary">无角色</Text>
          }
          return (
            <TagGroup
              maxTagCount={2}
              showPopover
              tagList={ids.map((id) => ({ tagKey: id, children: getRoleName(id), color: 'blue' }))}
            />
          )
        },
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as UserStatus | undefined
          if (!statusValue) return '-'
          return <Tag color={STATUS_TAG_COLORS[statusValue]}>{STATUS_LABELS[statusValue]}</Tag>
        },
      },
      {
        title: '最后登录',
        dataIndex: 'last_login_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    [getRoleName]
  )

  // Table row actions
  const tableActions: TableAction<UserRow>[] = useMemo(
    () => [
      {
        key: 'edit',
        label: '编辑',
        onClick: handleEdit,
      },
      {
        key: 'roles',
        label: '分配角色',
        onClick: handleAssignRoles,
      },
      {
        key: 'reset-password',
        label: '重置密码',
        icon: <IconKey size="small" />,
        onClick: handleResetPassword,
      },
      {
        key: 'activate',
        label: '激活',
        type: 'primary',
        onClick: handleActivate,
        hidden: (record) => record.status === 'active',
      },
      {
        key: 'deactivate',
        label: '停用',
        type: 'warning',
        onClick: handleDeactivate,
        hidden: (record) => record.status === 'deactivated',
      },
      {
        key: 'lock',
        label: '锁定',
        type: 'warning',
        icon: <IconLock size="small" />,
        onClick: handleLock,
        hidden: (record) => record.status === 'locked',
      },
      {
        key: 'unlock',
        label: '解锁',
        icon: <IconUnlock size="small" />,
        onClick: handleUnlock,
        hidden: (record) => record.status !== 'locked',
      },
      {
        key: 'delete',
        label: '删除',
        type: 'danger',
        onClick: handleDelete,
      },
    ],
    [
      handleEdit,
      handleAssignRoles,
      handleResetPassword,
      handleActivate,
      handleDeactivate,
      handleLock,
      handleUnlock,
      handleDelete,
    ]
  )

  // Row selection handler
  const onSelectionChange = useCallback((keys: string[], _rows: UserRow[]) => {
    setSelectedRowKeys(keys)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchUsers()
  }, [fetchUsers])

  // Role filter options
  const roleOptions = useMemo(() => {
    const options = [{ label: '全部角色', value: '' }]
    roles.forEach((role) => {
      options.push({ label: role.name, value: role.id })
    })
    return options
  }, [roles])

  return (
    <Container size="full" className="users-page">
      <Card className="users-card">
        <div className="users-header">
          <Title heading={4} style={{ margin: 0 }}>
            用户管理
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索用户名、姓名、邮箱、电话..."
          primaryAction={{
            label: '新增用户',
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
            <Space className="users-filter-container">
              <Select
                placeholder="状态筛选"
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="角色筛选"
                value={roleFilter}
                onChange={handleRoleChange}
                optionList={roleOptions}
                style={{ width: 160 }}
              />
            </Space>
          }
        />

        {selectedRowKeys.length > 0 && (
          <BulkActionBar
            selectedCount={selectedRowKeys.length}
            onCancel={() => setSelectedRowKeys([])}
          >
            <Tag color="blue" onClick={handleBulkActivate} style={{ cursor: 'pointer' }}>
              批量激活
            </Tag>
            <Tag color="orange" onClick={handleBulkDeactivate} style={{ cursor: 'pointer' }}>
              批量停用
            </Tag>
          </BulkActionBar>
        )}

        <Spin spinning={loading}>
          <DataTable<UserRow>
            data={userList}
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
            rowSelection={{
              selectedRowKeys,
              onChange: onSelectionChange,
            }}
            scroll={{ x: 1200 }}
          />
        </Spin>
      </Card>

      {/* Create/Edit User Modal */}
      <Modal
        title={modalMode === 'create' ? '新增用户' : '编辑用户'}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleModalSubmit}
        confirmLoading={modalLoading}
        width={600}
        okText={modalMode === 'create' ? '创建' : '保存'}
        cancelText="取消"
      >
        <Form
          getFormApi={(api) => {
            formApiRef.current = api
          }}
          initValues={
            editingUser
              ? {
                  username: editingUser.username,
                  email: editingUser.email,
                  phone: editingUser.phone,
                  display_name: editingUser.display_name,
                  role_ids: editingUser.role_ids,
                }
              : {}
          }
          labelPosition="left"
          labelWidth={100}
        >
          <Form.Input
            field="username"
            label="用户名"
            placeholder="请输入用户名"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少3个字符' },
              { max: 100, message: '用户名最多100个字符' },
            ]}
            disabled={modalMode === 'edit'}
          />
          {modalMode === 'create' && (
            <Form.Input
              field="password"
              label="密码"
              mode="password"
              placeholder="请输入密码"
              rules={[
                { required: true, message: '请输入密码' },
                { min: 8, message: '密码至少8个字符' },
                { max: 128, message: '密码最多128个字符' },
              ]}
              extraText={
                <Button
                  theme="borderless"
                  size="small"
                  onClick={() => {
                    formApiRef.current?.setValue('password', generatePassword())
                  }}
                >
                  生成随机密码
                </Button>
              }
            />
          )}
          <Form.Input field="display_name" label="显示名称" placeholder="请输入显示名称" />
          <Form.Input
            field="email"
            label="邮箱"
            placeholder="请输入邮箱"
            rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
          />
          <Form.Input field="phone" label="电话" placeholder="请输入电话号码" />
          {modalMode === 'create' && (
            <Form.CheckboxGroup field="role_ids" label="分配角色" direction="horizontal">
              {roles.map((role) => (
                <Checkbox key={role.id} value={role.id}>
                  {role.name}
                </Checkbox>
              ))}
            </Form.CheckboxGroup>
          )}
          <Form.TextArea field="notes" label="备注" placeholder="请输入备注信息" rows={3} />
        </Form>
      </Modal>

      {/* Reset Password Modal */}
      <Modal
        title="重置密码"
        visible={resetPasswordVisible}
        onCancel={() => setResetPasswordVisible(false)}
        onOk={handleSubmitResetPassword}
        okText="确认重置"
        cancelText="取消"
        width={500}
      >
        <div style={{ marginBottom: 16 }}>
          <Text>
            即将为用户{' '}
            <Text strong>{resetPasswordUser?.display_name || resetPasswordUser?.username}</Text>{' '}
            重置密码
          </Text>
        </div>
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary">新密码：</Text>
          <Input
            value={newPassword}
            onChange={(v) => setNewPassword(v)}
            style={{ marginTop: 8 }}
            suffix={
              <Button
                theme="borderless"
                size="small"
                onClick={() => setNewPassword(generatePassword())}
              >
                重新生成
              </Button>
            }
          />
        </div>
        <div>
          <Text type="warning">请记录新密码并安全地告知用户。密码一旦重置，旧密码将立即失效。</Text>
        </div>
      </Modal>

      {/* Role Assignment Modal */}
      <Modal
        title="分配角色"
        visible={roleModalVisible}
        onCancel={() => setRoleModalVisible(false)}
        onOk={handleSubmitRoles}
        okText="保存"
        cancelText="取消"
        width={500}
      >
        <div style={{ marginBottom: 16 }}>
          <Text>
            为用户 <Text strong>{roleAssignUser?.display_name || roleAssignUser?.username}</Text>{' '}
            分配角色
          </Text>
        </div>
        <div>
          <Checkbox.Group
            value={selectedRoleIds}
            onChange={(v) => setSelectedRoleIds(v as string[])}
            direction="vertical"
          >
            {roles.map((role) => (
              <Checkbox key={role.id} value={role.id} style={{ marginBottom: 8 }}>
                <div>
                  <Text strong>{role.name}</Text>
                  {role.description && (
                    <Text type="tertiary" style={{ marginLeft: 8 }}>
                      {role.description}
                    </Text>
                  )}
                  {role.is_system_role && (
                    <Tag size="small" color="blue" style={{ marginLeft: 8 }}>
                      系统角色
                    </Tag>
                  )}
                </div>
              </Checkbox>
            ))}
          </Checkbox.Group>
        </div>
      </Modal>
    </Container>
  )
}
