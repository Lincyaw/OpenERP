import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Toast } from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import { ProductForm } from '@/features/catalog/ProductForm'
import { getProductById } from '@/api/products/products'
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
        const response = await getProductById(id)
        if (response.status === 200 && response.data.success && response.data.data) {
          setProduct(response.data.data)
        } else {
          Toast.error(response.data.error?.message || t('products.messages.loadError'))
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
  }, [id, navigate, t])

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
