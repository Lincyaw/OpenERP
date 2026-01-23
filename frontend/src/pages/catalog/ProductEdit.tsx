import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Toast } from '@douyinfe/semi-ui'
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
  const api = useMemo(() => getProducts(), [])

  const [loading, setLoading] = useState(true)
  const [product, setProduct] = useState<HandlerProductResponse | null>(null)

  // Load product data
  useEffect(() => {
    if (!id) {
      Toast.error('商品ID无效')
      navigate('/catalog/products')
      return
    }

    const loadProduct = async () => {
      setLoading(true)
      try {
        const response = await api.getCatalogProductsId(id)
        if (response.success && response.data) {
          setProduct(response.data)
        } else {
          Toast.error(response.error?.message || '加载商品失败')
          navigate('/catalog/products')
        }
      } catch {
        Toast.error('加载商品失败')
        navigate('/catalog/products')
      } finally {
        setLoading(false)
      }
    }

    loadProduct()
  }, [id, api, navigate])

  if (loading) {
    return (
      <Container size="md" style={{ padding: '48px 0', textAlign: 'center' }}>
        <Spin size="large" tip="加载中..." />
      </Container>
    )
  }

  if (!product) {
    return null
  }

  return <ProductForm productId={id} initialData={product} />
}
