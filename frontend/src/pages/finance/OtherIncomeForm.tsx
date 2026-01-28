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
import { getIncomeIncome, createIncomeIncome, updateIncomeIncome } from '@/api/incomes/incomes'
import type { CreateIncomeIncomeBody } from '@/api/models'
import './OtherIncomeForm.css'

const { Title } = Typography

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
const createIncomeFormSchema = (t: (key: string) => string) =>
  z.object({
    category: createEnumSchema(CATEGORIES, true),
    amount: z
      .number()
      .positive(t('otherIncomeForm.validation.amountPositive'))
      .max(999999999.99, t('otherIncomeForm.validation.amountMax')),
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

type IncomeFormData = z.infer<ReturnType<typeof createIncomeFormSchema>>

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
  const { t } = useTranslation('finance')
  const isEditMode = Boolean(id)

  // Memoized schema with translations
  const incomeFormSchema = useMemo(() => createIncomeFormSchema(t), [t])

  // Memoized category options with translations
  const categoryOptions = useMemo(
    () => [
      { label: t('otherIncomes.category.INVESTMENT'), value: 'INVESTMENT' },
      { label: t('otherIncomes.category.SUBSIDY'), value: 'SUBSIDY' },
      { label: t('otherIncomes.category.INTEREST'), value: 'INTEREST' },
      { label: t('otherIncomes.category.RENTAL'), value: 'RENTAL' },
      { label: t('otherIncomes.category.REFUND'), value: 'REFUND' },
      { label: t('otherIncomes.category.COMPENSATION'), value: 'COMPENSATION' },
      { label: t('otherIncomes.category.ASSET_DISPOSAL'), value: 'ASSET_DISPOSAL' },
      { label: t('otherIncomes.category.OTHER'), value: 'OTHER' },
    ],
    [t]
  )

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
    successMessage: isEditMode
      ? t('otherIncomeForm.messages.updateSuccess')
      : t('otherIncomeForm.messages.createSuccess'),
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
          const response = await getIncomeIncome(id)
          if (response.status === 200 && response.data.success && response.data.data) {
            const income = response.data.data
            reset({
              category: income.category as IncomeFormData['category'],
              amount: income.amount,
              description: income.description,
              received_at: new Date(income.received_at || ''),
              remark: income.remark || '',
              attachment_urls: income.attachment_urls || '',
            })
          } else {
            Toast.error(t('otherIncomeForm.messages.loadError'))
            navigate('/finance/incomes')
          }
        } catch {
          Toast.error(t('otherIncomeForm.messages.loadError'))
          navigate('/finance/incomes')
        } finally {
          setInitialLoading(false)
        }
      }
      loadIncome()
    }
  }, [isEditMode, id, reset, navigate, t])

  // Handle form submission
  const onSubmit = useCallback(
    async (data: IncomeFormData) => {
      const request: CreateIncomeIncomeBody = {
        category: data.category as CreateIncomeIncomeBody['category'],
        amount: data.amount,
        description: data.description,
        received_at: data.received_at.toISOString(),
        remark: data.remark,
        attachment_urls: data.attachment_urls,
      }

      if (isEditMode && id) {
        const response = await updateIncomeIncome(id, request)
        if (response.status !== 200 || !response.data.success) {
          throw new Error(response.data.error?.message || t('otherIncomeForm.messages.updateError'))
        }
      } else {
        const response = await createIncomeIncome(request)
        if (response.status !== 201 || !response.data.success) {
          throw new Error(response.data.error?.message || t('otherIncomeForm.messages.createError'))
        }
      }
    },
    [id, isEditMode, t]
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
            {isEditMode ? t('otherIncomeForm.editTitle') : t('otherIncomeForm.createTitle')}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection
            title={t('otherIncomeForm.incomeInfo.title')}
            description={t('otherIncomeForm.incomeInfo.description')}
          >
            <FormRow cols={2}>
              <SelectField
                name="category"
                control={control}
                label={t('otherIncomeForm.incomeInfo.category')}
                placeholder={t('otherIncomeForm.incomeInfo.categoryPlaceholder')}
                options={categoryOptions}
                required
              />
              <NumberField
                name="amount"
                control={control}
                label={t('otherIncomeForm.incomeInfo.amount')}
                placeholder={t('otherIncomeForm.incomeInfo.amountPlaceholder')}
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
                label={t('otherIncomeForm.incomeInfo.receivedAt')}
                placeholder={t('otherIncomeForm.incomeInfo.receivedAtPlaceholder')}
                required
              />
              <div />
            </FormRow>
            <FormRow cols={1}>
              <TextField
                name="description"
                control={control}
                label={t('otherIncomeForm.incomeInfo.description')}
                placeholder={t('otherIncomeForm.incomeInfo.descriptionPlaceholder')}
                required
                maxLength={200}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('otherIncomeForm.otherInfo.title')}
            description={t('otherIncomeForm.otherInfo.description')}
          >
            <TextAreaField
              name="remark"
              control={control}
              label={t('otherIncomeForm.otherInfo.remark')}
              placeholder={t('otherIncomeForm.otherInfo.remarkPlaceholder')}
              rows={3}
              maxCount={500}
            />
            <TextAreaField
              name="attachment_urls"
              control={control}
              label={t('otherIncomeForm.otherInfo.attachmentUrls')}
              placeholder={t('otherIncomeForm.otherInfo.attachmentUrlsPlaceholder')}
              rows={2}
              helperText={t('otherIncomeForm.otherInfo.attachmentUrlsHelper')}
            />
          </FormSection>

          <FormActions
            submitText={
              isEditMode ? t('otherIncomeForm.actions.save') : t('otherIncomeForm.actions.create')
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
