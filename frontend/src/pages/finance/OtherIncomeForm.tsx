import { useState, useEffect, useCallback, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography, Toast, Spin } from '@douyinfe/semi-ui'
import { useNavigate, useParams } from 'react-router-dom'
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
import type { IncomeCategory, CreateOtherIncomeRecordRequest } from '@/api/finance'
import './OtherIncomeForm.css'

const { Title } = Typography

// Income category options
const CATEGORY_OPTIONS = [
  { label: '投资收益', value: 'INVESTMENT' },
  { label: '补贴收入', value: 'SUBSIDY' },
  { label: '利息收入', value: 'INTEREST' },
  { label: '租金收入', value: 'RENTAL' },
  { label: '退款收入', value: 'REFUND' },
  { label: '赔偿收入', value: 'COMPENSATION' },
  { label: '资产处置', value: 'ASSET_DISPOSAL' },
  { label: '其他收入', value: 'OTHER' },
]

// Category values
const CATEGORIES = [
  'INVESTMENT',
  'SUBSIDY',
  'INTEREST',
  'RENTAL',
  'REFUND',
  'COMPENSATION',
  'ASSET_DISPOSAL',
  'OTHER',
] as const

// Form validation schema
const incomeFormSchema = z.object({
  category: createEnumSchema(CATEGORIES, true),
  amount: z.number().positive('金额必须大于0').max(999999999.99, '金额不能超过999,999,999.99'),
  description: z
    .string()
    .min(1, validationMessages.required)
    .max(200, validationMessages.maxLength(200)),
  received_at: z.date({ message: validationMessages.required }),
  remark: z
    .string()
    .max(500, validationMessages.maxLength(500))
    .optional()
    .transform((val) => val || undefined),
  attachment_urls: z
    .string()
    .max(2000, validationMessages.maxLength(2000))
    .optional()
    .transform((val) => val || undefined),
})

type IncomeFormData = z.infer<typeof incomeFormSchema>

/**
 * Other Income entry form page (New/Edit)
 *
 * Features:
 * - Category selection
 * - Amount input with validation
 * - Description and date
 * - Optional remark and attachments
 * - Edit mode support
 */
export default function OtherIncomeFormPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEditMode = Boolean(id)
  const api = useMemo(() => getFinanceApi(), [])

  // State
  const [initialLoading, setInitialLoading] = useState(isEditMode)

  // Form setup
  const defaultValues: Partial<IncomeFormData> = useMemo(
    () => ({
      category: undefined,
      amount: undefined,
      description: '',
      received_at: new Date(),
      remark: '',
      attachment_urls: '',
    }),
    []
  )

  const { control, handleFormSubmit, isSubmitting, reset } = useFormWithValidation<IncomeFormData>({
    schema: incomeFormSchema,
    defaultValues,
    successMessage: isEditMode ? '收入更新成功' : '收入创建成功',
    onSuccess: () => {
      navigate('/finance/incomes')
    },
  })

  // Load income data for edit mode
  useEffect(() => {
    if (isEditMode && id) {
      const loadIncome = async () => {
        setInitialLoading(true)
        try {
          const response = await api.getFinanceIncomesId(id)
          if (response.success && response.data) {
            const income = response.data
            reset({
              category: income.category as IncomeFormData['category'],
              amount: income.amount,
              description: income.description,
              received_at: new Date(income.received_at),
              remark: income.remark || '',
              attachment_urls: income.attachment_urls || '',
            })
          } else {
            Toast.error('加载收入信息失败')
            navigate('/finance/incomes')
          }
        } catch {
          Toast.error('加载收入信息失败')
          navigate('/finance/incomes')
        } finally {
          setInitialLoading(false)
        }
      }
      loadIncome()
    }
  }, [isEditMode, id, api, reset, navigate])

  // Handle form submission
  const onSubmit = useCallback(
    async (data: IncomeFormData) => {
      const request: CreateOtherIncomeRecordRequest = {
        category: data.category as IncomeCategory,
        amount: data.amount,
        description: data.description,
        received_at: data.received_at.toISOString(),
        remark: data.remark,
        attachment_urls: data.attachment_urls,
      }

      if (isEditMode && id) {
        const response = await api.putFinanceIncomesId(id, request)
        if (!response.success) {
          throw new Error(response.error || '更新收入失败')
        }
      } else {
        const response = await api.postFinanceIncomes(request)
        if (!response.success) {
          throw new Error(response.error || '创建收入失败')
        }
      }
    },
    [api, id, isEditMode]
  )

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/finance/incomes')
  }, [navigate])

  if (initialLoading) {
    return (
      <Container size="md" className="other-income-form-page">
        <Card className="other-income-form-card">
          <div className="other-income-form-loading">
            <Spin size="large" />
          </div>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="md" className="other-income-form-page">
      <Card className="other-income-form-card">
        <div className="other-income-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? '编辑收入' : '新增收入'}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection title="收入信息" description="填写收入基本信息">
            <FormRow cols={2}>
              <SelectField
                name="category"
                control={control}
                label="收入分类"
                placeholder="请选择收入分类"
                options={CATEGORY_OPTIONS}
                required
              />
              <NumberField
                name="amount"
                control={control}
                label="金额"
                placeholder="请输入金额"
                required
                min={0.01}
                max={999999999.99}
                precision={2}
                prefix="¥"
              />
            </FormRow>
            <FormRow cols={2}>
              <DateField
                name="received_at"
                control={control}
                label="收入日期"
                placeholder="请选择收入日期"
                required
              />
              <div />
            </FormRow>
            <FormRow cols={1}>
              <TextField
                name="description"
                control={control}
                label="收入描述"
                placeholder="请输入收入描述"
                required
                maxLength={200}
              />
            </FormRow>
          </FormSection>

          <FormSection title="其他信息" description="备注和附件（可选）">
            <TextAreaField
              name="remark"
              control={control}
              label="备注"
              placeholder="请输入备注信息（可选）"
              rows={3}
              maxCount={500}
            />
            <TextAreaField
              name="attachment_urls"
              control={control}
              label="附件链接"
              placeholder="请输入附件URL，多个URL请用逗号分隔（可选）"
              rows={2}
              helperText="支持输入多个URL，用逗号分隔"
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
