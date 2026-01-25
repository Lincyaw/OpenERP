import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import { Card, Typography, Toast, Spin } from '@douyinfe/semi-ui-19'
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

// Form validation schema - moved inside component for i18n support
const createExpenseFormSchema = (t: (key: string) => string) =>
  z.object({
    category: createEnumSchema(CATEGORIES, true),
    amount: z
      .number()
      .positive(t('expenseForm.validation.amountPositive'))
      .max(999999999.99, t('expenseForm.validation.amountMax')),
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

type ExpenseFormData = z.infer<ReturnType<typeof createExpenseFormSchema>>

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
  const { t } = useTranslation('finance')
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEditMode = Boolean(id)
  const api = useMemo(() => getFinanceApi(), [])

  // Expense category options with translated labels
  const categoryOptions = useMemo(
    () => [
      { label: t('expenses.category.RENT'), value: 'RENT' },
      { label: t('expenses.category.UTILITIES'), value: 'UTILITIES' },
      { label: t('expenses.category.SALARY'), value: 'SALARY' },
      { label: t('expenses.category.OFFICE'), value: 'OFFICE' },
      { label: t('expenses.category.TRAVEL'), value: 'TRAVEL' },
      { label: t('expenses.category.MARKETING'), value: 'MARKETING' },
      { label: t('expenses.category.EQUIPMENT'), value: 'EQUIPMENT' },
      { label: t('expenses.category.MAINTENANCE'), value: 'MAINTENANCE' },
      { label: t('expenses.category.INSURANCE'), value: 'INSURANCE' },
      { label: t('expenses.category.TAX'), value: 'TAX' },
      { label: t('expenses.category.OTHER'), value: 'OTHER' },
    ],
    [t]
  )

  // Create schema with translated validation messages
  const expenseFormSchema = useMemo(() => createExpenseFormSchema(t), [t])

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
      successMessage: isEditMode
        ? t('expenseForm.messages.updateSuccess')
        : t('expenseForm.messages.createSuccess'),
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
            Toast.error(t('expenseForm.messages.loadError'))
            navigate('/finance/expenses')
          }
        } catch {
          Toast.error(t('expenseForm.messages.loadError'))
          navigate('/finance/expenses')
        } finally {
          setInitialLoading(false)
        }
      }
      loadExpense()
    }
  }, [isEditMode, id, api, reset, navigate, t])

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
          throw new Error(response.error || t('expenseForm.messages.updateError'))
        }
      } else {
        const response = await api.postFinanceExpenses(request)
        if (!response.success) {
          throw new Error(response.error || t('expenseForm.messages.createError'))
        }
      }
    },
    [api, id, isEditMode, t]
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
            {isEditMode ? t('expenseForm.editTitle') : t('expenseForm.createTitle')}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection
            title={t('expenseForm.expenseInfo.title')}
            description={t('expenseForm.expenseInfo.description')}
          >
            <FormRow cols={2}>
              <SelectField
                name="category"
                control={control}
                label={t('expenseForm.expenseInfo.category')}
                placeholder={t('expenseForm.expenseInfo.categoryPlaceholder')}
                options={categoryOptions}
                required
              />
              <NumberField
                name="amount"
                control={control}
                label={t('expenseForm.expenseInfo.amount')}
                placeholder={t('expenseForm.expenseInfo.amountPlaceholder')}
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
                label={t('expenseForm.expenseInfo.incurredAt')}
                placeholder={t('expenseForm.expenseInfo.incurredAtPlaceholder')}
                required
              />
              <div />
            </FormRow>
            <FormRow cols={1}>
              <TextField
                name="description"
                control={control}
                label={t('expenseForm.expenseInfo.description')}
                placeholder={t('expenseForm.expenseInfo.descriptionPlaceholder')}
                required
                maxLength={200}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('expenseForm.otherInfo.title')}
            description={t('expenseForm.otherInfo.description')}
          >
            <TextAreaField
              name="remark"
              control={control}
              label={t('expenseForm.otherInfo.remark')}
              placeholder={t('expenseForm.otherInfo.remarkPlaceholder')}
              rows={3}
              maxCount={500}
            />
            <TextAreaField
              name="attachment_urls"
              control={control}
              label={t('expenseForm.otherInfo.attachmentUrls')}
              placeholder={t('expenseForm.otherInfo.attachmentUrlsPlaceholder')}
              rows={2}
              helperText={t('expenseForm.otherInfo.attachmentUrlsHelper')}
            />
          </FormSection>

          <FormActions
            submitText={
              isEditMode ? t('expenseForm.actions.save') : t('expenseForm.actions.create')
            }
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}
