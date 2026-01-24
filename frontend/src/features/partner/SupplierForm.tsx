import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography, Rating } from '@douyinfe/semi-ui'
import { useNavigate } from 'react-router-dom'
import {
  Form,
  FormActions,
  FormSection,
  FormRow,
  TextField,
  NumberField,
  TextAreaField,
  SelectField,
  useFormWithValidation,
  validationMessages,
  createEnumSchema,
} from '@/components/common/form'
import { Container } from '@/components/common/layout'
import { getSuppliers } from '@/api/suppliers/suppliers'
import type { HandlerSupplierResponse } from '@/api/models'
import './SupplierForm.css'

const { Title, Text } = Typography

// Supplier type options
const SUPPLIER_TYPE_OPTIONS = [
  { label: '生产商', value: 'manufacturer' },
  { label: '经销商', value: 'distributor' },
  { label: '零售商', value: 'retailer' },
  { label: '服务商', value: 'service' },
]

// Supplier type values
const SUPPLIER_TYPES = ['manufacturer', 'distributor', 'retailer', 'service'] as const

// Form validation schema
const supplierFormSchema = z.object({
  code: z
    .string()
    .min(1, validationMessages.required)
    .max(50, validationMessages.maxLength(50))
    .regex(/^[A-Za-z0-9_-]+$/, '编码只能包含字母、数字、下划线和横线'),
  name: z.string().min(1, validationMessages.required).max(200, validationMessages.maxLength(200)),
  short_name: z
    .string()
    .max(100, validationMessages.maxLength(100))
    .optional()
    .transform((val) => val || undefined),
  type: createEnumSchema(SUPPLIER_TYPES, true),
  contact_name: z
    .string()
    .max(100, validationMessages.maxLength(100))
    .optional()
    .transform((val) => val || undefined),
  phone: z
    .string()
    .max(50, validationMessages.maxLength(50))
    .optional()
    .transform((val) => val || undefined),
  email: z
    .string()
    .email('请输入有效的邮箱地址')
    .max(200, validationMessages.maxLength(200))
    .optional()
    .or(z.literal(''))
    .transform((val) => val || undefined),
  tax_id: z
    .string()
    .max(50, validationMessages.maxLength(50))
    .optional()
    .transform((val) => val || undefined),
  country: z
    .string()
    .max(100, validationMessages.maxLength(100))
    .optional()
    .transform((val) => val || undefined),
  province: z
    .string()
    .max(100, validationMessages.maxLength(100))
    .optional()
    .transform((val) => val || undefined),
  city: z
    .string()
    .max(100, validationMessages.maxLength(100))
    .optional()
    .transform((val) => val || undefined),
  postal_code: z
    .string()
    .max(20, validationMessages.maxLength(20))
    .optional()
    .transform((val) => val || undefined),
  address: z
    .string()
    .max(500, validationMessages.maxLength(500))
    .optional()
    .transform((val) => val || undefined),
  bank_name: z
    .string()
    .max(200, validationMessages.maxLength(200))
    .optional()
    .transform((val) => val || undefined),
  bank_account: z
    .string()
    .max(100, validationMessages.maxLength(100))
    .optional()
    .transform((val) => val || undefined),
  credit_limit: z
    .number()
    .nonnegative(validationMessages.nonNegative)
    .optional()
    .nullable()
    .transform((val) => val ?? undefined),
  credit_days: z
    .number()
    .int(validationMessages.integer)
    .nonnegative(validationMessages.nonNegative)
    .optional()
    .nullable()
    .transform((val) => val ?? undefined),
  rating: z
    .number()
    .min(0, '评级最小为 0')
    .max(5, '评级最大为 5')
    .optional()
    .nullable()
    .transform((val) => val ?? undefined),
  sort_order: z
    .number()
    .int(validationMessages.integer)
    .nonnegative(validationMessages.nonNegative)
    .optional()
    .nullable()
    .transform((val) => val ?? undefined),
  notes: z
    .string()
    .max(2000, validationMessages.maxLength(2000))
    .optional()
    .transform((val) => val || undefined),
})

type SupplierFormData = z.infer<typeof supplierFormSchema>

interface SupplierFormProps {
  /** Supplier ID for edit mode, undefined for create mode */
  supplierId?: string
  /** Initial supplier data for edit mode */
  initialData?: HandlerSupplierResponse
}

/**
 * Supplier form component for creating and editing suppliers
 *
 * Features:
 * - Zod schema validation
 * - Create/edit modes
 * - Form sections for better organization (basic info, contact, address, banking, other settings)
 * - API integration with error handling
 * - Rating component for supplier evaluation
 */
export function SupplierForm({ supplierId, initialData }: SupplierFormProps) {
  const navigate = useNavigate()
  const api = useMemo(() => getSuppliers(), [])
  const isEditMode = Boolean(supplierId)

  // Transform API data to form values
  const defaultValues: Partial<SupplierFormData> = useMemo(() => {
    if (!initialData) {
      return {
        code: '',
        name: '',
        short_name: '',
        type: 'manufacturer' as const,
        contact_name: '',
        phone: '',
        email: '',
        tax_id: '',
        country: '中国',
        province: '',
        city: '',
        postal_code: '',
        address: '',
        bank_name: '',
        bank_account: '',
        credit_limit: undefined,
        credit_days: undefined,
        rating: undefined,
        sort_order: undefined,
        notes: '',
      }
    }
    return {
      code: initialData.code || '',
      name: initialData.name || '',
      short_name: initialData.short_name || '',
      type:
        (initialData.type as 'manufacturer' | 'distributor' | 'retailer' | 'service') ||
        'manufacturer',
      contact_name: initialData.contact_name || '',
      phone: initialData.phone || '',
      email: initialData.email || '',
      tax_id: initialData.tax_id || '',
      country: initialData.country || '中国',
      province: initialData.province || '',
      city: initialData.city || '',
      postal_code: initialData.postal_code || '',
      address: initialData.address || '',
      bank_name: initialData.bank_name || '',
      bank_account: initialData.bank_account || '',
      credit_limit: initialData.credit_limit ?? undefined,
      credit_days: initialData.payment_term_days ?? undefined,
      rating: initialData.rating ?? undefined,
      sort_order: initialData.sort_order ?? undefined,
      notes: initialData.notes || '',
    }
  }, [initialData])

  const { control, handleFormSubmit, isSubmitting, reset, watch, setValue } =
    useFormWithValidation<SupplierFormData>({
      schema: supplierFormSchema,
      defaultValues,
      successMessage: isEditMode ? '供应商更新成功' : '供应商创建成功',
      onSuccess: () => {
        navigate('/partner/suppliers')
      },
    })

  // Watch rating for the visual component
  const ratingValue = watch('rating')

  // Reset form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      reset(defaultValues)
    }
  }, [initialData, defaultValues, reset])

  // Handle form submission
  const onSubmit = async (data: SupplierFormData) => {
    if (isEditMode && supplierId) {
      // Update existing supplier
      const response = await api.putPartnerSuppliersId(supplierId, {
        name: data.name,
        short_name: data.short_name,
        contact_name: data.contact_name,
        phone: data.phone,
        email: data.email,
        tax_id: data.tax_id,
        country: data.country,
        province: data.province,
        city: data.city,
        postal_code: data.postal_code,
        address: data.address,
        bank_name: data.bank_name,
        bank_account: data.bank_account,
        credit_limit: data.credit_limit,
        credit_days: data.credit_days,
        rating: data.rating,
        sort_order: data.sort_order,
        notes: data.notes,
      })
      if (!response.success) {
        throw new Error(response.error?.message || '更新失败')
      }
    } else {
      // Create new supplier
      const response = await api.postPartnerSuppliers({
        code: data.code,
        name: data.name,
        short_name: data.short_name,
        type: data.type as 'manufacturer' | 'distributor' | 'retailer' | 'service',
        contact_name: data.contact_name,
        phone: data.phone,
        email: data.email,
        tax_id: data.tax_id,
        country: data.country,
        province: data.province,
        city: data.city,
        postal_code: data.postal_code,
        address: data.address,
        bank_name: data.bank_name,
        bank_account: data.bank_account,
        credit_limit: data.credit_limit,
        credit_days: data.credit_days,
        rating: data.rating,
        sort_order: data.sort_order,
        notes: data.notes,
      })
      if (!response.success) {
        throw new Error(response.error?.message || '创建失败')
      }
    }
  }

  const handleCancel = () => {
    navigate('/partner/suppliers')
  }

  const handleRatingChange = (value: number) => {
    setValue('rating', value)
  }

  return (
    <Container size="md" className="supplier-form-page">
      <Card className="supplier-form-card">
        <div className="supplier-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? '编辑供应商' : '新增供应商'}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection title="基本信息" description="供应商的基本属性和标识信息">
            <FormRow cols={2}>
              <TextField
                name="code"
                control={control}
                label="供应商编码"
                placeholder="请输入供应商编码"
                required
                disabled={isEditMode}
                helperText={isEditMode ? '编码创建后不可修改' : '支持字母、数字、下划线和横线'}
              />
              <TextField
                name="name"
                control={control}
                label="供应商名称"
                placeholder="请输入供应商全称"
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="short_name"
                control={control}
                label="简称"
                placeholder="请输入供应商简称 (可选)"
              />
              <SelectField
                name="type"
                control={control}
                label="供应商类型"
                placeholder="请选择供应商类型"
                options={SUPPLIER_TYPE_OPTIONS}
                required
                disabled={isEditMode}
              />
            </FormRow>
          </FormSection>

          <FormSection title="联系信息" description="供应商的联系人和联系方式">
            <FormRow cols={2}>
              <TextField
                name="contact_name"
                control={control}
                label="联系人"
                placeholder="请输入联系人姓名"
              />
              <TextField name="phone" control={control} label="电话" placeholder="请输入联系电话" />
            </FormRow>
            <FormRow cols={2}>
              <TextField name="email" control={control} label="邮箱" placeholder="请输入电子邮箱" />
              <TextField
                name="tax_id"
                control={control}
                label="税号"
                placeholder="请输入税务登记号"
              />
            </FormRow>
          </FormSection>

          <FormSection title="地址信息" description="供应商的地址和邮寄信息">
            <FormRow cols={3}>
              <TextField name="country" control={control} label="国家" placeholder="请输入国家" />
              <TextField name="province" control={control} label="省份" placeholder="请输入省份" />
              <TextField name="city" control={control} label="城市" placeholder="请输入城市" />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="address"
                control={control}
                label="详细地址"
                placeholder="请输入详细地址"
              />
              <TextField
                name="postal_code"
                control={control}
                label="邮政编码"
                placeholder="请输入邮政编码"
              />
            </FormRow>
          </FormSection>

          <FormSection title="银行信息" description="供应商的银行账户信息 (用于付款)">
            <FormRow cols={2}>
              <TextField
                name="bank_name"
                control={control}
                label="开户银行"
                placeholder="请输入开户银行"
              />
              <TextField
                name="bank_account"
                control={control}
                label="银行账号"
                placeholder="请输入银行账号"
              />
            </FormRow>
          </FormSection>

          <FormSection title="采购设置" description="信用额度、付款周期、评级等设置">
            <FormRow cols={2}>
              <NumberField
                name="credit_limit"
                control={control}
                label="信用额度"
                placeholder="请输入信用额度"
                min={0}
                precision={2}
                prefix="¥"
                helperText="供应商允许的最大赊购金额"
              />
              <NumberField
                name="credit_days"
                control={control}
                label="账期天数"
                placeholder="请输入账期天数"
                min={0}
                precision={0}
                suffix="天"
                helperText="付款到期天数"
              />
            </FormRow>
            <FormRow cols={2}>
              <div className="rating-field">
                <Text className="rating-label">供应商评级</Text>
                <div className="rating-wrapper">
                  <Rating
                    value={ratingValue ?? 0}
                    onChange={handleRatingChange}
                    allowHalf
                    size="default"
                  />
                  <Text type="tertiary" className="rating-value">
                    {ratingValue !== undefined ? `${ratingValue} 分` : '未评级'}
                  </Text>
                </div>
                <Text type="tertiary" className="rating-helper">
                  对供应商的综合评价 (0-5 分)
                </Text>
              </div>
              <NumberField
                name="sort_order"
                control={control}
                label="排序"
                placeholder="请输入排序值"
                min={0}
                precision={0}
                helperText="数值越小越靠前"
              />
            </FormRow>
            <TextAreaField
              name="notes"
              control={control}
              label="备注"
              placeholder="请输入备注信息 (可选)"
              rows={3}
              maxCount={2000}
            />
          </FormSection>

          <FormActions
            submitText={isEditMode ? '保存' : '创建'}
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}

export default SupplierForm
