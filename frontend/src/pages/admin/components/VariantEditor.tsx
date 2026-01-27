import { useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Button, Input, InputNumber, Slider, Select, Typography, Space } from '@douyinfe/semi-ui-19'
import { IconPlus, IconDelete } from '@douyinfe/semi-icons'
import type { Control, UseFormWatch, UseFormSetValue } from 'react-hook-form'

const { Text } = Typography

export interface Variant {
  name: string
  weight: number
}

interface VariantEditorProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  control: Control<any>
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  watch: UseFormWatch<any>
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setValue: UseFormSetValue<any>
}

// Color palette for variant segments
const VARIANT_COLORS = [
  '#0077FA', // Primary blue
  '#F5222D', // Red
  '#52C41A', // Green
  '#FA8C16', // Orange
  '#722ED1', // Purple
  '#13C2C2', // Cyan
  '#EB2F96', // Pink
  '#FAAD14', // Yellow
]

/**
 * VariantEditor component for managing feature flag variants
 *
 * Features:
 * - Add/remove variants
 * - Adjust variant weights with slider
 * - Real-time weight distribution visualization
 * - Default variant selection
 */
export function VariantEditor({ watch, setValue }: VariantEditorProps) {
  const { t } = useTranslation('admin')

  const watchedVariants = watch('variants')
  const watchedDefaultVariant = watch('defaultVariant')

  // Memoize variants to avoid dependency changes on every render
  const variants: Variant[] = useMemo(() => watchedVariants || [], [watchedVariants])
  const defaultVariant = useMemo(() => watchedDefaultVariant || '', [watchedDefaultVariant])

  // Calculate total weight
  const totalWeight = useMemo(() => {
    return variants.reduce((sum, v) => sum + v.weight, 0)
  }, [variants])

  // Check if weights are valid (sum to 100)
  const isWeightValid = useMemo(() => {
    return Math.abs(totalWeight - 100) < 0.01
  }, [totalWeight])

  // Add new variant
  const handleAddVariant = useCallback(() => {
    const newVariants = [...variants]
    const nextLetter = String.fromCharCode(65 + newVariants.length) // A, B, C, ...

    // Calculate remaining weight
    const remainingWeight = Math.max(0, 100 - totalWeight)
    newVariants.push({ name: nextLetter, weight: remainingWeight })

    setValue('variants', newVariants)

    // Set default variant if this is the first one
    if (!defaultVariant && newVariants.length > 0) {
      setValue('defaultVariant', newVariants[0].name)
    }
  }, [variants, totalWeight, defaultVariant, setValue])

  // Remove variant
  const handleRemoveVariant = useCallback(
    (index: number) => {
      if (variants.length <= 2) {
        return // Keep at least 2 variants
      }

      const newVariants = variants.filter((_, i) => i !== index)
      setValue('variants', newVariants)

      // Update default variant if the removed one was selected
      if (defaultVariant === variants[index].name && newVariants.length > 0) {
        setValue('defaultVariant', newVariants[0].name)
      }
    },
    [variants, defaultVariant, setValue]
  )

  // Update variant name
  const handleNameChange = useCallback(
    (index: number, name: string) => {
      const newVariants = [...variants]
      const oldName = newVariants[index].name
      newVariants[index] = { ...newVariants[index], name }
      setValue('variants', newVariants)

      // Update default variant if it was renamed
      if (defaultVariant === oldName) {
        setValue('defaultVariant', name)
      }
    },
    [variants, defaultVariant, setValue]
  )

  // Update variant weight
  const handleWeightChange = useCallback(
    (index: number, weight: number) => {
      const newVariants = [...variants]
      newVariants[index] = { ...newVariants[index], weight }
      setValue('variants', newVariants)
    },
    [variants, setValue]
  )

  // Auto-balance weights
  const handleAutoBalance = useCallback(() => {
    if (variants.length === 0) return

    const equalWeight = Math.floor(100 / variants.length)
    const remainder = 100 - equalWeight * variants.length

    const newVariants = variants.map((v, i) => ({
      ...v,
      weight: equalWeight + (i === 0 ? remainder : 0),
    }))

    setValue('variants', newVariants)
  }, [variants, setValue])

  // Variant options for default selection
  const variantOptions = useMemo(() => {
    return variants.map((v) => ({
      label: v.name,
      value: v.name,
    }))
  }, [variants])

  return (
    <div className="variant-editor">
      {/* Header */}
      <div className="variant-editor-header">
        <Text className="variant-editor-title">{t('featureFlags.form.variants', 'Variants')}</Text>
        <Space>
          <Button size="small" onClick={handleAutoBalance} disabled={variants.length === 0}>
            {t('featureFlags.form.autoBalance', 'Auto Balance')}
          </Button>
          <Button
            icon={<IconPlus />}
            size="small"
            onClick={handleAddVariant}
            disabled={variants.length >= 8}
          >
            {t('featureFlags.form.addVariant', 'Add Variant')}
          </Button>
        </Space>
      </div>

      {/* Variant List */}
      <div className="variant-editor-list">
        {variants.map((variant, index) => (
          <div key={index} className="variant-item">
            {/* Color indicator */}
            <div
              style={{
                width: 8,
                height: 32,
                borderRadius: 4,
                backgroundColor: VARIANT_COLORS[index % VARIANT_COLORS.length],
              }}
            />

            {/* Name input */}
            <div className="variant-item-name">
              <Input
                size="small"
                value={variant.name}
                onChange={(value) => handleNameChange(index, value)}
                placeholder={t('featureFlags.form.variantName', 'Variant name')}
                maxLength={50}
              />
            </div>

            {/* Weight slider */}
            <div className="variant-item-weight">
              <Slider
                min={0}
                max={100}
                step={1}
                value={variant.weight}
                onChange={(val) => handleWeightChange(index, val as number)}
                tipFormatter={(val) => `${val}%`}
              />
              <InputNumber
                size="small"
                min={0}
                max={100}
                value={variant.weight}
                onChange={(val) => handleWeightChange(index, val as number)}
                formatter={(val) => `${val}%`}
                parser={(val) => (val ? Number(val.replace('%', '')) : 0)}
              />
            </div>

            {/* Delete button */}
            <div className="variant-item-actions">
              <Button
                icon={<IconDelete />}
                type="danger"
                theme="borderless"
                size="small"
                onClick={() => handleRemoveVariant(index)}
                disabled={variants.length <= 2}
              />
            </div>
          </div>
        ))}
      </div>

      {/* Weight Distribution Chart */}
      {variants.length > 0 && (
        <div className="variant-weight-chart">
          <Text className="variant-weight-chart-title">
            {t('featureFlags.form.weightDistribution', 'Weight Distribution')}
          </Text>
          <div className="variant-weight-bar">
            {variants.map((variant, index) => (
              <div
                key={index}
                className="variant-weight-bar-segment"
                style={{
                  width: `${variant.weight}%`,
                  backgroundColor: VARIANT_COLORS[index % VARIANT_COLORS.length],
                }}
                title={`${variant.name}: ${variant.weight}%`}
              >
                {variant.weight >= 10 && `${variant.name}`}
              </div>
            ))}
          </div>
          <div className={`variant-weight-total ${isWeightValid ? 'success' : 'error'}`}>
            {t('featureFlags.form.totalWeight', 'Total')}: {totalWeight}%
            {!isWeightValid && ` (${t('featureFlags.form.mustBe100', 'must be 100%')})`}
          </div>
        </div>
      )}

      {/* Default Variant Selection */}
      {variants.length > 0 && (
        <div className="variant-default-select">
          <Text strong style={{ display: 'block', marginBottom: 8 }}>
            {t('featureFlags.form.defaultVariant', 'Default Variant')}
          </Text>
          <Select
            value={defaultVariant}
            onChange={(val) => setValue('defaultVariant', val as string)}
            optionList={variantOptions}
            placeholder={t('featureFlags.form.selectDefaultVariant', 'Select default variant')}
            style={{ width: '100%', maxWidth: 200 }}
          />
          <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 4 }}>
            {t(
              'featureFlags.form.defaultVariantHelp',
              'The variant served when no rules match or for new users'
            )}
          </Text>
        </div>
      )}
    </div>
  )
}

export default VariantEditor
