import { useState, useEffect, useCallback, useMemo } from 'react'
import { z } from 'zod'
import {
  Card,
  Typography,
  Toast,
  Spin,
  Select,
  Tag,
  Banner,
} from '@douyinfe/semi-ui'
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
import type {
  PaymentMethod,
  CreatePaymentVoucherRequest,
  AccountPayable,
} from '@/api/finance'
import type { HandlerSupplierResponse } from '@/api/models'
import './PaymentVoucherNew.css'

const { Title, Text } = Typography

// Payment method options
const PAYMENT_METHOD_OPTIONS = [
  { label: '现金', value: 'CASH' },
  { label: '银行转账', value: 'BANK_TRANSFER' },
  { label: '微信支付', value: 'WECHAT' },
  { label: '支付宝', value: 'ALIPAY' },
  { label: '支票', value: 'CHECK' },
  { label: '余额抵扣', value: 'BALANCE' },
  { label: '其他', value: 'OTHER' },
]

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
      successMessage: '付款单创建成功',
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
        const response = await supplierApi.getPartnerSuppliers({
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
        Toast.error('搜索供应商失败')
      } finally {
        setSupplierLoading(false)
      }
    },
    [supplierApi]
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
        const payablesResponse = await financeApi.getFinancePayables({
          supplier_id: supplierId,
          status: 'PENDING',
          page: 1,
          page_size: 100,
        })

        if (payablesResponse.success && payablesResponse.data) {
          // Filter payables for this supplier only
          const filteredPayables = payablesResponse.data.filter(
            (p) => p.supplier_id === supplierId && (p.status === 'PENDING' || p.status === 'PARTIAL')
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
          const response = await supplierApi.getPartnerSuppliersId(preSelectedSupplierId)
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
          Toast.error('加载供应商信息失败')
        }
      }
      loadSupplier()
    }
  }, [preSelectedSupplierId, supplierApi, setValue, fetchSupplierPayables])

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

    const response = await financeApi.postFinancePayments(request)
    if (!response.success) {
      throw new Error(response.error || '创建付款单失败')
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
            新增付款单
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
                        供应商 <strong>{selectedSupplier.name}</strong> 共有{' '}
                        <strong>{supplierPayables.length}</strong> 笔待付账款，待付总额:{' '}
                        <strong className="amount-highlight">{formatCurrency(totalOutstanding)}</strong>
                      </Text>
                    </div>
                  }
                />
              ) : (
                <Banner
                  type="success"
                  description={
                    <Text>
                      供应商 <strong>{selectedSupplier.name}</strong> 暂无待付账款
                    </Text>
                  }
                />
              )}
            </Spin>

            {/* Pending payables list */}
            {supplierPayables.length > 0 && (
              <div className="pending-payables-list">
                <Text type="secondary" className="list-title">
                  待核销应付账款:
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
                    <Tag color="grey">+{supplierPayables.length - 5} 更多</Tag>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection title="供应商信息" description="选择付款的供应商">
            <FormRow cols={1}>
              <div className="supplier-select-wrapper">
                <label className="semi-form-field-label">
                  <span className="semi-form-field-label-text">
                    <span className="semi-form-field-label-required">*</span>
                    供应商
                  </span>
                </label>
                <Select
                  placeholder="请输入供应商名称或编码搜索"
                  value={watchedSupplierId || undefined}
                  onChange={handleSupplierChange}
                  onSearch={handleSupplierSearch}
                  optionList={supplierOptions}
                  loading={supplierLoading}
                  filter={false}
                  remote
                  showClear
                  style={{ width: '100%' }}
                />
              </div>
            </FormRow>
          </FormSection>

          <FormSection title="付款信息" description="填写付款金额和方式">
            <FormRow cols={2}>
              <NumberField
                name="amount"
                control={control}
                label="付款金额"
                placeholder="请输入付款金额"
                required
                min={0.01}
                max={999999999.99}
                precision={2}
                prefix="¥"
                helperText={
                  totalOutstanding > 0
                    ? `待付总额: ${formatCurrency(totalOutstanding)}`
                    : undefined
                }
              />
              <SelectField
                name="payment_method"
                control={control}
                label="付款方式"
                placeholder="请选择付款方式"
                options={PAYMENT_METHOD_OPTIONS}
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <DateField
                name="payment_date"
                control={control}
                label="付款日期"
                placeholder="请选择付款日期"
                required
              />
              <TextField
                name="payment_reference"
                control={control}
                label="付款凭证号"
                placeholder="交易流水号、支票号等 (可选)"
                helperText="用于关联银行流水或支付平台交易"
              />
            </FormRow>
          </FormSection>

          <FormSection title="其他信息" description="备注说明">
            <TextAreaField
              name="remark"
              control={control}
              label="备注"
              placeholder="请输入备注信息 (可选)"
              rows={3}
              maxCount={500}
            />
          </FormSection>

          <FormActions
            submitText="创建"
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}
