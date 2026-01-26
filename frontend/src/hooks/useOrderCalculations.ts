import { useMemo } from 'react'

/**
 * Order item with amount for calculation purposes
 */
export interface OrderItemForCalculation {
  product_id: string
  amount: number
}

/**
 * Calculation results returned by the hook
 */
export interface OrderCalculations {
  /** Total before discount */
  subtotal: number
  /** Amount discounted */
  discountAmount: number
  /** Final total after discount */
  total: number
  /** Number of items with valid products */
  itemCount: number
}

/**
 * Hook to calculate order totals
 *
 * @param items - Array of order items with amount field
 * @param discountPercent - Discount percentage (0-100)
 * @returns Calculated subtotal, discount amount, total, and item count
 *
 * @example
 * function OrderForm() {
 *   const calculations = useOrderCalculations(orderItems, discount)
 *
 *   return (
 *     <div>
 *       <p>Subtotal: ¥{calculations.subtotal}</p>
 *       <p>Discount: -¥{calculations.discountAmount}</p>
 *       <p>Total: ¥{calculations.total}</p>
 *     </div>
 *   )
 * }
 */
export function useOrderCalculations(
  items: OrderItemForCalculation[],
  discountPercent: number
): OrderCalculations {
  return useMemo(() => {
    const subtotal = items.reduce((sum, item) => sum + item.amount, 0)
    const discountAmount = (subtotal * discountPercent) / 100
    const total = subtotal - discountAmount
    return {
      subtotal,
      discountAmount,
      total,
      itemCount: items.filter((item) => item.product_id).length,
    }
  }, [items, discountPercent])
}
