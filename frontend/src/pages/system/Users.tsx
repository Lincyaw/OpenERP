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
  Input,
  Button,
  Checkbox,
  TagGroup,
} from '@douyinfe/semi-ui-19'
import type { FormApi } from '@douyinfe/semi-ui-19/lib/es/form/interface'
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
import {
  listUsers,
  createUser,
  updateUser,
  activateUser,
  deactivateUser,
  lockUser,
  unlockUser,
  deleteUser,
  resetPasswordUser as resetPasswordUserApi,
  assignRolesUser,
} from '@/api/users/users'
import { listRoles } from '@/api/roles/roles'
import type {
  HandlerUserResponse,
  HandlerRoleResponse,
  ListUsersParams,
  ListUsersStatus,
  CreateUserBody,
  UpdateUserBody,
  ResetPasswordUserBody,
  AssignRolesUserBody,
  ListRolesParams,
} from '@/api/models'

// Type aliases for backward compatibility
type User = HandlerUserResponse
type Role = HandlerRoleResponse
type UserStatus = ListUsersStatus
type UserListQuery = ListUsersParams
type CreateUserRequest = CreateUserBody
type UpdateUserRequest = UpdateUserBody
import './Users.css'

const { Title, Text } = Typography

// User type with index signature for DataTable compatibility
type UserRow = User & Record<string, unknown>

/**
 * Format date for display
 */
function formatDate(dateStr?: string, locale?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString(locale || 'zh-CN', {
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
  const { t, i18n } = useTranslation('system')

  // Status tag color mapping
  const STATUS_TAG_COLORS: Record<UserStatus, 'white' | 'green' | 'red' | 'grey'> = {
    pending: 'white',
    active: 'green',
    locked: 'red',
    deactivated: 'grey',
  }

  // Status options for filter - using useMemo to react to language changes
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('users.allStatus'), value: '' },
      { label: t('users.status.pending'), value: 'pending' },
      { label: t('users.status.active'), value: 'active' },
      { label: t('users.status.locked'), value: 'locked' },
      { label: t('users.status.deactivated'), value: 'deactivated' },
    ],
    [t]
  )

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
      const params: ListRolesParams = { page_size: 100 }
      const response = await listRoles(params)
      if (response.status === 200 && response.data.success && response.data.data) {
        setRoles(response.data.data.roles || [])
      }
    } catch {
      // Silent fail for roles fetch
    }
  }, [])

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

      const response = await listUsers(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setUserList(response.data.data.users as UserRow[])
        setTotal(response.data.data.total || 0)
      }
    } catch {
      Toast.error(t('users.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    statusFilter,
    roleFilter,
    t,
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
        const response = await createUser(request)
        if (response.status === 201 && response.data.success) {
          Toast.success(t('users.messages.createSuccess'))
          setModalVisible(false)
          fetchUsers()
        } else {
          Toast.error(response.data.error?.message || t('users.messages.createError'))
        }
      } else if (editingUser) {
        const request: UpdateUserRequest = {
          email: values.email || undefined,
          phone: values.phone || undefined,
          display_name: values.display_name || undefined,
          notes: values.notes || undefined,
        }
        const response = await updateUser(editingUser.id!, request)
        if (response.status === 200 && response.data.success) {
          Toast.success(t('users.messages.updateSuccess'))
          setModalVisible(false)
          fetchUsers()
        } else {
          Toast.error(response.data.error?.message || t('users.messages.updateError'))
        }
      }
    } catch {
      // Validation failed or API error
    } finally {
      setModalLoading(false)
    }
  }, [modalMode, editingUser, fetchUsers, t])

  // Handle activate user
  const handleActivate = useCallback(
    async (user: UserRow) => {
      try {
        const response = await activateUser(user.id!, {})
        if (response.status === 200 && response.data.success) {
          Toast.success(
            t('users.messages.activateSuccess', { name: user.display_name || user.username })
          )
          fetchUsers()
        } else {
          Toast.error(response.data.error?.message || t('users.messages.activateError'))
        }
      } catch {
        Toast.error(t('users.messages.activateError'))
      }
    },
    [fetchUsers, t]
  )

  // Handle deactivate user
  const handleDeactivate = useCallback(
    async (user: UserRow) => {
      Modal.confirm({
        title: t('users.confirm.deactivateTitle'),
        content: t('users.confirm.deactivateContent', { name: user.display_name || user.username }),
        okText: t('users.confirm.deactivateOk'),
        cancelText: t('common.cancel'),
        okButtonProps: { type: 'warning' },
        onOk: async () => {
          try {
            const response = await deactivateUser(user.id!, {})
            if (response.status === 200 && response.data.success) {
              Toast.success(
                t('users.messages.deactivateSuccess', { name: user.display_name || user.username })
              )
              fetchUsers()
            } else {
              Toast.error(response.data.error?.message || t('users.messages.deactivateError'))
            }
          } catch {
            Toast.error(t('users.messages.deactivateError'))
          }
        },
      })
    },
    [fetchUsers, t]
  )

  // Handle lock user
  const handleLock = useCallback(
    async (user: UserRow) => {
      Modal.confirm({
        title: t('users.confirm.lockTitle'),
        content: t('users.confirm.lockContent', { name: user.display_name || user.username }),
        okText: t('users.confirm.lockOk'),
        cancelText: t('common.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            const response = await lockUser(user.id!, {})
            if (response.status === 200 && response.data.success) {
              Toast.success(
                t('users.messages.lockSuccess', { name: user.display_name || user.username })
              )
              fetchUsers()
            } else {
              Toast.error(response.data.error?.message || t('users.messages.lockError'))
            }
          } catch {
            Toast.error(t('users.messages.lockError'))
          }
        },
      })
    },
    [fetchUsers, t]
  )

  // Handle unlock user
  const handleUnlock = useCallback(
    async (user: UserRow) => {
      try {
        const response = await unlockUser(user.id!, {})
        if (response.status === 200 && response.data.success) {
          Toast.success(
            t('users.messages.unlockSuccess', { name: user.display_name || user.username })
          )
          fetchUsers()
        } else {
          Toast.error(response.data.error?.message || t('users.messages.unlockError'))
        }
      } catch {
        Toast.error(t('users.messages.unlockError'))
      }
    },
    [fetchUsers, t]
  )

  // Handle delete user
  const handleDelete = useCallback(
    async (user: UserRow) => {
      Modal.confirm({
        title: t('users.confirm.deleteTitle'),
        content: t('users.confirm.deleteContent', { name: user.display_name || user.username }),
        okText: t('users.confirm.deleteOk'),
        cancelText: t('common.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            const response = await deleteUser(user.id!)
            if (response.status === 200 && response.data.success) {
              Toast.success(
                t('users.messages.deleteSuccess', { name: user.display_name || user.username })
              )
              fetchUsers()
            } else {
              Toast.error(response.data.error?.message || t('users.messages.deleteError'))
            }
          } catch {
            Toast.error(t('users.messages.deleteError'))
          }
        },
      })
    },
    [fetchUsers, t]
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
      const body: ResetPasswordUserBody = { new_password: newPassword }
      const response = await resetPasswordUserApi(resetPasswordUser.id!, body)
      if (response.status === 200 && response.data.success) {
        Toast.success(t('users.messages.resetPasswordSuccess'))
        setResetPasswordVisible(false)
      } else {
        Toast.error(response.data.error?.message || t('users.messages.resetPasswordError'))
      }
    } catch {
      Toast.error(t('users.messages.resetPasswordError'))
    }
  }, [resetPasswordUser, newPassword, t])

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
      const body: AssignRolesUserBody = { role_ids: selectedRoleIds }
      const response = await assignRolesUser(roleAssignUser.id!, body)
      if (response.status === 200 && response.data.success) {
        Toast.success(t('users.messages.assignRolesSuccess'))
        setRoleModalVisible(false)
        fetchUsers()
      } else {
        Toast.error(response.data.error?.message || t('users.messages.assignRolesError'))
      }
    } catch {
      Toast.error(t('users.messages.assignRolesError'))
    }
  }, [roleAssignUser, selectedRoleIds, fetchUsers, t])

  // Handle bulk activate
  const handleBulkActivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => activateUser(id, {})))
      Toast.success(t('users.messages.batchActivateSuccess', { count: selectedRowKeys.length }))
      setSelectedRowKeys([])
      fetchUsers()
    } catch {
      Toast.error(t('users.messages.batchActivateError'))
    }
  }, [selectedRowKeys, fetchUsers, t])

  // Handle bulk deactivate
  const handleBulkDeactivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => deactivateUser(id, {})))
      Toast.success(t('users.messages.batchDeactivateSuccess', { count: selectedRowKeys.length }))
      setSelectedRowKeys([])
      fetchUsers()
    } catch {
      Toast.error(t('users.messages.batchDeactivateError'))
    }
  }, [selectedRowKeys, fetchUsers, t])

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
        title: t('users.columns.username'),
        dataIndex: 'username',
        width: 140,
        sortable: true,
        render: (username: unknown) => (
          <span className="user-username">{(username as string) || '-'}</span>
        ),
      },
      {
        title: t('users.columns.displayName'),
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
        title: t('users.columns.contact'),
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
        title: t('users.columns.role'),
        dataIndex: 'role_ids',
        width: 200,
        render: (roleIds: unknown) => {
          const ids = roleIds as string[] | undefined
          if (!ids || ids.length === 0) {
            return <Text type="tertiary">{t('users.noRole')}</Text>
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
        title: t('users.columns.status'),
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as UserStatus | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>{t(`users.status.${statusValue}`)}</Tag>
          )
        },
      },
      {
        title: t('users.columns.lastLogin'),
        dataIndex: 'last_login_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, i18n.language),
      },
      {
        title: t('users.columns.createdAt'),
        dataIndex: 'created_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, i18n.language),
      },
    ],
    [getRoleName, t, i18n.language, STATUS_TAG_COLORS]
  )

  // Table row actions
  const tableActions: TableAction<UserRow>[] = useMemo(
    () => [
      {
        key: 'edit',
        label: t('users.actions.edit'),
        onClick: handleEdit,
      },
      {
        key: 'roles',
        label: t('users.actions.assignRoles'),
        onClick: handleAssignRoles,
      },
      {
        key: 'reset-password',
        label: t('users.actions.resetPassword'),
        icon: <IconKey size="small" />,
        onClick: handleResetPassword,
      },
      {
        key: 'activate',
        label: t('users.actions.activate'),
        type: 'primary',
        onClick: handleActivate,
        hidden: (record) => record.status === 'active',
      },
      {
        key: 'deactivate',
        label: t('users.actions.deactivate'),
        type: 'warning',
        onClick: handleDeactivate,
        hidden: (record) => record.status === 'deactivated',
      },
      {
        key: 'lock',
        label: t('users.actions.lock'),
        type: 'warning',
        icon: <IconLock size="small" />,
        onClick: handleLock,
        hidden: (record) => record.status === 'locked',
      },
      {
        key: 'unlock',
        label: t('users.actions.unlock'),
        icon: <IconUnlock size="small" />,
        onClick: handleUnlock,
        hidden: (record) => record.status !== 'locked',
      },
      {
        key: 'delete',
        label: t('users.actions.delete'),
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
      t,
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
    const options = [{ label: t('users.allRoles'), value: '' }]
    roles.forEach((role) => {
      options.push({ label: role.name, value: role.id })
    })
    return options
  }, [roles, t])

  return (
    <Container size="full" className="users-page">
      <Card className="users-card">
        <div className="users-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('users.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('users.searchPlaceholder')}
          primaryAction={{
            label: t('users.addUser'),
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
            <Space className="users-filter-container">
              <Select
                placeholder={t('users.statusFilter')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('users.roleFilter')}
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
              {t('users.actions.batchActivate')}
            </Tag>
            <Tag color="orange" onClick={handleBulkDeactivate} style={{ cursor: 'pointer' }}>
              {t('users.actions.batchDeactivate')}
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
        title={modalMode === 'create' ? t('users.addUser') : t('users.editUser')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleModalSubmit}
        confirmLoading={modalLoading}
        width={600}
        okText={modalMode === 'create' ? t('common.create') : t('common.save')}
        cancelText={t('common.cancel')}
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
            label={t('users.form.username')}
            placeholder={t('users.form.usernamePlaceholder')}
            rules={[
              { required: true, message: t('users.form.usernamePlaceholder') },
              { min: 3, message: t('users.form.usernameMinError') },
              { max: 100, message: t('users.form.usernameMaxError') },
            ]}
            disabled={modalMode === 'edit'}
          />
          {modalMode === 'create' && (
            <Form.Input
              field="password"
              label={t('users.form.password')}
              mode="password"
              placeholder={t('users.form.passwordPlaceholder')}
              rules={[
                { required: true, message: t('users.form.passwordPlaceholder') },
                { min: 8, message: t('users.form.passwordMinError') },
                { max: 128, message: t('users.form.passwordMaxError') },
              ]}
              extraText={
                <Button
                  theme="borderless"
                  size="small"
                  onClick={() => {
                    formApiRef.current?.setValue('password', generatePassword())
                  }}
                >
                  {t('users.form.generatePassword')}
                </Button>
              }
            />
          )}
          <Form.Input
            field="display_name"
            label={t('users.form.displayName')}
            placeholder={t('users.form.displayNamePlaceholder')}
          />
          <Form.Input
            field="email"
            label={t('users.form.email')}
            placeholder={t('users.form.emailPlaceholder')}
            rules={[{ type: 'email', message: t('users.form.emailError') }]}
          />
          <Form.Input
            field="phone"
            label={t('users.form.phone')}
            placeholder={t('users.form.phonePlaceholder')}
          />
          {modalMode === 'create' && (
            <Form.CheckboxGroup
              field="role_ids"
              label={t('users.form.assignRoles')}
              direction="horizontal"
            >
              {roles.map((role) => (
                <Checkbox key={role.id} value={role.id}>
                  {role.name}
                </Checkbox>
              ))}
            </Form.CheckboxGroup>
          )}
          <Form.TextArea
            field="notes"
            label={t('users.form.notes')}
            placeholder={t('users.form.notesPlaceholder')}
            rows={3}
          />
        </Form>
      </Modal>

      {/* Reset Password Modal */}
      <Modal
        title={t('users.resetPassword.title')}
        visible={resetPasswordVisible}
        onCancel={() => setResetPasswordVisible(false)}
        onOk={handleSubmitResetPassword}
        okText={t('users.resetPassword.confirmReset')}
        cancelText={t('common.cancel')}
        width={500}
      >
        <div style={{ marginBottom: 16 }}>
          <Text>
            {t('users.resetPassword.description', {
              name: resetPasswordUser?.display_name || resetPasswordUser?.username,
            })}
          </Text>
        </div>
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary">{t('users.resetPassword.newPassword')}</Text>
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
                {t('users.resetPassword.regenerate')}
              </Button>
            }
          />
        </div>
        <div>
          <Text type="warning">{t('users.resetPassword.warning')}</Text>
        </div>
      </Modal>

      {/* Role Assignment Modal */}
      <Modal
        title={t('users.roleAssignment.title')}
        visible={roleModalVisible}
        onCancel={() => setRoleModalVisible(false)}
        onOk={handleSubmitRoles}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        width={500}
      >
        <div style={{ marginBottom: 16 }}>
          <Text>
            {t('users.roleAssignment.description', {
              name: roleAssignUser?.display_name || roleAssignUser?.username,
            })}
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
                      {t('users.roleAssignment.systemRole')}
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
