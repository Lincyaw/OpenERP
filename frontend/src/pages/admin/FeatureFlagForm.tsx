import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import {
  Card,
  Typography,
  Toast,
  Spin,
  Slider,
  Tag,
  Space,
  InputNumber,
} from '@douyinfe/semi-ui-19'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Form,
  FormActions,
  FormSection,
  FormRow,
  TextField,
  TextAreaField,
  SelectField,
  SwitchField,
  useFormWithValidation,
  validationMessages,
  createEnumSchema,
} from '@/components/common/form'
import { Container } from '@/components/common/layout'
import { listFeatureFlagFlags } from '@/api/feature-flags'
import type {
  FlagType,
  CreateFlagRequest,
  UpdateFlagRequest,
  FlagValue,
  TargetingRule,
} from '@/api/feature-flags'
import { VariantEditor, type Variant } from './components/VariantEditor'
import { RulesEditor } from './components/RulesEditor'
import { TagInputField } from './components/TagInputField'
import './FeatureFlagForm.css'

const { Title, Text } = Typography

// Flag type values
const FLAG_TYPES = ['boolean', 'percentage', 'variant', 'user_segment'] as const

// Helper type for translation function
type TranslateFunc = (key: string, fallback?: string) => string

// Form validation schema
const createFlagFormSchema = (t: TranslateFunc, isEditMode: boolean) =>
  z
    .object({
      key: isEditMode
        ? z.string() // Key is read-only in edit mode
        : z
            .string()
            .min(2, t('featureFlags.form.keyMinError', 'Key must be at least 2 characters'))
            .max(100, t('featureFlags.form.keyMaxError', 'Key must be at most 100 characters'))
            .regex(
              /^[a-z][a-z0-9_]*$/,
              t(
                'featureFlags.form.keyRegexError',
                'Key must start with lowercase letter and contain only lowercase letters, numbers, and underscores'
              )
            ),
      name: z
        .string()
        .min(2, t('featureFlags.form.nameMinError', 'Name must be at least 2 characters'))
        .max(200, t('featureFlags.form.nameMaxError', 'Name must be at most 200 characters')),
      description: z
        .string()
        .max(500, validationMessages.maxLength(500))
        .optional()
        .transform((val) => val || undefined),
      type: createEnumSchema(FLAG_TYPES, true),
      tags: z.array(z.string()).optional(),
      // Boolean type
      defaultEnabled: z.boolean().optional(),
      // Percentage type
      defaultPercentage: z.number().min(0).max(100).optional(),
      // Variant type - will be validated separately
      variants: z
        .array(
          z.object({
            name: z.string().min(1),
            weight: z.number().min(0).max(100),
          })
        )
        .optional(),
      defaultVariant: z.string().optional(),
    })
    .refine(
      (data) => {
        // Variant weights must sum to 100
        if (data.type === 'variant' && data.variants && data.variants.length > 0) {
          const totalWeight = data.variants.reduce((sum, v) => sum + v.weight, 0)
          return Math.abs(totalWeight - 100) < 0.01
        }
        return true
      },
      {
        message: t('featureFlags.form.variantWeightError', 'Variant weights must sum to 100%'),
        path: ['variants'],
      }
    )

type FlagFormData = z.infer<ReturnType<typeof createFlagFormSchema>>

/**
 * Feature Flag Create/Edit Form
 *
 * Features:
 * - Create and edit feature flags
 * - Dynamic default value field based on type
 * - Variant editor with weight distribution
 * - Rules editor for targeting
 * - Form validation with Zod
 */
export default function FeatureFlagFormPage() {
  const { t } = useTranslation('admin')
  const navigate = useNavigate()
  const { key } = useParams<{ key: string }>()
  const isEditMode = Boolean(key)
  const api = useMemo(() => listFeatureFlagFlags(), [])

  // State
  const [initialLoading, setInitialLoading] = useState(isEditMode)
  const [selectedType, setSelectedType] = useState<FlagType>('boolean')
  const [rules, setRules] = useState<TargetingRule[]>([])
  const [version, setVersion] = useState(1)

  // Type options
  const typeOptions = useMemo(
    () => [
      { label: t('featureFlags.type.boolean', 'Boolean'), value: 'boolean' },
      { label: t('featureFlags.type.percentage', 'Percentage'), value: 'percentage' },
      { label: t('featureFlags.type.variant', 'Variant'), value: 'variant' },
      { label: t('featureFlags.type.user_segment', 'User Segment'), value: 'user_segment' },
    ],
    [t]
  )

  // Create schema with translated validation messages
  // Create a wrapper function for translation that matches the expected signature
  const translateForSchema: TranslateFunc = (key, fallback) => {
    const result = t(key, { defaultValue: fallback })
    return typeof result === 'string' ? result : (fallback ?? key)
  }
  const flagFormSchema = useMemo(
    () => createFlagFormSchema(translateForSchema, isEditMode),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, isEditMode]
  )

  // Default form values
  const defaultValues: Partial<FlagFormData> = useMemo(
    () => ({
      key: '',
      name: '',
      description: '',
      type: 'boolean' as FlagType,
      tags: [],
      defaultEnabled: false,
      defaultPercentage: 50,
      variants: [
        { name: 'A', weight: 50 },
        { name: 'B', weight: 50 },
      ],
      defaultVariant: 'A',
    }),
    []
  )

  const { control, handleFormSubmit, isSubmitting, reset, watch, setValue } =
    useFormWithValidation<FlagFormData>({
      schema: flagFormSchema,
      defaultValues,
      successMessage: isEditMode
        ? t('featureFlags.messages.updateSuccess', 'Feature flag updated successfully')
        : t('featureFlags.messages.createSuccess', 'Feature flag created successfully'),
      onSuccess: () => {
        navigate('/admin/feature-flags')
      },
    })

  // Watch type field for dynamic rendering
  const watchedType = watch('type')

  // Update selected type when form value changes
  useEffect(() => {
    if (watchedType) {
      setSelectedType(watchedType as FlagType)
    }
  }, [watchedType])

  // Load flag data for edit mode
  useEffect(() => {
    if (isEditMode && key) {
      const loadFlag = async () => {
        setInitialLoading(true)
        try {
          const response = await api.getFlag(key)
          if (response.success && response.data) {
            const flag = response.data
            setSelectedType(flag.type)
            setVersion(flag.version)
            setRules(flag.rules || [])

            // Map flag data to form values
            const formData: Partial<FlagFormData> = {
              key: flag.key,
              name: flag.name,
              description: flag.description || '',
              type: flag.type,
              tags: flag.tags || [],
              defaultEnabled: flag.default_value.enabled,
              defaultPercentage: 50, // Will be parsed from metadata
              variants: [],
              defaultVariant: flag.default_value.variant,
            }

            // Parse type-specific values
            if (
              flag.type === 'percentage' &&
              flag.default_value.metadata?.percentage !== undefined
            ) {
              formData.defaultPercentage = flag.default_value.metadata.percentage as number
            }

            if (flag.type === 'variant' && flag.default_value.metadata?.variants) {
              formData.variants = flag.default_value.metadata.variants as Variant[]
            }

            reset(formData)
          } else {
            Toast.error(t('featureFlags.messages.loadError', 'Failed to load feature flag'))
            navigate('/admin/feature-flags')
          }
        } catch {
          Toast.error(t('featureFlags.messages.loadError', 'Failed to load feature flag'))
          navigate('/admin/feature-flags')
        } finally {
          setInitialLoading(false)
        }
      }
      loadFlag()
    }
  }, [isEditMode, key, api, reset, navigate, t])

  // Build FlagValue from form data
  const buildFlagValue = useCallback((data: FlagFormData): FlagValue => {
    const value: FlagValue = {
      enabled: data.defaultEnabled || false,
      variant: undefined,
      metadata: {},
    }

    switch (data.type) {
      case 'boolean':
        value.enabled = data.defaultEnabled || false
        break
      case 'percentage':
        value.enabled = true
        value.metadata = { percentage: data.defaultPercentage || 50 }
        break
      case 'variant':
        value.enabled = true
        value.variant = data.defaultVariant
        value.metadata = { variants: data.variants }
        break
      case 'user_segment':
        value.enabled = data.defaultEnabled || false
        break
    }

    return value
  }, [])

  // Handle form submission
  const onSubmit = useCallback(
    async (data: FlagFormData) => {
      const flagValue = buildFlagValue(data)

      if (isEditMode && key) {
        const request: UpdateFlagRequest = {
          name: data.name,
          description: data.description,
          default_value: flagValue,
          rules: rules.length > 0 ? rules : undefined,
          tags: data.tags,
          version, // For optimistic locking
        }

        const response = await api.updateFlag(key, request)
        if (!response.success) {
          throw new Error(
            response.error?.message ||
              t('featureFlags.messages.updateError', 'Failed to update feature flag')
          )
        }
      } else {
        const request: CreateFlagRequest = {
          key: data.key,
          name: data.name,
          description: data.description,
          type: data.type as FlagType,
          default_value: flagValue,
          rules: rules.length > 0 ? rules : undefined,
          tags: data.tags,
        }

        const response = await api.createFlag(request)
        if (!response.success) {
          throw new Error(
            response.error?.message ||
              t('featureFlags.messages.createError', 'Failed to create feature flag')
          )
        }
      }
    },
    [api, key, isEditMode, rules, version, buildFlagValue, t]
  )

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/admin/feature-flags')
  }, [navigate])

  // Render loading state
  if (initialLoading) {
    return (
      <Container size="md" className="feature-flag-form-page">
        <Card className="feature-flag-form-card">
          <div className="feature-flag-form-loading">
            <Spin size="large" />
          </div>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="md" className="feature-flag-form-page">
      <Card className="feature-flag-form-card">
        <div className="feature-flag-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode
              ? t('featureFlags.editTitle', 'Edit Feature Flag')
              : t('featureFlags.createTitle', 'Create Feature Flag')}
          </Title>
        </div>

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          {/* Basic Information Section */}
          <FormSection
            title={t('featureFlags.form.basicInfo', 'Basic Information')}
            description={t(
              'featureFlags.form.basicInfoDescription',
              'Configure the basic settings for the feature flag'
            )}
          >
            <FormRow cols={2}>
              <TextField
                name="key"
                control={control}
                label={t('featureFlags.form.key', 'Key')}
                placeholder={t(
                  'featureFlags.form.keyPlaceholder',
                  'Enter flag key, e.g., new_checkout_flow'
                )}
                required
                disabled={isEditMode}
                helperText={
                  isEditMode
                    ? t('featureFlags.form.keyReadonly', 'Key cannot be changed after creation')
                    : t('featureFlags.form.keyHelp', 'Use snake_case, e.g., new_feature_enabled')
                }
              />
              <SelectField
                name="type"
                control={control}
                label={t('featureFlags.form.type', 'Type')}
                placeholder={t('featureFlags.form.typePlaceholder', 'Select type')}
                options={typeOptions}
                required
                disabled={isEditMode}
                helperText={
                  isEditMode
                    ? t('featureFlags.form.typeReadonly', 'Type cannot be changed after creation')
                    : undefined
                }
              />
            </FormRow>
            <FormRow cols={1}>
              <TextField
                name="name"
                control={control}
                label={t('featureFlags.form.name', 'Name')}
                placeholder={t('featureFlags.form.namePlaceholder', 'Enter feature flag name')}
                required
                maxLength={200}
              />
            </FormRow>
            <FormRow cols={1}>
              <TextAreaField
                name="description"
                control={control}
                label={t('featureFlags.form.description', 'Description')}
                placeholder={t(
                  'featureFlags.form.descriptionPlaceholder',
                  'Enter description (optional)'
                )}
                rows={3}
                maxCount={500}
              />
            </FormRow>
            <FormRow cols={1}>
              <TagInputField
                name="tags"
                control={control}
                label={t('featureFlags.form.tags', 'Tags')}
                placeholder={t('featureFlags.form.tagsPlaceholder', 'Enter tags and press Enter')}
              />
            </FormRow>
          </FormSection>

          {/* Default Value Section - Dynamic based on type */}
          <FormSection
            title={t('featureFlags.form.defaultValue', 'Default Value')}
            description={t(
              'featureFlags.form.defaultValueDescription',
              'Configure the default value for users not matching any targeting rules'
            )}
          >
            {selectedType === 'boolean' && (
              <FormRow cols={1}>
                <SwitchField
                  name="defaultEnabled"
                  control={control}
                  label={t('featureFlags.form.enabled', 'Enabled')}
                  checkedText={t('featureFlags.status.enabled', 'Enabled')}
                  uncheckedText={t('featureFlags.status.disabled', 'Disabled')}
                />
              </FormRow>
            )}

            {selectedType === 'percentage' && (
              <FormRow cols={1}>
                <PercentageSlider
                  control={control}
                  setValue={setValue}
                  watch={watch}
                  label={t('featureFlags.form.percentage', 'Rollout Percentage')}
                />
              </FormRow>
            )}

            {selectedType === 'variant' && (
              <FormRow cols={1}>
                <VariantEditor control={control} setValue={setValue} watch={watch} />
              </FormRow>
            )}

            {selectedType === 'user_segment' && (
              <FormRow cols={1}>
                <SwitchField
                  name="defaultEnabled"
                  control={control}
                  label={t(
                    'featureFlags.form.defaultForNonMatching',
                    'Default for non-matching users'
                  )}
                  checkedText={t('featureFlags.status.enabled', 'Enabled')}
                  uncheckedText={t('featureFlags.status.disabled', 'Disabled')}
                  helperText={t(
                    'featureFlags.form.defaultForNonMatchingHelp',
                    'Value for users not matching any targeting rules'
                  )}
                />
              </FormRow>
            )}
          </FormSection>

          {/* Targeting Rules Section */}
          <FormSection
            title={t('featureFlags.form.targetingRules', 'Targeting Rules')}
            description={t(
              'featureFlags.form.targetingRulesDescription',
              'Configure rules to target specific users or segments'
            )}
            collapsible
            defaultCollapsed={rules.length === 0}
          >
            <RulesEditor rules={rules} onChange={setRules} flagType={selectedType} />
          </FormSection>

          <FormActions
            submitText={
              isEditMode
                ? t('featureFlags.actions.save', 'Save')
                : t('featureFlags.actions.create', 'Create')
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

/**
 * Percentage slider component with InputNumber
 */
interface PercentageSliderProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  control: any
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setValue: any
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  watch: any
  label: string
}

function PercentageSlider({ setValue, watch, label }: PercentageSliderProps) {
  const { t } = useTranslation('admin')
  const value = watch('defaultPercentage') || 50

  return (
    <div className="percentage-slider-container">
      <Text strong className="percentage-slider-label">
        {label}
      </Text>
      <div className="percentage-slider-content">
        <Slider
          min={0}
          max={100}
          step={1}
          value={value}
          onChange={(val) => setValue('defaultPercentage', val as number)}
          tipFormatter={(val) => `${val}%`}
          style={{ flex: 1 }}
        />
        <InputNumber
          min={0}
          max={100}
          value={value}
          onChange={(val) => setValue('defaultPercentage', val as number)}
          formatter={(val) => `${val}%`}
          parser={(val) => (val ? String(Number(val.replace('%', ''))) : '0')}
          style={{ width: 100, marginLeft: 16 }}
        />
      </div>
      <Space className="percentage-slider-preview">
        <Tag color="green">
          {t('featureFlags.form.enabledUsers', 'Enabled')}: {value}%
        </Tag>
        <Tag color="grey">
          {t('featureFlags.form.disabledUsers', 'Disabled')}: {100 - value}%
        </Tag>
      </Space>
    </div>
  )
}
