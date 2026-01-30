import { useMemo, useCallback } from 'react'
import { Table, Button, Typography, Empty, Tag } from '@douyinfe/semi-ui-19'
import { IconDownload } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import type { CsvimportRowError } from '@/api/models'
import type { ErrorTableProps } from './types'
import './ErrorTable.css'

const { Text } = Typography

/**
 * ErrorTable component displays validation and import errors
 * in a tabular format with export capability
 */
export function ErrorTable({
  errors,
  isTruncated = false,
  totalErrors = 0,
  showExport = true,
  onExport,
  maxRows = 100,
}: ErrorTableProps) {
  const { t } = useTranslation('common')

  // Limit displayed errors for performance
  const displayedErrors = useMemo(() => {
    return errors.slice(0, maxRows)
  }, [errors, maxRows])

  // Generate CSV content for export
  const handleExport = useCallback(() => {
    if (onExport) {
      onExport()
      return
    }

    // Default CSV export
    const headers = [
      t('import.errors.row'),
      t('import.errors.column'),
      t('import.errors.value'),
      t('import.errors.message'),
      t('import.errors.code'),
    ]
    const csvContent = [
      headers.join(','),
      ...errors.map((error) =>
        [
          error.row ?? '',
          `"${(error.column ?? '').replace(/"/g, '""')}"`,
          `"${(error.value ?? '').replace(/"/g, '""')}"`,
          `"${(error.message ?? '').replace(/"/g, '""')}"`,
          error.code ?? '',
        ].join(',')
      ),
    ].join('\n')

    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `import-errors-${new Date().toISOString().slice(0, 10)}.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }, [errors, onExport, t])

  // Get error severity color
  const getErrorColor = useCallback((code?: string) => {
    if (!code) return 'red'
    if (code.includes('REQUIRED') || code.includes('INVALID')) return 'red'
    if (code.includes('WARNING') || code.includes('DUPLICATE')) return 'orange'
    return 'red'
  }, [])

  // Table columns
  const columns = useMemo(
    () => [
      {
        title: t('import.errors.row'),
        dataIndex: 'row',
        width: 80,
        render: (row: number | undefined) => <Text strong>{row !== undefined ? row : '-'}</Text>,
      },
      {
        title: t('import.errors.column'),
        dataIndex: 'column',
        width: 120,
        render: (column: string | undefined) => <Tag color="blue">{column || '-'}</Tag>,
      },
      {
        title: t('import.errors.value'),
        dataIndex: 'value',
        width: 150,
        ellipsis: true,
        render: (value: string | undefined) => (
          <Text type="secondary" className="error-value">
            {value || '-'}
          </Text>
        ),
      },
      {
        title: t('import.errors.message'),
        dataIndex: 'message',
        render: (message: string | undefined, record: CsvimportRowError) => (
          <div className="error-message-cell">
            <Text type="danger">{message || '-'}</Text>
            {record.code && (
              <Tag size="small" color={getErrorColor(record.code)} className="error-code-tag">
                {record.code}
              </Tag>
            )}
          </div>
        ),
      },
    ],
    [t, getErrorColor]
  )

  if (errors.length === 0) {
    return <Empty description={t('import.errors.noErrors')} className="error-table-empty" />
  }

  return (
    <div className="error-table">
      <div className="error-table-header">
        <Text strong>
          {t('import.errors.title', { count: isTruncated ? totalErrors : errors.length })}
        </Text>
        {showExport && errors.length > 0 && (
          <Button size="small" icon={<IconDownload />} onClick={handleExport}>
            {t('import.errors.export')}
          </Button>
        )}
      </div>

      {isTruncated && (
        <div className="error-table-truncated">
          <Text type="warning">
            {t('import.errors.truncated', { shown: displayedErrors.length, total: totalErrors })}
          </Text>
        </div>
      )}

      <Table
        columns={columns}
        dataSource={displayedErrors}
        rowKey={(record?: CsvimportRowError, index?: number) =>
          `${record?.row ?? ''}-${record?.column ?? ''}-${index ?? 0}`
        }
        pagination={
          displayedErrors.length > 10
            ? {
                pageSize: 10,
                showTotal: true,
                showSizeChanger: false,
              }
            : false
        }
        size="small"
        className="error-table-content"
      />
    </div>
  )
}

export default ErrorTable
