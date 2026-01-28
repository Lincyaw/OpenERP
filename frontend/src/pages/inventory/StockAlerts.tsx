import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Spin,
  Empty,
  Button,
  Modal,
  Form,
} from '@douyinfe/semi-ui-19'
import { IconRefresh, IconAlertTriangle } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import { listInventoryBelowMinimum, setThresholdsInventory } from '@/api/inventory/inventory'
import { listWarehouses } from '@/api/warehouses/warehouses'
import { listProducts } from '@/api/products/products'
import type {
  HandlerInventoryItemResponse,
  ListInventoryBelowMinimumParams,
  ListInventoryBelowMinimumOrderDir,
  HandlerWarehouseListResponse,
  HandlerProductListResponse,
  HandlerSetThresholdsRequest,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './StockAlerts.css'
import { createScopedLogger } from '@/utils'

const log = createScopedLogger('StockAlerts')

const { Title, Text } = Typography

// Inventory item type with index signature for DataTable compatibility
type InventoryItem = HandlerInventoryItemResponse & Record<string, unknown>

type WarehouseOption = {
  label: string
  value: string
}

/**
 * Format quantity for display with 2 decimal places
 * Handles both number and string values from API
 */
function formatQuantity(quantity?: number | string): string {
  if (quantity === undefined || quantity === null) return '-'
  const num = typeof quantity === 'string' ? parseFloat(quantity) : quantity
  if (isNaN(num)) return '-'
  return num.toFixed(2)
}

/**
 * Calculate shortage amount (min_quantity - total_quantity)
 */
function calculateShortage(minQty?: number, totalQty?: number): number {
  if (minQty === undefined || totalQty === undefined) return 0
  const shortage = minQty - totalQty
  return shortage > 0 ? shortage : 0
}

/**
 * Stock Alerts Page
 *
 * Features:
 * - Display items below their minimum stock threshold
 * - Filter by warehouse
 * - Show shortage quantities
 * - Set threshold modal for individual items
 * - Summary cards showing alert counts
 */
export default function StockAlertsPage() {
  const navigate = useNavigate()
  const { t } = useTranslation(['inventory', 'common'])
  const { formatDate: formatDateBase } = useFormatters()

  // Wrapper function to handle undefined values
  const formatDate = useCallback(
    (date?: string, style?: 'date' | 'dateTime'): string =>
      date ? formatDateBase(date, style === 'dateTime' ? 'medium' : 'short') : '-',
    [formatDateBase]
  )

  // State for data
  const [alertList, setAlertList] = useState<InventoryItem[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [warehouseFilter, setWarehouseFilter] = useState<string>('')

  // Warehouse and product options for display
  const [warehouseOptions, setWarehouseOptions] = useState<WarehouseOption[]>([])
  const [warehouseMap, setWarehouseMap] = useState<Map<string, string>>(new Map())
  const [productMap, setProductMap] = useState<Map<string, string>>(new Map())

  // Threshold modal state
  const [thresholdModalVisible, setThresholdModalVisible] = useState(false)
  const [selectedItem, setSelectedItem] = useState<InventoryItem | null>(null)
  const [thresholdSaving, setThresholdSaving] = useState(false)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'updated_at',
    defaultSortOrder: 'desc',
  })

  // Fetch warehouses for filter dropdown
  const fetchWarehouses = useCallback(async () => {
    try {
      const response = await listWarehouses({
        page: 1,
        page_size: 100,
        status: 'enabled',
      })
      if (response.status === 200 && response.data.success && response.data.data) {
        const warehouses = response.data.data as HandlerWarehouseListResponse[]
        const options: WarehouseOption[] = [
          { label: t('alerts.allWarehouses'), value: '' },
          ...warehouses.map((w: HandlerWarehouseListResponse) => ({
            label: w.name || w.code || '',
            value: w.id || '',
          })),
        ]
        setWarehouseOptions(options)

        // Build warehouse map for display
        const map = new Map<string, string>()
        warehouses.forEach((w: HandlerWarehouseListResponse) => {
          if (w.id) {
            map.set(w.id, w.name || w.code || w.id)
          }
        })
        setWarehouseMap(map)
      }
    } catch {
      log.error('Failed to fetch warehouses')
    }
  }, [t])

  // Fetch products for display names
  const fetchProducts = useCallback(async () => {
    try {
      const response = await listProducts({ page: 1, page_size: 100 })
      if (response.status === 200 && response.data.success && response.data.data) {
        const products = response.data.data as HandlerProductListResponse[]
        const map = new Map<string, string>()
        products.forEach((p) => {
          if (p.id) {
            map.set(p.id, p.name || p.code || p.id)
          }
        })
        setProductMap(map)
      }
    } catch {
      log.error('Failed to fetch products')
    }
  }, [])

  // Fetch low stock alert items
  const fetchAlerts = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListInventoryBelowMinimumParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        warehouse_id: warehouseFilter || undefined,
        order_by: state.sort.field || 'updated_at',
        order_dir: (state.sort.order === 'asc'
          ? 'asc'
          : 'desc') as ListInventoryBelowMinimumOrderDir,
      }

      const response = await listInventoryBelowMinimum(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        // Client-side filtering by search keyword (product name)
        let items = response.data.data as InventoryItem[]
        if (searchKeyword && productMap.size > 0) {
          const keyword = searchKeyword.toLowerCase()
          items = items.filter((item) => {
            const productName = item.product_id ? productMap.get(item.product_id) || '' : ''
            return productName.toLowerCase().includes(keyword)
          })
        }
        setAlertList(items)
        if (response.data.meta) {
          setPaginationMeta({
            page: response.data.meta.page || 1,
            page_size: response.data.meta.page_size || 20,
            total: response.data.meta.total || 0,
            total_pages: response.data.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error(t('alerts.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    warehouseFilter,
    t,
    productMap,
  ])

  // Fetch warehouses and products on mount
  useEffect(() => {
    fetchWarehouses()
    fetchProducts()
  }, [fetchWarehouses, fetchProducts])

  // Fetch alerts on mount and when state changes
  useEffect(() => {
    fetchAlerts()
  }, [fetchAlerts])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle warehouse filter change
  const handleWarehouseChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const warehouseValue = typeof value === 'string' ? value : ''
      setWarehouseFilter(warehouseValue)
      setFilter('warehouse_id', warehouseValue || null)
    },
    [setFilter]
  )

  // Handle view detail
  const handleViewDetail = useCallback(
    (item: InventoryItem) => {
      if (item.id) {
        navigate(`/inventory/stock/${item.id}`)
      }
    },
    [navigate]
  )

  // Handle adjust stock
  const handleAdjustStock = useCallback(
    (item: InventoryItem) => {
      if (item.id) {
        navigate(
          `/inventory/adjust?warehouse_id=${item.warehouse_id}&product_id=${item.product_id}`
        )
      }
    },
    [navigate]
  )

  // Handle set threshold click
  const handleSetThreshold = useCallback((item: InventoryItem) => {
    setSelectedItem(item)
    setThresholdModalVisible(true)
  }, [])

  // Handle threshold modal close
  const handleThresholdModalClose = useCallback(() => {
    setThresholdModalVisible(false)
    setSelectedItem(null)
  }, [])

  // Handle threshold save
  const handleThresholdSave = useCallback(
    async (values: { min_quantity?: number; max_quantity?: number }) => {
      if (!selectedItem?.warehouse_id || !selectedItem?.product_id) {
        return
      }

      setThresholdSaving(true)
      try {
        const request: HandlerSetThresholdsRequest = {
          warehouse_id: selectedItem.warehouse_id,
          product_id: selectedItem.product_id,
          min_quantity: values.min_quantity,
          max_quantity: values.max_quantity,
        }

        const response = await setThresholdsInventory(request)

        if (response.status === 200 && response.data.success) {
          Toast.success(t('alerts.messages.setThresholdSuccess'))
          handleThresholdModalClose()
          fetchAlerts()
        } else {
          Toast.error(t('alerts.messages.setThresholdError'))
        }
      } catch {
        Toast.error(t('alerts.messages.setThresholdError'))
      } finally {
        setThresholdSaving(false)
      }
    },
    [selectedItem, t, handleThresholdModalClose, fetchAlerts]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchAlerts()
  }, [fetchAlerts])

  // Get warehouse name by ID
  const getWarehouseName = useCallback(
    (warehouseId?: string): string => {
      if (!warehouseId) return '-'
      return warehouseMap.get(warehouseId) || warehouseId.substring(0, 8) + '...'
    },
    [warehouseMap]
  )

  // Get product name by ID
  const getProductName = useCallback(
    (productId?: string): string => {
      if (!productId) return '-'
      return productMap.get(productId) || productId.substring(0, 8) + '...'
    },
    [productMap]
  )

  // Calculate summary statistics
  const summaryStats = useMemo(() => {
    const total = paginationMeta?.total || 0
    // Critical: stock is 0 or very low (less than 20% of min)
    const critical = alertList.filter((item) => {
      const totalQty = item.total_quantity || 0
      const minQty = item.min_quantity || 0
      return totalQty <= 0 || (minQty > 0 && totalQty < minQty * 0.2)
    }).length
    const warning = alertList.length - critical
    return { total, critical, warning }
  }, [alertList, paginationMeta])

  // Table columns
  const tableColumns: DataTableColumn<InventoryItem>[] = useMemo(
    () => [
      {
        title: t('alerts.columns.warehouse'),
        dataIndex: 'warehouse_id',
        width: 120,
        render: (warehouseId: unknown) => getWarehouseName(warehouseId as string | undefined),
      },
      {
        title: t('alerts.columns.product'),
        dataIndex: 'product_id',
        width: 180,
        ellipsis: true,
        render: (productId: unknown) => (
          <span className="product-name-cell">
            {getProductName(productId as string | undefined)}
          </span>
        ),
      },
      {
        title: t('alerts.columns.availableQuantity'),
        dataIndex: 'available_quantity',
        width: 110,
        align: 'right',
        sortable: true,
        render: (qty: unknown, record: InventoryItem) => {
          const availableQty = qty as number | undefined
          const isCritical =
            (availableQty || 0) <= 0 ||
            (record.min_quantity && (availableQty || 0) < record.min_quantity * 0.2)
          return (
            <div className="quantity-cell">
              <span className={isCritical ? 'quantity-critical' : 'quantity-warning'}>
                {formatQuantity(availableQty)}
              </span>
              <IconAlertTriangle className={isCritical ? 'critical-icon' : 'warning-icon'} />
            </div>
          )
        },
      },
      {
        title: t('alerts.columns.minQuantity'),
        dataIndex: 'min_quantity',
        width: 100,
        align: 'right',
        sortable: true,
        render: (qty: unknown) => formatQuantity(qty as number | undefined),
      },
      {
        title: t('alerts.columns.shortage'),
        dataIndex: 'shortage',
        width: 100,
        align: 'right',
        render: (_: unknown, record: InventoryItem) => {
          const shortage = calculateShortage(record.min_quantity, record.total_quantity)
          if (shortage <= 0) return '-'
          return <span className="quantity-shortage">-{formatQuantity(shortage)}</span>
        },
      },
      {
        title: t('alerts.columns.totalQuantity'),
        dataIndex: 'total_quantity',
        width: 100,
        align: 'right',
        sortable: true,
        render: (qty: unknown) => formatQuantity(qty as number | undefined),
      },
      {
        title: t('alerts.columns.status'),
        dataIndex: 'status',
        width: 110,
        align: 'center',
        render: (_: unknown, record: InventoryItem) => {
          const totalQty = record.total_quantity || 0
          const minQty = record.min_quantity || 0
          const isCritical = totalQty <= 0 || (minQty > 0 && totalQty < minQty * 0.2)

          if (isCritical) {
            return <Tag color="red">{t('alerts.status.critical')}</Tag>
          }
          return <Tag color="orange">{t('alerts.status.warning')}</Tag>
        },
      },
      {
        title: t('alerts.columns.updatedAt'),
        dataIndex: 'updated_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, 'dateTime'),
      },
    ],
    [getWarehouseName, getProductName, t, formatDate]
  )

  // Table row actions
  const tableActions: TableAction<InventoryItem>[] = useMemo(
    () => [
      {
        key: 'setThreshold',
        label: t('alerts.actions.setThreshold'),
        onClick: handleSetThreshold,
      },
      {
        key: 'view',
        label: t('alerts.actions.viewDetail'),
        onClick: handleViewDetail,
      },
      {
        key: 'adjust',
        label: t('alerts.actions.adjustStock'),
        onClick: handleAdjustStock,
      },
    ],
    [handleSetThreshold, handleViewDetail, handleAdjustStock, t]
  )

  return (
    <Container size="full" className="stock-alerts-page">
      <Card className="stock-alerts-card">
        <div className="stock-alerts-header">
          <div>
            <Title heading={4} style={{ margin: 0 }}>
              {t('alerts.title')}
            </Title>
            <Text type="tertiary" className="stock-alerts-description">
              {t('alerts.description')}
            </Text>
          </div>
        </div>

        {/* Summary Cards */}
        <div className="summary-cards">
          <div className="summary-card">
            <div className="summary-label">{t('alerts.summary.totalAlerts')}</div>
            <div className="summary-value">{summaryStats.total}</div>
          </div>
          <div className="summary-card summary-critical">
            <div className="summary-label">{t('alerts.summary.criticalCount')}</div>
            <div className="summary-value">{summaryStats.critical}</div>
          </div>
          <div className="summary-card summary-warning">
            <div className="summary-label">{t('alerts.summary.warningCount')}</div>
            <div className="summary-value">{summaryStats.warning}</div>
          </div>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('alerts.searchPlaceholder')}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('alerts.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder={t('alerts.selectWarehouse')}
                value={warehouseFilter}
                onChange={handleWarehouseChange}
                optionList={warehouseOptions}
                style={{ width: 150 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          {alertList.length === 0 && !loading ? (
            <Empty
              image={<IconAlertTriangle size="extra-large" style={{ color: '#52c41a' }} />}
              title={t('alerts.empty.title')}
              description={t('alerts.empty.description')}
              className="empty-state"
            />
          ) : (
            <DataTable<InventoryItem>
              data={alertList}
              columns={tableColumns}
              rowKey="id"
              loading={loading}
              pagination={paginationMeta}
              actions={tableActions}
              onStateChange={handleStateChange}
              sortState={state.sort}
              scroll={{ x: 1100 }}
            />
          )}
        </Spin>
      </Card>

      {/* Threshold Setting Modal */}
      <Modal
        title={t('alerts.thresholdModal.title')}
        visible={thresholdModalVisible}
        onCancel={handleThresholdModalClose}
        footer={null}
        width={480}
      >
        {selectedItem && (
          <Form
            onSubmit={(values) =>
              handleThresholdSave(values as { min_quantity?: number; max_quantity?: number })
            }
            initValues={{
              min_quantity: selectedItem.min_quantity,
              max_quantity: selectedItem.max_quantity,
            }}
          >
            <div className="threshold-modal-info">
              <div className="info-row">
                <span className="info-label">{t('alerts.thresholdModal.product')}:</span>
                <span className="info-value">{getProductName(selectedItem.product_id)}</span>
              </div>
              <div className="info-row">
                <span className="info-label">{t('alerts.thresholdModal.warehouse')}:</span>
                <span className="info-value">{getWarehouseName(selectedItem.warehouse_id)}</span>
              </div>
              <div className="info-row">
                <span className="info-label">{t('alerts.thresholdModal.currentStock')}:</span>
                <span className="info-value">{formatQuantity(selectedItem.total_quantity)}</span>
              </div>
            </div>

            <Form.InputNumber
              field="min_quantity"
              label={t('alerts.thresholdModal.minQuantity')}
              placeholder={t('alerts.thresholdModal.minQuantityPlaceholder')}
              min={0}
              extraText={t('alerts.thresholdModal.minQuantityHelper')}
              style={{ width: '100%' }}
            />

            <Form.InputNumber
              field="max_quantity"
              label={t('alerts.thresholdModal.maxQuantity')}
              placeholder={t('alerts.thresholdModal.maxQuantityPlaceholder')}
              min={0}
              extraText={t('alerts.thresholdModal.maxQuantityHelper')}
              style={{ width: '100%' }}
            />

            <div className="threshold-modal-actions">
              <Button onClick={handleThresholdModalClose}>
                {t('alerts.thresholdModal.cancel')}
              </Button>
              <Button type="primary" htmlType="submit" loading={thresholdSaving}>
                {t('alerts.thresholdModal.submit')}
              </Button>
            </div>
          </Form>
        )}
      </Modal>
    </Container>
  )
}
