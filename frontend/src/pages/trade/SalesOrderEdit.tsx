import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Toast } from '@douyinfe/semi-ui-19'
import { SalesOrderForm } from '@/features/trade/SalesOrderForm'
import { getSalesOrders } from '@/api/sales-orders/sales-orders'
import type { HandlerSalesOrderResponse } from '@/api/models'
import { useI18n } from '@/hooks/useI18n'

/**
 * Sales order edit page
 *
 * Fetches order data by ID and renders the SalesOrderForm in edit mode
 */
export default function SalesOrderEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const api = useMemo(() => getSalesOrders(), [])
  const [orderData, setOrderData] = useState<HandlerSalesOrderResponse | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchOrder = async () => {
      if (!id) {
        navigate('/trade/sales')
        return
      }

      setLoading(true)
      try {
        const response = await api.getTradeSalesOrdersId(id)
        if (response.success && response.data) {
          // Check if order is in draft status (only draft orders can be edited)
          if (response.data.status !== 'draft') {
            Toast.error(t('orderForm.messages.onlyDraftEditable'))
            navigate('/trade/sales')
            return
          }
          setOrderData(response.data)
        } else {
          Toast.error(t('salesOrder.messages.notExist'))
          navigate('/trade/sales')
        }
      } catch {
        Toast.error(t('salesOrder.messages.fetchDetailError'))
        navigate('/trade/sales')
      } finally {
        setLoading(false)
      }
    }

    fetchOrder()
  }, [id, api, navigate, t])

  if (loading) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '100%',
          minHeight: 400,
        }}
      >
        <Spin size="large" tip={t('receive.loading')} />
      </div>
    )
  }

  if (!orderData || !id) {
    return null
  }

  return <SalesOrderForm orderId={id} initialData={orderData} />
}
