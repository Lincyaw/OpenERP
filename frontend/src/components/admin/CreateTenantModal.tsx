import { useRef, useCallback, useEffect } from 'react'
import { Modal, Toast } from '@douyinfe/semi-ui-19'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { TenantForm, type CreateTenantFormData, type TenantFormHandle } from './TenantForm'
import { useCreateTenant, getListTenantsQueryKey } from '@/api/tenants/tenants'
import type { CreateTenantBody } from '@/api/models'

interface CreateTenantModalProps {
  visible: boolean
  onClose: () => void
  onSuccess?: () => void
}

/**
 * CreateTenantModal - Modal dialog for creating a new tenant
 *
 * Features:
 * - Form validation with Zod
 * - API integration with React Query
 * - Loading state during submission
 * - Success/error toast notifications
 * - Keyboard support (Enter to submit, Esc to cancel)
 * - Refreshes tenant list on success
 */
export function CreateTenantModal({ visible, onClose, onSuccess }: CreateTenantModalProps) {
  const { t } = useTranslation('admin')
  const queryClient = useQueryClient()
  const formRef = useRef<TenantFormHandle>(null)

  // Create tenant mutation
  const createMutation = useCreateTenant({
    mutation: {
      onSuccess: () => {
        Toast.success(t('tenants.messages.createSuccess', 'Tenant created successfully'))
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        onClose()
        onSuccess?.()
      },
      onError: (error: Error | null) => {
        const errorMessage =
          error?.message || t('tenants.messages.createError', 'Failed to create tenant')
        Toast.error(errorMessage)
      },
    },
  })

  // Handle form submission
  const handleSubmit = useCallback(
    (data: CreateTenantFormData) => {
      // Transform form data to API format
      const requestBody: CreateTenantBody = {
        name: data.name,
        code: data.code,
        short_name: data.short_name || undefined,
        contact_email: data.contact_email || undefined,
        contact_name: data.contact_name || undefined,
        contact_phone: data.contact_phone || undefined,
        address: data.address || undefined,
        domain: data.domain || undefined,
        plan: data.plan,
        trial_days: data.trial_days,
        notes: data.notes || undefined,
      }

      createMutation.mutate({ data: requestBody })
    },
    [createMutation]
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

  return (
    <Modal
      title={t('tenants.modal.createTitle', 'Create Tenant')}
      visible={visible}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={t('common.create', 'Create')}
      cancelText={t('common.cancel', 'Cancel')}
      confirmLoading={createMutation.isPending}
      width={600}
      closeOnEsc={false}
      maskClosable={false}
    >
      <TenantForm mode="create" onSubmit={handleSubmit} isLoading={false} ref={formRef} />
    </Modal>
  )
}

export default CreateTenantModal
