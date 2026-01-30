import { useMemo } from 'react'
import { Typography, Card, Button, Progress, Empty } from '@douyinfe/semi-ui-19'
import {
  IconPlusCircle,
  IconClose,
  IconTickCircle,
  IconMinusCircle,
  IconRefresh2,
  IconCrossCircleStroked,
} from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import type { ResultsStepProps } from './types'
import { ErrorTable } from './ErrorTable'
import './ResultsStep.css'

const { Title, Text } = Typography

/**
 * ResultsStep component displays import results
 */
export function ResultsStep({ result, onImportMore, onClose }: ResultsStepProps) {
  const { t } = useTranslation('common')

  // Calculate import status
  const importStatus = useMemo(() => {
    if (!result) return 'pending'
    if (result.errorRows === 0) return 'success'
    if (result.importedRows === 0 && result.updatedRows === 0) return 'error'
    return 'partial'
  }, [result])

  // Get status icon
  const StatusIcon = useMemo(() => {
    switch (importStatus) {
      case 'success':
        return <IconTickCircle size="extra-large" className="results-status-icon--success" />
      case 'partial':
        return <IconTickCircle size="extra-large" className="results-status-icon--warning" />
      case 'error':
        return <IconCrossCircleStroked size="extra-large" className="results-status-icon--error" />
      default:
        return null
    }
  }, [importStatus])

  // No result yet
  if (!result) {
    return (
      <div className="results-step results-step--empty">
        <Empty description={t('import.results.noResult')} />
      </div>
    )
  }

  // Calculate success rate
  const totalProcessed =
    result.importedRows + result.updatedRows + result.skippedRows + result.errorRows
  const successCount = result.importedRows + result.updatedRows
  const successRate = totalProcessed > 0 ? Math.round((successCount / totalProcessed) * 100) : 0

  return (
    <div className="results-step">
      {/* Status header */}
      <div
        className={`results-status-header results-status-header--${importStatus}`}
        role="status"
        aria-live="polite"
      >
        {StatusIcon}
        <div className="results-status-text">
          <Title heading={4}>
            {importStatus === 'success' && t('import.results.success')}
            {importStatus === 'partial' && t('import.results.partial')}
            {importStatus === 'error' && t('import.results.failed')}
          </Title>
          <Text type="secondary">
            {t('import.results.summary', {
              imported: result.importedRows,
              updated: result.updatedRows,
              skipped: result.skippedRows,
              errors: result.errorRows,
            })}
          </Text>
        </div>
      </div>

      {/* Statistics cards */}
      <div className="results-stats-grid">
        <Card className="results-stat-card results-stat-card--imported">
          <div className="results-stat-content">
            <IconPlusCircle className="results-stat-icon" />
            <div className="results-stat-value">
              <Title heading={2}>{result.importedRows}</Title>
              <Text type="secondary">{t('import.results.imported')}</Text>
            </div>
          </div>
        </Card>

        <Card className="results-stat-card results-stat-card--updated">
          <div className="results-stat-content">
            <IconRefresh2 className="results-stat-icon" />
            <div className="results-stat-value">
              <Title heading={2}>{result.updatedRows}</Title>
              <Text type="secondary">{t('import.results.updated')}</Text>
            </div>
          </div>
        </Card>

        <Card className="results-stat-card results-stat-card--skipped">
          <div className="results-stat-content">
            <IconMinusCircle className="results-stat-icon" />
            <div className="results-stat-value">
              <Title heading={2}>{result.skippedRows}</Title>
              <Text type="secondary">{t('import.results.skipped')}</Text>
            </div>
          </div>
        </Card>

        <Card className="results-stat-card results-stat-card--errors">
          <div className="results-stat-content">
            <IconCrossCircleStroked className="results-stat-icon" />
            <div className="results-stat-value">
              <Title heading={2}>{result.errorRows}</Title>
              <Text type="secondary">{t('import.results.errors')}</Text>
            </div>
          </div>
        </Card>
      </div>

      {/* Success rate */}
      <Card className="results-rate-card">
        <div className="results-rate-content">
          <div className="results-rate-info">
            <Text>{t('import.results.successRate')}</Text>
            <Text type="secondary">
              {t('import.results.processedTotal', { count: result.totalRows })}
            </Text>
          </div>
          <Progress
            percent={successRate}
            type="circle"
            size="default"
            showInfo
            format={(percent) => `${percent}%`}
            stroke={successRate === 100 ? 'var(--color-success)' : undefined}
          />
        </div>
      </Card>

      {/* Errors (if any) */}
      {result.errors && result.errors.length > 0 && (
        <Card className="results-errors-card">
          <ErrorTable
            errors={result.errors}
            isTruncated={result.isTruncated}
            totalErrors={result.totalErrors}
          />
        </Card>
      )}

      {/* Actions */}
      <div className="results-actions">
        <Button icon={<IconPlusCircle />} onClick={onImportMore}>
          {t('import.results.importMore')}
        </Button>
        <Button theme="solid" icon={<IconClose />} onClick={onClose}>
          {t('import.results.close')}
        </Button>
      </div>
    </div>
  )
}

export default ResultsStep
