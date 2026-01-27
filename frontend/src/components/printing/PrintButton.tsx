/**
 * PrintButton Component
 *
 * A button that opens the print preview modal.
 * Supports keyboard shortcut Ctrl+P (or Cmd+P on Mac).
 *
 * @example
 * <PrintButton
 *   documentType="SALES_ORDER"
 *   documentId={orderId}
 *   documentNumber={orderNumber}
 * />
 */

import { useState, useEffect, useCallback } from 'react'
import { Button, Tooltip } from '@douyinfe/semi-ui-19'
import { IconPrint } from '@douyinfe/semi-icons'
import { PrintPreviewModal } from './PrintPreviewModal'

export interface PrintButtonProps {
  /** Document type (e.g., 'SALES_ORDER', 'SALES_DELIVERY') */
  documentType: string
  /** Document UUID */
  documentId: string
  /** Document number for display (e.g., 'SO-2024-001') */
  documentNumber: string
  /** Additional data for template rendering */
  data?: unknown
  /** Button size */
  size?: 'small' | 'default' | 'large'
  /** Button type */
  type?: 'primary' | 'secondary' | 'tertiary' | 'warning' | 'danger'
  /** Show text label */
  showLabel?: boolean
  /** Custom label text */
  label?: string
  /** Icon only mode */
  iconOnly?: boolean
  /** Disable the button */
  disabled?: boolean
  /** Enable Ctrl+P keyboard shortcut */
  enableShortcut?: boolean
  /** Custom class name */
  className?: string
}

export function PrintButton({
  documentType,
  documentId,
  documentNumber,
  data,
  size = 'default',
  type = 'tertiary',
  showLabel = true,
  label = '打印',
  iconOnly = false,
  disabled = false,
  enableShortcut = true,
  className,
}: PrintButtonProps) {
  const [modalVisible, setModalVisible] = useState(false)

  // Handle Ctrl+P keyboard shortcut
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enableShortcut || disabled) return

      // Check for Ctrl+P (Windows/Linux) or Cmd+P (Mac)
      const isCtrlOrCmd = event.ctrlKey || event.metaKey
      if (isCtrlOrCmd && event.key === 'p') {
        event.preventDefault()
        setModalVisible(true)
      }
    },
    [enableShortcut, disabled]
  )

  // Add keyboard event listener
  useEffect(() => {
    if (enableShortcut) {
      document.addEventListener('keydown', handleKeyDown)
      return () => {
        document.removeEventListener('keydown', handleKeyDown)
      }
    }
  }, [enableShortcut, handleKeyDown])

  const handleClick = () => {
    setModalVisible(true)
  }

  const handleClose = () => {
    setModalVisible(false)
  }

  // Tooltip content with shortcut hint
  const tooltipContent = enableShortcut ? `${label} (Ctrl+P)` : label

  const buttonContent = iconOnly ? null : showLabel ? label : null

  const button = (
    <Button
      icon={<IconPrint />}
      size={size}
      type={type}
      disabled={disabled}
      onClick={handleClick}
      className={className}
      aria-label={label}
    >
      {buttonContent}
    </Button>
  )

  return (
    <>
      {iconOnly ? <Tooltip content={tooltipContent}>{button}</Tooltip> : button}

      <PrintPreviewModal
        visible={modalVisible}
        onClose={handleClose}
        documentType={documentType}
        documentId={documentId}
        documentNumber={documentNumber}
        data={data}
      />
    </>
  )
}
