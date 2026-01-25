import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Toast } from '@douyinfe/semi-ui-19'
import { CustomerForm } from '@/features/partner/CustomerForm'
import { getCustomers } from '@/api/customers/customers'
import type { HandlerCustomerResponse } from '@/api/models'
import { Container } from '@/components/common/layout'

/**
 * Customer edit page
 *
 * Loads customer data by ID and renders the CustomerForm in edit mode
 */
export default function CustomerEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const api = useMemo(() => getCustomers(), [])

  const [loading, setLoading] = useState(true)
  const [customer, setCustomer] = useState<HandlerCustomerResponse | null>(null)

  // Load customer data
  useEffect(() => {
    if (!id) {
      Toast.error('客户ID无效')
      navigate('/partner/customers')
      return
    }

    const loadCustomer = async () => {
      setLoading(true)
      try {
        const response = await api.getPartnerCustomersId(id)
        if (response.success && response.data) {
          setCustomer(response.data)
        } else {
          Toast.error(response.error?.message || '加载客户失败')
          navigate('/partner/customers')
        }
      } catch {
        Toast.error('加载客户失败')
        navigate('/partner/customers')
      } finally {
        setLoading(false)
      }
    }

    loadCustomer()
  }, [id, api, navigate])

  if (loading) {
    return (
      <Container size="md" style={{ padding: '48px 0', textAlign: 'center' }}>
        <Spin size="large" tip="加载中..." />
      </Container>
    )
  }

  if (!customer) {
    return null
  }

  return <CustomerForm customerId={id} initialData={customer} />
}
