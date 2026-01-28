import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Toast } from '@douyinfe/semi-ui-19'
import { WarehouseForm } from '@/features/partner'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type { HandlerWarehouseResponse } from '@/api/models'

/**
 * Warehouse edit page
 * Fetches warehouse data and passes to WarehouseForm in edit mode
 */
export default function WarehouseEditPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const api = useMemo(() => getWarehouses(), [])

  const [warehouse, setWarehouse] = useState<HandlerWarehouseResponse | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchWarehouse = async () => {
      if (!id) {
        Toast.error('仓库 ID 无效')
        navigate('/partner/warehouses')
        return
      }

      try {
        const response = await api.getWarehouseById(id)
        if (response.success && response.data) {
          setWarehouse(response.data)
        } else {
          Toast.error('仓库不存在')
          navigate('/partner/warehouses')
        }
      } catch {
        Toast.error('获取仓库信息失败')
        navigate('/partner/warehouses')
      } finally {
        setLoading(false)
      }
    }

    fetchWarehouse()
  }, [api, id, navigate])

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '100px 0' }}>
        <Spin size="large" tip="加载中..." />
      </div>
    )
  }

  if (!warehouse) {
    return null
  }

  return <WarehouseForm warehouseId={id} initialData={warehouse} />
}
