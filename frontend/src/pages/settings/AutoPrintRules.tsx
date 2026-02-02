/**
 * Auto Print Rules Configuration Page
 *
 * Allows users to configure automatic printing rules for different document types.
 * Rules can be created, edited, enabled/disabled, and deleted.
 *
 * Features:
 * - Display all auto print rules in a table
 * - Create new rules with document type, trigger event, template, and printer selection
 * - Edit existing rules
 * - Delete rules with confirmation
 * - Enable/disable rules via switch toggle
 * - Print service status indicator
 * - Printer list fetched from local print service
 */

import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Toast,
  Table,
  Button,
  Modal,
  Form,
  Switch,
  Popconfirm,
  Banner,
  Space,
  Tag,
  Tooltip,
  Empty,
  Spin,
} from '@douyinfe/semi-ui-19'
import type { ColumnProps } from '@douyinfe/semi-ui-19/lib/es/table'
import {
  IconPlus,
  IconEdit2,
  IconDelete,
  IconRefresh,
  IconLink,
  IconUnlink,
  IconPrint,
} from '@douyinfe/semi-icons'
import { Container } from '@/components/common/layout'
import { usePrintServiceStatus, type PrinterInfo } from '@/hooks'
import {
  useListAutoPrintRules,
  useCreateAutoPrintRule,
  useUpdateAutoPrintRule,
  useDeleteAutoPrintRule,
  useEnableAutoPrintRule,
  useDisableAutoPrintRule,
  useGetTriggerEventsForDocType,
  getListAutoPrintRulesQueryKey,
} from '@/api/auto-print-rules/auto-print-rules'
import type {
  HandlerAutoPrintRuleHTTPResponse,
  HandlerCreateAutoPrintRuleHTTPRequest,
} from '@/api/models'
import { useGetPrintReferenceDocumentTypes } from '@/api/print-reference/print-reference'
import type { HandlerDocumentTypeResponse } from '@/api/models'
import { useGetPrintTemplateTemplatesByDocType } from '@/api/print-templates/print-templates'
import type { HandlerTemplateResponse } from '@/api/models'
import { useQueryClient } from '@tanstack/react-query'

import './AutoPrintRules.css'

const { Title, Text } = Typography

/** Form values for creating/editing a rule */
interface RuleFormValues {
  document_type: string
  trigger_event: string
  template_id: string
  copies: number
  printer_name: string
  auto_print: boolean
}

/** Default form values */
const defaultFormValues: RuleFormValues = {
  document_type: '',
  trigger_event: '',
  template_id: '',
  copies: 1,
  printer_name: '',
  auto_print: true,
}

/**
 * Print Service Status Indicator Component
 */
function PrintServiceStatusIndicator() {
  const { t } = useTranslation('system')
  const { status, version, error, refresh, printers, isLoadingPrinters } = usePrintServiceStatus()

  const statusConfig = useMemo(() => {
    switch (status) {
      case 'connected':
        return {
          color: 'green' as const,
          icon: <IconLink />,
          label: t('autoPrintRules.printService.connected'),
        }
      case 'disconnected':
        return {
          color: 'red' as const,
          icon: <IconUnlink />,
          label: t('autoPrintRules.printService.disconnected'),
        }
      case 'checking':
      default:
        return {
          color: 'grey' as const,
          icon: <IconRefresh spin />,
          label: t('autoPrintRules.printService.checking'),
        }
    }
  }, [status, t])

  return (
    <div className="print-service-status">
      <Tooltip
        content={
          <div>
            <div>{statusConfig.label}</div>
            {version && <div>Version: {version}</div>}
            {error && <div style={{ color: 'var(--semi-color-danger)' }}>{error}</div>}
            {status === 'connected' && (
              <div>
                {isLoadingPrinters
                  ? t('autoPrintRules.printService.loadingPrinters')
                  : `${printers.length} ${t('autoPrintRules.printService.printersFound')}`}
              </div>
            )}
          </div>
        }
      >
        <Tag color={statusConfig.color} prefixIcon={statusConfig.icon}>
          <IconPrint style={{ marginRight: 4 }} />
          {statusConfig.label}
        </Tag>
      </Tooltip>
      <Button
        icon={<IconRefresh />}
        theme="borderless"
        size="small"
        onClick={() => refresh()}
        loading={status === 'checking'}
      />
    </div>
  )
}

/**
 * Auto Print Rules Page Component
 */
export default function AutoPrintRulesPage() {
  const { t } = useTranslation('system')
  const queryClient = useQueryClient()

  // Print service status
  const { status: printServiceStatus, printers, isLoadingPrinters } = usePrintServiceStatus()

  // Modal state
  const [isModalVisible, setIsModalVisible] = useState(false)
  const [editingRule, setEditingRule] = useState<HandlerAutoPrintRuleHTTPResponse | null>(null)
  const [formValues, setFormValues] = useState<RuleFormValues>(defaultFormValues)

  // Track selected document type for trigger events query
  const [selectedDocType, setSelectedDocType] = useState<string>('')

  // Fetch rules list
  const {
    data: rulesResponse,
    isLoading: isLoadingRules,
    refetch: refetchRules,
  } = useListAutoPrintRules()

  const rules = useMemo(() => {
    if (rulesResponse?.status === 200) {
      return rulesResponse.data.data || []
    }
    return []
  }, [rulesResponse])

  // Fetch document types
  const { data: docTypesResponse, isLoading: isLoadingDocTypes } =
    useGetPrintReferenceDocumentTypes()

  const documentTypes = useMemo(() => {
    if (docTypesResponse?.status === 200) {
      return docTypesResponse.data.data || []
    }
    return []
  }, [docTypesResponse])

  // Fetch trigger events for selected document type
  const { data: triggerEventsResponse, isLoading: isLoadingTriggerEvents } =
    useGetTriggerEventsForDocType(
      { document_type: selectedDocType },
      { query: { enabled: !!selectedDocType } }
    )

  const triggerEvents = useMemo(() => {
    if (triggerEventsResponse?.status === 200) {
      return triggerEventsResponse.data.data || []
    }
    return []
  }, [triggerEventsResponse])

  // Fetch templates for selected document type
  const { data: templatesResponse, isLoading: isLoadingTemplates } =
    useGetPrintTemplateTemplatesByDocType(selectedDocType, {
      query: { enabled: !!selectedDocType },
    })

  const templates = useMemo(() => {
    if (templatesResponse?.status === 200) {
      return templatesResponse.data.data || []
    }
    return []
  }, [templatesResponse])

  // Mutations
  const createMutation = useCreateAutoPrintRule()
  const updateMutation = useUpdateAutoPrintRule()
  const deleteMutation = useDeleteAutoPrintRule()
  const enableMutation = useEnableAutoPrintRule()
  const disableMutation = useDisableAutoPrintRule()

  // Invalidate queries helper
  const invalidateRulesQuery = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: getListAutoPrintRulesQueryKey() })
  }, [queryClient])

  // Open create modal
  const handleCreate = useCallback(() => {
    setEditingRule(null)
    setFormValues(defaultFormValues)
    setSelectedDocType('')
    setIsModalVisible(true)
  }, [])

  // Open edit modal
  const handleEdit = useCallback((rule: HandlerAutoPrintRuleHTTPResponse) => {
    setEditingRule(rule)
    setFormValues({
      document_type: rule.document_type || '',
      trigger_event: rule.trigger_event || '',
      template_id: rule.template_id || '',
      copies: rule.copies || 1,
      printer_name: rule.printer_name || '',
      auto_print: rule.auto_print ?? true,
    })
    setSelectedDocType(rule.document_type || '')
    setIsModalVisible(true)
  }, [])

  // Close modal
  const handleCloseModal = useCallback(() => {
    setIsModalVisible(false)
    setEditingRule(null)
    setFormValues(defaultFormValues)
    setSelectedDocType('')
  }, [])

  // Handle form field change
  const handleFormChange = useCallback(
    (values: Partial<RuleFormValues>) => {
      // If document type changes, reset trigger event and template
      if (values.document_type !== undefined && values.document_type !== formValues.document_type) {
        setSelectedDocType(values.document_type)
        if (!editingRule) {
          setFormValues((prev) => ({
            ...prev,
            ...values,
            trigger_event: '',
            template_id: '',
          }))
          return
        }
      }
      setFormValues((prev) => ({ ...prev, ...values }))
    },
    [formValues.document_type, editingRule]
  )

  // Validate form
  const validateForm = useCallback((): string | null => {
    if (!formValues.document_type) {
      return t('autoPrintRules.validation.documentTypeRequired')
    }
    if (!formValues.trigger_event) {
      return t('autoPrintRules.validation.triggerEventRequired')
    }
    if (!formValues.template_id) {
      return t('autoPrintRules.validation.templateRequired')
    }
    if (formValues.copies < 1 || formValues.copies > 100) {
      return t('autoPrintRules.validation.copiesRange')
    }
    return null
  }, [formValues, t])

  // Submit form
  const handleSubmit = useCallback(async () => {
    const error = validateForm()
    if (error) {
      Toast.error(error)
      return
    }

    const data: HandlerCreateAutoPrintRuleHTTPRequest = {
      document_type: formValues.document_type,
      trigger_event: formValues.trigger_event,
      template_id: formValues.template_id,
      copies: formValues.copies,
      printer_name: formValues.printer_name || undefined,
      auto_print: formValues.auto_print,
    }

    try {
      if (editingRule?.id) {
        await updateMutation.mutateAsync({ id: editingRule.id, data })
        Toast.success(t('autoPrintRules.messages.updateSuccess'))
      } else {
        await createMutation.mutateAsync({ data })
        Toast.success(t('autoPrintRules.messages.createSuccess'))
      }
      handleCloseModal()
      invalidateRulesQuery()
    } catch {
      Toast.error(
        editingRule
          ? t('autoPrintRules.messages.updateError')
          : t('autoPrintRules.messages.createError')
      )
    }
  }, [
    formValues,
    editingRule,
    validateForm,
    createMutation,
    updateMutation,
    handleCloseModal,
    invalidateRulesQuery,
    t,
  ])

  // Delete rule
  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteMutation.mutateAsync({ id })
        Toast.success(t('autoPrintRules.messages.deleteSuccess'))
        invalidateRulesQuery()
      } catch {
        Toast.error(t('autoPrintRules.messages.deleteError'))
      }
    },
    [deleteMutation, invalidateRulesQuery, t]
  )

  // Toggle rule enabled state
  const handleToggleEnabled = useCallback(
    async (rule: HandlerAutoPrintRuleHTTPResponse) => {
      if (!rule.id) return

      try {
        if (rule.enabled) {
          await disableMutation.mutateAsync({ id: rule.id })
          Toast.success(t('autoPrintRules.messages.disableSuccess'))
        } else {
          await enableMutation.mutateAsync({ id: rule.id })
          Toast.success(t('autoPrintRules.messages.enableSuccess'))
        }
        invalidateRulesQuery()
      } catch {
        Toast.error(t('autoPrintRules.messages.toggleError'))
      }
    },
    [enableMutation, disableMutation, invalidateRulesQuery, t]
  )

  // Get display name for document type
  const getDocTypeDisplayName = useCallback(
    (code: string): string => {
      const docType = documentTypes.find((dt: HandlerDocumentTypeResponse) => dt.code === code)
      return docType?.display_name || code
    },
    [documentTypes]
  )

  // Get display name for trigger event (placeholder for future use)
  const getTriggerEventDisplayName = useCallback((_code: string, _docType: string): string => {
    // Trigger events are loaded dynamically, fallback to code
    return _code
  }, [])

  // Table columns
  const columns: ColumnProps<HandlerAutoPrintRuleHTTPResponse>[] = useMemo(
    () => [
      {
        title: t('autoPrintRules.columns.documentType'),
        dataIndex: 'document_type',
        key: 'document_type',
        render: (value: string) => getDocTypeDisplayName(value),
      },
      {
        title: t('autoPrintRules.columns.triggerEvent'),
        dataIndex: 'trigger_event',
        key: 'trigger_event',
        render: (value: string, record: HandlerAutoPrintRuleHTTPResponse) =>
          getTriggerEventDisplayName(value, record.document_type || ''),
      },
      {
        title: t('autoPrintRules.columns.template'),
        dataIndex: 'template_id',
        key: 'template_id',
        render: (value: string) => value || '-',
      },
      {
        title: t('autoPrintRules.columns.copies'),
        dataIndex: 'copies',
        key: 'copies',
        width: 80,
        align: 'center' as const,
      },
      {
        title: t('autoPrintRules.columns.printer'),
        dataIndex: 'printer_name',
        key: 'printer_name',
        render: (value: string) => value || t('autoPrintRules.defaultPrinter'),
      },
      {
        title: t('autoPrintRules.columns.enabled'),
        dataIndex: 'enabled',
        key: 'enabled',
        width: 100,
        align: 'center' as const,
        render: (value: boolean, record: HandlerAutoPrintRuleHTTPResponse) => (
          <Switch
            checked={value}
            onChange={() => handleToggleEnabled(record)}
            loading={enableMutation.isPending || disableMutation.isPending}
          />
        ),
      },
      {
        title: t('autoPrintRules.columns.actions'),
        key: 'actions',
        width: 120,
        fixed: 'right' as const,
        render: (_: unknown, record: HandlerAutoPrintRuleHTTPResponse) => (
          <Space>
            <Button
              icon={<IconEdit2 />}
              theme="borderless"
              size="small"
              onClick={() => handleEdit(record)}
            />
            <Popconfirm
              title={t('autoPrintRules.confirmDelete')}
              onConfirm={() => {
                if (record.id) handleDelete(record.id)
              }}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
            >
              <Button icon={<IconDelete />} theme="borderless" size="small" type="danger" />
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [
      t,
      getDocTypeDisplayName,
      getTriggerEventDisplayName,
      handleToggleEnabled,
      handleEdit,
      handleDelete,
      enableMutation.isPending,
      disableMutation.isPending,
    ]
  )

  // Document type options
  const docTypeOptions = useMemo(
    () =>
      documentTypes.map((dt: HandlerDocumentTypeResponse) => ({
        label: dt.display_name || dt.code,
        value: dt.code,
      })),
    [documentTypes]
  )

  // Trigger event options
  const triggerEventOptions = useMemo(
    () =>
      triggerEvents.map((te: { code?: string; display_name?: string }) => ({
        label: te.display_name || te.code,
        value: te.code,
      })),
    [triggerEvents]
  )

  // Template options
  const templateOptions = useMemo(
    () =>
      templates.map((t: HandlerTemplateResponse) => ({
        label: t.name || t.id,
        value: t.id,
      })),
    [templates]
  )

  // Printer options
  const printerOptions = useMemo(
    () => [
      { label: t('autoPrintRules.defaultPrinter'), value: '' },
      ...printers.map((p: PrinterInfo) => ({
        label: p.displayName || p.name,
        value: p.name,
      })),
    ],
    [printers, t]
  )

  // Check if form is loading data
  const isFormLoading = isLoadingDocTypes || isLoadingTriggerEvents || isLoadingTemplates

  return (
    <Container size="lg" className="auto-print-rules-page">
      <Card className="auto-print-rules-card">
        {/* Header */}
        <div className="auto-print-rules-header">
          <div className="auto-print-rules-title">
            <Title heading={4} style={{ margin: 0 }}>
              {t('autoPrintRules.title')}
            </Title>
            <Text type="tertiary">{t('autoPrintRules.subtitle')}</Text>
          </div>
          <div className="auto-print-rules-actions">
            <PrintServiceStatusIndicator />
            <Button icon={<IconRefresh />} onClick={() => refetchRules()} loading={isLoadingRules}>
              {t('common.refresh')}
            </Button>
            <Button
              icon={<IconPlus />}
              type="primary"
              theme="solid"
              onClick={handleCreate}
              disabled={printServiceStatus === 'disconnected'}
            >
              {t('autoPrintRules.createRule')}
            </Button>
          </div>
        </div>

        {/* Print Service Warning */}
        {printServiceStatus === 'disconnected' && (
          <Banner
            type="warning"
            description={t('autoPrintRules.printServiceWarning')}
            className="auto-print-rules-banner"
          />
        )}

        {/* Rules Table */}
        <Table
          columns={columns}
          dataSource={rules}
          rowKey="id"
          loading={isLoadingRules}
          pagination={false}
          empty={
            <Empty
              image={<IconPrint style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
              title={t('autoPrintRules.empty.title')}
              description={t('autoPrintRules.empty.description')}
            />
          }
        />
      </Card>

      {/* Create/Edit Modal */}
      <Modal
        title={editingRule ? t('autoPrintRules.editRule') : t('autoPrintRules.createRule')}
        visible={isModalVisible}
        onOk={handleSubmit}
        onCancel={handleCloseModal}
        okText={editingRule ? t('common.save') : t('common.create')}
        cancelText={t('common.cancel')}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
        width={560}
        maskClosable={false}
      >
        <Spin spinning={isFormLoading}>
          <Form labelPosition="left" labelWidth={120} className="auto-print-rules-form">
            {/* Document Type */}
            <Form.Select
              field="document_type"
              label={t('autoPrintRules.form.documentType')}
              placeholder={t('autoPrintRules.form.documentTypePlaceholder')}
              optionList={docTypeOptions}
              initValue={formValues.document_type}
              onChange={(value) => handleFormChange({ document_type: value as string })}
              rules={[
                { required: true, message: t('autoPrintRules.validation.documentTypeRequired') },
              ]}
              disabled={!!editingRule}
              showClear
            />

            {/* Trigger Event */}
            <Form.Select
              field="trigger_event"
              label={t('autoPrintRules.form.triggerEvent')}
              placeholder={t('autoPrintRules.form.triggerEventPlaceholder')}
              optionList={triggerEventOptions}
              initValue={formValues.trigger_event}
              onChange={(value) => handleFormChange({ trigger_event: value as string })}
              rules={[
                { required: true, message: t('autoPrintRules.validation.triggerEventRequired') },
              ]}
              disabled={!formValues.document_type || !!editingRule}
              loading={isLoadingTriggerEvents}
              showClear
            />

            {/* Template */}
            <Form.Select
              field="template_id"
              label={t('autoPrintRules.form.template')}
              placeholder={t('autoPrintRules.form.templatePlaceholder')}
              optionList={templateOptions}
              initValue={formValues.template_id}
              onChange={(value) => handleFormChange({ template_id: value as string })}
              rules={[{ required: true, message: t('autoPrintRules.validation.templateRequired') }]}
              disabled={!formValues.document_type}
              loading={isLoadingTemplates}
              showClear
            />

            {/* Copies */}
            <Form.InputNumber
              field="copies"
              label={t('autoPrintRules.form.copies')}
              initValue={formValues.copies}
              onChange={(value) => handleFormChange({ copies: value as number })}
              min={1}
              max={100}
              rules={[{ required: true, message: t('autoPrintRules.validation.copiesRequired') }]}
            />

            {/* Printer */}
            <Form.Select
              field="printer_name"
              label={t('autoPrintRules.form.printer')}
              placeholder={t('autoPrintRules.form.printerPlaceholder')}
              optionList={printerOptions}
              initValue={formValues.printer_name}
              onChange={(value) => handleFormChange({ printer_name: value as string })}
              loading={isLoadingPrinters}
              disabled={printServiceStatus !== 'connected'}
              showClear
            />

            {/* Auto Print Toggle */}
            <Form.Switch
              field="auto_print"
              label={t('autoPrintRules.form.autoPrint')}
              initValue={formValues.auto_print}
              onChange={(value) => handleFormChange({ auto_print: value as boolean })}
            />
          </Form>

          {printServiceStatus !== 'connected' && (
            <Banner
              type="info"
              description={t('autoPrintRules.printerUnavailable')}
              style={{ marginTop: 16 }}
            />
          )}
        </Spin>
      </Modal>
    </Container>
  )
}
