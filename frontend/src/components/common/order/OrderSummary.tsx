import { InputNumber, Typography } from '@douyinfe/semi-ui-19'
import type { OrderCalculations } from '@/hooks/useOrderCalculations'
import { safeToFixed } from '@/utils'

const { Text } = Typography

/**
 * Props for OrderSummary component
 */
export interface OrderSummaryProps {
  /** Calculated order totals */
  calculations: OrderCalculations
  /** Current discount percentage */
  discount: number
  /** Handler for discount changes */
  onDiscountChange: (value: number | string | undefined) => void
  /** Translation function */
  t: (key: string) => string
  /** Optional className */
  className?: string
}

/**
 * Shared order summary component
 *
 * Displays order totals including item count, subtotal, discount, and final total.
 * Includes a discount input for percentage-based discounts.
 *
 * @example
 * <OrderSummary
 *   calculations={calculations}
 *   discount={formData.discount}
 *   onDiscountChange={handleDiscountChange}
 *   t={t}
 * />
 */
export function OrderSummary({
  calculations,
  discount,
  onDiscountChange,
  t,
  className,
}: OrderSummaryProps) {
  return (
    <div className={`summary-row ${className || ''}`}>
      <div className="form-field discount-field">
        <label className="form-label">{t('orderForm.summary.discount')} (%)</label>
        <InputNumber
          value={discount}
          onChange={onDiscountChange}
          min={0}
          max={100}
          precision={2}
          suffix="%"
          style={{ width: 120 }}
        />
      </div>
      <div className="summary-totals">
        <div className="summary-item">
          <Text type="tertiary">{t('orderForm.summary.itemCount')}</Text>
          <Text>
            {calculations.itemCount} {t('orderForm.summary.itemCountUnit')}
          </Text>
        </div>
        <div className="summary-item">
          <Text type="tertiary">{t('orderForm.summary.subtotal')}</Text>
          <Text>¥{safeToFixed(calculations.subtotal)}</Text>
        </div>
        {discount > 0 && (
          <div className="summary-item">
            <Text type="tertiary">
              {t('orderForm.summary.discount')} ({discount}%):
            </Text>
            <Text type="danger">-¥{safeToFixed(calculations.discountAmount)}</Text>
          </div>
        )}
        <div className="summary-item total">
          <Text strong>{t('orderForm.summary.payableAmount')}</Text>
          <Text strong className="total-amount">
            ¥{safeToFixed(calculations.total)}
          </Text>
        </div>
      </div>
    </div>
  )
}

export default OrderSummary
