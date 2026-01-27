import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Modal,
  Form,
  InputNumber,
  TextArea,
  Input,
  Toast,
  Typography,
  Descriptions,
  Select,
} from '@douyinfe/semi-ui-19'
import { getBalance } from '@/api/balance/balance'
import type { HandlerRechargeRequest, HandlerRechargeRequestPaymentMethod } from '@/api/models'
import { useFormatters } from '@/hooks/useFormatters'
import './RechargeModal.css'

const { Text } = Typography

interface RechargeModalProps {
  visible: boolean
  customerId: string
  customerName: string
  currentBalance: number
  onClose: () => void
  onSuccess: () => void
}

/**
 * RechargeModal - Modal component for recharging customer balance
 *
 * Features:
 * - Amount input with validation (positive number required)
 * - Optional reference number
 * - Optional remark
 * - Preview of balance after recharge
 */
export default function RechargeModal({
  visible,
  customerId,
  customerName,
  currentBalance,
  onClose,
  onSuccess,
}: RechargeModalProps) {
  const { t } = useTranslation(['partner', 'common'])
  const { formatCurrency } = useFormatters()
  const balanceApi = useMemo(() => getBalance(), [])

  // Form state
  const [amount, setAmount] = useState<number | null>(null)
  const [paymentMethod, setPaymentMethod] = useState<HandlerRechargeRequestPaymentMethod>('CASH')
  const [reference, setReference] = useState('')
  const [remark, setRemark] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // Calculate balance after recharge
  const balanceAfter = useMemo(() => {
    if (amount && amount > 0) {
      return currentBalance + amount
    }
    return currentBalance
  }, [currentBalance, amount])

  // Reset form when modal closes
  const handleClose = useCallback(() => {
    setAmount(null)
    setPaymentMethod('CASH')
    setReference('')
    setRemark('')
    onClose()
  }, [onClose])

  // Handle form submission
  const handleSubmit = useCallback(async () => {
    // Validate amount
    if (!amount || amount <= 0) {
      Toast.error(t('balance.rechargeModal.amountRequired'))
      return
    }

    setSubmitting(true)
    try {
      const request: HandlerRechargeRequest = {
        amount,
        payment_method: paymentMethod,
        reference: reference || undefined,
        remark: remark || undefined,
      }

      const response = await balanceApi.postPartnerCustomersIdBalanceRecharge(customerId, request)

      if (response.success) {
        handleClose()
        onSuccess()
      } else {
        Toast.error(t('balance.rechargeModal.rechargeError'))
      }
    } catch {
      Toast.error(t('balance.rechargeModal.rechargeError'))
    } finally {
      setSubmitting(false)
    }
  }, [amount, paymentMethod, reference, remark, customerId, balanceApi, handleClose, onSuccess, t])

  // Handle amount change
  const handleAmountChange = useCallback((value: number | string) => {
    const numValue = typeof value === 'string' ? parseFloat(value) : value
    setAmount(isNaN(numValue) ? null : numValue)
  }, [])

  return (
    <Modal
      title={t('balance.rechargeModal.title')}
      visible={visible}
      onCancel={handleClose}
      onOk={handleSubmit}
      okText={t('balance.rechargeModal.confirmRecharge')}
      cancelText={t('common:actions.cancel')}
      confirmLoading={submitting}
      className="recharge-modal"
      maskClosable={false}
    >
      <div className="recharge-modal-content">
        {/* Customer Info */}
        <div className="customer-info-section">
          <Descriptions
            row
            data={[
              { key: t('balance.rechargeModal.customerName'), value: customerName },
              {
                key: t('balance.rechargeModal.currentBalance'),
                value: <Text strong>{formatCurrency(currentBalance)}</Text>,
              },
            ]}
          />
        </div>

        {/* Recharge Form */}
        <Form className="recharge-form" labelPosition="top">
          <Form.Slot label={`${t('balance.rechargeModal.amount')} *`}>
            <InputNumber
              value={amount ?? undefined}
              onChange={handleAmountChange}
              placeholder={t('balance.rechargeModal.amountPlaceholder')}
              prefix="¥"
              min={0.01}
              precision={2}
              style={{ width: '100%' }}
              size="large"
              className="amount-input"
            />
          </Form.Slot>

          <Form.Slot label={`${t('balance.rechargeModal.paymentMethod')} *`}>
            <Select
              value={paymentMethod}
              onChange={(value) => setPaymentMethod(value as HandlerRechargeRequestPaymentMethod)}
              optionList={[
                { label: t('balance.paymentMethods.CASH'), value: 'CASH' },
                { label: t('balance.paymentMethods.WECHAT'), value: 'WECHAT' },
                { label: t('balance.paymentMethods.ALIPAY'), value: 'ALIPAY' },
                { label: t('balance.paymentMethods.BANK'), value: 'BANK' },
              ]}
              style={{ width: '100%' }}
            />
          </Form.Slot>

          <Form.Slot label={t('balance.rechargeModal.reference')}>
            <Input
              value={reference}
              onChange={(value) => setReference(value)}
              placeholder={t('balance.rechargeModal.referencePlaceholder')}
              maxLength={100}
            />
          </Form.Slot>

          <Form.Slot label={t('balance.rechargeModal.remark')}>
            <TextArea
              value={remark}
              onChange={(value) => setRemark(value)}
              placeholder={t('balance.rechargeModal.remarkPlaceholder')}
              maxLength={500}
              rows={3}
              showClear
            />
          </Form.Slot>
        </Form>

        {/* Balance Preview */}
        {amount && amount > 0 && (
          <div className="balance-preview">
            <div className="balance-preview-row">
              <span className="balance-preview-label">
                {t('balance.rechargeModal.preview.currentBalance')}
              </span>
              <span className="balance-preview-value">{formatCurrency(currentBalance)}</span>
            </div>
            <div className="balance-preview-row">
              <span className="balance-preview-label">
                {t('balance.rechargeModal.preview.rechargeAmount')}
              </span>
              <span className="balance-preview-value balance-add">+{formatCurrency(amount)}</span>
            </div>
            <div className="balance-preview-divider" />
            <div className="balance-preview-row balance-preview-total">
              <span className="balance-preview-label">
                {t('balance.rechargeModal.preview.balanceAfter')}
              </span>
              <span className="balance-preview-value balance-after">
                {formatCurrency(balanceAfter)}
              </span>
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
}
