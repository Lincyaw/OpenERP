import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ImportWizard } from './ImportWizard'

// Mock axios
vi.mock('@/services/axios-instance', () => ({
  axiosInstance: {
    post: vi.fn(),
  },
}))

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const translations: Record<string, string> = {
        'import.wizard.title': `Import ${params?.entity || ''}`,
        'import.steps.upload': 'Upload File',
        'import.steps.validate': 'Validate',
        'import.steps.import': 'Import',
        'import.steps.results': 'Results',
        'import.entityTypes.products': 'Products',
        'import.entityTypes.customers': 'Customers',
        'import.upload.title': `Upload ${params?.entity || ''} Data`,
        'import.upload.dragDropText': 'Drag and drop CSV file here',
        'import.upload.orClickText': 'or click to select file',
        'import.upload.selectFile': 'Select File',
        'import.upload.fileRequirements': `CSV format, max ${params?.maxSize || ''}`,
        'import.upload.dropzone': 'Drop file here',
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

describe('ImportWizard', () => {
  const mockOnClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders wizard modal when visible', () => {
    render(<ImportWizard visible={true} onClose={mockOnClose} entityType="products" />)

    expect(screen.getByText('Import Products')).toBeInTheDocument()
    expect(screen.getByText('Upload File')).toBeInTheDocument()
    expect(screen.getByText('Validate')).toBeInTheDocument()
    expect(screen.getByText('Import')).toBeInTheDocument()
    expect(screen.getByText('Results')).toBeInTheDocument()
  })

  it('does not render when not visible', () => {
    render(<ImportWizard visible={false} onClose={mockOnClose} entityType="products" />)

    expect(screen.queryByText('Import Products')).not.toBeInTheDocument()
  })

  it('starts at upload step', () => {
    render(<ImportWizard visible={true} onClose={mockOnClose} entityType="products" />)

    expect(screen.getByText('Upload Products Data')).toBeInTheDocument()
    expect(screen.getByText('Drag and drop CSV file here')).toBeInTheDocument()
  })

  it('renders correct title for different entity types', () => {
    const { rerender } = render(
      <ImportWizard visible={true} onClose={mockOnClose} entityType="products" />
    )
    expect(screen.getByText('Import Products')).toBeInTheDocument()

    rerender(<ImportWizard visible={true} onClose={mockOnClose} entityType="customers" />)
    expect(screen.getByText('Import Customers')).toBeInTheDocument()
  })

  it('has file input for uploads', () => {
    render(<ImportWizard visible={true} onClose={mockOnClose} entityType="products" />)

    const input = document.querySelector('input[type="file"]')
    expect(input).toBeInTheDocument()
    expect(input).toHaveAttribute('accept')
  })
})
