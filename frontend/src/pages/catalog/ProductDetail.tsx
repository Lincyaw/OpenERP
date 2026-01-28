import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Descriptions,
  Tag,
  Toast,
  Button,
  Space,
  Spin,
  Empty,
  Modal,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconEdit, IconPlay, IconStop, IconDelete } from '@douyinfe/semi-icons'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import { getProducts } from '@/api/products/products'
import type { HandlerProductResponse, HandlerProductResponseStatus } from '@/api/models'
import './ProductDetail.css'

const { Title, Text } = Typography

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerProductResponseStatus, 'green' | 'grey' | 'red'> = {
  active: 'green',
  inactive: 'grey',
  discontinued: 'red',
}

/**
 * Product Detail Page
 *
 * Features:
 * - Display complete product information
 * - Display price information
 * - Status action buttons (activate, deactivate, discontinue)
 * - Navigate to edit page
 */
export default function ProductDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation(['catalog', 'common'])
  const { formatCurrency, formatDateTime } = useFormatters()
  const productsApi = useMemo(() => getProducts(), [])

  const [product, setProduct] = useState<HandlerProductResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)

  // Fetch product details
  const fetchProduct = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await productsApi.getProductById(id)
      if (response.success && response.data) {
        setProduct(response.data)
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
  }, [id, productsApi, navigate, t])

  useEffect(() => {
    fetchProduct()
  }, [fetchProduct])

  // Handle activate product
  const handleActivate = useCallback(async () => {
    if (!product?.id) return
    setActionLoading(true)
    try {
      await productsApi.activateProduct(product.id)
      Toast.success(t('products.messages.activateSuccess', { name: product.name }))
      fetchProduct()
    } catch {
      Toast.error(t('products.messages.activateError'))
    } finally {
      setActionLoading(false)
    }
  }, [product, productsApi, fetchProduct, t])

  // Handle deactivate product
  const handleDeactivate = useCallback(async () => {
    if (!product?.id) return
    setActionLoading(true)
    try {
      await productsApi.deactivateProduct(product.id)
      Toast.success(t('products.messages.deactivateSuccess', { name: product.name }))
      fetchProduct()
    } catch {
      Toast.error(t('products.messages.deactivateError'))
    } finally {
      setActionLoading(false)
    }
  }, [product, productsApi, fetchProduct, t])

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
          await productsApi.discontinueProduct(product.id!)
          Toast.success(t('products.messages.discontinueSuccess', { name: product.name }))
          fetchProduct()
        } catch {
          Toast.error(t('products.messages.discontinueError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [product, productsApi, fetchProduct, t])

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
          await productsApi.deleteProduct(product.id!)
          Toast.success(t('products.messages.deleteSuccess', { name: product.name }))
          navigate('/catalog/products')
        } catch {
          Toast.error(t('products.messages.deleteError'))
        } finally {
          setActionLoading(false)
        }
      },
    })
  }, [product, productsApi, navigate, t])

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

  // Render basic info
  const renderBasicInfo = () => {
    if (!product) return null

    const data = [
      { key: t('productDetail.fields.code'), value: product.code || '-' },
      { key: t('productDetail.fields.name'), value: product.name || '-' },
      { key: t('productDetail.fields.unit'), value: product.unit || '-' },
      { key: t('productDetail.fields.barcode'), value: product.barcode || '-' },
      {
        key: t('productDetail.fields.status'),
        value: product.status ? (
          <Tag color={STATUS_TAG_COLORS[product.status]}>
            {t(`products.status.${product.status}`)}
          </Tag>
        ) : (
          '-'
        ),
      },
      { key: t('productDetail.fields.description'), value: product.description || '-' },
    ]

    return <Descriptions data={data} row className="product-basic-info" />
  }

  // Render price info
  const renderPriceInfo = () => {
    if (!product) return null

    const data = [
      {
        key: t('productDetail.fields.purchasePrice'),
        value: product.purchase_price !== undefined ? formatCurrency(product.purchase_price) : '-',
      },
      {
        key: t('productDetail.fields.sellingPrice'),
        value: product.selling_price !== undefined ? formatCurrency(product.selling_price) : '-',
      },
      {
        key: t('productDetail.fields.profitMargin'),
        value: profitMargin !== null ? `${profitMargin.toFixed(2)}%` : '-',
      },
    ]

    return <Descriptions data={data} row className="product-price-info" />
  }

  // Render stock settings
  const renderStockSettings = () => {
    if (!product) return null

    const data = [
      {
        key: t('productDetail.fields.minStock'),
        value: product.min_stock !== undefined ? product.min_stock.toString() : '-',
      },
      {
        key: t('productDetail.fields.sortOrder'),
        value: product.sort_order !== undefined ? product.sort_order.toString() : '-',
      },
    ]

    return <Descriptions data={data} row className="product-stock-settings" />
  }

  // Render timestamps
  const renderTimestamps = () => {
    if (!product) return null

    const data = [
      {
        key: t('productDetail.fields.createdAt'),
        value: product.created_at ? formatDateTime(product.created_at) : '-',
      },
      {
        key: t('productDetail.fields.updatedAt'),
        value: product.updated_at ? formatDateTime(product.updated_at) : '-',
      },
    ]

    return <Descriptions data={data} row className="product-timestamps" />
  }

  // Render action buttons based on status
  const renderActions = () => {
    if (!product) return null

    const status = product.status

    return (
      <Space>
        {status !== 'discontinued' && (
          <Button icon={<IconEdit />} onClick={handleEdit} disabled={actionLoading}>
            {t('common:actions.edit')}
          </Button>
        )}
        {status === 'inactive' && (
          <Button
            type="primary"
            icon={<IconPlay />}
            onClick={handleActivate}
            loading={actionLoading}
          >
            {t('products.actions.activate')}
          </Button>
        )}
        {status === 'active' && (
          <Button
            type="warning"
            icon={<IconStop />}
            onClick={handleDeactivate}
            loading={actionLoading}
          >
            {t('products.actions.deactivate')}
          </Button>
        )}
        {status !== 'discontinued' && (
          <Button type="danger" onClick={handleDiscontinue} loading={actionLoading}>
            {t('products.actions.discontinue')}
          </Button>
        )}
        <Button type="danger" icon={<IconDelete />} onClick={handleDelete} loading={actionLoading}>
          {t('products.actions.delete')}
        </Button>
      </Space>
    )
  }

  if (loading) {
    return (
      <Container size="lg" className="product-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
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
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={() => navigate('/catalog/products')}
          >
            {t('productDetail.back')}
          </Button>
          <Title heading={4} className="page-title">
            {t('productDetail.title')}
          </Title>
          {product.status && (
            <Tag color={STATUS_TAG_COLORS[product.status]} size="large">
              {t(`products.status.${product.status}`)}
            </Tag>
          )}
        </div>
        <div className="header-right">{renderActions()}</div>
      </div>

      {/* Product Name Display */}
      <Card className="product-name-card">
        <div className="product-name-display">
          <Text className="product-code">{product.code}</Text>
          <Title heading={3} className="product-name">
            {product.name}
          </Title>
          {product.barcode && <Text type="secondary">{product.barcode}</Text>}
        </div>
      </Card>

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
