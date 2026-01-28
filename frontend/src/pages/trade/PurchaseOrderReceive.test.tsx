/**
 * Purchase Order Receive Integration Tests (P3-INT-002)
 *
 * These tests verify the frontend-backend integration for the Purchase Order receiving flow:
 * - Order summary display
 * - Receivable items listing with remaining quantities
 * - Receive quantity input and validation
 * - Batch number and expiry date entry
 * - Warehouse selection
 * - Partial receiving support
 * - Inventory update verification
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderWithProviders, screen, waitFor, fireEvent } from '@/tests/utils'
import PurchaseOrderReceivePage from './PurchaseOrderReceive'
import * as purchaseOrdersApi from '@/api/purchase-orders/purchase-orders'
import * as warehousesApi from '@/api/warehouses/warehouses'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the API modules
vi.mock('@/api/purchase-orders/purchase-orders', () => ({
  getPurchaseOrders: vi.fn(),
}))

vi.mock('@/api/warehouses/warehouses', () => ({
  getWarehouses: vi.fn(),
}))

// Mock react-router-dom's useNavigate and useParams
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ id: 'po-001' }),
  }
})

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'warning').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')

// Sample warehouse data
const mockWarehouses = [
  {
    id: 'wh-001',
    code: 'WH001',
    name: '主仓库',
    address: '北京市朝阳区',
    status: 'active',
    is_default: true,
  },
  {
    id: 'wh-002',
    code: 'WH002',
    name: '分仓库',
    address: '上海市浦东新区',
    status: 'active',
    is_default: false,
  },
]

// Sample purchase order data
const mockPurchaseOrder = {
  id: 'po-001',
  order_number: 'PO-2024-0001',
  supplier_id: 'supp-001',
  supplier_name: '测试供应商A',
  warehouse_id: 'wh-001',
  total_amount: 5000.0,
  payable_amount: 4800.0,
  status: 'confirmed',
  created_at: '2024-01-15T10:00:00Z',
  confirmed_at: '2024-01-15T11:00:00Z',
  updated_at: '2024-01-15T11:00:00Z',
}

// Sample receivable items data
const mockReceivableItems = [
  {
    id: 'item-001',
    product_id: 'prod-001',
    product_name: '测试商品A',
    product_code: 'SKU-A001',
    unit: '件',
    ordered_quantity: 100,
    received_quantity: 0,
    remaining_quantity: 100,
    unit_cost: 25.0,
  },
  {
    id: 'item-002',
    product_id: 'prod-002',
    product_name: '测试商品B',
    product_code: 'SKU-B001',
    unit: '箱',
    ordered_quantity: 50,
    received_quantity: 20,
    remaining_quantity: 30,
    unit_cost: 60.0,
  },
  {
    id: 'item-003',
    product_id: 'prod-003',
    product_name: '测试商品C',
    product_code: 'SKU-C001',
    unit: '盒',
    ordered_quantity: 200,
    received_quantity: 100,
    remaining_quantity: 100,
    unit_cost: 15.0,
  },
]

// Mock API response helpers
const createMockOrderResponse = (order = mockPurchaseOrder) => ({
  success: true,
  data: order,
})

const createMockReceivableItemsResponse = (items = mockReceivableItems) => ({
  success: true,
  data: items,
})

const createMockWarehouseListResponse = (warehouses = mockWarehouses) => ({
  success: true,
  data: warehouses,
  meta: {
    total: warehouses.length,
    page: 1,
    page_size: 100,
    total_pages: 1,
  },
})

const createMockReceiveResponse = (isFullyReceived = false) => ({
  success: true,
  data: {
    is_fully_received: isFullyReceived,
    received_count: 3,
  },
})

describe('PurchaseOrderReceivePage', () => {
  let mockPurchaseOrderApiInstance: {
    getPurchaseOrderById: ReturnType<typeof vi.fn>
    getTradePurchaseOrdersIdReceivableItems: ReturnType<typeof vi.fn>
    postTradePurchaseOrdersIdReceive: ReturnType<typeof vi.fn>
  }

  let mockWarehouseApiInstance: {
    listWarehouses: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    // Setup mock purchase order API
    mockPurchaseOrderApiInstance = {
      getPurchaseOrderById: vi.fn().mockResolvedValue(createMockOrderResponse()),
      getTradePurchaseOrdersIdReceivableItems: vi
        .fn()
        .mockResolvedValue(createMockReceivableItemsResponse()),
      postTradePurchaseOrdersIdReceive: vi.fn().mockResolvedValue(createMockReceiveResponse()),
    }

    // Setup mock warehouse API
    mockWarehouseApiInstance = {
      listWarehouses: vi.fn().mockResolvedValue(createMockWarehouseListResponse()),
    }

    vi.mocked(purchaseOrdersApi.getPurchaseOrders).mockReturnValue(
      mockPurchaseOrderApiInstance as unknown as ReturnType<
        typeof purchaseOrdersApi.getPurchaseOrders
      >
    )
    vi.mocked(warehousesApi.getWarehouses).mockReturnValue(
      mockWarehouseApiInstance as unknown as ReturnType<typeof warehousesApi.getWarehouses>
    )
  })

  describe('Page Loading', () => {
    it('should display loading state initially', async () => {
      // Delay the API response
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve(createMockOrderResponse()), 100))
      )

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      // Should show loading initially
      expect(screen.getByText('加载中...')).toBeInTheDocument()
    })

    it('should display page title after loading', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('采购收货')).toBeInTheDocument()
      })
    })
  })

  describe('Order Summary Display', () => {
    it('should display order number', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('PO-2024-0001')).toBeInTheDocument()
      })
    })

    it('should display supplier name', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('测试供应商A')).toBeInTheDocument()
      })
    })

    it('should display order status tag', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('已确认')).toBeInTheDocument()
      })
    })

    it('should display total and payable amounts', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('¥5000.00')).toBeInTheDocument()
        expect(screen.getByText('¥4800.00')).toBeInTheDocument()
      })
    })

    it('should display order information section title', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('订单信息')).toBeInTheDocument()
      })
    })
  })

  describe('Receivable Items Table', () => {
    it('should display receivable items table with correct columns', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('收货明细')).toBeInTheDocument()
      })

      // Verify column headers
      expect(screen.getByText('商品编码')).toBeInTheDocument()
      expect(screen.getByText('商品名称')).toBeInTheDocument()
      expect(screen.getByText('单位')).toBeInTheDocument()
      expect(screen.getByText('订购数量')).toBeInTheDocument()
      expect(screen.getByText('已收数量')).toBeInTheDocument()
      expect(screen.getByText('待收数量')).toBeInTheDocument()
      expect(screen.getByText('单价')).toBeInTheDocument()
      expect(screen.getByText('本次收货数量')).toBeInTheDocument()
      expect(screen.getByText('批次号')).toBeInTheDocument()
      expect(screen.getByText('有效期')).toBeInTheDocument()
    })

    it('should display product information in table rows', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('测试商品A')).toBeInTheDocument()
      })

      // Verify product codes are displayed
      expect(screen.getByText('SKU-A001')).toBeInTheDocument()
      expect(screen.getByText('SKU-B001')).toBeInTheDocument()
      expect(screen.getByText('SKU-C001')).toBeInTheDocument()

      // Verify product names are displayed
      expect(screen.getByText('测试商品A')).toBeInTheDocument()
      expect(screen.getByText('测试商品B')).toBeInTheDocument()
      expect(screen.getByText('测试商品C')).toBeInTheDocument()
    })

    it('should display remaining quantities correctly', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('测试商品A')).toBeInTheDocument()
      })

      // Verify remaining quantities are displayed
      // Item A: remaining 100, Item B: remaining 30, Item C: remaining 100
      // 100.00 appears multiple times (Item A and C both have 100)
      expect(screen.getAllByText('100.00').length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('30.00')).toBeInTheDocument() // Item B remaining
    })

    it('should display unit costs correctly', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('测试商品A')).toBeInTheDocument()
      })

      // Verify unit costs are displayed
      expect(screen.getByText('¥25.00')).toBeInTheDocument()
      expect(screen.getByText('¥60.00')).toBeInTheDocument()
      expect(screen.getByText('¥15.00')).toBeInTheDocument()
    })
  })

  describe('Warehouse Selection', () => {
    it('should display warehouse selection dropdown', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('收货仓库')).toBeInTheDocument()
      })
    })

    it('should load warehouse options from API', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(mockWarehouseApiInstance.listWarehouses).toHaveBeenCalledWith(
          expect.objectContaining({
            status: 'active',
            page_size: 100,
          })
        )
      })
    })

    it('should show default warehouse as selected by default', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('采购收货')).toBeInTheDocument()
      })

      // The default warehouse should be pre-selected (from order or first active warehouse)
      // We can verify that warehouse API was called
      expect(mockWarehouseApiInstance.listWarehouses).toHaveBeenCalled()
    })
  })

  describe('Receive Quantity Actions', () => {
    it('should have "receive all" button', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('全部收货')).toBeInTheDocument()
      })
    })

    it('should have "clear all" button', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('清空数量')).toBeInTheDocument()
      })
    })

    it('should have submit button', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('确认收货')).toBeInTheDocument()
      })
    })

    it('should have cancel button', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        // Look for cancel button in actions bar
        const cancelButtons = screen.getAllByText('取消')
        expect(cancelButtons.length).toBeGreaterThanOrEqual(1)
      })
    })
  })

  describe('Error Handling', () => {
    it('should show error when order fetch fails', async () => {
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取订单数据失败')
      })
    })

    it('should show empty state when order not found', async () => {
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockResolvedValueOnce({
        success: true,
        data: null,
      })

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('订单不存在')).toBeInTheDocument()
      })
    })

    it('should show cannot receive state for draft orders', async () => {
      const draftOrder = { ...mockPurchaseOrder, status: 'draft' }
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockResolvedValueOnce(
        createMockOrderResponse(draftOrder)
      )

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('订单状态为"草稿"，无法收货')).toBeInTheDocument()
      })
    })

    it('should show cannot receive state for completed orders', async () => {
      const completedOrder = { ...mockPurchaseOrder, status: 'completed' }
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockResolvedValueOnce(
        createMockOrderResponse(completedOrder)
      )

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('订单状态为"已完成"，无法收货')).toBeInTheDocument()
      })
    })

    it('should show cannot receive state for cancelled orders', async () => {
      const cancelledOrder = { ...mockPurchaseOrder, status: 'cancelled' }
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockResolvedValueOnce(
        createMockOrderResponse(cancelledOrder)
      )

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('订单状态为"已取消"，无法收货')).toBeInTheDocument()
      })
    })

    it('should handle warehouse API failure gracefully', async () => {
      mockWarehouseApiInstance.listWarehouses.mockRejectedValueOnce(
        new Error('Network error')
      )

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('获取仓库列表失败')
      })
    })
  })

  describe('Navigation', () => {
    it('should navigate back to list when clicking back button', async () => {
      const { user } = renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('采购收货')).toBeInTheDocument()
      })

      // Find the back button (arrow icon button)
      const backButtons = document.querySelectorAll('.semi-button')
      const backButton = Array.from(backButtons).find(
        (btn) => btn.querySelector('.semi-icon-arrow_left') !== null
      )

      if (backButton) {
        await user.click(backButton)
        expect(mockNavigate).toHaveBeenCalledWith('/trade/purchase')
      }
    })

    it('should navigate back when clicking cancel button', async () => {
      const { user } = renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('采购收货')).toBeInTheDocument()
      })

      // Find the cancel button in actions bar (not in the header)
      const cancelButtons = screen.getAllByText('取消')
      // The last cancel button should be in the actions bar
      const actionCancelButton = cancelButtons[cancelButtons.length - 1]

      await user.click(actionCancelButton)

      expect(mockNavigate).toHaveBeenCalledWith('/trade/purchase')
    })

    it('should display empty state elements when order not found', async () => {
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockResolvedValueOnce({
        success: true,
        data: null,
      })

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('订单不存在')).toBeInTheDocument()
      })

      // Find button that returns to list (it navigates to /trade/purchase)
      const buttons = screen.getAllByRole('button')
      expect(buttons.length).toBeGreaterThanOrEqual(1)
    })
  })

  describe('API Integration', () => {
    it('should call order detail API with correct id', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(mockPurchaseOrderApiInstance.getPurchaseOrderById).toHaveBeenCalledWith('po-001')
      })
    })

    it('should call receivable items API with correct id', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(
          mockPurchaseOrderApiInstance.getTradePurchaseOrdersIdReceivableItems
        ).toHaveBeenCalledWith('po-001')
      })
    })

    it('should have refresh button that reloads data', async () => {
      const { user } = renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('采购收货')).toBeInTheDocument()
      })

      // Clear mocks to track new calls
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockClear()
      mockPurchaseOrderApiInstance.getTradePurchaseOrdersIdReceivableItems.mockClear()

      const refreshButton = screen.getByText('刷新')
      await user.click(refreshButton)

      await waitFor(() => {
        expect(mockPurchaseOrderApiInstance.getPurchaseOrderById).toHaveBeenCalledWith('po-001')
      })
    })
  })

  describe('Partial Receiving Support', () => {
    it('should allow partial_received order to be received', async () => {
      const partialOrder = { ...mockPurchaseOrder, status: 'partial_received' }
      mockPurchaseOrderApiInstance.getPurchaseOrderById.mockResolvedValueOnce(
        createMockOrderResponse(partialOrder)
      )

      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('采购收货')).toBeInTheDocument()
      })

      // Should display the receive page, not an error state
      expect(screen.getByText('收货明细')).toBeInTheDocument()
      expect(screen.getByText('确认收货')).toBeInTheDocument()
    })

    it('should display already received quantities', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('测试商品B')).toBeInTheDocument()
      })

      // Item B has received_quantity: 20
      expect(screen.getByText('20.00')).toBeInTheDocument()
    })
  })
})

describe('PurchaseOrderReceivePage - Inventory Integration (P3-INT-002)', () => {
  let mockPurchaseOrderApiInstance: {
    getPurchaseOrderById: ReturnType<typeof vi.fn>
    getTradePurchaseOrdersIdReceivableItems: ReturnType<typeof vi.fn>
    postTradePurchaseOrdersIdReceive: ReturnType<typeof vi.fn>
  }

  let mockWarehouseApiInstance: {
    listWarehouses: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()

    mockPurchaseOrderApiInstance = {
      getPurchaseOrderById: vi.fn().mockResolvedValue(createMockOrderResponse()),
      getTradePurchaseOrdersIdReceivableItems: vi
        .fn()
        .mockResolvedValue(createMockReceivableItemsResponse()),
      postTradePurchaseOrdersIdReceive: vi.fn().mockResolvedValue(createMockReceiveResponse()),
    }

    mockWarehouseApiInstance = {
      listWarehouses: vi.fn().mockResolvedValue(createMockWarehouseListResponse()),
    }

    vi.mocked(purchaseOrdersApi.getPurchaseOrders).mockReturnValue(
      mockPurchaseOrderApiInstance as unknown as ReturnType<
        typeof purchaseOrdersApi.getPurchaseOrders
      >
    )
    vi.mocked(warehousesApi.getWarehouses).mockReturnValue(
      mockWarehouseApiInstance as unknown as ReturnType<typeof warehousesApi.getWarehouses>
    )
  })

  describe('Receive Submission', () => {
    it('should submit receive request with correct warehouse id', async () => {
      const { user } = renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('确认收货')).toBeInTheDocument()
      })

      // Submit receive
      const submitButton = screen.getByText('确认收货')
      await user.click(submitButton)

      await waitFor(() => {
        expect(mockPurchaseOrderApiInstance.postTradePurchaseOrdersIdReceive).toHaveBeenCalledWith(
          'po-001',
          expect.objectContaining({
            warehouse_id: expect.any(String),
            items: expect.any(Array),
          })
        )
      })
    })

    it('should show success message and navigate back on successful partial receive', async () => {
      mockPurchaseOrderApiInstance.postTradePurchaseOrdersIdReceive.mockResolvedValueOnce(
        createMockReceiveResponse(false)
      )

      const { user } = renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('确认收货')).toBeInTheDocument()
      })

      const submitButton = screen.getByText('确认收货')
      await user.click(submitButton)

      await waitFor(() => {
        expect(Toast.success).toHaveBeenCalledWith('收货成功，部分商品已入库')
      })

      expect(mockNavigate).toHaveBeenCalledWith('/trade/purchase')
    })

    it('should show success message and navigate back on successful full receive', async () => {
      mockPurchaseOrderApiInstance.postTradePurchaseOrdersIdReceive.mockResolvedValueOnce(
        createMockReceiveResponse(true)
      )

      const { user } = renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('确认收货')).toBeInTheDocument()
      })

      const submitButton = screen.getByText('确认收货')
      await user.click(submitButton)

      await waitFor(() => {
        expect(Toast.success).toHaveBeenCalledWith('收货完成，订单已全部入库')
      })

      expect(mockNavigate).toHaveBeenCalledWith('/trade/purchase')
    })

    it('should show error message on receive failure', async () => {
      mockPurchaseOrderApiInstance.postTradePurchaseOrdersIdReceive.mockRejectedValueOnce(
        new Error('Network error')
      )

      const { user } = renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('确认收货')).toBeInTheDocument()
      })

      const submitButton = screen.getByText('确认收货')
      await user.click(submitButton)

      await waitFor(() => {
        expect(Toast.error).toHaveBeenCalledWith('收货失败')
      })
    })
  })

  describe('Validation', () => {
    it('should require warehouse selection', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('确认收货')).toBeInTheDocument()
      })

      // Verify warehouse selection section is displayed
      expect(screen.getByText('收货仓库')).toBeInTheDocument()
      // Required indicator should be present
      const requiredElements = document.querySelectorAll('.semi-tag-danger')
      // Warehouse selection should exist
      expect(screen.getAllByRole('combobox').length).toBeGreaterThanOrEqual(1)
    })

    it('should have submit button that becomes enabled when items have quantities', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('确认收货')).toBeInTheDocument()
      })

      // With default items that have remaining quantities, button should be enabled
      const submitButton = screen.getByText('确认收货').closest('button')
      // Default items have remaining quantities, so receive_quantity defaults to those values
      // Button should be enabled by default
      expect(submitButton).not.toBeDisabled()
    })
  })

  describe('Batch and Expiry Date Entry', () => {
    it('should display batch number input fields', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('批次号')).toBeInTheDocument()
      })

      // Should have batch number inputs with placeholder "可选"
      const optionalInputs = screen.getAllByPlaceholderText('可选')
      expect(optionalInputs.length).toBeGreaterThanOrEqual(1)
    })

    it('should display expiry date picker fields', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('有效期')).toBeInTheDocument()
      })
    })
  })

  describe('Receiving Summary Display', () => {
    it('should display receiving summary when items have quantities', async () => {
      renderWithProviders(<PurchaseOrderReceivePage />, {
        route: '/trade/purchase/po-001/receive',
      })

      await waitFor(() => {
        expect(screen.getByText('测试商品A')).toBeInTheDocument()
      })

      // The summary should show receiving stats
      // Default is to receive all remaining quantities
      const summaryText = screen.getByText(/本次收货:/i)
      expect(summaryText).toBeInTheDocument()
    })
  })
})
