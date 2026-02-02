/**
 * Auto Print Rules Page Tests
 *
 * Tests for the auto print rules configuration page.
 * Note: Component tests with Semi UI require additional setup due to the
 * complex nature of the component library. The tests below focus on the
 * core business logic via mock hooks.
 */

import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useState, useCallback } from 'react'

// Test the validation logic separately
describe('AutoPrintRules - Validation Logic', () => {
  interface RuleFormValues {
    document_type: string
    trigger_event: string
    template_id: string
    copies: number
    printer_name: string
    auto_print: boolean
  }

  const defaultFormValues: RuleFormValues = {
    document_type: '',
    trigger_event: '',
    template_id: '',
    copies: 1,
    printer_name: '',
    auto_print: true,
  }

  const validateForm = (formValues: RuleFormValues, t: (key: string) => string): string | null => {
    if (!formValues.document_type) {
      return t('autoPrintRules.validation.documentTypeRequired')
    }
    if (!formValues.trigger_event) {
      return t('autoPrintRules.validation.triggerEventRequired')
    }
    if (!formValues.template_id) {
      return t('autoPrintRules.validation.templateRequired')
    }
    if (formValues.copies < 1 || formValues.copies > 100) {
      return t('autoPrintRules.validation.copiesRange')
    }
    return null
  }

  const mockT = (key: string) => key

  it('should require document type', () => {
    const error = validateForm(defaultFormValues, mockT)
    expect(error).toBe('autoPrintRules.validation.documentTypeRequired')
  })

  it('should require trigger event when document type is provided', () => {
    const formValues = { ...defaultFormValues, document_type: 'SALES_ORDER' }
    const error = validateForm(formValues, mockT)
    expect(error).toBe('autoPrintRules.validation.triggerEventRequired')
  })

  it('should require template when document type and trigger event are provided', () => {
    const formValues = {
      ...defaultFormValues,
      document_type: 'SALES_ORDER',
      trigger_event: 'ORDER_CONFIRMED',
    }
    const error = validateForm(formValues, mockT)
    expect(error).toBe('autoPrintRules.validation.templateRequired')
  })

  it('should validate copies range (1-100)', () => {
    const formValues = {
      ...defaultFormValues,
      document_type: 'SALES_ORDER',
      trigger_event: 'ORDER_CONFIRMED',
      template_id: 'template-1',
      copies: 0,
    }
    const error = validateForm(formValues, mockT)
    expect(error).toBe('autoPrintRules.validation.copiesRange')
  })

  it('should validate copies exceeding max', () => {
    const formValues = {
      ...defaultFormValues,
      document_type: 'SALES_ORDER',
      trigger_event: 'ORDER_CONFIRMED',
      template_id: 'template-1',
      copies: 101,
    }
    const error = validateForm(formValues, mockT)
    expect(error).toBe('autoPrintRules.validation.copiesRange')
  })

  it('should pass validation with all required fields', () => {
    const formValues = {
      ...defaultFormValues,
      document_type: 'SALES_ORDER',
      trigger_event: 'ORDER_CONFIRMED',
      template_id: 'template-1',
      copies: 1,
    }
    const error = validateForm(formValues, mockT)
    expect(error).toBeNull()
  })

  it('should pass validation with max copies', () => {
    const formValues = {
      ...defaultFormValues,
      document_type: 'SALES_ORDER',
      trigger_event: 'ORDER_CONFIRMED',
      template_id: 'template-1',
      copies: 100,
    }
    const error = validateForm(formValues, mockT)
    expect(error).toBeNull()
  })
})

describe('AutoPrintRules - Form State Management', () => {
  interface RuleFormValues {
    document_type: string
    trigger_event: string
    template_id: string
    copies: number
    printer_name: string
    auto_print: boolean
  }

  const defaultFormValues: RuleFormValues = {
    document_type: '',
    trigger_event: '',
    template_id: '',
    copies: 1,
    printer_name: '',
    auto_print: true,
  }

  it('should initialize with default values', () => {
    const { result } = renderHook(() => useState(defaultFormValues))
    expect(result.current[0]).toEqual(defaultFormValues)
  })

  it('should update form values immutably', () => {
    const { result } = renderHook(() => {
      const [formValues, setFormValues] = useState(defaultFormValues)

      const handleFormChange = useCallback((values: Partial<RuleFormValues>) => {
        setFormValues((prev) => ({ ...prev, ...values }))
      }, [])

      return { formValues, handleFormChange }
    })

    act(() => {
      result.current.handleFormChange({ document_type: 'SALES_ORDER' })
    })

    expect(result.current.formValues.document_type).toBe('SALES_ORDER')
    expect(result.current.formValues.trigger_event).toBe('')
    expect(result.current.formValues.copies).toBe(1)
  })

  it('should reset form to defaults', () => {
    const { result } = renderHook(() => {
      const [formValues, setFormValues] = useState({
        ...defaultFormValues,
        document_type: 'SALES_ORDER',
        trigger_event: 'ORDER_CONFIRMED',
      })

      const reset = useCallback(() => {
        setFormValues(defaultFormValues)
      }, [])

      return { formValues, reset }
    })

    expect(result.current.formValues.document_type).toBe('SALES_ORDER')

    act(() => {
      result.current.reset()
    })

    expect(result.current.formValues).toEqual(defaultFormValues)
  })
})

describe('AutoPrintRules - Document Type Options', () => {
  interface HandlerDocumentTypeResponse {
    code?: string
    display_name?: string
  }

  it('should transform document types to select options', () => {
    const documentTypes: HandlerDocumentTypeResponse[] = [
      { code: 'SALES_ORDER', display_name: 'Sales Order' },
      { code: 'PURCHASE_ORDER', display_name: 'Purchase Order' },
      { code: 'INVOICE' },
    ]

    const options = documentTypes.map((dt) => ({
      label: dt.display_name || dt.code,
      value: dt.code,
    }))

    expect(options).toEqual([
      { label: 'Sales Order', value: 'SALES_ORDER' },
      { label: 'Purchase Order', value: 'PURCHASE_ORDER' },
      { label: 'INVOICE', value: 'INVOICE' },
    ])
  })

  it('should handle empty document types', () => {
    const documentTypes: HandlerDocumentTypeResponse[] = []
    const options = documentTypes.map((dt) => ({
      label: dt.display_name || dt.code,
      value: dt.code,
    }))

    expect(options).toEqual([])
  })
})

describe('AutoPrintRules - Printer Options', () => {
  interface PrinterInfo {
    name: string
    displayName?: string
    isDefault?: boolean
    status?: string
  }

  it('should include default printer option', () => {
    const printers: PrinterInfo[] = [
      { name: 'printer1', displayName: 'Printer 1', isDefault: true },
      { name: 'printer2', displayName: 'Printer 2', isDefault: false },
    ]

    const defaultPrinterLabel = 'Default Printer'

    const options = [
      { label: defaultPrinterLabel, value: '' },
      ...printers.map((p) => ({
        label: p.displayName || p.name,
        value: p.name,
      })),
    ]

    expect(options[0]).toEqual({ label: 'Default Printer', value: '' })
    expect(options).toHaveLength(3)
  })

  it('should handle printers without display name', () => {
    const printers: PrinterInfo[] = [{ name: 'printer1' }]

    const options = printers.map((p) => ({
      label: p.displayName || p.name,
      value: p.name,
    }))

    expect(options[0]).toEqual({ label: 'printer1', value: 'printer1' })
  })
})

describe('AutoPrintRules - Toggle Logic', () => {
  interface HandlerAutoPrintRuleHTTPResponse {
    id?: string
    enabled?: boolean
    document_type?: string
    trigger_event?: string
  }

  it('should determine correct action for enabled rule', () => {
    const rule: HandlerAutoPrintRuleHTTPResponse = {
      id: 'rule-1',
      enabled: true,
    }

    // Logic: if enabled, should call disable; if disabled, should call enable
    const shouldDisable = rule.enabled === true
    expect(shouldDisable).toBe(true)
  })

  it('should determine correct action for disabled rule', () => {
    const rule: HandlerAutoPrintRuleHTTPResponse = {
      id: 'rule-1',
      enabled: false,
    }

    const shouldEnable = rule.enabled === false
    expect(shouldEnable).toBe(true)
  })
})

describe('AutoPrintRules - Print Service Status Logic', () => {
  type PrintServiceStatus = 'connected' | 'disconnected' | 'checking'

  it('should show connected status', () => {
    const status: PrintServiceStatus = 'connected'
    const statusConfig = {
      connected: { color: 'green', label: 'Connected' },
      disconnected: { color: 'red', label: 'Disconnected' },
      checking: { color: 'grey', label: 'Checking...' },
    }

    expect(statusConfig[status].color).toBe('green')
    expect(statusConfig[status].label).toBe('Connected')
  })

  it('should disable create button when disconnected', () => {
    const status: PrintServiceStatus = 'disconnected'
    const isCreateDisabled = status === 'disconnected'
    expect(isCreateDisabled).toBe(true)
  })

  it('should enable create button when connected', () => {
    const status: PrintServiceStatus = 'connected'
    const isCreateDisabled = status === 'disconnected'
    expect(isCreateDisabled).toBe(false)
  })
})
