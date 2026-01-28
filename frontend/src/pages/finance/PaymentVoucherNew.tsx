import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import { Card, Typography, Toast, Spin, Select, Tag, Banner } from '@douyinfe/semi-ui-19'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Form,
  FormActions,
  FormSection,
  FormRow,
  TextField,
  NumberField,
  TextAreaField,
  SelectField,
  DateField,
  useFormWithValidation,
  validationMessages,
  createEnumSchema,
} from '@/components/common/form'
import { Container } from '@/components/common/layout'
import { getFinanceApi } from '@/api/finance'
import { getSuppliers } from '@/api/suppliers/suppliers'
import type { PaymentMethod, CreatePaymentVoucherRequest, AccountPayable } from '@/api/finance'
import type { HandlerSupplierResponse } from '@/api/models'
import './PaymentVoucherNew.css'

const { Title, Text } = Typography

// Payment method values
const PAYMENT_METHODS = [
  'CASH',
  'BANK_TRANSFER',
  'WECHAT',
  'ALIPAY',
  'CHECK',
  'BALANCE',
  'OTHER',
] as const

// Form validation schema
const paymentVoucherFormSchema = z.object({
  supplier_id: z.string().min(1, validationMessages.required),
  supplier_name: z.string().min(1, validationMessages.required),
  amount: z
    .number()
    .positive('付款金额必须大于0')
    .max(999999999.99, '付款金额不能超过999,999,999.99'),
  payment_method: createEnumSchema(PAYMENT_METHODS, true),
  payment_reference: z
    .string()
    .max(100, validationMessages.maxLength(100))
    .optional()
    .transform((val) => val || undefined),
  payment_date: z.date({ message: validationMessages.required }),
  remark: z
    .string()
    .max(500, validationMessages.maxLength(500))
    .optional()
    .transform((val) => val || undefined),
})

type PaymentVoucherFormData = z.infer<typeof paymentVoucherFormSchema>

/**
 * Format currency for display
 */
function formatCurrency(amount?: number): string {
  if (amount === undefined || amount === null) return '-'
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

/**
 * Payment Voucher Creation Page
 *
 * Features:
 * - Supplier selection with search
 * - Payment method selection
 * - Amount input with validation
 * - Payment date picker
 * - Optional payment reference (transaction ID, check number, etc.)
 * - Shows supplier's outstanding payables summary
 */
export default function PaymentVoucherNewPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { t } = useTranslation('finance')
  const financeApi = useMemo(() => getFinanceApi(), [])
  const supplierApi = useMemo(() => getSuppliers(), [])

  // URL params for pre-filling
  const preSelectedSupplierId = searchParams.get('supplier_id') || ''

  // State
  const [supplierOptions, setSupplierOptions] = useState<
    Array<{ label: string; value: string; supplier: HandlerSupplierResponse }>
  >([])
  const [supplierLoading, setSupplierLoading] = useState(false)
  const [selectedSupplier, setSelectedSupplier] = useState<HandlerSupplierResponse | null>(null)
  const [supplierPayables, setSupplierPayables] = useState<AccountPayable[]>([])
  const [payablesLoading, setPayablesLoading] = useState(false)

  // Payment method options with translated labels
  const paymentMethodOptions = useMemo(
    () => [
      { label: t('paymentMethod.CASH'), value: 'CASH' },
      { label: t('paymentMethod.BANK_TRANSFER'), value: 'BANK_TRANSFER' },
      { label: t('paymentMethod.WECHAT'), value: 'WECHAT' },
      { label: t('paymentMethod.ALIPAY'), value: 'ALIPAY' },
      { label: t('paymentMethod.CHECK'), value: 'CHECK' },
      { label: t('paymentMethod.BALANCE'), value: 'BALANCE' },
      { label: t('paymentMethod.OTHER'), value: 'OTHER' },
    ],
    [t]
  )

  // Form setup
  const defaultValues: Partial<PaymentVoucherFormData> = useMemo(
    () => ({
      supplier_id: preSelectedSupplierId,
      supplier_name: '',
      amount: undefined,
      payment_method: 'BANK_TRANSFER' as const,
      payment_reference: '',
      payment_date: new Date(),
      remark: '',
    }),
    [preSelectedSupplierId]
  )

  const { control, handleFormSubmit, isSubmitting, setValue, watch } =
    useFormWithValidation<PaymentVoucherFormData>({
      schema: paymentVoucherFormSchema,
      defaultValues,
      successMessage: t('paymentVoucher.messages.createSuccess'),
      onSuccess: () => {
        navigate('/finance/payables')
      },
    })

  const watchedSupplierId = watch('supplier_id')

  // Search suppliers
  const searchSuppliers = useCallback(
    async (keyword: string) => {
      if (!keyword || keyword.length < 1) {
        setSupplierOptions([])
        return
      }

      setSupplierLoading(true)
      try {
        const response = await supplierApi.listSuppliers({
          search: keyword,
          status: 'active',
          page: 1,
          page_size: 20,
        })

        if (response.success && response.data) {
          const options = response.data.map((supplier) => ({
            label: `${supplier.name} (${supplier.code})`,
            value: supplier.id || '',
            supplier,
          }))
          setSupplierOptions(options)
        }
      } catch {
        Toast.error(t('paymentVoucher.messages.searchSupplierError'))
      } finally {
        setSupplierLoading(false)
      }
    },
    [supplierApi, t]
  )

  // Fetch supplier payables when supplier is selected
  const fetchSupplierPayables = useCallback(
    async (supplierId: string) => {
      if (!supplierId) {
        setSupplierPayables([])
        return
      }

      setPayablesLoading(true)
      try {
        const payablesResponse = await financeApi.listFinancePayablesPayables({
          supplier_id: supplierId,
          status: 'PENDING',
          page: 1,
          page_size: 100,
        })

        if (payablesResponse.success && payablesResponse.data) {
          // Filter payables for this supplier only
          const filteredPayables = payablesResponse.data.filter(
            (p) =>
              p.supplier_id === supplierId && (p.status === 'PENDING' || p.status === 'PARTIAL')
          )
          setSupplierPayables(filteredPayables)
        }
      } catch {
        // Silent fail
      } finally {
        setPayablesLoading(false)
      }
    },
    [financeApi]
  )

  // Load pre-selected supplier
  useEffect(() => {
    if (preSelectedSupplierId) {
      const loadSupplier = async () => {
        try {
          const response = await supplierApi.getSupplierById(preSelectedSupplierId)
          if (response.success && response.data) {
            const supplier = response.data
            setSelectedSupplier(supplier)
            setValue('supplier_id', supplier.id || '')
            setValue('supplier_name', supplier.name || '')
            setSupplierOptions([
              {
                label: `${supplier.name} (${supplier.code})`,
                value: supplier.id || '',
                supplier,
              },
            ])
            fetchSupplierPayables(supplier.id || '')
          }
        } catch {
          Toast.error(t('paymentVoucher.messages.loadSupplierError'))
        }
      }
      loadSupplier()
    }
  }, [preSelectedSupplierId, supplierApi, setValue, fetchSupplierPayables, t])

  // Watch for supplier changes
  useEffect(() => {
    if (watchedSupplierId && watchedSupplierId !== selectedSupplier?.id) {
      fetchSupplierPayables(watchedSupplierId)
    }
  }, [watchedSupplierId, selectedSupplier?.id, fetchSupplierPayables])

  // Handle supplier selection
  const handleSupplierChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const supplierId = typeof value === 'string' ? value : ''
      const selectedOption = supplierOptions.find((opt) => opt.value === supplierId)

      if (selectedOption) {
        setSelectedSupplier(selectedOption.supplier)
        setValue('supplier_id', supplierId)
        setValue('supplier_name', selectedOption.supplier.name || '')
      } else {
        setSelectedSupplier(null)
        setValue('supplier_id', '')
        setValue('supplier_name', '')
      }
    },
    [supplierOptions, setValue]
  )

  // Handle supplier search
  const handleSupplierSearch = useCallback(
    (inputValue: string) => {
      searchSuppliers(inputValue)
    },
    [searchSuppliers]
  )

  // Handle form submission
  const onSubmit = async (data: PaymentVoucherFormData) => {
    const request: CreatePaymentVoucherRequest = {
      supplier_id: data.supplier_id,
      supplier_name: data.supplier_name,
      amount: data.amount,
      payment_method: data.payment_method as PaymentMethod,
      payment_reference: data.payment_reference,
      payment_date: data.payment_date.toISOString().split('T')[0],
      remark: data.remark,
    }

    const response = await financeApi.createFinancePaymentPaymentVoucher(request)
    if (!response.success) {
      throw new Error(response.error || t('paymentVoucher.messages.createError'))
    }
  }

  // Handle cancel
  const handleCancel = () => {
    navigate('/finance/payables')
  }

  // Calculate total outstanding for selected supplier
  const totalOutstanding = useMemo(() => {
    return supplierPayables.reduce((sum, p) => sum + (p.outstanding_amount || 0), 0)
  }, [supplierPayables])

  return (
    <Container size="md" className="payment-voucher-new-page">
      <Card className="payment-voucher-form-card">
        <div className="payment-voucher-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('paymentVoucher.title')}
          </Title>
        </div>

        {/* Supplier payables summary */}
        {selectedSupplier && (
          <div className="supplier-payables-summary">
            <Spin spinning={payablesLoading}>
              {supplierPayables.length > 0 ? (
                <Banner
                  type="info"
                  description={
                    <div className="payables-info">
                      <Text>
                        {t('paymentVoucher.supplierSummary.hasPayables', {
                          name: selectedSupplier.name,
                          count: supplierPayables.length,
                        })}{' '}
                        <strong className="amount-highlight">
                          {formatCurrency(totalOutstanding)}
                        </strong>
                      </Text>
                    </div>
                  }
                />
              ) : (
                <Banner
                  type="success"
                  description={
                    <Text>
                      {t('paymentVoucher.supplierSummary.noPayables', {
                        name: selectedSupplier.name,
                      })}
                    </Text>
                  }
                />
              )}
            </Spin>

            {/* Pending payables list */}
            {supplierPayables.length > 0 && (
              <div className="pending-payables-list">
                <Text type="secondary" className="list-title">
                  {t('paymentVoucher.supplierSummary.pendingPayables')}
                </Text>
                <div className="payables-tags">
                  {supplierPayables.slice(0, 5).map((payable) => (
                    <Tag
                      key={payable.id}
                      color={payable.status === 'PARTIAL' ? 'blue' : 'orange'}
                      className="payable-tag"
                    >
                      {payable.payable_number}: {formatCurrency(payable.outstanding_amount)}
                    </Tag>
                  ))}
                  {supplierPayables.length > 5 && (
                    <Tag color="grey">
                      {t('paymentVoucher.supplierSummary.more', {
                        count: supplierPayables.length - 5,
                      })}
                    </Tag>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection
            title={t('paymentVoucher.supplierInfo.title')}
            description={t('paymentVoucher.supplierInfo.description')}
          >
            <FormRow cols={1}>
              <div className="supplier-select-wrapper">
                <label className="semi-form-field-label">
                  <span className="semi-form-field-label-text">
                    <span className="semi-form-field-label-required">*</span>
                    {t('paymentVoucher.supplierInfo.label')}
                  </span>
                </label>
                <Select
                  placeholder={t('paymentVoucher.supplierInfo.placeholder')}
                  value={watchedSupplierId || undefined}
                  onChange={handleSupplierChange}
                  onSearch={handleSupplierSearch}
                  optionList={supplierOptions}
                  loading={supplierLoading}
                  filter
                  remote
                  showClear
                  style={{ width: '100%' }}
                />
              </div>
            </FormRow>
          </FormSection>

          <FormSection
            title={t('paymentVoucher.paymentInfo.title')}
            description={t('paymentVoucher.paymentInfo.description')}
          >
            <FormRow cols={2}>
              <NumberField
                name="amount"
                control={control}
                label={t('paymentVoucher.paymentInfo.amount')}
                placeholder={t('paymentVoucher.paymentInfo.amountPlaceholder')}
                required
                min={0.01}
                max={999999999.99}
                precision={2}
                prefix="¥"
                helperText={
                  totalOutstanding > 0
                    ? t('paymentVoucher.paymentInfo.outstandingHelper', {
                        amount: formatCurrency(totalOutstanding),
                      })
                    : undefined
                }
              />
              <SelectField
                name="payment_method"
                control={control}
                label={t('paymentVoucher.paymentInfo.method')}
                placeholder={t('paymentVoucher.paymentInfo.methodPlaceholder')}
                options={paymentMethodOptions}
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <DateField
                name="payment_date"
                control={control}
                label={t('paymentVoucher.paymentInfo.date')}
                placeholder={t('paymentVoucher.paymentInfo.datePlaceholder')}
                required
              />
              <TextField
                name="payment_reference"
                control={control}
                label={t('paymentVoucher.paymentInfo.reference')}
                placeholder={t('paymentVoucher.paymentInfo.referencePlaceholder')}
                helperText={t('paymentVoucher.paymentInfo.referenceHelper')}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('paymentVoucher.otherInfo.title')}
            description={t('paymentVoucher.otherInfo.description')}
          >
            <TextAreaField
              name="remark"
              control={control}
              label={t('paymentVoucher.otherInfo.remark')}
              placeholder={t('paymentVoucher.otherInfo.remarkPlaceholder')}
              rows={3}
              maxCount={500}
            />
          </FormSection>

          <FormActions
            submitText={t('paymentVoucher.actions.create')}
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}
