import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Toast } from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import { ProductForm } from '@/features/catalog/ProductForm'
import { getProducts } from '@/api/products/products'
import type { HandlerProductResponse } from '@/api/models'
import { Container } from '@/components/common/layout'

/**
 * Product edit page
 *
 * Loads product data by ID and renders the ProductForm in edit mode
 */
export default function ProductEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation(['catalog', 'common'])
  const api = useMemo(() => getProducts(), [])

  const [loading, setLoading] = useState(true)
  const [product, setProduct] = useState<HandlerProductResponse | null>(null)

  // Load product data
  useEffect(() => {
    if (!id) {
      Toast.error(t('products.messages.invalidId'))
      navigate('/catalog/products')
      return
    }

    const loadProduct = async () => {
      setLoading(true)
      try {
        const response = await api.getProductById(id)
        if (response.success && response.data) {
          setProduct(response.data)
        } else {
          Toast.error(response.error?.message || t('products.messages.loadError'))
          navigate('/catalog/products')
        }
      } catch {
        Toast.error(t('products.messages.loadError'))
        navigate('/catalog/products')
      } finally {
        setLoading(false)
      }
    }

    loadProduct()
  }, [id, api, navigate, t])

  if (loading) {
    return (
      <Container size="md" style={{ padding: '48px 0', textAlign: 'center' }}>
        <Spin size="large" tip={t('common:messages.loading')} />
      </Container>
    )
  }

  if (!product) {
    return null
  }

  return <ProductForm productId={id} initialData={product} />
}
