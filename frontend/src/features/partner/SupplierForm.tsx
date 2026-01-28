import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography, Rating } from '@douyinfe/semi-ui-19'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
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

// Supplier type values
const SUPPLIER_TYPES = ['manufacturer', 'distributor', 'retailer', 'service'] as const

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
  const { t } = useTranslation(['partner', 'common'])
  const api = useMemo(() => getSuppliers(), [])
  const isEditMode = Boolean(supplierId)

  // Supplier type options with translations
  const SUPPLIER_TYPE_OPTIONS = useMemo(
    () => [
      { label: t('suppliers.type.manufacturer'), value: 'manufacturer' },
      { label: t('suppliers.type.distributor'), value: 'distributor' },
      { label: t('suppliers.type.retailer'), value: 'retailer' },
      { label: t('suppliers.type.service'), value: 'service' },
    ],
    [t]
  )

  // Form validation schema with translations
  const supplierFormSchema = useMemo(
    () =>
      z.object({
        code: z
          .string()
          .min(1, validationMessages.required)
          .max(50, validationMessages.maxLength(50))
          .regex(/^[A-Za-z0-9_-]+$/, t('suppliers.form.codeRegexError')),
        name: z
          .string()
          .min(1, validationMessages.required)
          .max(200, validationMessages.maxLength(200)),
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
          .email(t('suppliers.form.emailError'))
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
          .min(0, t('suppliers.form.ratingMinError'))
          .max(5, t('suppliers.form.ratingMaxError'))
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
      }),
    [t]
  )

  type SupplierFormData = z.infer<typeof supplierFormSchema>

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
        ((initialData as { type?: string }).type as
          | 'manufacturer'
          | 'distributor'
          | 'retailer'
          | 'service') || 'manufacturer',
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
      successMessage: isEditMode
        ? t('suppliers.messages.updateSuccess')
        : t('suppliers.messages.createSuccess'),
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
      const response = await api.updateSupplier(supplierId, {
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
        throw new Error(response.error?.message || t('suppliers.messages.updateError'))
      }
    } else {
      // Create new supplier
      const response = await api.createSupplier({
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
        throw new Error(response.error?.message || t('suppliers.messages.createError'))
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
            {isEditMode ? t('suppliers.editSupplier') : t('suppliers.addSupplier')}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection
            title={t('suppliers.form.basicInfo')}
            description={t('suppliers.form.basicInfoDesc')}
          >
            <FormRow cols={2}>
              <TextField
                name="code"
                control={control}
                label={t('suppliers.form.code')}
                placeholder={t('suppliers.form.codePlaceholder')}
                required
                disabled={isEditMode}
                helperText={
                  isEditMode
                    ? t('suppliers.form.codeHelperEdit')
                    : t('suppliers.form.codeHelperCreate')
                }
              />
              <TextField
                name="name"
                control={control}
                label={t('suppliers.form.name')}
                placeholder={t('suppliers.form.namePlaceholder')}
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="short_name"
                control={control}
                label={t('suppliers.form.shortName')}
                placeholder={t('suppliers.form.shortNamePlaceholder')}
              />
              <SelectField
                name="type"
                control={control}
                label={t('suppliers.form.supplierType')}
                placeholder={t('suppliers.form.supplierTypePlaceholder')}
                options={SUPPLIER_TYPE_OPTIONS}
                required
                disabled={isEditMode}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('suppliers.form.contactInfo')}
            description={t('suppliers.form.contactInfoDesc')}
          >
            <FormRow cols={2}>
              <TextField
                name="contact_name"
                control={control}
                label={t('suppliers.form.contactName')}
                placeholder={t('suppliers.form.contactNamePlaceholder')}
              />
              <TextField
                name="phone"
                control={control}
                label={t('suppliers.form.phone')}
                placeholder={t('suppliers.form.phonePlaceholder')}
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="email"
                control={control}
                label={t('suppliers.form.email')}
                placeholder={t('suppliers.form.emailPlaceholder')}
              />
              <TextField
                name="tax_id"
                control={control}
                label={t('suppliers.form.taxId')}
                placeholder={t('suppliers.form.taxIdPlaceholder')}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('suppliers.form.addressInfo')}
            description={t('suppliers.form.addressInfoDesc')}
          >
            <FormRow cols={3}>
              <TextField
                name="country"
                control={control}
                label={t('suppliers.form.country')}
                placeholder={t('suppliers.form.countryPlaceholder')}
              />
              <TextField
                name="province"
                control={control}
                label={t('suppliers.form.province')}
                placeholder={t('suppliers.form.provincePlaceholder')}
              />
              <TextField
                name="city"
                control={control}
                label={t('suppliers.form.city')}
                placeholder={t('suppliers.form.cityPlaceholder')}
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="address"
                control={control}
                label={t('suppliers.form.address')}
                placeholder={t('suppliers.form.addressPlaceholder')}
              />
              <TextField
                name="postal_code"
                control={control}
                label={t('suppliers.form.postalCode')}
                placeholder={t('suppliers.form.postalCodePlaceholder')}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('suppliers.form.bankInfo')}
            description={t('suppliers.form.bankInfoDesc')}
          >
            <FormRow cols={2}>
              <TextField
                name="bank_name"
                control={control}
                label={t('suppliers.form.bankName')}
                placeholder={t('suppliers.form.bankNamePlaceholder')}
              />
              <TextField
                name="bank_account"
                control={control}
                label={t('suppliers.form.bankAccount')}
                placeholder={t('suppliers.form.bankAccountPlaceholder')}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('suppliers.form.purchaseSettings')}
            description={t('suppliers.form.purchaseSettingsDesc')}
          >
            <FormRow cols={2}>
              <NumberField
                name="credit_limit"
                control={control}
                label={t('suppliers.form.creditLimit')}
                placeholder={t('suppliers.form.creditLimitPlaceholder')}
                min={0}
                precision={2}
                prefix="¥"
                helperText={t('suppliers.form.creditLimitHelper')}
              />
              <NumberField
                name="credit_days"
                control={control}
                label={t('suppliers.form.creditDays')}
                placeholder={t('suppliers.form.creditDaysPlaceholder')}
                min={0}
                precision={0}
                suffix={t('suppliers.form.creditDaysSuffix')}
                helperText={t('suppliers.form.creditDaysHelper')}
              />
            </FormRow>
            <FormRow cols={2}>
              <div className="rating-field">
                <Text className="rating-label">{t('suppliers.form.rating')}</Text>
                <div className="rating-wrapper">
                  <Rating
                    value={ratingValue ?? 0}
                    onChange={handleRatingChange}
                    allowHalf
                    size="default"
                  />
                  <Text type="tertiary" className="rating-value">
                    {ratingValue !== undefined
                      ? t('suppliers.form.ratingScore', { score: ratingValue })
                      : t('suppliers.form.ratingNotRated')}
                  </Text>
                </div>
                <Text type="tertiary" className="rating-helper">
                  {t('suppliers.form.ratingHelper')}
                </Text>
              </div>
              <NumberField
                name="sort_order"
                control={control}
                label={t('suppliers.form.sortOrder')}
                placeholder={t('suppliers.form.sortOrderPlaceholder')}
                min={0}
                precision={0}
                helperText={t('suppliers.form.sortOrderHelper')}
              />
            </FormRow>
            <TextAreaField
              name="notes"
              control={control}
              label={t('suppliers.form.notes')}
              placeholder={t('suppliers.form.notesPlaceholder')}
              rows={3}
              maxCount={2000}
            />
          </FormSection>

          <FormActions
            submitText={isEditMode ? t('common:actions.save') : t('common:actions.create')}
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
