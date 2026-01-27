/**
 * PrintButton Component Tests
 *
 * Tests for the print button component and keyboard shortcut.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderWithProviders, screen, fireEvent, waitFor } from '@/tests/utils'
import { PrintButton } from './PrintButton'
import * as printApi from '@/api/printing'
import { Toast } from '@douyinfe/semi-ui-19'

// Mock the print API
vi.mock('@/api/printing', () => ({
  previewDocument: vi.fn(),
  generatePDF: vi.fn(),
  getTemplatesByDocType: vi.fn(),
}))

// Spy on Toast methods
vi.spyOn(Toast, 'success').mockImplementation(() => '')
vi.spyOn(Toast, 'error').mockImplementation(() => '')
vi.spyOn(Toast, 'info').mockImplementation(() => '')

describe('PrintButton', () => {
  const defaultProps = {
    documentType: 'SALES_ORDER',
    documentId: 'doc-1',
    documentNumber: 'SO-2024-001',
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(printApi.getTemplatesByDocType).mockResolvedValue([])
    vi.mocked(printApi.previewDocument).mockResolvedValue({
      html: '<html><body>Test</body></html>',
      templateId: 'template-1',
      paperSize: 'A4',
      orientation: 'PORTRAIT',
      margins: { top: 10, right: 10, bottom: 10, left: 10 },
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should render with default props', () => {
    renderWithProviders(<PrintButton {...defaultProps} />)

    const button = screen.getByRole('button', { name: /打印/i })
    expect(button).toBeInTheDocument()
    expect(button).toHaveTextContent('打印')
  })

  it('should render with custom label', () => {
    renderWithProviders(<PrintButton {...defaultProps} label="Print Order" />)

    expect(screen.getByRole('button', { name: /Print Order/i })).toBeInTheDocument()
  })

  it('should render as icon only button', () => {
    renderWithProviders(<PrintButton {...defaultProps} iconOnly />)

    const button = screen.getByRole('button', { name: /打印/i })
    expect(button).toBeInTheDocument()
    // Icon-only button should not have text content
    expect(button.textContent).toBe('')
  })

  it('should be disabled when disabled prop is true', () => {
    renderWithProviders(<PrintButton {...defaultProps} disabled />)

    const button = screen.getByRole('button', { name: /打印/i })
    expect(button).toBeDisabled()
  })

  it('should open modal on click', async () => {
    renderWithProviders(<PrintButton {...defaultProps} />)

    const button = screen.getByRole('button', { name: /打印/i })
    fireEvent.click(button)

    await waitFor(() => {
      expect(screen.getByText('打印预览 - SO-2024-001')).toBeInTheDocument()
    })
  })

  it('should open modal on Ctrl+P when enableShortcut is true', async () => {
    renderWithProviders(<PrintButton {...defaultProps} enableShortcut />)

    // Simulate Ctrl+P
    fireEvent.keyDown(document, { key: 'p', ctrlKey: true })

    await waitFor(() => {
      expect(screen.getByText('打印预览 - SO-2024-001')).toBeInTheDocument()
    })
  })

  it('should not open modal on Ctrl+P when enableShortcut is false', async () => {
    renderWithProviders(<PrintButton {...defaultProps} enableShortcut={false} />)

    // Simulate Ctrl+P
    fireEvent.keyDown(document, { key: 'p', ctrlKey: true })

    // Modal should not appear
    await waitFor(
      () => {
        expect(screen.queryByText('打印预览 - SO-2024-001')).not.toBeInTheDocument()
      },
      { timeout: 100 }
    )
  })

  it('should not respond to Ctrl+P when disabled', async () => {
    renderWithProviders(<PrintButton {...defaultProps} disabled enableShortcut />)

    // Simulate Ctrl+P
    fireEvent.keyDown(document, { key: 'p', ctrlKey: true })

    // Modal should not appear
    await waitFor(
      () => {
        expect(screen.queryByText('打印预览 - SO-2024-001')).not.toBeInTheDocument()
      },
      { timeout: 100 }
    )
  })

  it('should support Cmd+P on Mac', async () => {
    renderWithProviders(<PrintButton {...defaultProps} enableShortcut />)

    // Simulate Cmd+P (metaKey for Mac)
    fireEvent.keyDown(document, { key: 'p', metaKey: true })

    await waitFor(() => {
      expect(screen.getByText('打印预览 - SO-2024-001')).toBeInTheDocument()
    })
  })

  it('should prevent default browser print dialog', () => {
    renderWithProviders(<PrintButton {...defaultProps} enableShortcut />)

    const event = new KeyboardEvent('keydown', {
      key: 'p',
      ctrlKey: true,
      bubbles: true,
      cancelable: true,
    })
    const preventDefaultSpy = vi.spyOn(event, 'preventDefault')

    document.dispatchEvent(event)

    expect(preventDefaultSpy).toHaveBeenCalled()
  })

  it('should have cancel button in modal', async () => {
    renderWithProviders(<PrintButton {...defaultProps} />)

    // Open modal
    const button = screen.getByRole('button', { name: /打印/i })
    fireEvent.click(button)

    await waitFor(() => {
      expect(screen.getByText('打印预览 - SO-2024-001')).toBeInTheDocument()
    })

    // Verify cancel button is present and clickable
    const cancelButton = screen.getByRole('button', { name: /取消/i })
    expect(cancelButton).toBeInTheDocument()
    expect(cancelButton).toBeEnabled()
  })

  it('should apply custom className', () => {
    renderWithProviders(<PrintButton {...defaultProps} className="custom-class" />)

    const button = screen.getByRole('button', { name: /打印/i })
    expect(button).toHaveClass('custom-class')
  })
})
