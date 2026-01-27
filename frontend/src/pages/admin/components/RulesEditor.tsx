import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Button,
  Input,
  InputNumber,
  Select,
  Switch,
  Typography,
  Space,
  Tag,
  Tooltip,
} from '@douyinfe/semi-ui-19'
import { IconPlus, IconDelete, IconArrowUp, IconArrowDown, IconHandle } from '@douyinfe/semi-icons'
import type { TargetingRule, Condition, FlagValue, FlagType } from '@/api/feature-flags'

const { Text } = Typography

// Generate unique rule ID
function generateRuleId(): string {
  return `rule_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

// Available condition operators
const OPERATORS = [
  { value: 'eq', label: '=' },
  { value: 'neq', label: '≠' },
  { value: 'contains', label: 'contains' },
  { value: 'not_contains', label: 'not contains' },
  { value: 'starts_with', label: 'starts with' },
  { value: 'ends_with', label: 'ends with' },
  { value: 'gt', label: '>' },
  { value: 'gte', label: '≥' },
  { value: 'lt', label: '<' },
  { value: 'lte', label: '≤' },
  { value: 'in', label: 'in' },
  { value: 'not_in', label: 'not in' },
  { value: 'regex', label: 'regex' },
]

// Common user attributes
const COMMON_ATTRIBUTES = [
  { value: 'user_id', label: 'User ID' },
  { value: 'email', label: 'Email' },
  { value: 'country', label: 'Country' },
  { value: 'region', label: 'Region' },
  { value: 'device_type', label: 'Device Type' },
  { value: 'os', label: 'OS' },
  { value: 'browser', label: 'Browser' },
  { value: 'app_version', label: 'App Version' },
  { value: 'plan', label: 'Plan' },
  { value: 'role', label: 'Role' },
  { value: 'tenant_id', label: 'Tenant ID' },
]

interface RulesEditorProps {
  rules: TargetingRule[]
  onChange: (rules: TargetingRule[]) => void
  flagType: FlagType
}

/**
 * RulesEditor component for managing feature flag targeting rules
 *
 * Features:
 * - Add/remove targeting rules
 * - Configure conditions (attribute/operator/value)
 * - Set rule values based on flag type
 * - Reorder rules by priority
 */
export function RulesEditor({ rules, onChange, flagType }: RulesEditorProps) {
  const { t } = useTranslation('admin')

  // Add new rule
  const handleAddRule = useCallback(() => {
    const newRule: TargetingRule = {
      rule_id: generateRuleId(),
      priority: rules.length + 1,
      conditions: [
        {
          attribute: 'user_id',
          operator: 'eq',
          values: [''],
        },
      ],
      value: {
        enabled: true,
        variant: undefined,
      },
      percentage: 100,
    }

    onChange([...rules, newRule])
  }, [rules, onChange])

  // Remove rule
  const handleRemoveRule = useCallback(
    (index: number) => {
      const newRules = rules.filter((_, i) => i !== index)
      // Update priorities
      const reorderedRules = newRules.map((rule, i) => ({
        ...rule,
        priority: i + 1,
      }))
      onChange(reorderedRules)
    },
    [rules, onChange]
  )

  // Move rule up
  const handleMoveUp = useCallback(
    (index: number) => {
      if (index === 0) return
      const newRules = [...rules]
      ;[newRules[index - 1], newRules[index]] = [newRules[index], newRules[index - 1]]
      // Update priorities
      const reorderedRules = newRules.map((rule, i) => ({
        ...rule,
        priority: i + 1,
      }))
      onChange(reorderedRules)
    },
    [rules, onChange]
  )

  // Move rule down
  const handleMoveDown = useCallback(
    (index: number) => {
      if (index === rules.length - 1) return
      const newRules = [...rules]
      ;[newRules[index], newRules[index + 1]] = [newRules[index + 1], newRules[index]]
      // Update priorities
      const reorderedRules = newRules.map((rule, i) => ({
        ...rule,
        priority: i + 1,
      }))
      onChange(reorderedRules)
    },
    [rules, onChange]
  )

  // Update rule
  const handleUpdateRule = useCallback(
    (index: number, updates: Partial<TargetingRule>) => {
      const newRules = [...rules]
      newRules[index] = { ...newRules[index], ...updates }
      onChange(newRules)
    },
    [rules, onChange]
  )

  // Add condition to rule
  const handleAddCondition = useCallback(
    (ruleIndex: number) => {
      const rule = rules[ruleIndex]
      const newCondition: Condition = {
        attribute: 'user_id',
        operator: 'eq',
        values: [''],
      }
      handleUpdateRule(ruleIndex, {
        conditions: [...rule.conditions, newCondition],
      })
    },
    [rules, handleUpdateRule]
  )

  // Remove condition from rule
  const handleRemoveCondition = useCallback(
    (ruleIndex: number, conditionIndex: number) => {
      const rule = rules[ruleIndex]
      if (rule.conditions.length <= 1) return // Keep at least one condition
      const newConditions = rule.conditions.filter((_, i) => i !== conditionIndex)
      handleUpdateRule(ruleIndex, { conditions: newConditions })
    },
    [rules, handleUpdateRule]
  )

  // Update condition
  const handleUpdateCondition = useCallback(
    (ruleIndex: number, conditionIndex: number, updates: Partial<Condition>) => {
      const rule = rules[ruleIndex]
      const newConditions = [...rule.conditions]
      newConditions[conditionIndex] = { ...newConditions[conditionIndex], ...updates }
      handleUpdateRule(ruleIndex, { conditions: newConditions })
    },
    [rules, handleUpdateRule]
  )

  // Empty state
  if (rules.length === 0) {
    return (
      <div className="rules-editor">
        <div className="rules-editor-empty">
          <Text className="rules-editor-empty-text">
            {t(
              'featureFlags.form.noRules',
              'No targeting rules. All users will receive the default value.'
            )}
          </Text>
          <Button icon={<IconPlus />} onClick={handleAddRule}>
            {t('featureFlags.form.addRule', 'Add Rule')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="rules-editor">
      {/* Header */}
      <div className="rules-editor-header">
        <Text type="secondary">
          {t(
            'featureFlags.form.rulesDescription',
            'Rules are evaluated in order. First matching rule wins.'
          )}
        </Text>
        <Button icon={<IconPlus />} size="small" onClick={handleAddRule}>
          {t('featureFlags.form.addRule', 'Add Rule')}
        </Button>
      </div>

      {/* Rules List */}
      <div className="rules-editor-list">
        {rules.map((rule, ruleIndex) => (
          <RuleItem
            key={rule.rule_id}
            rule={rule}
            index={ruleIndex}
            flagType={flagType}
            isFirst={ruleIndex === 0}
            isLast={ruleIndex === rules.length - 1}
            onUpdate={(updates) => handleUpdateRule(ruleIndex, updates)}
            onRemove={() => handleRemoveRule(ruleIndex)}
            onMoveUp={() => handleMoveUp(ruleIndex)}
            onMoveDown={() => handleMoveDown(ruleIndex)}
            onAddCondition={() => handleAddCondition(ruleIndex)}
            onRemoveCondition={(conditionIndex) => handleRemoveCondition(ruleIndex, conditionIndex)}
            onUpdateCondition={(conditionIndex, updates) =>
              handleUpdateCondition(ruleIndex, conditionIndex, updates)
            }
          />
        ))}
      </div>
    </div>
  )
}

// Single rule item component
interface RuleItemProps {
  rule: TargetingRule
  index: number
  flagType: FlagType
  isFirst: boolean
  isLast: boolean
  onUpdate: (updates: Partial<TargetingRule>) => void
  onRemove: () => void
  onMoveUp: () => void
  onMoveDown: () => void
  onAddCondition: () => void
  onRemoveCondition: (conditionIndex: number) => void
  onUpdateCondition: (conditionIndex: number, updates: Partial<Condition>) => void
}

function RuleItem({
  rule,
  index,
  flagType,
  isFirst,
  isLast,
  onUpdate,
  onRemove,
  onMoveUp,
  onMoveDown,
  onAddCondition,
  onRemoveCondition,
  onUpdateCondition,
}: RuleItemProps) {
  const { t } = useTranslation('admin')

  // Operator options with localized labels
  const operatorOptions = useMemo(
    () =>
      OPERATORS.map((op) => ({
        value: op.value,
        label: op.label,
      })),
    []
  )

  // Attribute options with localized labels
  const attributeOptions = useMemo(
    () =>
      COMMON_ATTRIBUTES.map((attr) => ({
        value: attr.value,
        label: t(`featureFlags.form.attributes.${attr.value}`, attr.label),
      })),
    [t]
  )

  return (
    <div className="rule-item">
      {/* Header with priority and actions */}
      <div className="rule-item-header">
        <div className="rule-item-priority">
          <IconHandle style={{ color: 'var(--semi-color-text-2)', cursor: 'grab' }} />
          <div className="rule-item-priority-badge">{index + 1}</div>
          <Tag size="small" color="blue">
            {t('featureFlags.form.rulePriority', 'Rule {{n}}', { n: index + 1 })}
          </Tag>
        </div>
        <div className="rule-item-actions">
          <Tooltip content={t('featureFlags.form.moveUp', 'Move up')}>
            <Button
              icon={<IconArrowUp />}
              theme="borderless"
              size="small"
              disabled={isFirst}
              onClick={onMoveUp}
            />
          </Tooltip>
          <Tooltip content={t('featureFlags.form.moveDown', 'Move down')}>
            <Button
              icon={<IconArrowDown />}
              theme="borderless"
              size="small"
              disabled={isLast}
              onClick={onMoveDown}
            />
          </Tooltip>
          <Tooltip content={t('featureFlags.form.deleteRule', 'Delete rule')}>
            <Button
              icon={<IconDelete />}
              type="danger"
              theme="borderless"
              size="small"
              onClick={onRemove}
            />
          </Tooltip>
        </div>
      </div>

      {/* Conditions */}
      <div className="rule-conditions">
        <div className="rule-conditions-title">
          {t('featureFlags.form.conditions', 'Conditions')}
          <Text type="tertiary" size="small" style={{ marginLeft: 8 }}>
            ({t('featureFlags.form.allMustMatch', 'all must match')})
          </Text>
        </div>
        <div className="rule-condition-list">
          {rule.conditions.map((condition, conditionIndex) => (
            <div key={conditionIndex} className="rule-condition-item">
              {conditionIndex > 0 && (
                <Tag size="small" style={{ marginRight: 4 }}>
                  AND
                </Tag>
              )}
              <Select
                size="small"
                value={condition.attribute}
                onChange={(val) => onUpdateCondition(conditionIndex, { attribute: val as string })}
                optionList={attributeOptions}
                filter
                style={{ width: 140 }}
              />
              <Select
                size="small"
                value={condition.operator}
                onChange={(val) => onUpdateCondition(conditionIndex, { operator: val as string })}
                optionList={operatorOptions}
                style={{ width: 120 }}
              />
              <Input
                size="small"
                value={condition.values.join(', ')}
                onChange={(val) =>
                  onUpdateCondition(conditionIndex, {
                    values: val.split(',').map((v) => v.trim()),
                  })
                }
                placeholder={t('featureFlags.form.conditionValue', 'Value(s), comma separated')}
                style={{ minWidth: 180, flex: 1 }}
              />
              <Button
                icon={<IconDelete />}
                type="tertiary"
                theme="borderless"
                size="small"
                disabled={rule.conditions.length <= 1}
                onClick={() => onRemoveCondition(conditionIndex)}
              />
            </div>
          ))}
        </div>
        <Button
          icon={<IconPlus />}
          size="small"
          theme="borderless"
          onClick={onAddCondition}
          style={{ marginTop: 8 }}
        >
          {t('featureFlags.form.addCondition', 'Add Condition')}
        </Button>
      </div>

      {/* Rule Value */}
      <div className="rule-value-section">
        <div className="rule-value-title">
          {t('featureFlags.form.ruleValue', 'Value when matched')}
        </div>
        <RuleValueEditor
          flagType={flagType}
          value={rule.value}
          percentage={rule.percentage}
          onChange={(value, percentage) => onUpdate({ value, percentage })}
        />
      </div>
    </div>
  )
}

// Rule value editor based on flag type
interface RuleValueEditorProps {
  flagType: FlagType
  value: FlagValue
  percentage: number
  onChange: (value: FlagValue, percentage: number) => void
}

function RuleValueEditor({ flagType, value, percentage, onChange }: RuleValueEditorProps) {
  const { t } = useTranslation('admin')

  if (flagType === 'boolean' || flagType === 'user_segment') {
    return (
      <Space>
        <Switch
          checked={value.enabled}
          onChange={(checked) => onChange({ ...value, enabled: checked }, percentage)}
          checkedText={t('featureFlags.status.enabled', 'Enabled')}
          uncheckedText={t('featureFlags.status.disabled', 'Disabled')}
        />
      </Space>
    )
  }

  if (flagType === 'percentage') {
    return (
      <Space align="center">
        <Text>{t('featureFlags.form.percentageValue', 'Rollout')}</Text>
        <InputNumber
          size="small"
          min={0}
          max={100}
          value={percentage}
          onChange={(val) => onChange({ ...value, enabled: true }, val as number)}
          formatter={(val) => `${val}%`}
          parser={(val) => (val ? Number(val.replace('%', '')) : 0)}
          style={{ width: 100 }}
        />
        <Text type="tertiary">{t('featureFlags.form.ofMatchingUsers', 'of matching users')}</Text>
      </Space>
    )
  }

  if (flagType === 'variant') {
    return (
      <Space align="center">
        <Text>{t('featureFlags.form.serveVariant', 'Serve variant')}</Text>
        <Input
          size="small"
          value={value.variant || ''}
          onChange={(val) => onChange({ ...value, variant: val, enabled: true }, percentage)}
          placeholder={t('featureFlags.form.variantName', 'Variant name')}
          style={{ width: 120 }}
        />
      </Space>
    )
  }

  return null
}

export default RulesEditor
