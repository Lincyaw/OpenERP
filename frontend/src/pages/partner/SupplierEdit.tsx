import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Toast } from '@douyinfe/semi-ui-19'
import { SupplierForm } from '@/features/partner/SupplierForm'
import { getSupplierById } from '@/api/suppliers/suppliers'
import type { HandlerSupplierResponse } from '@/api/models'
import { Container } from '@/components/common/layout'

/**
 * Supplier edit page
 *
 * Loads supplier data by ID and renders the SupplierForm in edit mode
 */
export default function SupplierEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [loading, setLoading] = useState(true)
  const [supplier, setSupplier] = useState<HandlerSupplierResponse | null>(null)

  // Load supplier data
  useEffect(() => {
    if (!id) {
      Toast.error('供应商ID无效')
      navigate('/partner/suppliers')
      return
    }

    const loadSupplier = async () => {
      setLoading(true)
      try {
        const response = await getSupplierById(id)
        if (response.status === 200 && response.data.success && response.data.data) {
          setSupplier(response.data.data)
        } else {
          Toast.error((response.data.error as { message?: string })?.message || '加载供应商失败')
          navigate('/partner/suppliers')
        }
      } catch {
        Toast.error('加载供应商失败')
        navigate('/partner/suppliers')
      } finally {
        setLoading(false)
      }
    }

    loadSupplier()
  }, [id, navigate])

  if (loading) {
    return (
      <Container size="md" style={{ padding: '48px 0', textAlign: 'center' }}>
        <Spin size="large" tip="加载中..." />
      </Container>
    )
  }

  if (!supplier) {
    return null
  }

  return <SupplierForm supplierId={id} initialData={supplier} />
}
