import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography } from '@douyinfe/semi-ui'
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
import { getCustomers } from '@/api/customers/customers'
import type { HandlerCustomerResponse } from '@/api/models'
import './CustomerForm.css'

const { Title } = Typography

// Customer type options
const CUSTOMER_TYPE_OPTIONS = [
  { label: '个人', value: 'individual' },
  { label: '企业/组织', value: 'organization' },
]

// Customer level options (only for edit mode)
const CUSTOMER_LEVEL_OPTIONS = [
  { label: '普通', value: 'normal' },
  { label: '白银', value: 'silver' },
  { label: '黄金', value: 'gold' },
  { label: '铂金', value: 'platinum' },
  { label: 'VIP', value: 'vip' },
]

// Customer type and level values
const CUSTOMER_TYPES = ['individual', 'organization'] as const
const CUSTOMER_LEVELS = ['normal', 'silver', 'gold', 'platinum', 'vip'] as const

// Form validation schema
const customerFormSchema = z.object({
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
  type: createEnumSchema(CUSTOMER_TYPES, true),
  level: createEnumSchema(CUSTOMER_LEVELS, false)
    .nullable()
    .transform((val) => val ?? undefined),
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
  credit_limit: z
    .number()
    .nonnegative(validationMessages.nonNegative)
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

type CustomerFormData = z.infer<typeof customerFormSchema>

interface CustomerFormProps {
  /** Customer ID for edit mode, undefined for create mode */
  customerId?: string
  /** Initial customer data for edit mode */
  initialData?: HandlerCustomerResponse
}

/**
 * Customer form component for creating and editing customers
 *
 * Features:
 * - Zod schema validation
 * - Create/edit modes
 * - Form sections for better organization (basic info, contact, address, other settings)
 * - API integration with error handling
 */
export function CustomerForm({ customerId, initialData }: CustomerFormProps) {
  const navigate = useNavigate()
  const api = useMemo(() => getCustomers(), [])
  const isEditMode = Boolean(customerId)

  // Transform API data to form values
  const defaultValues: Partial<CustomerFormData> = useMemo(() => {
    if (!initialData) {
      return {
        code: '',
        name: '',
        short_name: '',
        type: 'individual' as const,
        level: undefined,
        contact_name: '',
        phone: '',
        email: '',
        tax_id: '',
        country: '中国',
        province: '',
        city: '',
        postal_code: '',
        address: '',
        credit_limit: undefined,
        sort_order: undefined,
        notes: '',
      }
    }
    return {
      code: initialData.code || '',
      name: initialData.name || '',
      short_name: initialData.short_name || '',
      type: (initialData.type as 'individual' | 'organization') || 'individual',
      level: (initialData.level as 'normal' | 'silver' | 'gold' | 'platinum' | 'vip') || undefined,
      contact_name: initialData.contact_name || '',
      phone: initialData.phone || '',
      email: initialData.email || '',
      tax_id: initialData.tax_id || '',
      country: initialData.country || '中国',
      province: initialData.province || '',
      city: initialData.city || '',
      postal_code: initialData.postal_code || '',
      address: initialData.address || '',
      credit_limit: initialData.credit_limit ?? undefined,
      sort_order: initialData.sort_order ?? undefined,
      notes: initialData.notes || '',
    }
  }, [initialData])

  const { control, handleFormSubmit, isSubmitting, reset } =
    useFormWithValidation<CustomerFormData>({
      schema: customerFormSchema,
      defaultValues,
      successMessage: isEditMode ? '客户更新成功' : '客户创建成功',
      onSuccess: () => {
        navigate('/partner/customers')
      },
    })

  // Reset form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      reset(defaultValues)
    }
  }, [initialData, defaultValues, reset])

  // Handle form submission
  const onSubmit = async (data: CustomerFormData) => {
    if (isEditMode && customerId) {
      // Update existing customer
      const response = await api.putPartnerCustomersId(customerId, {
        name: data.name,
        short_name: data.short_name,
        level: data.level,
        contact_name: data.contact_name,
        phone: data.phone,
        email: data.email,
        tax_id: data.tax_id,
        country: data.country,
        province: data.province,
        city: data.city,
        postal_code: data.postal_code,
        address: data.address,
        credit_limit: data.credit_limit,
        sort_order: data.sort_order,
        notes: data.notes,
      })
      if (!response.success) {
        throw new Error(response.error?.message || '更新失败')
      }
    } else {
      // Create new customer
      const response = await api.postPartnerCustomers({
        code: data.code,
        name: data.name,
        short_name: data.short_name,
        type: data.type as 'individual' | 'organization',
        contact_name: data.contact_name,
        phone: data.phone,
        email: data.email,
        tax_id: data.tax_id,
        country: data.country,
        province: data.province,
        city: data.city,
        postal_code: data.postal_code,
        address: data.address,
        credit_limit: data.credit_limit,
        sort_order: data.sort_order,
        notes: data.notes,
      })
      if (!response.success) {
        throw new Error(response.error?.message || '创建失败')
      }
    }
  }

  const handleCancel = () => {
    navigate('/partner/customers')
  }

  return (
    <Container size="md" className="customer-form-page">
      <Card className="customer-form-card">
        <div className="customer-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? '编辑客户' : '新增客户'}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection title="基本信息" description="客户的基本属性和标识信息">
            <FormRow cols={2}>
              <TextField
                name="code"
                control={control}
                label="客户编码"
                placeholder="请输入客户编码"
                required
                disabled={isEditMode}
                helperText={isEditMode ? '编码创建后不可修改' : '支持字母、数字、下划线和横线'}
              />
              <TextField
                name="name"
                control={control}
                label="客户名称"
                placeholder="请输入客户全称"
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="short_name"
                control={control}
                label="简称"
                placeholder="请输入客户简称 (可选)"
              />
              <SelectField
                name="type"
                control={control}
                label="客户类型"
                placeholder="请选择客户类型"
                options={CUSTOMER_TYPE_OPTIONS}
                required
                disabled={isEditMode}
              />
            </FormRow>
            {isEditMode && (
              <FormRow cols={2}>
                <SelectField
                  name="level"
                  control={control}
                  label="客户等级"
                  placeholder="请选择客户等级"
                  options={CUSTOMER_LEVEL_OPTIONS}
                  allowClear
                />
                <div /> {/* Empty placeholder for grid alignment */}
              </FormRow>
            )}
          </FormSection>

          <FormSection title="联系信息" description="客户的联系人和联系方式">
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
                placeholder="请输入税务登记号 (企业客户)"
              />
            </FormRow>
          </FormSection>

          <FormSection title="地址信息" description="客户的地址和邮寄信息">
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

          <FormSection title="其他设置" description="信用额度、排序等其他设置">
            <FormRow cols={2}>
              <NumberField
                name="credit_limit"
                control={control}
                label="信用额度"
                placeholder="请输入信用额度"
                min={0}
                precision={2}
                prefix="¥"
                helperText="客户允许的最大赊账金额"
              />
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

export default CustomerForm
