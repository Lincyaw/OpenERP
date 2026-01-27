// Custom hooks
// Reusable React hooks for the application

// Theme and accessibility hooks
export {
  useThemeManager,
  useFontScale,
  useAccessibilityPreferences,
  applyTheme,
  applyFontScale,
  getSystemTheme,
  prefersReducedMotion,
  prefersHighContrast,
  type Theme,
  type FontScale,
  type ThemeConfig,
} from './useTheme'

// Internationalization hooks
export { useI18n } from './useI18n'
export {
  useFormatters,
  useDateFormatter,
  useNumberFormatter,
  type DateFormatStyle,
  type NumberFormatOptions,
} from './useFormatters'

// Order form hooks
export {
  useOrderCalculations,
  type OrderItemForCalculation,
  type OrderCalculations,
} from './useOrderCalculations'

export {
  useOrderForm,
  createItemKey,
  createEmptySalesOrderItem,
  createEmptyPurchaseOrderItem,
  type BaseOrderItemFormData,
  type SalesOrderItemFormData,
  type PurchaseOrderItemFormData,
  type OrderItemFormData,
  type BaseOrderFormData,
  type SalesOrderFormData,
  type PurchaseOrderFormData,
  type OrderFormDataType,
  type UseOrderFormOptions,
  type UseOrderFormReturn,
} from './useOrderForm'

// Print hooks
export { usePrint, ZOOM_LEVELS, type UsePrintOptions, type UsePrintReturn } from './usePrint'
