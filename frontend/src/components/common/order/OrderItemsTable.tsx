import { useMemo, useCallback } from 'react'
import {
  Table,
  InputNumber,
  Input,
  Select,
  Typography,
  Button,
  Popconfirm,
  Empty,
} from '@douyinfe/semi-ui-19'
import { IconDelete, IconSearch } from '@douyinfe/semi-icons'
import type {
  SalesOrderItemFormData,
  PurchaseOrderItemFormData,
  OrderItemFormData,
} from '@/hooks/useOrderForm'
import { safeToFixed } from '@/utils'

const { Text } = Typography

/**
 * Product option for the select dropdown
 */
export interface ProductOption {
  value: string
  label: string
  code?: string
  name?: string
  unit?: string
  price?: number
}

/**
 * Props for OrderItemsTable component
 */
export interface OrderItemsTableProps<T extends OrderItemFormData> {
  /** Order items data */
  items: T[]
  /** Product options for dropdown */
  productOptions: ProductOption[]
  /** Whether products are loading */
  productsLoading: boolean
  /** Product search handler */
  onProductSearch: (search: string) => void
  /** Handle product selection for an item */
  onProductSelect: (itemKey: string, productId: string, productOption: ProductOption) => void
  /** Handle quantity change */
  onQuantityChange: (itemKey: string, quantity: number | string | undefined) => void
  /** Handle price/cost change */
  onPriceChange: (itemKey: string, price: number | string | undefined) => void
  /** Handle item remark change */
  onItemRemarkChange: (itemKey: string, remark: string) => void
  /** Handle item removal */
  onRemoveItem: (itemKey: string) => void
  /** Translation function */
  t: (key: string) => string
  /** Order type: 'sales' for selling price, 'purchase' for cost price */
  orderType: 'sales' | 'purchase'
  /** Optional className */
  className?: string
}

/**
 * Shared order items table component
 *
 * Displays and manages order line items for both sales and purchase orders.
 * Supports product selection, quantity/price editing, and item removal.
 *
 * @example
 * <OrderItemsTable
 *   items={formData.items}
 *   productOptions={productOptions}
 *   productsLoading={productsLoading}
 *   onProductSearch={setProductSearch}
 *   onProductSelect={handleProductSelect}
 *   onQuantityChange={handleQuantityChange}
 *   onPriceChange={handleUnitPriceChange}
 *   onItemRemarkChange={handleItemRemarkChange}
 *   onRemoveItem={handleRemoveItem}
 *   t={t}
 *   orderType="sales"
 * />
 */
export function OrderItemsTable<T extends OrderItemFormData>({
  items,
  productOptions,
  productsLoading,
  onProductSearch,
  onProductSelect,
  onQuantityChange,
  onPriceChange,
  onItemRemarkChange,
  onRemoveItem,
  t,
  orderType,
  className,
}: OrderItemsTableProps<T>) {
  // Get price field name based on order type
  const priceFieldLabel =
    orderType === 'sales'
      ? t('orderForm.items.columns.unitPrice')
      : t('orderForm.items.columns.purchasePrice')

  // Get price value from item based on order type
  const getPriceValue = useCallback(
    (item: T): number => {
      if (orderType === 'sales') {
        return (item as SalesOrderItemFormData).unit_price
      }
      return (item as PurchaseOrderItemFormData).unit_cost
    },
    [orderType]
  )

  // Handle product selection wrapper
  const handleProductSelectWrapper = useCallback(
    (itemKey: string, productId: string) => {
      const option = productOptions.find((p) => p.value === productId)
      if (option) {
        onProductSelect(itemKey, productId, option)
      }
    },
    [productOptions, onProductSelect]
  )

  // Table columns
  const columns = useMemo(
    () => [
      {
        title: t('orderForm.items.columns.product'),
        dataIndex: 'product_id',
        width: 280,
        render: (_: unknown, record: T) => (
          <Select
            value={record.product_id || undefined}
            placeholder={t('orderForm.items.columns.productPlaceholder')}
            onChange={(value) => handleProductSelectWrapper(record.key, value as string)}
            optionList={productOptions}
            filter
            remote
            onSearch={onProductSearch}
            loading={productsLoading}
            style={{ width: '100%' }}
            prefix={<IconSearch />}
            renderSelectedItem={(option: { label?: string }) => (
              <span className="selected-product">{option.label}</span>
            )}
          />
        ),
      },
      {
        title: t('orderForm.items.columns.unit'),
        dataIndex: 'unit',
        width: 80,
        render: (unit: string) => <Text>{unit || '-'}</Text>,
      },
      {
        title: priceFieldLabel,
        dataIndex: orderType === 'sales' ? 'unit_price' : 'unit_cost',
        width: 120,
        render: (_: unknown, record: T) => (
          <InputNumber
            value={getPriceValue(record)}
            onChange={(value) => onPriceChange(record.key, value)}
            min={0}
            precision={2}
            prefix="¥"
            style={{ width: '100%' }}
            disabled={!record.product_id}
          />
        ),
      },
      {
        title: t('orderForm.items.columns.quantity'),
        dataIndex: 'quantity',
        width: 100,
        render: (qty: number, record: T) => (
          <InputNumber
            value={qty}
            onChange={(value) => onQuantityChange(record.key, value)}
            min={0.01}
            precision={2}
            style={{ width: '100%' }}
            disabled={!record.product_id}
          />
        ),
      },
      {
        title: t('orderForm.items.columns.amount'),
        dataIndex: 'amount',
        width: 120,
        align: 'right' as const,
        render: (amount: number) => (
          <Text strong className="item-amount">
            ¥{safeToFixed(amount)}
          </Text>
        ),
      },
      {
        title: t('orderForm.items.columns.remark'),
        dataIndex: 'remark',
        width: 150,
        render: (remark: string, record: T) => (
          <Input
            value={remark}
            onChange={(value) => onItemRemarkChange(record.key, value)}
            placeholder={t('orderForm.items.columns.remarkPlaceholder')}
            disabled={!record.product_id}
          />
        ),
      },
      {
        title: t('orderForm.items.columns.operation'),
        dataIndex: 'actions',
        width: 60,
        render: (_: unknown, record: T) => (
          <Popconfirm
            title={t('orderForm.items.remove')}
            onConfirm={() => onRemoveItem(record.key)}
            position="left"
          >
            <Button icon={<IconDelete />} type="danger" theme="borderless" size="small" />
          </Popconfirm>
        ),
      },
    ],
    [
      t,
      priceFieldLabel,
      orderType,
      productOptions,
      productsLoading,
      handleProductSelectWrapper,
      getPriceValue,
      onProductSearch,
      onPriceChange,
      onQuantityChange,
      onItemRemarkChange,
      onRemoveItem,
    ]
  )

  return (
    <Table
      columns={columns}
      dataSource={items}
      rowKey="key"
      pagination={false}
      size="small"
      className={className}
      empty={<Empty description={t('orderForm.items.empty')} />}
    />
  )
}

export default OrderItemsTable
