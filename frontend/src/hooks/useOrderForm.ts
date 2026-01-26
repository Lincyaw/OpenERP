import { useState, useCallback } from 'react'
import { z, ZodSchema } from 'zod'

/**
 * Base order item type for form management
 */
export interface BaseOrderItemFormData {
  key: string
  product_id: string
  product_code: string
  product_name: string
  unit: string
  quantity: number
  amount: number
  remark?: string
}

/**
 * Sales order specific item with unit_price
 */
export interface SalesOrderItemFormData extends BaseOrderItemFormData {
  unit_price: number
}

/**
 * Purchase order specific item with unit_cost
 */
export interface PurchaseOrderItemFormData extends BaseOrderItemFormData {
  unit_cost: number
}

/**
 * Union type for all order item types
 */
export type OrderItemFormData = SalesOrderItemFormData | PurchaseOrderItemFormData

/**
 * Base order form data structure
 */
export interface BaseOrderFormData<T extends OrderItemFormData> {
  warehouse_id?: string
  discount: number
  remark?: string
  items: T[]
}

/**
 * Sales order form data
 */
export interface SalesOrderFormData extends BaseOrderFormData<SalesOrderItemFormData> {
  customer_id: string
  customer_name: string
}

/**
 * Purchase order form data
 */
export interface PurchaseOrderFormData extends BaseOrderFormData<PurchaseOrderItemFormData> {
  supplier_id: string
  supplier_name: string
}

/**
 * Generic order form data type
 */
export type OrderFormDataType = SalesOrderFormData | PurchaseOrderFormData

/**
 * Create a unique key for new order items
 */
export function createItemKey(): string {
  return `item-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

/**
 * Create an empty sales order item
 */
export function createEmptySalesOrderItem(): SalesOrderItemFormData {
  return {
    key: createItemKey(),
    product_id: '',
    product_code: '',
    product_name: '',
    unit: '',
    unit_price: 0,
    quantity: 1,
    amount: 0,
    remark: '',
  }
}

/**
 * Create an empty purchase order item
 */
export function createEmptyPurchaseOrderItem(): PurchaseOrderItemFormData {
  return {
    key: createItemKey(),
    product_id: '',
    product_code: '',
    product_name: '',
    unit: '',
    unit_cost: 0,
    quantity: 1,
    amount: 0,
    remark: '',
  }
}

/**
 * Options for useOrderForm hook
 */
export interface UseOrderFormOptions<T extends OrderFormDataType> {
  initialData: T
  schema?: ZodSchema
  createEmptyItem: () => T['items'][number]
}

/**
 * Return type for useOrderForm hook
 */
export interface UseOrderFormReturn<T extends OrderFormDataType> {
  formData: T
  setFormData: React.Dispatch<React.SetStateAction<T>>
  errors: Record<string, string>
  setErrors: React.Dispatch<React.SetStateAction<Record<string, string>>>
  isSubmitting: boolean
  setIsSubmitting: React.Dispatch<React.SetStateAction<boolean>>
  clearError: (field: string) => void
  clearAllErrors: () => void
  validateForm: () => boolean
  resetForm: () => void
  // Item management functions
  addItem: () => void
  removeItem: (key: string) => void
  updateItem: <K extends keyof T['items'][number]>(
    key: string,
    field: K,
    value: T['items'][number][K]
  ) => void
  updateItemWithAmount: (
    key: string,
    updates: Partial<T['items'][number]>,
    priceField: 'unit_price' | 'unit_cost'
  ) => void
  // Form field handlers
  handleDiscountChange: (value: number | string | undefined) => void
  handleRemarkChange: (value: string) => void
  handleWarehouseChange: (
    value: string | number | (string | number)[] | Record<string, unknown> | undefined
  ) => void
}

/**
 * Hook for managing order form state and operations
 *
 * @param options - Configuration options including initial data, schema, and item factory
 * @returns Form state and handler functions
 *
 * @example
 * function SalesOrderForm() {
 *   const {
 *     formData,
 *     errors,
 *     addItem,
 *     removeItem,
 *     handleDiscountChange,
 *   } = useOrderForm({
 *     initialData: defaultSalesOrderData,
 *     schema: salesOrderSchema,
 *     createEmptyItem: createEmptySalesOrderItem,
 *   })
 *
 *   return <form>...</form>
 * }
 */
export function useOrderForm<T extends OrderFormDataType>({
  initialData,
  schema,
  createEmptyItem,
}: UseOrderFormOptions<T>): UseOrderFormReturn<T> {
  const [formData, setFormData] = useState<T>(initialData)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Clear a specific error
  const clearError = useCallback((field: string) => {
    setErrors((prev) => {
      const newErrors = { ...prev }
      delete newErrors[field]
      return newErrors
    })
  }, [])

  // Clear all errors
  const clearAllErrors = useCallback(() => {
    setErrors({})
  }, [])

  // Reset form to initial state
  const resetForm = useCallback(() => {
    setFormData(initialData)
    setErrors({})
    setIsSubmitting(false)
  }, [initialData])

  // Validate form against schema
  const validateForm = useCallback((): boolean => {
    if (!schema) return true

    const dataToValidate = {
      ...formData,
      items: formData.items.filter((item) => item.product_id),
    }

    const result = schema.safeParse(dataToValidate)

    if (!result.success) {
      const newErrors: Record<string, string> = {}
      result.error.issues.forEach((issue: z.ZodIssue) => {
        const path = issue.path.join('.')
        newErrors[path] = issue.message
      })
      setErrors(newErrors)
      return false
    }

    setErrors({})
    return true
  }, [formData, schema])

  // Add a new empty item
  const addItem = useCallback(() => {
    setFormData((prev) => ({
      ...prev,
      items: [...prev.items, createEmptyItem()] as T['items'],
    }))
  }, [createEmptyItem])

  // Remove an item by key
  const removeItem = useCallback(
    (key: string) => {
      setFormData((prev) => {
        const newItems = prev.items.filter((item) => item.key !== key)
        // Always keep at least one row
        if (newItems.length === 0) {
          return { ...prev, items: [createEmptyItem()] as T['items'] }
        }
        return { ...prev, items: newItems as T['items'] }
      })
    },
    [createEmptyItem]
  )

  // Update a specific field on an item
  const updateItem = useCallback(
    <K extends keyof T['items'][number]>(key: string, field: K, value: T['items'][number][K]) => {
      setFormData((prev) => ({
        ...prev,
        items: prev.items.map((item) => {
          if (item.key !== key) return item
          return { ...item, [field]: value }
        }) as T['items'],
      }))
    },
    []
  )

  // Update item with automatic amount recalculation
  const updateItemWithAmount = useCallback(
    (key: string, updates: Partial<T['items'][number]>, priceField: 'unit_price' | 'unit_cost') => {
      setFormData((prev) => ({
        ...prev,
        items: prev.items.map((item) => {
          if (item.key !== key) return item
          const updatedItem = { ...item, ...updates }
          const price = (updatedItem as Record<string, unknown>)[priceField] as number
          const quantity = updatedItem.quantity
          return { ...updatedItem, amount: price * quantity }
        }) as T['items'],
      }))
    },
    []
  )

  // Handle discount change
  const handleDiscountChange = useCallback((value: number | string | undefined) => {
    const discount = typeof value === 'number' ? value : parseFloat(String(value)) || 0
    setFormData((prev) => ({ ...prev, discount }))
  }, [])

  // Handle remark change
  const handleRemarkChange = useCallback((value: string) => {
    setFormData((prev) => ({ ...prev, remark: value }))
  }, [])

  // Handle warehouse change
  const handleWarehouseChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const warehouseId = typeof value === 'string' ? value : undefined
      setFormData((prev) => ({ ...prev, warehouse_id: warehouseId || undefined }))
    },
    []
  )

  return {
    formData,
    setFormData,
    errors,
    setErrors,
    isSubmitting,
    setIsSubmitting,
    clearError,
    clearAllErrors,
    validateForm,
    resetForm,
    addItem,
    removeItem,
    updateItem,
    updateItemWithAmount,
    handleDiscountChange,
    handleRemarkChange,
    handleWarehouseChange,
  }
}
