import { useCallback, useImperativeHandle, forwardRef } from 'react'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Form, Space, Spin } from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import type { HandlerTenantResponse } from '@/api/models'

// Plan options
const PLAN_OPTIONS = [
  { label: 'Free', value: 'free' },
  { label: 'Basic', value: 'basic' },
  { label: 'Pro', value: 'pro' },
  { label: 'Enterprise', value: 'enterprise' },
]

// Create tenant form schema
export const createTenantSchema = z.object({
  name: z.string().min(1, 'Name is required').max(200, 'Name must be at most 200 characters'),
  code: z
    .string()
    .min(2, 'Code must be at least 2 characters')
    .max(50, 'Code must be at most 50 characters')
    .regex(
      /^[a-z][a-z0-9_-]*$/,
      'Code must start with lowercase letter and contain only lowercase letters, numbers, underscores, and hyphens'
    ),
  short_name: z
    .string()
    .max(100, 'Short name must be at most 100 characters')
    .optional()
    .or(z.literal('')),
  contact_email: z
    .string()
    .email('Invalid email format')
    .max(200, 'Email must be at most 200 characters')
    .optional()
    .or(z.literal('')),
  contact_name: z
    .string()
    .max(100, 'Contact name must be at most 100 characters')
    .optional()
    .or(z.literal('')),
  contact_phone: z
    .string()
    .max(50, 'Phone must be at most 50 characters')
    .optional()
    .or(z.literal('')),
  address: z
    .string()
    .max(500, 'Address must be at most 500 characters')
    .optional()
    .or(z.literal('')),
  domain: z.string().max(200, 'Domain must be at most 200 characters').optional().or(z.literal('')),
  plan: z.enum(['free', 'basic', 'pro', 'enterprise']).optional(),
  trial_days: z
    .number()
    .int()
    .min(1, 'Trial days must be at least 1')
    .max(365, 'Trial days must be at most 365')
    .optional(),
  notes: z.string().optional().or(z.literal('')),
})

// Edit tenant form schema (subset of create schema)
export const editTenantSchema = z.object({
  name: z.string().min(1, 'Name is required').max(200, 'Name must be at most 200 characters'),
  short_name: z
    .string()
    .max(100, 'Short name must be at most 100 characters')
    .optional()
    .or(z.literal('')),
  contact_email: z
    .string()
    .email('Invalid email format')
    .max(200, 'Email must be at most 200 characters')
    .optional()
    .or(z.literal('')),
  contact_name: z
    .string()
    .max(100, 'Contact name must be at most 100 characters')
    .optional()
    .or(z.literal('')),
  contact_phone: z
    .string()
    .max(50, 'Phone must be at most 50 characters')
    .optional()
    .or(z.literal('')),
  address: z
    .string()
    .max(500, 'Address must be at most 500 characters')
    .optional()
    .or(z.literal('')),
  domain: z.string().max(200, 'Domain must be at most 200 characters').optional().or(z.literal('')),
  notes: z.string().optional().or(z.literal('')),
})

export type CreateTenantFormData = z.infer<typeof createTenantSchema>
export type EditTenantFormData = z.infer<typeof editTenantSchema>

// Form handle type for imperative methods
export interface TenantFormHandle {
  submit: () => void
  reset: () => void
}

interface TenantFormBaseProps {
  isLoading?: boolean
}

interface CreateTenantFormProps extends TenantFormBaseProps {
  mode: 'create'
  initialData?: never
  onSubmit: (data: CreateTenantFormData) => void
}

interface EditTenantFormProps extends TenantFormBaseProps {
  mode: 'edit'
  initialData?: HandlerTenantResponse
  onSubmit: (data: EditTenantFormData) => void
}

type TenantFormProps = CreateTenantFormProps | EditTenantFormProps

/**
 * TenantForm component for creating and editing tenants
 *
 * Features:
 * - React Hook Form + Zod validation
 * - Different fields for create vs edit mode
 * - Loading state support
 * - Keyboard navigation (Enter to submit)
 */
function TenantFormInner(
  { mode, initialData, onSubmit, isLoading = false }: TenantFormProps,
  ref: React.ForwardedRef<TenantFormHandle>
) {
  const { t } = useTranslation('admin')

  // Use appropriate schema based on mode
  const isCreateMode = mode === 'create'

  const {
    control,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateTenantFormData | EditTenantFormData>({
    resolver: zodResolver(isCreateMode ? createTenantSchema : editTenantSchema),
    defaultValues: isCreateMode
      ? {
          name: '',
          code: '',
          short_name: '',
          contact_email: '',
          contact_name: '',
          contact_phone: '',
          address: '',
          domain: '',
          plan: 'free',
          trial_days: 14,
          notes: '',
        }
      : {
          name: initialData?.name || '',
          short_name: initialData?.short_name || '',
          contact_email: initialData?.contact_email || '',
          contact_name: initialData?.contact_name || '',
          contact_phone: initialData?.contact_phone || '',
          address: initialData?.address || '',
          domain: initialData?.domain || '',
          notes: initialData?.notes || '',
        },
  })

  // Expose submit and reset methods via useImperativeHandle
  useImperativeHandle(
    ref,
    () => ({
      submit: handleSubmit(onSubmit as (data: CreateTenantFormData | EditTenantFormData) => void),
      reset: () => reset(),
    }),
    [handleSubmit, onSubmit, reset]
  )

  // Handle keyboard submit
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey && e.target instanceof HTMLInputElement) {
        e.preventDefault()
        handleSubmit(onSubmit as (data: CreateTenantFormData | EditTenantFormData) => void)()
      }
    },
    [handleSubmit, onSubmit]
  )

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: 24 }}>
        <Spin size="large" />
      </div>
    )
  }

  return (
    <Form onKeyDown={handleKeyDown} labelPosition="top" style={{ width: '100%' }}>
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        {/* Name - Required */}
        <Controller
          name="name"
          control={control}
          render={({ field }) => (
            <Form.Input
              {...field}
              field="name"
              label={t('tenants.form.name', 'Name')}
              placeholder={t('tenants.form.namePlaceholder', 'Enter tenant name')}
              validateStatus={errors.name ? 'error' : undefined}
              extraText={errors.name?.message}
              style={{ width: '100%' }}
              showClear
            />
          )}
        />

        {/* Code - Only for create mode */}
        {isCreateMode && (
          <Controller
            name="code"
            control={control}
            render={({ field }) => (
              <Form.Input
                {...field}
                field="code"
                label={t('tenants.form.code', 'Code')}
                placeholder={t(
                  'tenants.form.codePlaceholder',
                  'Enter tenant code (e.g., acme-corp)'
                )}
                validateStatus={
                  (errors as { code?: { message?: string } }).code ? 'error' : undefined
                }
                extraText={
                  (errors as { code?: { message?: string } }).code?.message ||
                  t(
                    'tenants.form.codeHelp',
                    'Unique identifier, lowercase letters, numbers, underscores, and hyphens'
                  )
                }
                style={{ width: '100%' }}
                showClear
              />
            )}
          />
        )}

        {/* Short Name */}
        <Controller
          name="short_name"
          control={control}
          render={({ field }) => (
            <Form.Input
              {...field}
              field="short_name"
              label={t('tenants.form.shortName', 'Short Name')}
              placeholder={t('tenants.form.shortNamePlaceholder', 'Enter short name (optional)')}
              validateStatus={errors.short_name ? 'error' : undefined}
              extraText={errors.short_name?.message}
              style={{ width: '100%' }}
              showClear
            />
          )}
        />

        {/* Contact Email */}
        <Controller
          name="contact_email"
          control={control}
          render={({ field }) => (
            <Form.Input
              {...field}
              field="contact_email"
              label={t('tenants.form.contactEmail', 'Contact Email')}
              placeholder={t('tenants.form.contactEmailPlaceholder', 'Enter contact email')}
              validateStatus={errors.contact_email ? 'error' : undefined}
              extraText={errors.contact_email?.message}
              style={{ width: '100%' }}
              showClear
            />
          )}
        />

        {/* Contact Name */}
        <Controller
          name="contact_name"
          control={control}
          render={({ field }) => (
            <Form.Input
              {...field}
              field="contact_name"
              label={t('tenants.form.contactName', 'Contact Name')}
              placeholder={t('tenants.form.contactNamePlaceholder', 'Enter contact name')}
              validateStatus={errors.contact_name ? 'error' : undefined}
              extraText={errors.contact_name?.message}
              style={{ width: '100%' }}
              showClear
            />
          )}
        />

        {/* Contact Phone */}
        <Controller
          name="contact_phone"
          control={control}
          render={({ field }) => (
            <Form.Input
              {...field}
              field="contact_phone"
              label={t('tenants.form.contactPhone', 'Contact Phone')}
              placeholder={t('tenants.form.contactPhonePlaceholder', 'Enter contact phone')}
              validateStatus={errors.contact_phone ? 'error' : undefined}
              extraText={errors.contact_phone?.message}
              style={{ width: '100%' }}
              showClear
            />
          )}
        />

        {/* Plan - Only for create mode */}
        {isCreateMode && (
          <Controller
            name="plan"
            control={control}
            render={({ field }) => (
              <Form.Select
                {...field}
                field="plan"
                label={t('tenants.form.plan', 'Initial Plan')}
                placeholder={t('tenants.form.planPlaceholder', 'Select a plan')}
                optionList={PLAN_OPTIONS}
                style={{ width: '100%' }}
              />
            )}
          />
        )}

        {/* Trial Days - Only for create mode */}
        {isCreateMode && (
          <Controller
            name="trial_days"
            control={control}
            render={({ field }) => (
              <Form.InputNumber
                {...field}
                field="trial_days"
                label={t('tenants.form.trialDays', 'Trial Days')}
                placeholder={t('tenants.form.trialDaysPlaceholder', 'Enter trial days')}
                min={1}
                max={365}
                validateStatus={
                  (errors as { trial_days?: { message?: string } }).trial_days ? 'error' : undefined
                }
                extraText={(errors as { trial_days?: { message?: string } }).trial_days?.message}
                style={{ width: '100%' }}
              />
            )}
          />
        )}

        {/* Domain */}
        <Controller
          name="domain"
          control={control}
          render={({ field }) => (
            <Form.Input
              {...field}
              field="domain"
              label={t('tenants.form.domain', 'Domain')}
              placeholder={t('tenants.form.domainPlaceholder', 'Enter domain (optional)')}
              validateStatus={errors.domain ? 'error' : undefined}
              extraText={errors.domain?.message}
              style={{ width: '100%' }}
              showClear
            />
          )}
        />

        {/* Address */}
        <Controller
          name="address"
          control={control}
          render={({ field }) => (
            <Form.TextArea
              {...field}
              field="address"
              label={t('tenants.form.address', 'Address')}
              placeholder={t('tenants.form.addressPlaceholder', 'Enter address (optional)')}
              validateStatus={errors.address ? 'error' : undefined}
              extraText={errors.address?.message}
              style={{ width: '100%' }}
              rows={2}
            />
          )}
        />

        {/* Notes */}
        <Controller
          name="notes"
          control={control}
          render={({ field }) => (
            <Form.TextArea
              {...field}
              field="notes"
              label={t('tenants.form.notes', 'Notes')}
              placeholder={t('tenants.form.notesPlaceholder', 'Enter notes (optional)')}
              validateStatus={errors.notes ? 'error' : undefined}
              extraText={errors.notes?.message}
              style={{ width: '100%' }}
              rows={3}
            />
          )}
        />
      </Space>
    </Form>
  )
}

// Export forwardRef wrapped component
export const TenantForm = forwardRef(TenantFormInner) as <T extends 'create' | 'edit'>(
  props: (T extends 'create' ? CreateTenantFormProps : EditTenantFormProps) & {
    ref?: React.ForwardedRef<TenantFormHandle>
  }
) => React.ReactElement

export default TenantForm
