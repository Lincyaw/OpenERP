import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Toast } from '@douyinfe/semi-ui-19'
import { PurchaseOrderForm } from '@/features/trade/PurchaseOrderForm'
import { getPurchaseOrders } from '@/api/purchase-orders/purchase-orders'
import type { HandlerPurchaseOrderResponse } from '@/api/models'
import { useI18n } from '@/hooks/useI18n'

/**
 * Purchase order edit page
 *
 * Fetches order data by ID and renders the PurchaseOrderForm in edit mode.
 * Only draft orders can be edited.
 */
export default function PurchaseOrderEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const api = useMemo(() => getPurchaseOrders(), [])
  const [orderData, setOrderData] = useState<HandlerPurchaseOrderResponse | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchOrder = async () => {
      if (!id) {
        navigate('/trade/purchase')
        return
      }

      setLoading(true)
      try {
        const response = await api.getPurchaseOrderById(id)
        if (response.success && response.data) {
          // Check if order is in draft status (only draft orders can be edited)
          if (response.data.status !== 'draft') {
            Toast.error(t('orderForm.messages.onlyDraftEditable'))
            navigate('/trade/purchase')
            return
          }
          setOrderData(response.data)
        } else {
          Toast.error(t('purchaseOrderDetail.messages.notExist'))
          navigate('/trade/purchase')
        }
      } catch {
        Toast.error(t('purchaseOrderDetail.messages.fetchError'))
        navigate('/trade/purchase')
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

  return <PurchaseOrderForm orderId={id} initialData={orderData} />
}
