import { useMemo } from 'react'
import { Typography, Card, Radio, RadioGroup, Button, Banner } from '@douyinfe/semi-ui-19'
import { IconPlay, IconAlertCircle } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import type { ImportStepProps, ConflictMode } from './types'
import './ImportStep.css'

const { Title, Text } = Typography

/**
 * ImportStep component for configuring and starting import
 */
export function ImportStep({
  conflictMode,
  onConflictModeChange,
  onImport,
  validRowCount,
  loading,
}: ImportStepProps) {
  const { t } = useTranslation('common')

  // Conflict mode options
  const conflictModeOptions = useMemo(
    () => [
      {
        value: 'skip' as ConflictMode,
        label: t('import.conflictMode.skip'),
        description: t('import.conflictMode.skipDescription'),
      },
      {
        value: 'update' as ConflictMode,
        label: t('import.conflictMode.update'),
        description: t('import.conflictMode.updateDescription'),
      },
      {
        value: 'fail' as ConflictMode,
        label: t('import.conflictMode.fail'),
        description: t('import.conflictMode.failDescription'),
      },
    ],
    [t]
  )

  return (
    <div className="import-step">
      <div className="import-step-header">
        <Title heading={5}>{t('import.import.title')}</Title>
        <Text type="secondary">{t('import.import.description', { count: validRowCount })}</Text>
      </div>

      {/* Conflict mode selection */}
      <Card className="import-conflict-card">
        <div className="import-conflict-header">
          <Title heading={6}>{t('import.conflictMode.title')}</Title>
          <Text type="tertiary">{t('import.conflictMode.description')}</Text>
        </div>

        <RadioGroup
          value={conflictMode}
          onChange={(e) => onConflictModeChange(e.target.value as ConflictMode)}
          direction="vertical"
          className="import-conflict-options"
          aria-label={t('import.conflictMode.title')}
        >
          {conflictModeOptions.map((option) => (
            <Radio key={option.value} value={option.value} className="import-conflict-option">
              <div className="import-conflict-option-content">
                <Text strong>{option.label}</Text>
                <Text type="tertiary" size="small">
                  {option.description}
                </Text>
              </div>
            </Radio>
          ))}
        </RadioGroup>
      </Card>

      {/* Warning banner */}
      <Banner
        type="warning"
        icon={<IconAlertCircle />}
        description={t('import.import.warning')}
        className="import-warning-banner"
      />

      {/* Summary and action */}
      <Card className="import-summary-card">
        <div className="import-summary">
          <div className="import-summary-info">
            <Text type="secondary">{t('import.import.readyToImport')}</Text>
            <Title heading={2}>{validRowCount}</Title>
            <Text type="tertiary">{t('import.import.rows')}</Text>
          </div>
          <Button
            theme="solid"
            size="large"
            icon={<IconPlay />}
            onClick={onImport}
            loading={loading}
            className="import-start-button"
          >
            {loading ? t('import.import.importing') : t('import.import.startImport')}
          </Button>
        </div>
      </Card>
    </div>
  )
}

export default ImportStep
