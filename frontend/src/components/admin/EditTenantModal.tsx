import { useRef, useCallback, useEffect, useMemo } from 'react'
import { Modal, Toast, Spin } from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { TenantForm, type EditTenantFormData, type TenantFormHandle } from './TenantForm'
import {
  useGetTenantById,
  useUpdateTenant,
  getListTenantsQueryKey,
  getGetTenantByIdQueryKey,
} from '@/api/tenants/tenants'
import type { UpdateTenantBody } from '@/api/models'

interface EditTenantModalProps {
  visible: boolean
  tenantId: string | null
  onClose: () => void
  onSuccess?: () => void
}

/**
 * EditTenantModal - Modal dialog for editing an existing tenant
 *
 * Features:
 * - Fetches tenant data on open
 * - Form validation with Zod
 * - API integration with React Query
 * - Loading state during fetch and submission
 * - Success/error toast notifications
 * - Keyboard support (Enter to submit, Esc to cancel)
 * - Refreshes tenant list and detail on success
 */
export function EditTenantModal({ visible, tenantId, onClose, onSuccess }: EditTenantModalProps) {
  const { t } = useTranslation('admin')
  const queryClient = useQueryClient()
  const formRef = useRef<TenantFormHandle>(null)

  // Fetch tenant data
  const {
    data: tenantResponse,
    isLoading: isFetching,
    isError,
  } = useGetTenantById(tenantId || '', {
    query: {
      enabled: visible && !!tenantId,
    },
  })

  // Extract tenant data
  const tenant = useMemo(() => {
    if (tenantResponse?.status === 200 && tenantResponse.data.data) {
      return tenantResponse.data.data
    }
    return undefined
  }, [tenantResponse])

  // Update tenant mutation
  const updateMutation = useUpdateTenant({
    mutation: {
      onSuccess: () => {
        Toast.success(t('tenants.messages.updateSuccess', 'Tenant updated successfully'))
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        if (tenantId) {
          queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(tenantId) })
        }
        onClose()
        onSuccess?.()
      },
      onError: (error: Error | null) => {
        const errorMessage =
          error?.message || t('tenants.messages.updateError', 'Failed to update tenant')
        Toast.error(errorMessage)
      },
    },
  })

  // Handle form submission
  const handleSubmit = useCallback(
    (data: EditTenantFormData) => {
      if (!tenantId) return

      // Transform form data to API format
      const requestBody: UpdateTenantBody = {
        name: data.name,
        short_name: data.short_name || undefined,
        contact_email: data.contact_email || undefined,
        contact_name: data.contact_name || undefined,
        contact_phone: data.contact_phone || undefined,
        address: data.address || undefined,
        domain: data.domain || undefined,
        notes: data.notes || undefined,
      }

      updateMutation.mutate({ id: tenantId, data: requestBody })
    },
    [tenantId, updateMutation]
  )

  // Handle modal OK button
  const handleOk = useCallback(() => {
    formRef.current?.submit()
  }, [])

  // Handle modal cancel/close
  const handleCancel = useCallback(() => {
    formRef.current?.reset()
    onClose()
  }, [onClose])

  // Handle keyboard events
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!visible) return

      if (e.key === 'Escape') {
        e.preventDefault()
        handleCancel()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [visible, handleCancel])

  // Show error state
  if (isError && visible) {
    return (
      <Modal
        title={t('tenants.modal.editTitle', 'Edit Tenant')}
        visible={visible}
        onCancel={handleCancel}
        footer={null}
        width={600}
      >
        <div style={{ textAlign: 'center', padding: 24 }}>
          {t('tenants.messages.loadError', 'Failed to load tenant data')}
        </div>
      </Modal>
    )
  }

  return (
    <Modal
      title={t('tenants.modal.editTitle', 'Edit Tenant')}
      visible={visible}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('common.save', 'Save')}
      cancelText={t('common.cancel', 'Cancel')}
      confirmLoading={updateMutation.isPending}
      okButtonProps={{ disabled: isFetching || !tenant }}
      width={600}
      closeOnEsc={false}
      maskClosable={false}
    >
      {isFetching ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : tenant ? (
        <TenantForm
          mode="edit"
          initialData={tenant}
          onSubmit={handleSubmit}
          isLoading={false}
          ref={formRef}
        />
      ) : null}
    </Modal>
  )
}

export default EditTenantModal
