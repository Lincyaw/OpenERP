import { useMemo } from 'react'
import {
  Typography,
  Card,
  Spin,
  Progress,
  Tag,
  Space,
  Button,
  Empty,
  Table,
  Collapse,
} from '@douyinfe/semi-ui-19'
import {
  IconAlertTriangle,
  IconRefresh,
  IconTickCircle,
  IconCrossCircleStroked,
} from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import type { ValidationStepProps } from './types'
import { ErrorTable } from './ErrorTable'
import './ValidationStep.css'

const { Title, Text } = Typography

/**
 * ValidationStep component displays validation results
 * and allows user to proceed or retry
 */
export function ValidationStep({
  result,
  loading,
  error,
  onRetry,
  onProceed,
  hasValidRows,
}: ValidationStepProps) {
  const { t } = useTranslation('common')

  // Calculate validation status
  const validationStatus = useMemo(() => {
    if (!result) return 'pending'
    if (result.errorRows === 0) return 'success'
    if (result.validRows === 0) return 'error'
    return 'warning'
  }, [result])

  // Get status icon
  const StatusIcon = useMemo(() => {
    switch (validationStatus) {
      case 'success':
        return <IconTickCircle size="extra-large" className="validation-status-icon--success" />
      case 'warning':
        return <IconAlertTriangle size="extra-large" className="validation-status-icon--warning" />
      case 'error':
        return (
          <IconCrossCircleStroked size="extra-large" className="validation-status-icon--error" />
        )
      default:
        return null
    }
  }, [validationStatus])

  // Preview table columns
  const previewColumns = useMemo(() => {
    if (!result?.preview || result.preview.length === 0) return []

    const firstRow = result.preview[0]
    return Object.keys(firstRow).map((key) => ({
      title: key,
      dataIndex: key,
      width: 120,
      ellipsis: true,
      render: (value: unknown) => {
        if (value === null || value === undefined) return '-'
        return String(value)
      },
    }))
  }, [result])

  // Loading state
  if (loading) {
    return (
      <div className="validation-step validation-step--loading">
        <Spin size="large" tip={t('import.validation.validating')} />
        <Text type="secondary">{t('import.validation.validatingHint')}</Text>
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="validation-step validation-step--error">
        <IconCrossCircleStroked size="extra-large" className="validation-status-icon--error" />
        <Title heading={5}>{t('import.validation.failed')}</Title>
        <Text type="danger">{error}</Text>
        <Button icon={<IconRefresh />} onClick={onRetry}>
          {t('import.validation.retry')}
        </Button>
      </div>
    )
  }

  // No result yet
  if (!result) {
    return (
      <div className="validation-step validation-step--empty">
        <Empty description={t('import.validation.noResult')} />
      </div>
    )
  }

  // Calculate progress percentage
  const validPercentage =
    result.totalRows > 0 ? Math.round((result.validRows / result.totalRows) * 100) : 0

  return (
    <div className="validation-step">
      {/* Status header */}
      <div className="validation-status-header">
        {StatusIcon}
        <div className="validation-status-text">
          <Title heading={5}>
            {validationStatus === 'success' && t('import.validation.allValid')}
            {validationStatus === 'warning' && t('import.validation.someErrors')}
            {validationStatus === 'error' && t('import.validation.allInvalid')}
          </Title>
          <Text type="secondary">
            {t('import.validation.summary', {
              valid: result.validRows,
              total: result.totalRows,
              errors: result.errorRows,
            })}
          </Text>
        </div>
      </div>

      {/* Statistics */}
      <Card className="validation-stats-card">
        <div className="validation-stats">
          <div className="validation-stat">
            <Text type="secondary">{t('import.validation.totalRows')}</Text>
            <Title heading={3}>{result.totalRows}</Title>
          </div>
          <div className="validation-stat validation-stat--success">
            <Text type="secondary">{t('import.validation.validRows')}</Text>
            <Title heading={3} className="validation-stat-value--success">
              <IconTickCircle /> {result.validRows}
            </Title>
          </div>
          <div className="validation-stat validation-stat--error">
            <Text type="secondary">{t('import.validation.errorRows')}</Text>
            <Title heading={3} className="validation-stat-value--error">
              <IconCrossCircleStroked /> {result.errorRows}
            </Title>
          </div>
          <div className="validation-stat">
            <Text type="secondary">{t('import.validation.validRate')}</Text>
            <Progress
              percent={validPercentage}
              type="circle"
              size="small"
              showInfo
              format={(percent) => `${percent}%`}
              stroke={validPercentage === 100 ? 'var(--color-success)' : undefined}
            />
          </div>
        </div>
      </Card>

      {/* Warnings */}
      {result.warnings && result.warnings.length > 0 && (
        <Collapse>
          <Collapse.Panel
            header={
              <Space>
                <IconAlertTriangle style={{ color: 'var(--color-warning)' }} />
                <Text>{t('import.validation.warnings', { count: result.warnings.length })}</Text>
              </Space>
            }
            itemKey="warnings"
          >
            <div className="validation-warnings">
              {result.warnings.slice(0, 10).map((warning, index) => (
                <Tag key={index} color="orange" className="validation-warning-tag">
                  {warning}
                </Tag>
              ))}
              {result.warnings.length > 10 && (
                <Text type="secondary">
                  {t('import.validation.moreWarnings', { count: result.warnings.length - 10 })}
                </Text>
              )}
            </div>
          </Collapse.Panel>
        </Collapse>
      )}

      {/* Preview */}
      {result.preview && result.preview.length > 0 && (
        <Collapse defaultActiveKey={['preview']}>
          <Collapse.Panel
            header={t('import.validation.preview', { count: result.preview.length })}
            itemKey="preview"
          >
            <Table
              columns={previewColumns}
              dataSource={result.preview}
              rowKey={(_record?: Record<string, unknown>, index?: number) =>
                `preview-${index ?? 0}`
              }
              pagination={false}
              size="small"
              scroll={{ x: 'max-content' }}
              className="validation-preview-table"
            />
          </Collapse.Panel>
        </Collapse>
      )}

      {/* Errors */}
      {result.errors && result.errors.length > 0 && (
        <Collapse defaultActiveKey={validationStatus === 'error' ? ['errors'] : []}>
          <Collapse.Panel
            header={
              <Space>
                <IconCrossCircleStroked style={{ color: 'var(--color-danger)' }} />
                <Text type="danger">
                  {t('import.validation.errors', { count: result.errorRows })}
                </Text>
              </Space>
            }
            itemKey="errors"
          >
            <ErrorTable
              errors={result.errors}
              isTruncated={result.isTruncated}
              totalErrors={result.totalErrors}
            />
          </Collapse.Panel>
        </Collapse>
      )}

      {/* Actions */}
      <div className="validation-actions">
        <Button icon={<IconRefresh />} onClick={onRetry}>
          {t('import.validation.reupload')}
        </Button>
        {hasValidRows && (
          <Button theme="solid" onClick={onProceed}>
            {t('import.validation.proceed', { count: result.validRows })}
          </Button>
        )}
      </div>
    </div>
  )
}

export default ValidationStep
