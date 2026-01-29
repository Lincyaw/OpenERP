import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FileUploadStep } from './FileUploadStep'

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, string>) => {
      const translations: Record<string, string> = {
        'import.entityTypes.products': 'Products',
        'import.entityTypes.customers': 'Customers',
        'import.upload.title': `Upload ${params?.entity || ''} Data`,
        'import.upload.description': 'Please upload a CSV file',
        'import.upload.selectFile': 'Select File',
        'import.upload.dropzone': 'Drop file here',
        'import.upload.dragDropText': 'Drag and drop CSV file here',
        'import.upload.orClickText': 'or click to select file',
        'import.upload.fileRequirements': `CSV format, max ${params?.maxSize || ''}`,
        'import.upload.removeFile': 'Remove file',
        'import.upload.templateHint': 'Not sure about the format?',
        'import.upload.downloadTemplate': 'Download Template',
        'import.errors.fileTooLarge': 'File too large',
        'import.errors.invalidFileType': 'Invalid file type',
      }
      return translations[key] || key
    },
  }),
}))

// Mock Semi UI Toast
vi.mock('@douyinfe/semi-ui-19', async () => {
  const actual = await vi.importActual('@douyinfe/semi-ui-19')
  return {
    ...actual,
    Toast: {
      error: vi.fn(),
      success: vi.fn(),
    },
  }
})

describe('FileUploadStep', () => {
  const mockOnFileSelect = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders upload step with empty state', () => {
    render(<FileUploadStep file={null} onFileSelect={mockOnFileSelect} entityType="products" />)

    expect(screen.getByText('Upload Products Data')).toBeInTheDocument()
    expect(screen.getByText('Drag and drop CSV file here')).toBeInTheDocument()
    expect(screen.getByText('or click to select file')).toBeInTheDocument()
  })

  it('renders template download button when templateUrl is provided', () => {
    render(
      <FileUploadStep
        file={null}
        onFileSelect={mockOnFileSelect}
        entityType="products"
        templateUrl="/templates/products.csv"
      />
    )

    expect(screen.getByText('Download Template')).toBeInTheDocument()
  })

  it('does not render template button when templateUrl is not provided', () => {
    render(<FileUploadStep file={null} onFileSelect={mockOnFileSelect} entityType="products" />)

    expect(screen.queryByText('Download Template')).not.toBeInTheDocument()
  })

  it('renders selected file info when file is provided', () => {
    const mockFile = new File(['test content'], 'test.csv', { type: 'text/csv' })

    render(<FileUploadStep file={mockFile} onFileSelect={mockOnFileSelect} entityType="products" />)

    expect(screen.getByText('test.csv')).toBeInTheDocument()
  })

  it('calls onFileSelect when file input changes with valid CSV', async () => {
    render(<FileUploadStep file={null} onFileSelect={mockOnFileSelect} entityType="products" />)

    const mockFile = new File(['test content'], 'test.csv', { type: 'text/csv' })
    const input = document.querySelector('input[type="file"]') as HTMLInputElement

    await userEvent.upload(input, mockFile)

    expect(mockOnFileSelect).toHaveBeenCalledWith(mockFile)
  })

  it('is disabled when disabled prop is true', () => {
    render(
      <FileUploadStep
        file={null}
        onFileSelect={mockOnFileSelect}
        entityType="products"
        disabled={true}
      />
    )

    const dropzone = screen.getByRole('button')
    expect(dropzone).toHaveAttribute('tabIndex', '-1')
  })

  it('handles drag and drop', async () => {
    render(<FileUploadStep file={null} onFileSelect={mockOnFileSelect} entityType="products" />)

    const dropzone = screen.getByRole('button')

    // Simulate drag over
    fireEvent.dragOver(dropzone)
    expect(dropzone).toHaveClass('file-upload-dropzone--drag-over')

    // Simulate drag leave
    fireEvent.dragLeave(dropzone)
    expect(dropzone).not.toHaveClass('file-upload-dropzone--drag-over')
  })

  it('opens template URL when download template is clicked', async () => {
    const mockOpen = vi.fn()
    window.open = mockOpen

    render(
      <FileUploadStep
        file={null}
        onFileSelect={mockOnFileSelect}
        entityType="products"
        templateUrl="/templates/products.csv"
      />
    )

    const downloadButton = screen.getByText('Download Template')
    await userEvent.click(downloadButton)

    expect(mockOpen).toHaveBeenCalledWith('/templates/products.csv', '_blank')
  })

  it('handles keyboard navigation', async () => {
    render(<FileUploadStep file={null} onFileSelect={mockOnFileSelect} entityType="products" />)

    const dropzone = screen.getByRole('button')

    // Tab to dropzone and press Enter
    dropzone.focus()
    fireEvent.keyDown(dropzone, { key: 'Enter' })

    // The file input should be triggered (hard to test directly)
    // But we can verify the dropzone is focusable
    expect(dropzone).toHaveAttribute('tabIndex', '0')
  })

  it('displays different entity types correctly', () => {
    render(<FileUploadStep file={null} onFileSelect={mockOnFileSelect} entityType="customers" />)

    expect(screen.getByText('Upload Customers Data')).toBeInTheDocument()
  })
})
