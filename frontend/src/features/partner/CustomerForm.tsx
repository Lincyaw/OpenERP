import { useEffect, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography } from '@douyinfe/semi-ui'
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
import { getCustomers } from '@/api/customers/customers'
import type { HandlerCustomerResponse } from '@/api/models'
import './CustomerForm.css'

const { Title } = Typography

// Customer type and level values
const CUSTOMER_TYPES = ['individual', 'organization'] as const
const CUSTOMER_LEVELS = ['normal', 'silver', 'gold', 'platinum', 'vip'] as const

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
  const { t } = useTranslation(['partner', 'common'])
  const api = useMemo(() => getCustomers(), [])
  const isEditMode = Boolean(customerId)

  // Customer type options with translations
  const CUSTOMER_TYPE_OPTIONS = useMemo(
    () => [
      { label: t('customers.type.individual'), value: 'individual' },
      { label: t('customers.type.organization'), value: 'organization' },
    ],
    [t]
  )

  // Customer level options with translations
  const CUSTOMER_LEVEL_OPTIONS = useMemo(
    () => [
      { label: t('customers.level.normal'), value: 'normal' },
      { label: t('customers.level.silver'), value: 'silver' },
      { label: t('customers.level.gold'), value: 'gold' },
      { label: t('customers.level.platinum'), value: 'platinum' },
      { label: t('customers.level.vip'), value: 'vip' },
    ],
    [t]
  )

  // Form validation schema with translations
  const customerFormSchema = useMemo(
    () =>
      z.object({
        code: z
          .string()
          .min(1, validationMessages.required)
          .max(50, validationMessages.maxLength(50))
          .regex(/^[A-Za-z0-9_-]+$/, t('customers.form.codeRegexError')),
        name: z
          .string()
          .min(1, validationMessages.required)
          .max(200, validationMessages.maxLength(200)),
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
          .email(t('customers.form.emailError'))
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
      }),
    [t]
  )

  type CustomerFormData = z.infer<typeof customerFormSchema>

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
      successMessage: isEditMode
        ? t('customers.messages.updateSuccess')
        : t('customers.messages.createSuccess'),
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
        throw new Error(response.error?.message || t('customers.messages.updateError'))
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
        throw new Error(response.error?.message || t('customers.messages.createError'))
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
            {isEditMode ? t('customers.editCustomer') : t('customers.addCustomer')}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection
            title={t('customers.form.basicInfo')}
            description={t('customers.form.basicInfoDesc')}
          >
            <FormRow cols={2}>
              <TextField
                name="code"
                control={control}
                label={t('customers.form.code')}
                placeholder={t('customers.form.codePlaceholder')}
                required
                disabled={isEditMode}
                helperText={
                  isEditMode
                    ? t('customers.form.codeHelperEdit')
                    : t('customers.form.codeHelperCreate')
                }
              />
              <TextField
                name="name"
                control={control}
                label={t('customers.form.name')}
                placeholder={t('customers.form.namePlaceholder')}
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="short_name"
                control={control}
                label={t('customers.form.shortName')}
                placeholder={t('customers.form.shortNamePlaceholder')}
              />
              <SelectField
                name="type"
                control={control}
                label={t('customers.form.customerType')}
                placeholder={t('customers.form.customerTypePlaceholder')}
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
                  label={t('customers.form.customerLevel')}
                  placeholder={t('customers.form.customerLevelPlaceholder')}
                  options={CUSTOMER_LEVEL_OPTIONS}
                  allowClear
                />
                <div /> {/* Empty placeholder for grid alignment */}
              </FormRow>
            )}
          </FormSection>

          <FormSection
            title={t('customers.form.contactInfo')}
            description={t('customers.form.contactInfoDesc')}
          >
            <FormRow cols={2}>
              <TextField
                name="contact_name"
                control={control}
                label={t('customers.form.contactName')}
                placeholder={t('customers.form.contactNamePlaceholder')}
              />
              <TextField
                name="phone"
                control={control}
                label={t('customers.form.phone')}
                placeholder={t('customers.form.phonePlaceholder')}
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="email"
                control={control}
                label={t('customers.form.email')}
                placeholder={t('customers.form.emailPlaceholder')}
              />
              <TextField
                name="tax_id"
                control={control}
                label={t('customers.form.taxId')}
                placeholder={t('customers.form.taxIdPlaceholder')}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('customers.form.addressInfo')}
            description={t('customers.form.addressInfoDesc')}
          >
            <FormRow cols={3}>
              <TextField
                name="country"
                control={control}
                label={t('customers.form.country')}
                placeholder={t('customers.form.countryPlaceholder')}
              />
              <TextField
                name="province"
                control={control}
                label={t('customers.form.province')}
                placeholder={t('customers.form.provincePlaceholder')}
              />
              <TextField
                name="city"
                control={control}
                label={t('customers.form.city')}
                placeholder={t('customers.form.cityPlaceholder')}
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="address"
                control={control}
                label={t('customers.form.address')}
                placeholder={t('customers.form.addressPlaceholder')}
              />
              <TextField
                name="postal_code"
                control={control}
                label={t('customers.form.postalCode')}
                placeholder={t('customers.form.postalCodePlaceholder')}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('customers.form.otherSettings')}
            description={t('customers.form.otherSettingsDesc')}
          >
            <FormRow cols={2}>
              <NumberField
                name="credit_limit"
                control={control}
                label={t('customers.form.creditLimit')}
                placeholder={t('customers.form.creditLimitPlaceholder')}
                min={0}
                precision={2}
                prefix="¥"
                helperText={t('customers.form.creditLimitHelper')}
              />
              <NumberField
                name="sort_order"
                control={control}
                label={t('customers.form.sortOrder')}
                placeholder={t('customers.form.sortOrderPlaceholder')}
                min={0}
                precision={0}
                helperText={t('customers.form.sortOrderHelper')}
              />
            </FormRow>
            <TextAreaField
              name="notes"
              control={control}
              label={t('customers.form.notes')}
              placeholder={t('customers.form.notesPlaceholder')}
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

export default CustomerForm
