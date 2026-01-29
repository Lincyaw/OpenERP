import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import { Card, Typography, Switch } from '@douyinfe/semi-ui-19'
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
import { createWarehouse, updateWarehouse } from '@/api/warehouses/warehouses'
import type { HandlerWarehouseResponse } from '@/api/models'
import './WarehouseForm.css'

const { Title, Text } = Typography

// Warehouse type values (matching API)
const WAREHOUSE_TYPES = ['physical', 'virtual', 'consign', 'transit'] as const

interface WarehouseFormProps {
  /** Warehouse ID for edit mode, undefined for create mode */
  warehouseId?: string
  /** Initial warehouse data for edit mode */
  initialData?: HandlerWarehouseResponse
}

/**
 * Warehouse form component for creating and editing warehouses
 *
 * Features:
 * - Zod schema validation
 * - Create/edit modes
 * - Form sections for better organization (basic info, contact, address, settings)
 * - API integration with error handling
 * - Default warehouse toggle
 * - Full i18n support
 */
export function WarehouseForm({ warehouseId, initialData }: WarehouseFormProps) {
  const { t } = useTranslation(['partner', 'common'])
  const navigate = useNavigate()
  const isEditMode = Boolean(warehouseId)

  // Memoized warehouse type options with translations
  const warehouseTypeOptions = useMemo(
    () => [
      { label: t('warehouses.type.physical'), value: 'physical' },
      { label: t('warehouses.type.virtual'), value: 'virtual' },
      { label: t('warehouses.type.consign'), value: 'consign' },
      { label: t('warehouses.type.transit'), value: 'transit' },
    ],
    [t]
  )

  // Form validation schema with translated error messages
  const warehouseFormSchema = useMemo(
    () =>
      z.object({
        code: z
          .string()
          .min(1, validationMessages.required)
          .max(50, validationMessages.maxLength(50))
          .regex(/^[A-Za-z0-9_-]+$/, t('warehouses.form.codeRegexError')),
        name: z
          .string()
          .min(1, validationMessages.required)
          .max(200, validationMessages.maxLength(200)),
        short_name: z
          .string()
          .max(100, validationMessages.maxLength(100))
          .optional()
          .transform((val) => val || undefined),
        type: createEnumSchema(WAREHOUSE_TYPES, true),
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
          .email(t('warehouses.form.emailError'))
          .max(200, validationMessages.maxLength(200))
          .optional()
          .or(z.literal(''))
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
        capacity: z
          .number()
          .int(validationMessages.integer)
          .nonnegative(validationMessages.nonNegative)
          .optional()
          .nullable()
          .transform((val) => val ?? undefined),
        is_default: z.boolean().optional().default(false),
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

  type WarehouseFormData = z.infer<typeof warehouseFormSchema>

  // Transform API data to form values
  const defaultValues: Partial<WarehouseFormData> = useMemo(() => {
    if (!initialData) {
      return {
        code: '',
        name: '',
        short_name: '',
        type: 'physical' as const,
        contact_name: '',
        phone: '',
        email: '',
        country: '中国',
        province: '',
        city: '',
        postal_code: '',
        address: '',
        capacity: undefined,
        is_default: false,
        sort_order: undefined,
        notes: '',
      }
    }
    return {
      code: initialData.code || '',
      name: initialData.name || '',
      short_name: initialData.short_name || '',
      type: (initialData.type as 'physical' | 'virtual' | 'consign' | 'transit') || 'physical',
      contact_name: initialData.manager_name || '',
      phone: initialData.phone || '',
      email: initialData.email || '',
      country: initialData.country || '中国',
      province: initialData.province || '',
      city: initialData.city || '',
      postal_code: initialData.postal_code || '',
      address: initialData.address || '',
      capacity: initialData.sort_order ?? undefined, // API doesn't return capacity, using sort_order as placeholder
      is_default: initialData.is_default ?? false,
      sort_order: initialData.sort_order ?? undefined,
      notes: initialData.notes || '',
    }
  }, [initialData])

  const { control, handleFormSubmit, isSubmitting, reset, watch, setValue } =
    useFormWithValidation<WarehouseFormData>({
      schema: warehouseFormSchema,
      defaultValues,
      successMessage: isEditMode
        ? t('warehouses.messages.updateSuccess')
        : t('warehouses.messages.createSuccess'),
      onSuccess: () => {
        navigate('/partner/warehouses')
      },
    })

  // Watch is_default for the switch component
  const isDefaultValue = watch('is_default')

  // Reset form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      reset(defaultValues)
    }
  }, [initialData, defaultValues, reset])

  // Handle form submission
  const onSubmit = async (data: WarehouseFormData) => {
    if (isEditMode && warehouseId) {
      // Update existing warehouse
      const response = await updateWarehouse(warehouseId, {
        name: data.name,
        short_name: data.short_name,
        contact_name: data.contact_name,
        phone: data.phone,
        email: data.email,
        country: data.country,
        province: data.province,
        city: data.city,
        postal_code: data.postal_code,
        address: data.address,
        capacity: data.capacity,
        is_default: data.is_default,
        sort_order: data.sort_order,
        notes: data.notes,
      })
      if (response.status !== 200 || !response.data.success) {
        const error = response.data.error as { message?: string } | undefined
        throw new Error(error?.message || t('warehouses.messages.updateError'))
      }
    } else {
      // Create new warehouse
      const response = await createWarehouse({
        code: data.code,
        name: data.name,
        short_name: data.short_name,
        type: data.type as 'physical' | 'virtual' | 'consign' | 'transit',
        contact_name: data.contact_name,
        phone: data.phone,
        email: data.email,
        country: data.country,
        province: data.province,
        city: data.city,
        postal_code: data.postal_code,
        address: data.address,
        capacity: data.capacity,
        is_default: data.is_default,
        sort_order: data.sort_order,
        notes: data.notes,
      })
      if (response.status !== 201 || !response.data.success) {
        const error = response.data.error as { message?: string } | undefined
        throw new Error(error?.message || t('warehouses.messages.createError'))
      }
    }
  }

  const handleCancel = () => {
    navigate('/partner/warehouses')
  }

  const handleDefaultChange = (checked: boolean) => {
    setValue('is_default', checked)
  }

  return (
    <Container size="md" className="warehouse-form-page">
      <Card className="warehouse-form-card">
        <div className="warehouse-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? t('warehouses.editWarehouse') : t('warehouses.addWarehouse')}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection
            title={t('warehouses.form.basicInfo')}
            description={t('warehouses.form.basicInfoDesc')}
          >
            <FormRow cols={2}>
              <TextField
                name="code"
                control={control}
                label={t('warehouses.form.code')}
                placeholder={t('warehouses.form.codePlaceholder')}
                required
                disabled={isEditMode}
                helperText={
                  isEditMode
                    ? t('warehouses.form.codeHelperEdit')
                    : t('warehouses.form.codeHelperCreate')
                }
              />
              <TextField
                name="name"
                control={control}
                label={t('warehouses.form.name')}
                placeholder={t('warehouses.form.namePlaceholder')}
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="short_name"
                control={control}
                label={t('warehouses.form.shortName')}
                placeholder={t('warehouses.form.shortNamePlaceholder')}
              />
              <SelectField
                name="type"
                control={control}
                label={t('warehouses.form.warehouseType')}
                placeholder={t('warehouses.form.warehouseTypePlaceholder')}
                options={warehouseTypeOptions}
                required
                disabled={isEditMode}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('warehouses.form.contactInfo')}
            description={t('warehouses.form.contactInfoDesc')}
          >
            <FormRow cols={2}>
              <TextField
                name="contact_name"
                control={control}
                label={t('warehouses.form.manager')}
                placeholder={t('warehouses.form.managerPlaceholder')}
              />
              <TextField
                name="phone"
                control={control}
                label={t('warehouses.form.phone')}
                placeholder={t('warehouses.form.phonePlaceholder')}
              />
            </FormRow>
            <TextField
              name="email"
              control={control}
              label={t('warehouses.form.email')}
              placeholder={t('warehouses.form.emailPlaceholder')}
            />
          </FormSection>

          <FormSection
            title={t('warehouses.form.addressInfo')}
            description={t('warehouses.form.addressInfoDesc')}
          >
            <FormRow cols={3}>
              <TextField
                name="country"
                control={control}
                label={t('warehouses.form.country')}
                placeholder={t('warehouses.form.countryPlaceholder')}
              />
              <TextField
                name="province"
                control={control}
                label={t('warehouses.form.province')}
                placeholder={t('warehouses.form.provincePlaceholder')}
              />
              <TextField
                name="city"
                control={control}
                label={t('warehouses.form.city')}
                placeholder={t('warehouses.form.cityPlaceholder')}
              />
            </FormRow>
            <FormRow cols={2}>
              <TextField
                name="address"
                control={control}
                label={t('warehouses.form.address')}
                placeholder={t('warehouses.form.addressPlaceholder')}
              />
              <TextField
                name="postal_code"
                control={control}
                label={t('warehouses.form.postalCode')}
                placeholder={t('warehouses.form.postalCodePlaceholder')}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('warehouses.form.warehouseSettings')}
            description={t('warehouses.form.warehouseSettingsDesc')}
          >
            <FormRow cols={2}>
              <NumberField
                name="capacity"
                control={control}
                label={t('warehouses.form.capacity')}
                placeholder={t('warehouses.form.capacityPlaceholder')}
                min={0}
                precision={0}
                suffix={t('warehouses.form.capacitySuffix')}
                helperText={t('warehouses.form.capacityHelper')}
              />
              <NumberField
                name="sort_order"
                control={control}
                label={t('warehouses.form.sortOrder')}
                placeholder={t('warehouses.form.sortOrderPlaceholder')}
                min={0}
                precision={0}
                helperText={t('warehouses.form.sortOrderHelper')}
              />
            </FormRow>
            <div className="default-field">
              <div className="default-field-content">
                <div className="default-field-label">
                  <Text strong>{t('warehouses.form.isDefault')}</Text>
                  <Text type="tertiary" className="default-field-helper">
                    {t('warehouses.form.isDefaultHelper')}
                  </Text>
                </div>
                <Switch
                  checked={isDefaultValue}
                  onChange={handleDefaultChange}
                  checkedText={t('warehouses.form.isDefaultYes')}
                  uncheckedText={t('warehouses.form.isDefaultNo')}
                />
              </div>
            </div>
            <TextAreaField
              name="notes"
              control={control}
              label={t('warehouses.form.notes')}
              placeholder={t('warehouses.form.notesPlaceholder')}
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

export default WarehouseForm
