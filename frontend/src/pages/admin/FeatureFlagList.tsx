import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
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
  Switch,
  Form,
  TagInput,
  Tooltip,
} from '@douyinfe/semi-ui-19'
import type { FormApi } from '@douyinfe/semi-ui-19/lib/es/form/interface'
import { IconPlus, IconRefresh, IconEdit, IconDelete } from '@douyinfe/semi-icons'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import {
  listFeatureFlagFlags,
  createFeatureFlagFlag,
  enableFlagFeatureFlag,
  disableFlagFeatureFlag,
  archiveFlagFeatureFlag,
} from '@/api/feature-flags/feature-flags'
import type {
  DtoFlagResponse,
  ListFeatureFlagFlagsParams,
  HandlerCreateFlagHTTPRequestType,
  ListFeatureFlagFlagsStatus,
  CreateFeatureFlagFlagBody,
} from '@/api/models'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import './FeatureFlagList.css'

const { Title, Text } = Typography

// Type aliases for cleaner code
type FlagType = HandlerCreateFlagHTTPRequestType
type FlagStatus = ListFeatureFlagFlagsStatus
type FeatureFlag = DtoFlagResponse

// Feature Flag row type with index signature for DataTable compatibility
type FlagRow = FeatureFlag & Record<string, unknown>

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
 * Get color for flag type badge
 */
function getTypeColor(type: FlagType): TagColor {
  switch (type) {
    case 'boolean':
      return 'blue'
    case 'percentage':
      return 'orange'
    case 'variant':
      return 'purple'
    case 'user_segment':
      return 'cyan'
    default:
      return 'grey'
  }
}

/**
 * Get color for flag status
 */
function getStatusColor(status: FlagStatus): TagColor {
  switch (status) {
    case 'enabled':
      return 'green'
    case 'disabled':
      return 'grey'
    case 'archived':
      return 'red'
    default:
      return 'grey'
  }
}

/**
 * Feature Flag List Page
 *
 * Admin interface for managing feature flags.
 *
 * Features:
 * - Paginated list of feature flags
 * - Search by key/name
 * - Filter by status and type
 * - Filter by tags
 * - Toggle enable/disable status
 * - Create new flags
 * - Edit existing flags
 * - Archive flags
 */
export default function FeatureFlagListPage() {
  const { t, i18n } = useTranslation('admin')
  const navigate = useNavigate()

  // Status options for filter
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('featureFlags.allStatus', '全部状态'), value: '' },
      { label: t('featureFlags.status.enabled', '已启用'), value: 'enabled' },
      { label: t('featureFlags.status.disabled', '已禁用'), value: 'disabled' },
      { label: t('featureFlags.status.archived', '已归档'), value: 'archived' },
    ],
    [t]
  )

  // Type options for filter
  const TYPE_OPTIONS = useMemo(
    () => [
      { label: t('featureFlags.allTypes', '全部类型'), value: '' },
      { label: t('featureFlags.type.boolean', '布尔型'), value: 'boolean' },
      { label: t('featureFlags.type.percentage', '百分比'), value: 'percentage' },
      { label: t('featureFlags.type.variant', '多变体'), value: 'variant' },
      { label: t('featureFlags.type.user_segment', '用户分群'), value: 'user_segment' },
    ],
    [t]
  )

  // State for data
  const [flagList, setFlagList] = useState<FlagRow[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [tagsFilter, setTagsFilter] = useState<string[]>([])

  // Create modal state
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [createLoading, setCreateLoading] = useState(false)
  const [formApiRef, setFormApiRef] = useState<FormApi | null>(null)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'updated_at',
    defaultSortOrder: 'desc',
  })

  // Fetch feature flags
  const fetchFlags = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListFeatureFlagFlagsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter as FlagStatus) || undefined,
        type: (typeFilter as FlagType) || undefined,
        tags: tagsFilter.length > 0 ? tagsFilter.join(',') : undefined,
      }

      const response = await listFeatureFlagFlags(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setFlagList((response.data.data.flags || []) as FlagRow[])
        setTotal(response.data.data.total || 0)
      }
    } catch {
      Toast.error(t('featureFlags.messages.fetchError', '获取功能开关列表失败'))
    } finally {
      setLoading(false)
    }
  }, [
    state.pagination.page,
    state.pagination.pageSize,
    searchKeyword,
    statusFilter,
    typeFilter,
    tagsFilter,
    t,
  ])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchFlags()
  }, [fetchFlags])

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

  // Handle type filter change
  const handleTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const typeValue = typeof value === 'string' ? value : ''
      setTypeFilter(typeValue)
      setFilter('type', typeValue || null)
    },
    [setFilter]
  )

  // Handle tags filter change
  const handleTagsChange = useCallback(
    (value: string[] | undefined) => {
      setTagsFilter(value || [])
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle create flag
  const handleCreate = useCallback(() => {
    setCreateModalVisible(true)
  }, [])

  // Handle create modal submit
  const handleCreateSubmit = useCallback(async () => {
    if (!formApiRef) return

    try {
      await formApiRef.validate()
      const values = formApiRef.getValues()
      setCreateLoading(true)

      const request: CreateFeatureFlagFlagBody = {
        key: values.key,
        name: values.name,
        description: values.description || undefined,
        type: values.type || 'boolean',
        default_value: {
          enabled: false,
          variant: undefined,
        },
        tags: values.tags || undefined,
      }

      const response = await createFeatureFlagFlag(request)
      if (response.status === 201 && response.data.success) {
        Toast.success(t('featureFlags.messages.createSuccess', '功能开关创建成功'))
        setCreateModalVisible(false)
        fetchFlags()
      } else {
        const error = response.data.error as { message?: string } | undefined
        Toast.error(error?.message || t('featureFlags.messages.createError', '创建功能开关失败'))
      }
    } catch {
      // Validation failed or API error
    } finally {
      setCreateLoading(false)
    }
  }, [formApiRef, fetchFlags, t])

  // Handle toggle status (enable/disable)
  const handleToggleStatus = useCallback(
    async (flag: FlagRow, checked: boolean) => {
      try {
        const response = checked
          ? await enableFlagFeatureFlag(flag.key || '', {})
          : await disableFlagFeatureFlag(flag.key || '', {})

        if (response.status === 200 && response.data.success) {
          Toast.success(
            checked
              ? t('featureFlags.messages.enableSuccess', '功能开关已启用')
              : t('featureFlags.messages.disableSuccess', '功能开关已禁用')
          )
          fetchFlags()
        } else {
          const error = response.data.error as { message?: string } | undefined
          Toast.error(
            error?.message ||
              (checked
                ? t('featureFlags.messages.enableError', '启用功能开关失败')
                : t('featureFlags.messages.disableError', '禁用功能开关失败'))
          )
        }
      } catch {
        Toast.error(
          checked
            ? t('featureFlags.messages.enableError', '启用功能开关失败')
            : t('featureFlags.messages.disableError', '禁用功能开关失败')
        )
      }
    },
    [fetchFlags, t]
  )

  // Handle view flag detail
  const handleViewDetail = useCallback(
    (flag: FlagRow) => {
      navigate(`/admin/feature-flags/${flag.key}`)
    },
    [navigate]
  )

  // Handle edit flag
  const handleEdit = useCallback(
    (flag: FlagRow) => {
      navigate(`/admin/feature-flags/${flag.key}/edit`)
    },
    [navigate]
  )

  // Handle archive flag
  const handleArchive = useCallback(
    async (flag: FlagRow) => {
      if (flag.status === 'archived') {
        Toast.warning(t('featureFlags.messages.alreadyArchived', '该功能开关已归档'))
        return
      }

      Modal.confirm({
        title: t('featureFlags.confirm.archiveTitle', '确认归档'),
        content: t(
          'featureFlags.confirm.archiveContent',
          '确定要归档功能开关 "{{key}}" 吗？归档后将无法被评估。',
          {
            key: flag.key,
          }
        ),
        okText: t('featureFlags.confirm.archiveOk', '确认归档'),
        cancelText: t('common.cancel', '取消'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            const response = await archiveFlagFeatureFlag(flag.key || '')
            if (response.status === 204) {
              Toast.success(t('featureFlags.messages.archiveSuccess', '功能开关已归档'))
              fetchFlags()
            } else {
              Toast.error(t('featureFlags.messages.archiveError', '归档功能开关失败'))
            }
          } catch {
            Toast.error(t('featureFlags.messages.archiveError', '归档功能开关失败'))
          }
        },
      })
    },
    [fetchFlags, t]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchFlags()
  }, [fetchFlags])

  // Table columns
  const tableColumns: DataTableColumn<FlagRow>[] = useMemo(
    () => [
      {
        title: t('featureFlags.columns.key', 'Key'),
        dataIndex: 'key',
        width: 200,
        render: (key: unknown, record: FlagRow) => (
          <Tooltip content={t('featureFlags.clickToViewDetail', '点击查看详情')}>
            <a
              onClick={() => handleViewDetail(record)}
              className="flag-key-link"
              style={{ cursor: 'pointer', color: 'var(--semi-color-primary)' }}
            >
              {key as string}
            </a>
          </Tooltip>
        ),
      },
      {
        title: t('featureFlags.columns.name', '名称'),
        dataIndex: 'name',
        ellipsis: true,
        render: (name: unknown) => <span className="flag-name">{name as string}</span>,
      },
      {
        title: t('featureFlags.columns.type', '类型'),
        dataIndex: 'type',
        width: 100,
        align: 'center',
        render: (type: unknown) => {
          const flagType = type as FlagType
          return (
            <Tag color={getTypeColor(flagType)} size="small">
              {t(`featureFlags.type.${flagType}`, flagType)}
            </Tag>
          )
        },
      },
      {
        title: t('featureFlags.columns.status', '状态'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown, record: FlagRow) => {
          const flagStatus = status as FlagStatus
          if (flagStatus === 'archived') {
            return (
              <Tag color={getStatusColor(flagStatus)} size="small">
                {t('featureFlags.status.archived', '已归档')}
              </Tag>
            )
          }
          return (
            <Switch
              checked={flagStatus === 'enabled'}
              onChange={(checked) => handleToggleStatus(record, checked)}
              size="small"
              aria-label={t('featureFlags.toggleStatus', '切换状态')}
            />
          )
        },
      },
      {
        title: t('featureFlags.columns.tags', '标签'),
        dataIndex: 'tags',
        width: 200,
        render: (tags: unknown) => {
          const tagList = tags as string[] | undefined
          if (!tagList || tagList.length === 0) {
            return <Text type="tertiary">-</Text>
          }
          return (
            <Space wrap>
              {tagList.slice(0, 3).map((tag) => (
                <Tag key={tag} size="small" color="light-blue">
                  {tag}
                </Tag>
              ))}
              {tagList.length > 3 && (
                <Tooltip content={tagList.slice(3).join(', ')}>
                  <Tag size="small" color="light-blue">
                    +{tagList.length - 3}
                  </Tag>
                </Tooltip>
              )}
            </Space>
          )
        },
      },
      {
        title: t('featureFlags.columns.updatedAt', '更新时间'),
        dataIndex: 'updated_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, i18n.language),
      },
    ],
    [t, i18n.language, handleViewDetail, handleToggleStatus]
  )

  // Table row actions
  const tableActions: TableAction<FlagRow>[] = useMemo(
    () => [
      {
        key: 'edit',
        label: t('featureFlags.actions.edit', '编辑'),
        icon: <IconEdit size="small" />,
        onClick: handleEdit,
        hidden: (record) => record.status === 'archived',
      },
      {
        key: 'archive',
        label: t('featureFlags.actions.archive', '归档'),
        icon: <IconDelete size="small" />,
        type: 'danger',
        onClick: handleArchive,
        hidden: (record) => record.status === 'archived',
      },
    ],
    [t, handleEdit, handleArchive]
  )

  return (
    <Container size="full" className="feature-flag-list-page">
      <Card className="feature-flag-list-card">
        <div className="feature-flag-list-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('featureFlags.title', '功能开关管理')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('featureFlags.searchPlaceholder', '搜索 Key 或名称...')}
          primaryAction={{
            label: t('featureFlags.addFlag', '新建开关'),
            icon: <IconPlus />,
            onClick: handleCreate,
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('common.refresh', '刷新'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="feature-flag-filter-container">
              <Select
                placeholder={t('featureFlags.statusFilter', '状态筛选')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('featureFlags.typeFilter', '类型筛选')}
                value={typeFilter}
                onChange={handleTypeChange}
                optionList={TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              <TagInput
                placeholder={t('featureFlags.tagsFilter', '标签筛选')}
                value={tagsFilter}
                onChange={handleTagsChange}
                style={{ width: 200 }}
                maxTagCount={2}
                showRestTagsPopover
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<FlagRow>
            data={flagList}
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
            scroll={{ x: 1000 }}
          />
        </Spin>
      </Card>

      {/* Create Feature Flag Modal */}
      <Modal
        title={t('featureFlags.createModal.title', '新建功能开关')}
        visible={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        onOk={handleCreateSubmit}
        confirmLoading={createLoading}
        width={500}
        okText={t('common.create', '创建')}
        cancelText={t('common.cancel', '取消')}
      >
        <Form
          getFormApi={(api) => setFormApiRef(api)}
          initValues={{ type: 'boolean' }}
          labelPosition="left"
          labelWidth={80}
        >
          <Form.Input
            field="key"
            label={t('featureFlags.form.key', 'Key')}
            placeholder={t(
              'featureFlags.form.keyPlaceholder',
              '请输入功能开关 Key，如 new_checkout_flow'
            )}
            rules={[
              { required: true, message: t('featureFlags.form.keyRequired', '请输入 Key') },
              { min: 2, message: t('featureFlags.form.keyMinError', 'Key 至少2个字符') },
              { max: 100, message: t('featureFlags.form.keyMaxError', 'Key 最多100个字符') },
              {
                pattern: /^[a-z][a-z0-9_]*$/,
                message: t(
                  'featureFlags.form.keyRegexError',
                  'Key 必须以小写字母开头，只能包含小写字母、数字和下划线'
                ),
              },
            ]}
          />
          <Form.Input
            field="name"
            label={t('featureFlags.form.name', '名称')}
            placeholder={t('featureFlags.form.namePlaceholder', '请输入功能开关名称')}
            rules={[
              { required: true, message: t('featureFlags.form.nameRequired', '请输入名称') },
              { min: 2, message: t('featureFlags.form.nameMinError', '名称至少2个字符') },
              { max: 200, message: t('featureFlags.form.nameMaxError', '名称最多200个字符') },
            ]}
          />
          <Form.TextArea
            field="description"
            label={t('featureFlags.form.description', '描述')}
            placeholder={t(
              'featureFlags.form.descriptionPlaceholder',
              '请输入功能开关描述（可选）'
            )}
            rows={3}
            maxLength={500}
          />
          <Form.Select
            field="type"
            label={t('featureFlags.form.type', '类型')}
            placeholder={t('featureFlags.form.typePlaceholder', '请选择类型')}
            optionList={[
              { label: t('featureFlags.type.boolean', '布尔型'), value: 'boolean' },
              { label: t('featureFlags.type.percentage', '百分比'), value: 'percentage' },
              { label: t('featureFlags.type.variant', '多变体'), value: 'variant' },
              { label: t('featureFlags.type.user_segment', '用户分群'), value: 'user_segment' },
            ]}
            rules={[{ required: true, message: t('featureFlags.form.typeRequired', '请选择类型') }]}
          />
          <Form.TagInput
            field="tags"
            label={t('featureFlags.form.tags', '标签')}
            placeholder={t('featureFlags.form.tagsPlaceholder', '输入标签后按回车')}
          />
        </Form>
      </Modal>
    </Container>
  )
}
