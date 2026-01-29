import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Toast, Modal, Empty } from '@douyinfe/semi-ui-19'
import { IconEdit, IconPlay, IconStop, IconDelete } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import {
  DetailPageHeader,
  type DetailPageHeaderAction,
  type DetailPageHeaderStatus,
  type DetailPageHeaderMetric,
} from '@/components/common'
import { useFormatters } from '@/hooks/useFormatters'
import {
  getProductById,
  activateProduct,
  deactivateProduct,
  discontinueProduct,
  deleteProduct,
} from '@/api/products/products'
import type { HandlerProductResponse, HandlerProductResponseStatus } from '@/api/models'
import './ProductDetail.css'

const { Text } = Typography

// Status variant mapping for DetailPageHeader
const STATUS_VARIANTS: Record<HandlerProductResponseStatus, 'default' | 'success' | 'danger'> = {
  active: 'success',
  inactive: 'default',
  discontinued: 'danger',
}

/**
 * Product Detail Page
 *
 * Features:
 * - Display complete product information using DetailPageHeader
 * - Display price information
 * - Status action buttons (activate, deactivate, discontinue)
 * - Navigate to edit page
 */
export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation(['catalog', 'common'])
  const { formatCurrency, formatDateTime } = useFormatters()

  const [product, setProduct] = useState<HandlerProductResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch product details
  const fetchProduct = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await getProductById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setProduct(response.data.data)
      } else {
        Toast.error(t('products.messages.loadError'))
        navigate('/catalog/products')
      }
    } catch {
      Toast.error(t('products.messages.loadError'))
      navigate('/catalog/products')
    } finally {
      setLoading(false)
    }
  }, [id, navigate, t])

  useEffect(() => {
    fetchProduct()
  }, [fetchProduct])

  // Handle activate product
  const handleActivate = useCallback(async () => {
    if (!product?.id) return
    setActionLoading(true)
    try {
      await activateProduct(product.id, {})
      Toast.success(t('products.messages.activateSuccess', { name: product.name }))
      fetchProduct()
    } catch {
      Toast.error(t('products.messages.activateError'))
    } finally {
      setActionLoading(false)
    }
  }, [product, fetchProduct, t])

  // Handle deactivate product
  const handleDeactivate = useCallback(async () => {
    if (!product?.id) return
    setActionLoading(true)
    try {
      await deactivateProduct(product.id, {})
      Toast.success(t('products.messages.deactivateSuccess', { name: product.name }))
      fetchProduct()
    } catch {
      Toast.error(t('products.messages.deactivateError'))
    } finally {
      setActionLoading(false)
    }
  }, [product, fetchProduct, t])

  // Handle discontinue product
  const handleDiscontinue = useCallback(async () => {
    if (!product?.id) return
    Modal.confirm({
      title: t('products.confirm.discontinueTitle'),
      content: t('products.confirm.discontinueContent', { name: product.name }),
      okText: t('products.confirm.discontinueOk'),
      cancelText: t('common:actions.cancel'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await discontinueProduct(product.id!, {})
          Toast.success(t('products.messages.discontinueSuccess', { name: product.name }))
          fetchProduct()
        } catch {
          Toast.error(t('products.messages.discontinueError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [product, fetchProduct, t])

  // Handle delete product
  const handleDelete = useCallback(async () => {
    if (!product?.id) return
    Modal.confirm({
      title: t('products.confirm.deleteTitle'),
      content: t('products.confirm.deleteContent', { name: product.name }),
      okText: t('products.confirm.deleteOk'),
      cancelText: t('common:actions.cancel'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        setActionLoading(true)
        try {
          await deleteProduct(product.id!)
          Toast.success(t('products.messages.deleteSuccess', { name: product.name }))
          navigate('/catalog/products')
        } catch {
          Toast.error(t('products.messages.deleteError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [product, navigate, t])

  // Handle edit product
  const handleEdit = useCallback(() => {
    if (product?.id) {
      navigate(`/catalog/products/${product.id}/edit`)
    }
  }, [product, navigate])

  // Calculate profit margin
  const profitMargin = useMemo(() => {
    if (!product?.selling_price || !product?.purchase_price || product.purchase_price === 0) {
      return null
    }
    return ((product.selling_price - product.purchase_price) / product.purchase_price) * 100
  }, [product])

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    if (!product?.status) return undefined
    return {
      label: t(`products.status.${product.status}`),
      variant: STATUS_VARIANTS[product.status] || 'default',
    }
  }, [product, t])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!product) return []
    return [
      {
        label: t('productDetail.fields.unit'),
        value: product.unit || '-',
      },
      {
        label: t('productDetail.fields.purchasePrice'),
        value: product.purchase_price !== undefined ? formatCurrency(product.purchase_price) : '-',
      },
      {
        label: t('productDetail.fields.sellingPrice'),
        value: product.selling_price !== undefined ? formatCurrency(product.selling_price) : '-',
        variant: 'primary',
      },
      {
        label: t('productDetail.fields.profitMargin'),
        value: profitMargin !== null ? `${profitMargin.toFixed(1)}%` : '-',
        variant: profitMargin !== null && profitMargin > 0 ? 'success' : 'default',
      },
    ]
  }, [product, t, formatCurrency, profitMargin])

  // Build primary action for DetailPageHeader
  const primaryAction = useMemo((): DetailPageHeaderAction | undefined => {
    if (!product) return undefined
    const status = product.status

    if (status === 'inactive') {
      return {
        key: 'activate',
        label: t('products.actions.activate'),
        icon: <IconPlay />,
        type: 'primary',
        onClick: handleActivate,
        loading: actionLoading,
      }
    }
    if (status === 'active') {
      return {
        key: 'deactivate',
        label: t('products.actions.deactivate'),
        icon: <IconStop />,
        type: 'warning',
        onClick: handleDeactivate,
        loading: actionLoading,
      }
    }
    return undefined
  }, [product, t, handleActivate, handleDeactivate, actionLoading])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    if (!product) return []
    const status = product.status
    const actions: DetailPageHeaderAction[] = []

    if (status !== 'discontinued') {
      actions.push({
        key: 'edit',
        label: t('common:actions.edit'),
        icon: <IconEdit />,
        onClick: handleEdit,
        disabled: actionLoading,
      })
      actions.push({
        key: 'discontinue',
        label: t('products.actions.discontinue'),
        type: 'danger',
        onClick: handleDiscontinue,
        loading: actionLoading,
      })
    }
    actions.push({
      key: 'delete',
      label: t('products.actions.delete'),
      icon: <IconDelete />,
      type: 'danger',
      onClick: handleDelete,
      loading: actionLoading,
    })

    return actions
  }, [product, t, handleEdit, handleDiscontinue, handleDelete, actionLoading])

  // Render basic info
  const renderBasicInfo = () => {
    if (!product) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.code')}
          </Text>
          <Text strong className="info-value code-value">
            {product.code || '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.name')}
          </Text>
          <Text className="info-value">{product.name || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.unit')}
          </Text>
          <Text className="info-value">{product.unit || '-'}</Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.barcode')}
          </Text>
          <Text className="info-value code-value">{product.barcode || '-'}</Text>
        </div>
        <div className="info-item info-item--full">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.description')}
          </Text>
          <Text className="info-value">{product.description || '-'}</Text>
        </div>
      </div>
    )
  }

  // Render price info
  const renderPriceInfo = () => {
    if (!product) return null

    const profitMarginClass =
      profitMargin !== null ? (profitMargin > 0 ? 'positive' : 'negative') : ''

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.purchasePrice')}
          </Text>
          <Text className="info-value">
            {product.purchase_price !== undefined ? formatCurrency(product.purchase_price) : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.sellingPrice')}
          </Text>
          <Text className="info-value price-value">
            {product.selling_price !== undefined ? formatCurrency(product.selling_price) : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.profitMargin')}
          </Text>
          <Text className={`info-value profit-value ${profitMarginClass}`}>
            {profitMargin !== null ? `${profitMargin.toFixed(2)}%` : '-'}
          </Text>
        </div>
      </div>
    )
  }

  // Render stock settings
  const renderStockSettings = () => {
    if (!product) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.minStock')}
          </Text>
          <Text className="info-value">
            {product.min_stock !== undefined ? product.min_stock.toString() : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.sortOrder')}
          </Text>
          <Text className="info-value">
            {product.sort_order !== undefined ? product.sort_order.toString() : '-'}
          </Text>
        </div>
      </div>
    )
  }

  // Render timestamps
  const renderTimestamps = () => {
    if (!product) return null

    return (
      <div className="info-grid">
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.createdAt')}
          </Text>
          <Text className="info-value">
            {product.created_at ? formatDateTime(product.created_at) : '-'}
          </Text>
        </div>
        <div className="info-item">
          <Text type="secondary" className="info-label">
            {t('productDetail.fields.updatedAt')}
          </Text>
          <Text className="info-value">
            {product.updated_at ? formatDateTime(product.updated_at) : '-'}
          </Text>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="product-detail-page">
        <DetailPageHeader
          title={t('productDetail.title')}
          loading={true}
          showBack={true}
          onBack={() => navigate('/catalog/products')}
          backLabel={t('productDetail.back')}
        />
      </Container>
    )
  }

  if (!product) {
    return (
      <Container size="lg" className="product-detail-page">
        <Empty title={t('productDetail.notExist')} description={t('productDetail.notExistDesc')} />
      </Container>
    )
  }

  return (
    <Container size="lg" className="product-detail-page">
      {/* Unified Header */}
      <DetailPageHeader
        title={t('productDetail.title')}
        documentNumber={product.code}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={() => navigate('/catalog/products')}
        backLabel={t('productDetail.back')}
        titleSuffix={
          product.barcode ? (
            <span className="product-barcode-suffix">{product.barcode}</span>
          ) : undefined
        }
      />

      {/* Basic Info Card */}
      <Card className="info-card" title={t('products.form.basicInfo')}>
        {renderBasicInfo()}
      </Card>

      {/* Price Info Card */}
      <Card className="info-card" title={t('products.form.priceInfo')}>
        {renderPriceInfo()}
      </Card>

      {/* Stock Settings Card */}
      <Card className="info-card" title={t('products.form.stockSettings')}>
        {renderStockSettings()}
      </Card>

      {/* Timestamps Card */}
      <Card className="info-card" title={t('productDetail.timestamps')}>
        {renderTimestamps()}
      </Card>
    </Container>
  )
}
