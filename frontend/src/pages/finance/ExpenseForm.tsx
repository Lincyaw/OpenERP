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
import type { ExpenseCategory, CreateExpenseRecordRequest } from '@/api/finance'
import './ExpenseForm.css'

const { Title } = Typography

// Expense category options
const CATEGORY_OPTIONS = [
  { label: '房租', value: 'RENT' },
  { label: '水电费', value: 'UTILITIES' },
  { label: '工资', value: 'SALARY' },
  { label: '办公费', value: 'OFFICE' },
  { label: '差旅费', value: 'TRAVEL' },
  { label: '市场营销', value: 'MARKETING' },
  { label: '设备费', value: 'EQUIPMENT' },
  { label: '维修费', value: 'MAINTENANCE' },
  { label: '保险费', value: 'INSURANCE' },
  { label: '税费', value: 'TAX' },
  { label: '其他费用', value: 'OTHER' },
]

// Category values
const CATEGORIES = [
  'RENT',
  'UTILITIES',
  'SALARY',
  'OFFICE',
  'TRAVEL',
  'MARKETING',
  'EQUIPMENT',
  'MAINTENANCE',
  'INSURANCE',
  'TAX',
  'OTHER',
] as const

// Form validation schema
const expenseFormSchema = z.object({
  category: createEnumSchema(CATEGORIES, true),
  amount: z.number().positive('金额必须大于0').max(999999999.99, '金额不能超过999,999,999.99'),
  description: z
    .string()
    .min(1, validationMessages.required)
    .max(200, validationMessages.maxLength(200)),
  incurred_at: z.date({ message: validationMessages.required }),
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

type ExpenseFormData = z.infer<typeof expenseFormSchema>

/**
 * Expense entry form page (New/Edit)
 *
 * Features:
 * - Category selection
 * - Amount input with validation
 * - Description and date
 * - Optional remark and attachments
 * - Edit mode support
 */
export default function ExpenseFormPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEditMode = Boolean(id)
  const api = useMemo(() => getFinanceApi(), [])

  // State
  const [initialLoading, setInitialLoading] = useState(isEditMode)

  // Form setup
  const defaultValues: Partial<ExpenseFormData> = useMemo(
    () => ({
      category: undefined,
      amount: undefined,
      description: '',
      incurred_at: new Date(),
      remark: '',
      attachment_urls: '',
    }),
    []
  )

  const { control, handleFormSubmit, isSubmitting, reset } = useFormWithValidation<ExpenseFormData>(
    {
      schema: expenseFormSchema,
      defaultValues,
      successMessage: isEditMode ? '费用更新成功' : '费用创建成功',
      onSuccess: () => {
        navigate('/finance/expenses')
      },
    }
  )

  // Load expense data for edit mode
  useEffect(() => {
    if (isEditMode && id) {
      const loadExpense = async () => {
        setInitialLoading(true)
        try {
          const response = await api.getFinanceExpensesId(id)
          if (response.success && response.data) {
            const expense = response.data
            reset({
              category: expense.category as ExpenseFormData['category'],
              amount: expense.amount,
              description: expense.description,
              incurred_at: new Date(expense.incurred_at),
              remark: expense.remark || '',
              attachment_urls: expense.attachment_urls || '',
            })
          } else {
            Toast.error('加载费用信息失败')
            navigate('/finance/expenses')
          }
        } catch {
          Toast.error('加载费用信息失败')
          navigate('/finance/expenses')
        } finally {
          setInitialLoading(false)
        }
      }
      loadExpense()
    }
  }, [isEditMode, id, api, reset, navigate])

  // Handle form submission
  const onSubmit = useCallback(
    async (data: ExpenseFormData) => {
      const request: CreateExpenseRecordRequest = {
        category: data.category as ExpenseCategory,
        amount: data.amount,
        description: data.description,
        incurred_at: data.incurred_at.toISOString(),
        remark: data.remark,
        attachment_urls: data.attachment_urls,
      }

      if (isEditMode && id) {
        const response = await api.putFinanceExpensesId(id, request)
        if (!response.success) {
          throw new Error(response.error || '更新费用失败')
        }
      } else {
        const response = await api.postFinanceExpenses(request)
        if (!response.success) {
          throw new Error(response.error || '创建费用失败')
        }
      }
    },
    [api, id, isEditMode]
  )

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/finance/expenses')
  }, [navigate])

  if (initialLoading) {
    return (
      <Container size="md" className="expense-form-page">
        <Card className="expense-form-card">
          <div className="expense-form-loading">
            <Spin size="large" />
          </div>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="md" className="expense-form-page">
      <Card className="expense-form-card">
        <div className="expense-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? '编辑费用' : '新增费用'}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection title="费用信息" description="填写费用基本信息">
            <FormRow cols={2}>
              <SelectField
                name="category"
                control={control}
                label="费用分类"
                placeholder="请选择费用分类"
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
                name="incurred_at"
                control={control}
                label="发生日期"
                placeholder="请选择发生日期"
                required
              />
              <div />
            </FormRow>
            <FormRow cols={1}>
              <TextField
                name="description"
                control={control}
                label="费用描述"
                placeholder="请输入费用描述"
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
