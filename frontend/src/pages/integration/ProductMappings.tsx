import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Toast,
  Select,
  Space,
  Modal,
  Spin,
  Tag,
  Form,
  Checkbox,
  Empty,
  Popconfirm,
} from '@douyinfe/semi-ui-19'
import {
  IconPlus,
  IconRefresh,
  IconLink,
  IconUnlink,
  IconSearch,
  IconDelete,
} from '@douyinfe/semi-icons'
import {
  DataTable,
  TableToolbar,
  BulkActionBar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import type { PaginationMeta } from '@/types/api'
import './ProductMappings.css'

const { Title, Text } = Typography

/**
 * Platform codes matching backend domain model
 */
type PlatformCode = 'TAOBAO' | 'DOUYIN' | 'JD' | 'PDD' | 'WECHAT' | 'KUAISHOU'

/**
 * Product mapping entity interface
 */
interface ProductMapping {
  id: string
  localProductId: string
  localProductCode: string
  localProductName: string
  platformCode: PlatformCode
  platformProductId: string
  platformSku: string
  platformProductName?: string
  syncPrice: boolean
  syncInventory: boolean
  lastSyncAt?: string
  createdAt: string
  updatedAt: string
}

/**
 * Local product for mapping selection
 */
interface LocalProduct {
  id: string
  code: string
  name: string
  unit: string
  sellingPrice?: number
}

/**
 * Platform product for mapping
 */
interface PlatformProduct {
  id: string
  sku: string
  title: string
  price?: number
}

/**
 * Platform metadata for display
 */
interface PlatformMeta {
  code: PlatformCode
  name: string
  icon: string
  color: string
}

/**
 * Available platforms with metadata
 */
const PLATFORMS: PlatformMeta[] = [
  { code: 'TAOBAO', name: '淘宝/天猫', icon: 'TB', color: '#FF5000' },
  { code: 'DOUYIN', name: '抖音', icon: 'DY', color: '#000000' },
  { code: 'JD', name: '京东', icon: 'JD', color: '#E2231A' },
  { code: 'PDD', name: '拼多多', icon: 'PDD', color: '#E02E24' },
  { code: 'WECHAT', name: '微信小商店', icon: 'WX', color: '#07C160' },
  { code: 'KUAISHOU', name: '快手', icon: 'KS', color: '#FF4906' },
]

/**
 * Extended type for DataTable compatibility
 */
type MappingRow = ProductMapping & Record<string, unknown>

/**
 * Product Mapping Management Page
 *
 * Features:
 * - Display mapping list with pagination
 * - Filter by platform and sync status
 * - Manual mapping: select local product + platform product
 * - Batch mapping: map multiple products at once
 * - Edit/delete existing mappings
 */
export default function ProductMappingsPage() {
  const { t } = useTranslation(['integration', 'common'])
  const { formatDate } = useFormatters()

  // Data states
  const [mappings, setMappings] = useState<MappingRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // Filter states
  const [searchKeyword, setSearchKeyword] = useState('')
  const [platformFilter, setPlatformFilter] = useState<string>('')
  const [syncStatusFilter, setSyncStatusFilter] = useState<string>('')

  // Modal states
  const [mappingModalVisible, setMappingModalVisible] = useState(false)
  const [batchMappingModalVisible, setBatchMappingModalVisible] = useState(false)
  const [editingMapping, setEditingMapping] = useState<ProductMapping | null>(null)

  // Form states for manual mapping
  const [selectedLocalProduct, setSelectedLocalProduct] = useState<LocalProduct | null>(null)
  const [selectedPlatformProduct, setSelectedPlatformProduct] = useState<PlatformProduct | null>(
    null
  )
  const [selectedPlatform, setSelectedPlatform] = useState<PlatformCode>('TAOBAO')
  const [syncPriceEnabled, setSyncPriceEnabled] = useState(true)
  const [syncInventoryEnabled, setSyncInventoryEnabled] = useState(true)

  // Batch mapping states
  const [batchPlatform, setBatchPlatform] = useState<PlatformCode>('TAOBAO')
  const [batchSyncPrice, setBatchSyncPrice] = useState(true)
  const [batchSyncInventory, setBatchSyncInventory] = useState(true)
  const [unmappedProducts, setUnmappedProducts] = useState<LocalProduct[]>([])
  const [selectedUnmappedKeys, setSelectedUnmappedKeys] = useState<string[]>([])

  // Search states for autocomplete
  const [localProductOptions, setLocalProductOptions] = useState<LocalProduct[]>([])
  const [platformProductOptions, setPlatformProductOptions] = useState<PlatformProduct[]>([])
  const [searchingLocal, setSearchingLocal] = useState(false)
  const [searchingPlatform, setSearchingPlatform] = useState(false)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  /**
   * Fetch product mappings from backend
   */
  const fetchMappings = useCallback(async () => {
    setLoading(true)
    try {
      // TODO: Replace with actual API call
      // const response = await api.getProductMappings(params)
      await new Promise((resolve) => setTimeout(resolve, 500))

      // Mock data for demo
      const mockMappings: MappingRow[] = [
        {
          id: '1',
          localProductId: 'p001',
          localProductCode: 'SKU-001',
          localProductName: '示例商品A',
          platformCode: 'TAOBAO',
          platformProductId: 'tb-001',
          platformSku: 'TB-SKU-001',
          platformProductName: '淘宝商品A',
          syncPrice: true,
          syncInventory: true,
          lastSyncAt: new Date(Date.now() - 3600000).toISOString(),
          createdAt: new Date(Date.now() - 86400000 * 7).toISOString(),
          updatedAt: new Date().toISOString(),
        },
        {
          id: '2',
          localProductId: 'p002',
          localProductCode: 'SKU-002',
          localProductName: '示例商品B',
          platformCode: 'JD',
          platformProductId: 'jd-001',
          platformSku: 'JD-SKU-001',
          platformProductName: '京东商品B',
          syncPrice: true,
          syncInventory: false,
          lastSyncAt: new Date(Date.now() - 7200000).toISOString(),
          createdAt: new Date(Date.now() - 86400000 * 5).toISOString(),
          updatedAt: new Date().toISOString(),
        },
        {
          id: '3',
          localProductId: 'p003',
          localProductCode: 'SKU-003',
          localProductName: '示例商品C',
          platformCode: 'DOUYIN',
          platformProductId: 'dy-001',
          platformSku: 'DY-SKU-001',
          platformProductName: '抖音商品C',
          syncPrice: false,
          syncInventory: true,
          createdAt: new Date(Date.now() - 86400000 * 3).toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ]

      // Apply filters
      let filtered = mockMappings
      if (searchKeyword) {
        const keyword = searchKeyword.toLowerCase()
        filtered = filtered.filter(
          (m) =>
            m.localProductCode.toLowerCase().includes(keyword) ||
            m.localProductName.toLowerCase().includes(keyword) ||
            m.platformProductId.toLowerCase().includes(keyword) ||
            m.platformSku.toLowerCase().includes(keyword)
        )
      }
      if (platformFilter) {
        filtered = filtered.filter((m) => m.platformCode === platformFilter)
      }

      setMappings(filtered)
      setPaginationMeta({
        page: 1,
        page_size: 20,
        total: filtered.length,
        total_pages: 1,
      })
    } catch {
      Toast.error(t('productMappings.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [searchKeyword, platformFilter, t])

  // Fetch on mount and when filters change
  useEffect(() => {
    fetchMappings()
  }, [fetchMappings])

  /**
   * Search local products for autocomplete
   */
  const searchLocalProducts = useCallback(
    async (keyword: string) => {
      if (!keyword || keyword.length < 2) {
        setLocalProductOptions([])
        return
      }

      setSearchingLocal(true)
      try {
        // TODO: Replace with actual API call
        // const response = await api.searchProducts({ keyword })
        await new Promise((resolve) => setTimeout(resolve, 300))

        // Mock data
        const mockProducts: LocalProduct[] = [
          { id: 'p001', code: 'SKU-001', name: '示例商品A', unit: '件', sellingPrice: 99.0 },
          { id: 'p002', code: 'SKU-002', name: '示例商品B', unit: '件', sellingPrice: 199.0 },
          { id: 'p003', code: 'SKU-003', name: '示例商品C', unit: '个', sellingPrice: 299.0 },
          { id: 'p004', code: 'SKU-004', name: '测试商品D', unit: '件', sellingPrice: 399.0 },
          { id: 'p005', code: 'SKU-005', name: '测试商品E', unit: '件', sellingPrice: 499.0 },
        ]

        const filtered = mockProducts.filter(
          (p) =>
            p.code.toLowerCase().includes(keyword.toLowerCase()) ||
            p.name.toLowerCase().includes(keyword.toLowerCase())
        )

        setLocalProductOptions(filtered)
      } catch {
        Toast.error(t('productMappings.messages.searchLocalError'))
      } finally {
        setSearchingLocal(false)
      }
    },
    [t]
  )

  /**
   * Search platform products for autocomplete
   */
  const searchPlatformProducts = useCallback(
    async (keyword: string, platform: PlatformCode) => {
      if (!keyword || keyword.length < 2) {
        setPlatformProductOptions([])
        return
      }

      setSearchingPlatform(true)
      try {
        // TODO: Replace with actual API call to fetch products from platform
        // const response = await api.searchPlatformProducts({ platform, keyword })
        await new Promise((resolve) => setTimeout(resolve, 300))

        // Mock data based on platform
        const prefix = platform.toLowerCase().slice(0, 2)
        const mockProducts: PlatformProduct[] = [
          {
            id: `${prefix}-001`,
            sku: `${prefix.toUpperCase()}-SKU-001`,
            title: `${platform}商品1`,
          },
          {
            id: `${prefix}-002`,
            sku: `${prefix.toUpperCase()}-SKU-002`,
            title: `${platform}商品2`,
          },
          {
            id: `${prefix}-003`,
            sku: `${prefix.toUpperCase()}-SKU-003`,
            title: `${platform}商品3`,
          },
        ]

        const filtered = mockProducts.filter(
          (p) =>
            p.sku.toLowerCase().includes(keyword.toLowerCase()) ||
            p.title.toLowerCase().includes(keyword.toLowerCase())
        )

        setPlatformProductOptions(filtered)
      } catch {
        Toast.error(t('productMappings.messages.searchPlatformError'))
      } finally {
        setSearchingPlatform(false)
      }
    },
    [t]
  )

  /**
   * Fetch unmapped products for batch mapping
   */
  const fetchUnmappedProducts = useCallback(
    async (_platform: PlatformCode) => {
      try {
        // TODO: Replace with actual API call
        // const response = await api.getUnmappedProducts({ platform: _platform })
        await new Promise((resolve) => setTimeout(resolve, 300))

        // Mock unmapped products
        const mockProducts: LocalProduct[] = [
          { id: 'p006', code: 'SKU-006', name: '未映射商品A', unit: '件', sellingPrice: 599.0 },
          { id: 'p007', code: 'SKU-007', name: '未映射商品B', unit: '件', sellingPrice: 699.0 },
          { id: 'p008', code: 'SKU-008', name: '未映射商品C', unit: '个', sellingPrice: 799.0 },
        ]

        setUnmappedProducts(mockProducts)
      } catch {
        Toast.error(t('productMappings.messages.fetchUnmappedError'))
      }
    },
    [t]
  )

  /**
   * Handle search
   */
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  /**
   * Handle platform filter change
   */
  const handlePlatformFilterChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const platformValue = typeof value === 'string' ? value : ''
      setPlatformFilter(platformValue)
      setFilter('platform', platformValue || null)
    },
    [setFilter]
  )

  /**
   * Handle sync status filter change
   */
  const handleSyncStatusFilterChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const statusValue = typeof value === 'string' ? value : ''
      setSyncStatusFilter(statusValue)
      setFilter('syncStatus', statusValue || null)
    },
    [setFilter]
  )

  /**
   * Open manual mapping modal (create or edit)
   */
  const openMappingModal = useCallback((mapping?: ProductMapping) => {
    if (mapping) {
      setEditingMapping(mapping)
      setSelectedPlatform(mapping.platformCode)
      setSelectedLocalProduct({
        id: mapping.localProductId,
        code: mapping.localProductCode,
        name: mapping.localProductName,
        unit: '',
      })
      setSelectedPlatformProduct({
        id: mapping.platformProductId,
        sku: mapping.platformSku,
        title: mapping.platformProductName || '',
      })
      setSyncPriceEnabled(mapping.syncPrice)
      setSyncInventoryEnabled(mapping.syncInventory)
    } else {
      setEditingMapping(null)
      setSelectedLocalProduct(null)
      setSelectedPlatformProduct(null)
      setSyncPriceEnabled(true)
      setSyncInventoryEnabled(true)
    }
    setMappingModalVisible(true)
  }, [])

  /**
   * Close manual mapping modal
   */
  const closeMappingModal = useCallback(() => {
    setMappingModalVisible(false)
    setEditingMapping(null)
    setSelectedLocalProduct(null)
    setSelectedPlatformProduct(null)
    setLocalProductOptions([])
    setPlatformProductOptions([])
  }, [])

  /**
   * Save manual mapping
   */
  const handleSaveMapping = useCallback(async () => {
    if (!selectedLocalProduct) {
      Toast.error(t('productMappings.errors.localProductRequired'))
      return
    }
    if (!selectedPlatformProduct) {
      Toast.error(t('productMappings.errors.platformProductRequired'))
      return
    }

    try {
      // TODO: Replace with actual API call
      // await api.createOrUpdateProductMapping({
      //   id: editingMapping?.id,
      //   localProductId: selectedLocalProduct.id,
      //   platformCode: selectedPlatform,
      //   platformProductId: selectedPlatformProduct.id,
      //   platformSku: selectedPlatformProduct.sku,
      //   syncPrice: syncPriceEnabled,
      //   syncInventory: syncInventoryEnabled,
      // })
      await new Promise((resolve) => setTimeout(resolve, 500))

      Toast.success(
        editingMapping
          ? t('productMappings.messages.updateSuccess')
          : t('productMappings.messages.createSuccess')
      )
      closeMappingModal()
      fetchMappings()
    } catch {
      Toast.error(t('productMappings.messages.saveError'))
    }
  }, [
    selectedLocalProduct,
    selectedPlatformProduct,
    selectedPlatform,
    syncPriceEnabled,
    syncInventoryEnabled,
    editingMapping,
    closeMappingModal,
    fetchMappings,
    t,
  ])

  /**
   * Open batch mapping modal
   */
  const openBatchMappingModal = useCallback(() => {
    setBatchMappingModalVisible(true)
    setBatchSyncPrice(true)
    setBatchSyncInventory(true)
    setSelectedUnmappedKeys([])
    fetchUnmappedProducts(batchPlatform)
  }, [batchPlatform, fetchUnmappedProducts])

  /**
   * Close batch mapping modal
   */
  const closeBatchMappingModal = useCallback(() => {
    setBatchMappingModalVisible(false)
    setUnmappedProducts([])
    setSelectedUnmappedKeys([])
  }, [])

  /**
   * Handle batch platform change
   */
  const handleBatchPlatformChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const platformValue = typeof value === 'string' ? (value as PlatformCode) : 'TAOBAO'
      setBatchPlatform(platformValue)
      setSelectedUnmappedKeys([])
      fetchUnmappedProducts(platformValue)
    },
    [fetchUnmappedProducts]
  )

  /**
   * Execute batch mapping
   */
  const handleBatchMapping = useCallback(async () => {
    if (selectedUnmappedKeys.length === 0) {
      Toast.error(t('productMappings.errors.noProductSelected'))
      return
    }

    try {
      // TODO: Replace with actual API call
      // await api.batchCreateMappings({
      //   productIds: selectedUnmappedKeys,
      //   platformCode: batchPlatform,
      //   syncPrice: batchSyncPrice,
      //   syncInventory: batchSyncInventory,
      // })
      await new Promise((resolve) => setTimeout(resolve, 500))

      Toast.success(
        t('productMappings.messages.batchCreateSuccess', { count: selectedUnmappedKeys.length })
      )
      closeBatchMappingModal()
      fetchMappings()
    } catch {
      Toast.error(t('productMappings.messages.batchCreateError'))
    }
  }, [
    selectedUnmappedKeys,
    batchPlatform,
    batchSyncPrice,
    batchSyncInventory,
    closeBatchMappingModal,
    fetchMappings,
    t,
  ])

  /**
   * Delete a mapping
   */
  const handleDeleteMapping = useCallback(
    async (_mapping: MappingRow) => {
      try {
        // TODO: Replace with actual API call
        // await api.deleteProductMapping(_mapping.id)
        await new Promise((resolve) => setTimeout(resolve, 300))

        Toast.success(t('productMappings.messages.deleteSuccess'))
        fetchMappings()
      } catch {
        Toast.error(t('productMappings.messages.deleteError'))
      }
    },
    [fetchMappings, t]
  )

  /**
   * Bulk delete mappings
   */
  const handleBulkDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) return

    try {
      // TODO: Replace with actual API call
      // await Promise.all(selectedRowKeys.map(id => api.deleteProductMapping(id)))
      await new Promise((resolve) => setTimeout(resolve, 500))

      Toast.success(
        t('productMappings.messages.batchDeleteSuccess', { count: selectedRowKeys.length })
      )
      setSelectedRowKeys([])
      fetchMappings()
    } catch {
      Toast.error(t('productMappings.messages.batchDeleteError'))
    }
  }, [selectedRowKeys, fetchMappings, t])

  /**
   * Get platform display info
   */
  const getPlatformInfo = useCallback((code: PlatformCode): PlatformMeta => {
    return PLATFORMS.find((p) => p.code === code) || PLATFORMS[0]
  }, [])

  // Platform filter options
  const platformOptions = useMemo(
    () => [
      { label: t('productMappings.allPlatforms'), value: '' },
      ...PLATFORMS.map((p) => ({ label: p.name, value: p.code })),
    ],
    [t]
  )

  // Sync status filter options
  const syncStatusOptions = useMemo(
    () => [
      { label: t('productMappings.allSyncStatus'), value: '' },
      { label: t('productMappings.syncStatus.synced'), value: 'synced' },
      { label: t('productMappings.syncStatus.notSynced'), value: 'notSynced' },
    ],
    [t]
  )

  // Table columns
  const tableColumns: DataTableColumn<MappingRow>[] = useMemo(
    () => [
      {
        title: t('productMappings.columns.localProduct'),
        dataIndex: 'localProductName',
        sortable: true,
        ellipsis: true,
        render: (_: unknown, record: MappingRow) => (
          <div className="product-cell">
            <span className="product-name">{record.localProductName}</span>
            <span className="product-code">{record.localProductCode}</span>
          </div>
        ),
      },
      {
        title: t('productMappings.columns.platform'),
        dataIndex: 'platformCode',
        width: 120,
        align: 'center',
        render: (code: unknown) => {
          const platform = getPlatformInfo(code as PlatformCode)
          return (
            <Tag style={{ backgroundColor: platform.color, color: '#fff' }}>{platform.name}</Tag>
          )
        },
      },
      {
        title: t('productMappings.columns.platformProduct'),
        dataIndex: 'platformProductId',
        ellipsis: true,
        render: (_: unknown, record: MappingRow) => (
          <div className="product-cell">
            <span className="product-name">{record.platformProductName || '-'}</span>
            <span className="product-code">{record.platformSku}</span>
          </div>
        ),
      },
      {
        title: t('productMappings.columns.syncSettings'),
        dataIndex: 'syncPrice',
        width: 150,
        align: 'center',
        render: (_: unknown, record: MappingRow) => (
          <Space>
            {record.syncPrice && (
              <Tag color="blue" size="small">
                {t('productMappings.syncPrice')}
              </Tag>
            )}
            {record.syncInventory && (
              <Tag color="green" size="small">
                {t('productMappings.syncInventory')}
              </Tag>
            )}
            {!record.syncPrice && !record.syncInventory && <Text type="tertiary">-</Text>}
          </Space>
        ),
      },
      {
        title: t('productMappings.columns.lastSync'),
        dataIndex: 'lastSyncAt',
        width: 160,
        sortable: true,
        render: (date: unknown) => {
          const dateStr = date as string | undefined
          if (!dateStr) return <Text type="tertiary">-</Text>
          return formatDate(new Date(dateStr), 'datetime')
        },
      },
      {
        title: t('productMappings.columns.createdAt'),
        dataIndex: 'createdAt',
        width: 120,
        sortable: true,
        render: (date: unknown) => {
          const dateStr = date as string | undefined
          if (!dateStr) return '-'
          return formatDate(new Date(dateStr), 'short')
        },
      },
    ],
    [t, getPlatformInfo, formatDate]
  )

  // Table row actions
  const tableActions: TableAction<MappingRow>[] = useMemo(
    () => [
      {
        key: 'edit',
        label: t('productMappings.actions.edit'),
        onClick: (record) => openMappingModal(record),
      },
      {
        key: 'delete',
        label: t('productMappings.actions.delete'),
        type: 'danger',
        onClick: handleDeleteMapping,
      },
    ],
    [t, openMappingModal, handleDeleteMapping]
  )

  // Row selection handler
  const onSelectionChange = useCallback((keys: string[], _rows: MappingRow[]) => {
    setSelectedRowKeys(keys)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchMappings()
  }, [fetchMappings])

  return (
    <Container size="full" className="product-mappings-page">
      <Card className="product-mappings-card">
        <div className="product-mappings-header">
          <div>
            <Title heading={4} style={{ margin: 0 }}>
              {t('productMappings.title')}
            </Title>
            <Text type="tertiary">{t('productMappings.subtitle')}</Text>
          </div>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('productMappings.searchPlaceholder')}
          primaryAction={{
            label: t('productMappings.addMapping'),
            icon: <IconPlus />,
            onClick: () => openMappingModal(),
          }}
          secondaryActions={[
            {
              key: 'batchMapping',
              label: t('productMappings.batchMapping'),
              icon: <IconLink />,
              onClick: openBatchMappingModal,
            },
            {
              key: 'refresh',
              label: t('common:actions.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder={t('productMappings.platformFilter')}
                value={platformFilter}
                onChange={handlePlatformFilterChange}
                optionList={platformOptions}
                style={{ width: 140 }}
              />
              <Select
                placeholder={t('productMappings.syncStatusFilter')}
                value={syncStatusFilter}
                onChange={handleSyncStatusFilterChange}
                optionList={syncStatusOptions}
                style={{ width: 120 }}
              />
            </Space>
          }
        />

        {selectedRowKeys.length > 0 && (
          <BulkActionBar
            selectedCount={selectedRowKeys.length}
            onCancel={() => setSelectedRowKeys([])}
          >
            <Popconfirm
              title={t('productMappings.confirm.batchDeleteTitle')}
              content={t('productMappings.confirm.batchDeleteContent', {
                count: selectedRowKeys.length,
              })}
              onConfirm={handleBulkDelete}
              okText={t('common:confirm')}
              cancelText={t('common:cancel')}
            >
              <Tag color="red" style={{ cursor: 'pointer' }}>
                <IconUnlink style={{ marginRight: 4 }} />
                {t('productMappings.actions.batchDelete')}
              </Tag>
            </Popconfirm>
          </BulkActionBar>
        )}

        <Spin spinning={loading}>
          <DataTable<MappingRow>
            data={mappings}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            rowSelection={{
              selectedRowKeys,
              onChange: onSelectionChange,
            }}
            scroll={{ x: 1000 }}
            resizable
          />
        </Spin>
      </Card>

      {/* Manual Mapping Modal */}
      <Modal
        title={
          editingMapping ? t('productMappings.editMapping') : t('productMappings.createMapping')
        }
        visible={mappingModalVisible}
        onOk={handleSaveMapping}
        onCancel={closeMappingModal}
        okText={t('common:save')}
        cancelText={t('common:cancel')}
        width={600}
        className="mapping-modal"
      >
        <Form labelPosition="left" labelWidth={120}>
          {/* Platform Selection */}
          <Form.Select
            field="platform"
            label={t('productMappings.fields.platform')}
            value={selectedPlatform}
            onChange={(value) => {
              setSelectedPlatform(value as PlatformCode)
              setSelectedPlatformProduct(null)
              setPlatformProductOptions([])
            }}
            optionList={PLATFORMS.map((p) => ({ label: p.name, value: p.code }))}
            disabled={!!editingMapping}
            style={{ width: '100%' }}
          />

          {/* Local Product Selection */}
          <Form.Select
            field="localProduct"
            label={t('productMappings.fields.localProduct')}
            placeholder={t('productMappings.placeholders.searchLocalProduct')}
            value={selectedLocalProduct?.id}
            onSearch={(keyword) => searchLocalProducts(keyword)}
            onChange={(value) => {
              const product = localProductOptions.find((p) => p.id === value)
              setSelectedLocalProduct(product || null)
            }}
            optionList={localProductOptions.map((p) => ({
              label: `${p.code} - ${p.name}`,
              value: p.id,
            }))}
            loading={searchingLocal}
            filter={false}
            remote
            showClear
            disabled={!!editingMapping}
            style={{ width: '100%' }}
            prefix={<IconSearch />}
          />

          {/* Platform Product Selection */}
          <Form.Select
            field="platformProduct"
            label={t('productMappings.fields.platformProduct')}
            placeholder={t('productMappings.placeholders.searchPlatformProduct')}
            value={selectedPlatformProduct?.id}
            onSearch={(keyword) => searchPlatformProducts(keyword, selectedPlatform)}
            onChange={(value) => {
              const product = platformProductOptions.find((p) => p.id === value)
              setSelectedPlatformProduct(product || null)
            }}
            optionList={platformProductOptions.map((p) => ({
              label: `${p.sku} - ${p.title}`,
              value: p.id,
            }))}
            loading={searchingPlatform}
            filter={false}
            remote
            showClear
            style={{ width: '100%' }}
            prefix={<IconSearch />}
          />

          {/* Sync Settings */}
          <div className="sync-settings-section">
            <Title heading={6} style={{ marginBottom: 12 }}>
              {t('productMappings.sections.syncSettings')}
            </Title>
            <Space vertical align="start">
              <Checkbox
                checked={syncPriceEnabled}
                onChange={(e) => setSyncPriceEnabled(e.target.checked)}
              >
                {t('productMappings.fields.syncPrice')}
              </Checkbox>
              <Checkbox
                checked={syncInventoryEnabled}
                onChange={(e) => setSyncInventoryEnabled(e.target.checked)}
              >
                {t('productMappings.fields.syncInventory')}
              </Checkbox>
            </Space>
          </div>
        </Form>
      </Modal>

      {/* Batch Mapping Modal */}
      <Modal
        title={t('productMappings.batchMappingTitle')}
        visible={batchMappingModalVisible}
        onOk={handleBatchMapping}
        onCancel={closeBatchMappingModal}
        okText={t('productMappings.confirmBatchMapping')}
        cancelText={t('common:cancel')}
        width={700}
        className="batch-mapping-modal"
      >
        <div className="batch-mapping-content">
          {/* Platform Selection */}
          <div className="batch-platform-section">
            <Form labelPosition="left" labelWidth={100}>
              <Form.Select
                field="batchPlatform"
                label={t('productMappings.fields.platform')}
                value={batchPlatform}
                onChange={handleBatchPlatformChange}
                optionList={PLATFORMS.map((p) => ({ label: p.name, value: p.code }))}
                style={{ width: 200 }}
              />
            </Form>
          </div>

          {/* Sync Settings */}
          <div className="batch-sync-section">
            <Space>
              <Checkbox
                checked={batchSyncPrice}
                onChange={(e) => setBatchSyncPrice(e.target.checked)}
              >
                {t('productMappings.fields.syncPrice')}
              </Checkbox>
              <Checkbox
                checked={batchSyncInventory}
                onChange={(e) => setBatchSyncInventory(e.target.checked)}
              >
                {t('productMappings.fields.syncInventory')}
              </Checkbox>
            </Space>
          </div>

          {/* Unmapped Products List */}
          <div className="unmapped-products-section">
            <Title heading={6}>{t('productMappings.unmappedProducts')}</Title>
            <Text type="tertiary" size="small">
              {t('productMappings.unmappedProductsHint')}
            </Text>

            {unmappedProducts.length > 0 ? (
              <div className="unmapped-products-list">
                <Checkbox.Group
                  value={selectedUnmappedKeys}
                  onChange={(values) => setSelectedUnmappedKeys(values as string[])}
                  direction="vertical"
                >
                  {unmappedProducts.map((product) => (
                    <Checkbox key={product.id} value={product.id}>
                      <div className="unmapped-product-item">
                        <span className="product-code">{product.code}</span>
                        <span className="product-name">{product.name}</span>
                        <span className="product-unit">{product.unit}</span>
                      </div>
                    </Checkbox>
                  ))}
                </Checkbox.Group>
              </div>
            ) : (
              <Empty
                image={<IconDelete size="extra-large" />}
                title={t('productMappings.noUnmappedProducts')}
                description={t('productMappings.noUnmappedProductsHint')}
              />
            )}
          </div>

          {selectedUnmappedKeys.length > 0 && (
            <div className="batch-selection-info">
              <Text type="secondary">
                {t('productMappings.selectedCount', { count: selectedUnmappedKeys.length })}
              </Text>
            </div>
          )}
        </div>
      </Modal>
    </Container>
  )
}
