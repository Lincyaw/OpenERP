import { useState, useCallback, useMemo } from 'react'
import {
  Modal,
  Form,
  InputNumber,
  TextArea,
  Input,
  Toast,
  Typography,
  Descriptions,
} from '@douyinfe/semi-ui'
import { getBalance } from '@/api/balance/balance'
import type { HandlerRechargeRequest } from '@/api/models'
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
 * Format currency for display
 */
function formatCurrency(amount?: number): string {
  if (amount === undefined || amount === null) return '¥0.00'
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
  }).format(amount)
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
  const balanceApi = useMemo(() => getBalance(), [])

  // Form state
  const [amount, setAmount] = useState<number | null>(null)
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
    setReference('')
    setRemark('')
    onClose()
  }, [onClose])

  // Handle form submission
  const handleSubmit = useCallback(async () => {
    // Validate amount
    if (!amount || amount <= 0) {
      Toast.error('请输入有效的充值金额')
      return
    }

    setSubmitting(true)
    try {
      const request: HandlerRechargeRequest = {
        amount,
        reference: reference || undefined,
        remark: remark || undefined,
      }

      const response = await balanceApi.postPartnerCustomersCustomerIdBalanceRecharge(
        customerId,
        request
      )

      if (response.success) {
        handleClose()
        onSuccess()
      } else {
        Toast.error('充值失败，请稍后重试')
      }
    } catch {
      Toast.error('充值失败，请稍后重试')
    } finally {
      setSubmitting(false)
    }
  }, [amount, reference, remark, customerId, balanceApi, handleClose, onSuccess])

  // Handle amount change
  const handleAmountChange = useCallback((value: number | string) => {
    const numValue = typeof value === 'string' ? parseFloat(value) : value
    setAmount(isNaN(numValue) ? null : numValue)
  }, [])

  return (
    <Modal
      title="客户充值"
      visible={visible}
      onCancel={handleClose}
      onOk={handleSubmit}
      okText="确认充值"
      cancelText="取消"
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
              { key: '客户名称', value: customerName },
              { key: '当前余额', value: <Text strong>{formatCurrency(currentBalance)}</Text> },
            ]}
          />
        </div>

        {/* Recharge Form */}
        <Form className="recharge-form" labelPosition="top">
          <Form.Slot label="充值金额" required>
            <InputNumber
              value={amount ?? undefined}
              onChange={handleAmountChange}
              placeholder="请输入充值金额"
              prefix="¥"
              min={0.01}
              precision={2}
              style={{ width: '100%' }}
              size="large"
              className="amount-input"
            />
          </Form.Slot>

          <Form.Slot label="参考号">
            <Input
              value={reference}
              onChange={(value) => setReference(value)}
              placeholder="可选，如收据编号、流水号等"
              maxLength={100}
            />
          </Form.Slot>

          <Form.Slot label="备注">
            <TextArea
              value={remark}
              onChange={(value) => setRemark(value)}
              placeholder="可选，充值备注信息"
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
              <span className="balance-preview-label">当前余额</span>
              <span className="balance-preview-value">{formatCurrency(currentBalance)}</span>
            </div>
            <div className="balance-preview-row">
              <span className="balance-preview-label">充值金额</span>
              <span className="balance-preview-value balance-add">+{formatCurrency(amount)}</span>
            </div>
            <div className="balance-preview-divider" />
            <div className="balance-preview-row balance-preview-total">
              <span className="balance-preview-label">充值后余额</span>
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
