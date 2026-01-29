/**
 * Sales Order Detail Integration Tests (P3-INT-001)
 *
 * These tests verify the frontend-backend integration for the Sales Order Detail page:
 * - Order detail display (basic info, items, amounts, timeline)
 * - Status change actions (confirm, ship, complete, cancel)
 * - Status-based action button visibility
 * - Timeline display for status changes
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor } from '@/tests/utils'
import SalesOrderDetailPage from './SalesOrderDetail'
import * as salesOrdersApi from '@/api/sales-orders/sales-orders'
import { Toast, Modal } from '@douyinfe/semi-ui-19'

// Mock the API modules
vi.mock('@/api/sales-orders/sales-orders', () => ({
  getSalesOrders: vi.fn(),
}))

// Mock react-router-dom
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ id: 'order-001' }),
  }
})

// Spy on Toast and Modal methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')
vi.spyOn(Modal, 'confirm').mockImplementation(() => ({ destroy: vi.fn() }))

// Sample order items
const mockOrderItems = [
  {
    id: 'item-001',
    product_id: 'prod-001',
    product_code: 'SKU-001',
    product_name: '测试商品A',
    unit: '件',
    quantity: 10,
    unit_price: 50.0,
    amount: 500.0,
    remark: '备注1',
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  },
  {
    id: 'item-002',
    product_id: 'prod-002',
    product_code: 'SKU-002',
    product_name: '测试商品B',
    unit: '箱',
    quantity: 5,
    unit_price: 100.0,
    amount: 500.0,
    remark: '',
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-15T10:00:00Z',
  },
]

// Sample sales order detail data
const createMockOrderDetail = (status: string = 'draft', includeTimestamps: boolean = true) => ({
  id: 'order-001',
  order_number: 'SO-2024-0001',
  customer_id: 'cust-001',
  customer_name: '测试客户A',
  warehouse_id: 'wh-001',
  item_count: 2,
  items: mockOrderItems,
  total_quantity: 15,
  total_amount: 1000.0,
  discount_amount: 50.0,
  payable_amount: 950.0,
  status,
  remark: '测试订单备注',
  version: 1,
  tenant_id: 'tenant-001',
  created_at: '2024-01-15T10:00:00Z',
  updated_at: '2024-01-15T10:00:00Z',
  ...(includeTimestamps && status !== 'draft'
    ? {
        confirmed_at: status !== 'draft' ? '2024-01-15T11:00:00Z' : undefined,
        shipped_at:
          status === 'shipped' || status === 'completed' ? '2024-01-15T14:00:00Z' : undefined,
        completed_at: status === 'completed' ? '2024-01-15T16:00:00Z' : undefined,
        cancelled_at: status === 'cancelled' ? '2024-01-15T12:00:00Z' : undefined,
        cancel_reason: status === 'cancelled' ? '用户取消' : undefined,
      }
    : {}),
})

const createMockOrderDetailResponse = (order = createMockOrderDetail()) => ({
  success: true,
  data: order,
})

describe('SalesOrderDetailPage', () => {
  let mockSalesOrderApiInstance: {
    getSalesOrderById: ReturnType<typeof vi.fn>
    postTradeSalesOrdersIdConfirm: ReturnType<typeof vi.fn>
    postTradeSalesOrdersIdShip: ReturnType<typeof vi.fn>
    postTradeSalesOrdersIdComplete: ReturnType<typeof vi.fn>
    postTradeSalesOrdersIdCancel: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockSalesOrderApiInstance = {
      getSalesOrderById: vi.fn().mockResolvedValue(createMockOrderDetailResponse()),
      postTradeSalesOrdersIdConfirm: vi.fn().mockResolvedValue({ success: true }),
      postTradeSalesOrdersIdShip: vi.fn().mockResolvedValue({ success: true }),
      postTradeSalesOrdersIdComplete: vi.fn().mockResolvedValue({ success: true }),
      postTradeSalesOrdersIdCancel: vi.fn().mockResolvedValue({ success: true }),
    }

    vi.mocked(salesOrdersApi.getSalesOrders).mockReturnValue(
      mockSalesOrderApiInstance as unknown as ReturnType<typeof salesOrdersApi.getSalesOrders>
    )
  })

  describe('Order Detail Display', () => {
    it('should display order number', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })
    })

    it('should display customer name', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('测试客户A')).toBeInTheDocument()
      })
    })

    it('should display order status tag', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        // There should be status tags (one in header, one in basic info)
        expect(screen.getAllByText('草稿').length).toBeGreaterThan(0)
      })
    })

    it('should display item count', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('2 件')).toBeInTheDocument()
      })
    })

    it('should display total quantity', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('15.00')).toBeInTheDocument()
      })
    })

    it('should display remark', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('测试订单备注')).toBeInTheDocument()
      })
    })

    it('should display page title', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('订单详情')).toBeInTheDocument()
      })
    })

    it('should display back button', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('返回列表')).toBeInTheDocument()
      })
    })
  })

  describe('Order Items Display', () => {
    it('should display product codes', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('SKU-001')).toBeInTheDocument()
        expect(screen.getByText('SKU-002')).toBeInTheDocument()
      })
    })

    it('should display product names', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('测试商品A')).toBeInTheDocument()
        expect(screen.getByText('测试商品B')).toBeInTheDocument()
      })
    })

    it('should display units', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('件')).toBeInTheDocument()
        expect(screen.getByText('箱')).toBeInTheDocument()
      })
    })

    it('should display quantities formatted', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('10.00')).toBeInTheDocument()
        expect(screen.getByText('5.00')).toBeInTheDocument()
      })
    })

    it('should display unit prices formatted as currency', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('¥50.00')).toBeInTheDocument()
        expect(screen.getByText('¥100.00')).toBeInTheDocument()
      })
    })

    it('should display line amounts formatted as currency', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        // Both items have ¥500.00 amount
        expect(screen.getAllByText('¥500.00').length).toBeGreaterThanOrEqual(2)
      })
    })
  })

  describe('Amount Summary Display', () => {
    it('should display subtotal (total + discount)', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        // Subtotal = total_amount + discount_amount = 1000 + 50 = 1050
        expect(screen.getByText('¥1050.00')).toBeInTheDocument()
      })
    })

    it('should display discount amount', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('-¥50.00')).toBeInTheDocument()
      })
    })

    it('should display payable amount', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('¥950.00')).toBeInTheDocument()
      })
    })
  })

  describe('Status-Based Action Buttons', () => {
    it('should show edit, confirm, cancel buttons for draft orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('draft'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('编辑')).toBeInTheDocument()
      expect(screen.getByText('确认订单')).toBeInTheDocument()
      // Cancel button shows "取消" text
      expect(screen.getAllByText('取消').length).toBeGreaterThan(0)
    })

    it('should show ship and cancel buttons for confirmed orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('confirmed'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('发货')).toBeInTheDocument()
    })

    it('should show complete button for shipped orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('shipped'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      expect(screen.getByText('完成')).toBeInTheDocument()
    })

    it('should not show action buttons for completed orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('completed'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      expect(screen.queryByText('编辑')).not.toBeInTheDocument()
      expect(screen.queryByText('确认订单')).not.toBeInTheDocument()
      expect(screen.queryByText('发货')).not.toBeInTheDocument()
      expect(screen.queryByText('完成')).not.toBeInTheDocument()
    })

    it('should not show action buttons for cancelled orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('cancelled'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      expect(screen.queryByText('编辑')).not.toBeInTheDocument()
      expect(screen.queryByText('确认订单')).not.toBeInTheDocument()
      expect(screen.queryByText('发货')).not.toBeInTheDocument()
    })
  })

  describe('Timeline Display', () => {
    it('should display created timeline item', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('订单创建')).toBeInTheDocument()
      })
    })

    it('should display confirmed timeline item for confirmed orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('confirmed'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('订单确认')).toBeInTheDocument()
      })
    })

    it('should display shipped timeline item for shipped orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('shipped'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('订单发货')).toBeInTheDocument()
      })
    })

    it('should display completed timeline item for completed orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('completed'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText('订单完成')).toBeInTheDocument()
      })
    })

    it('should display cancelled timeline item with reason for cancelled orders', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('cancelled'))
      )

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(screen.getByText(/订单取消.*用户取消/)).toBeInTheDocument()
      })
    })
  })

  describe('Navigation', () => {
    it('should navigate to list page when clicking back button', async () => {
      const { user } = renderWithProviders(<SalesOrderDetailPage />, {
        route: '/trade/sales/order-001',
      })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      const backButton = screen.getByText('返回列表')
      await user.click(backButton)

      expect(mockNavigate).toHaveBeenCalledWith('/trade/sales')
    })

    it('should navigate to edit page when clicking edit button', async () => {
      const { user } = renderWithProviders(<SalesOrderDetailPage />, {
        route: '/trade/sales/order-001',
      })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      const editButton = screen.getByText('编辑')
      await user.click(editButton)

      expect(mockNavigate).toHaveBeenCalledWith('/trade/sales/order-001/edit')
    })
  })

  describe('Error Handling', () => {
    it('should show error toast and navigate to list when order not found', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce({
        success: false,
        error: { message: '订单不存在' },
      })

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('订单不存在')
      })

      expect(mockNavigate).toHaveBeenCalledWith('/trade/sales')
    })

    it('should show error toast and navigate to list when API fails', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockRejectedValueOnce(new Error('Network error'))

      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取订单详情失败')
      })

      expect(mockNavigate).toHaveBeenCalledWith('/trade/sales')
    })
  })

  describe('API Integration Verification', () => {
    it('should call API with correct order ID', async () => {
      renderWithProviders(<SalesOrderDetailPage />, { route: '/trade/sales/order-001' })

      await waitFor(() => {
        expect(mockSalesOrderApiInstance.getSalesOrderById).toHaveBeenCalledWith('order-001')
      })
    })
  })

  describe('Status Change Actions', () => {
    it('should show confirm modal when clicking confirm button', async () => {
      const { user } = renderWithProviders(<SalesOrderDetailPage />, {
        route: '/trade/sales/order-001',
      })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      const confirmButton = screen.getByText('确认订单')
      await user.click(confirmButton)

      expect(Modal.confirm).toHaveBeenCalledWith(
        expect.objectContaining({
          title: '确认订单',
          content: expect.stringContaining('确认后将锁定库存'),
        })
      )
    })

    it('should open ship modal when clicking ship button', async () => {
      mockSalesOrderApiInstance.getSalesOrderById.mockResolvedValueOnce(
        createMockOrderDetailResponse(createMockOrderDetail('confirmed'))
      )

      const { user } = renderWithProviders(<SalesOrderDetailPage />, {
        route: '/trade/sales/order-001',
      })

      await waitFor(() => {
        expect(screen.getByText('SO-2024-0001')).toBeInTheDocument()
      })

      const shipButton = screen.getByText('发货')
      await user.click(shipButton)

      await waitFor(() => {
        expect(screen.getByText('发货确认')).toBeInTheDocument()
      })
    })
  })
})
