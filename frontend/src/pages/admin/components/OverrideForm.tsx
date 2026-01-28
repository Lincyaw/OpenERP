import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Form, Button, Toast, Space } from '@douyinfe/semi-ui-19'
import type { FormApi } from '@douyinfe/semi-ui-19/lib/es/form/interface'
import { createFeatureFlagOverride } from '@/api/feature-flags/feature-flags'
import type {
  DtoFlagValueDTO,
  HandlerCreateFlagHTTPRequestType,
  HandlerCreateOverrideHTTPRequest,
  HandlerCreateOverrideHTTPRequestTargetType,
  DtoOverrideResponse,
} from '@/api/models'

// Type aliases for cleaner code
type FlagType = HandlerCreateFlagHTTPRequestType
type FlagValue = DtoFlagValueDTO
type Override = DtoOverrideResponse
type OverrideTargetType = HandlerCreateOverrideHTTPRequestTargetType

interface OverrideFormProps {
  flagKey: string
  flagType: FlagType | string | undefined
  override?: Override | null
  onSuccess: () => void
  onCancel: () => void
}

/**
 * Override Form Component
 *
 * Features:
 * - Create new override
 * - Target type selection (User/Tenant)
 * - Target ID input
 * - Value configuration based on flag type
 * - Reason input
 * - Expiration date (optional)
 */
export function OverrideForm({
  flagKey,
  flagType,
  override,
  onSuccess,
  onCancel,
}: OverrideFormProps) {
  const { t } = useTranslation('admin')

  // State
  const [loading, setLoading] = useState(false)
  const [formApi, setFormApi] = useState<FormApi | null>(null)

  // Target type options
  const targetTypeOptions = useMemo(
    () => [
      { label: t('featureFlags.overrides.targetTypes.user', 'User'), value: 'user' },
      { label: t('featureFlags.overrides.targetTypes.tenant', 'Tenant'), value: 'tenant' },
    ],
    [t]
  )

  // Build flag value from form data
  const buildFlagValue = useCallback(
    (formData: Record<string, unknown>): FlagValue => {
      const value: FlagValue = {
        enabled: false,
        variant: undefined,
        metadata: {},
      }

      if (flagType === 'boolean' || flagType === 'user_segment') {
        value.enabled = Boolean(formData.valueEnabled)
      } else if (flagType === 'percentage') {
        value.enabled = true
        value.metadata = { percentage: formData.valuePercentage as number }
      } else if (flagType === 'variant') {
        value.enabled = true
        value.variant = formData.valueVariant as string
      }

      return value
    },
    [flagType]
  )

  // Handle submit
  const handleSubmit = useCallback(async () => {
    if (!formApi) return

    try {
      await formApi.validate()
      const values = formApi.getValues()
      setLoading(true)

      const flagValue = buildFlagValue(values)

      const request: HandlerCreateOverrideHTTPRequest = {
        target_type: values.targetType as OverrideTargetType,
        target_id: values.targetId as string,
        value: flagValue,
        reason: (values.reason as string) || undefined,
        expires_at: values.expiresAt ? new Date(values.expiresAt as Date).toISOString() : undefined,
      }

      const response = await createFeatureFlagOverride(flagKey, request)

      if (response.status === 201 && response.data.success) {
        Toast.success(t('featureFlags.overrides.createSuccess', 'Override created successfully'))
        onSuccess()
      } else {
        Toast.error(
          response.data.error?.message ||
            t('featureFlags.overrides.createError', 'Failed to create override')
        )
      }
    } catch {
      // Validation failed or API error
      Toast.error(t('featureFlags.overrides.createError', 'Failed to create override'))
    } finally {
      setLoading(false)
    }
  }, [formApi, flagKey, buildFlagValue, t, onSuccess])

  // Initial form values
  const initValues = useMemo(() => {
    if (override) {
      return {
        targetType: override.target_type,
        targetId: override.target_id,
        valueEnabled: override.value?.enabled,
        valuePercentage: (override.value?.metadata?.percentage as number) || 50,
        valueVariant: override.value?.variant || '',
        reason: override.reason || '',
        expiresAt: override.expires_at ? new Date(override.expires_at) : undefined,
      }
    }
    return {
      targetType: 'user',
      targetId: '',
      valueEnabled: true,
      valuePercentage: 100,
      valueVariant: '',
      reason: '',
      expiresAt: undefined,
    }
  }, [override])

  return (
    <Form getFormApi={setFormApi} initValues={initValues} labelPosition="left" labelWidth={120}>
      {/* Target Type */}
      <Form.Select
        field="targetType"
        label={t('featureFlags.overrides.targetType', 'Target Type')}
        placeholder={t('featureFlags.overrides.selectTargetType', 'Select target type')}
        optionList={targetTypeOptions}
        rules={[
          {
            required: true,
            message: t('featureFlags.overrides.targetTypeRequired', 'Please select target type'),
          },
        ]}
      />

      {/* Target ID */}
      <Form.Input
        field="targetId"
        label={t('featureFlags.overrides.targetId', 'Target ID')}
        placeholder={t('featureFlags.overrides.targetIdPlaceholder', 'Enter user ID or tenant ID')}
        rules={[
          {
            required: true,
            message: t('featureFlags.overrides.targetIdRequired', 'Please enter target ID'),
          },
        ]}
      />

      {/* Value Section based on flag type */}
      <div className="override-value-section">
        <Form.Label text={t('featureFlags.overrides.value', 'Override Value')} />
        {(flagType === 'boolean' || flagType === 'user_segment') && (
          <Form.Switch
            field="valueEnabled"
            checkedText={t('featureFlags.status.enabled', 'Enabled')}
            uncheckedText={t('featureFlags.status.disabled', 'Disabled')}
          />
        )}
        {flagType === 'percentage' && (
          <Form.InputNumber
            field="valuePercentage"
            min={0}
            max={100}
            formatter={(val) => `${val}%`}
            parser={(val) => (val ? String(Number(val.replace('%', ''))) : '0')}
            style={{ width: 120 }}
          />
        )}
        {flagType === 'variant' && (
          <Form.Input
            field="valueVariant"
            placeholder={t('featureFlags.form.variantName', 'Variant name')}
            rules={[
              {
                required: true,
                message: t('featureFlags.overrides.variantRequired', 'Please enter variant name'),
              },
            ]}
          />
        )}
      </div>

      {/* Reason */}
      <Form.TextArea
        field="reason"
        label={t('featureFlags.overrides.reason', 'Reason')}
        placeholder={t(
          'featureFlags.overrides.reasonPlaceholder',
          'Why is this override needed? (optional)'
        )}
        rows={3}
        maxLength={500}
      />

      {/* Expiration Date */}
      <Form.DatePicker
        field="expiresAt"
        label={t('featureFlags.overrides.expiresAt', 'Expires At')}
        placeholder={t(
          'featureFlags.overrides.expiresAtPlaceholder',
          'Select expiration date (optional)'
        )}
        type="dateTime"
        style={{ width: '100%' }}
        disabledDate={(date) => (date ? date < new Date() : false)}
      />

      {/* Actions */}
      <div className="override-form-actions">
        <Space>
          <Button onClick={onCancel}>{t('common.cancel', 'Cancel')}</Button>
          <Button theme="solid" onClick={handleSubmit} loading={loading}>
            {t('common.create', 'Create')}
          </Button>
        </Space>
      </div>
    </Form>
  )
}

export default OverrideForm
