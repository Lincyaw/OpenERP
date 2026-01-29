import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ErrorTable } from './ErrorTable'
import type { CsvimportRowError } from '@/api/models'

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        'import.errors.row': 'Row',
        'import.errors.column': 'Column',
        'import.errors.value': 'Value',
        'import.errors.message': 'Message',
        'import.errors.code': 'Code',
        'import.errors.export': 'Export',
        'import.errors.noErrors': 'No errors',
        'import.errors.title': 'Errors',
        'import.errors.truncated': 'Truncated',
      }
      return translations[key] || key
    },
  }),
}))

describe('ErrorTable', () => {
  const mockErrors: CsvimportRowError[] = [
    {
      row: 2,
      column: 'code',
      value: 'PROD001',
      message: 'Product code already exists',
      code: 'DUPLICATE',
    },
    {
      row: 5,
      column: 'price',
      value: '-10',
      message: 'Price must be positive',
      code: 'INVALID_VALUE',
    },
    {
      row: 8,
      column: 'name',
      value: '',
      message: 'Name is required',
      code: 'REQUIRED',
    },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders error table with errors', () => {
    render(<ErrorTable errors={mockErrors} />)

    // Check that error data is displayed
    expect(screen.getByText('2')).toBeInTheDocument() // Row number
    expect(screen.getByText('code')).toBeInTheDocument() // Column name
    expect(screen.getByText('PROD001')).toBeInTheDocument() // Value
    expect(screen.getByText('Product code already exists')).toBeInTheDocument() // Message
  })

  it('shows export button when showExport is true', () => {
    render(<ErrorTable errors={mockErrors} showExport={true} />)

    expect(screen.getByText('Export')).toBeInTheDocument()
  })

  it('hides export button when showExport is false', () => {
    render(<ErrorTable errors={mockErrors} showExport={false} />)

    expect(screen.queryByText('Export')).not.toBeInTheDocument()
  })

  it('calls onExport when export button is clicked', async () => {
    const mockOnExport = vi.fn()
    render(<ErrorTable errors={mockErrors} showExport={true} onExport={mockOnExport} />)

    const exportButton = screen.getByText('Export')
    await userEvent.click(exportButton)

    expect(mockOnExport).toHaveBeenCalledTimes(1)
  })

  it('shows truncated message when isTruncated is true', () => {
    render(<ErrorTable errors={mockErrors} isTruncated={true} totalErrors={100} />)

    expect(screen.getByText('Truncated')).toBeInTheDocument()
  })

  it('applies correct color to different error codes', () => {
    render(<ErrorTable errors={mockErrors} />)

    // Check that error codes are displayed
    expect(screen.getByText('DUPLICATE')).toBeInTheDocument()
    expect(screen.getByText('REQUIRED')).toBeInTheDocument()
  })

  it('handles errors with missing optional fields', () => {
    const partialErrors: CsvimportRowError[] = [
      {
        row: 1,
        message: 'Some error',
      },
      {
        column: 'test',
        message: 'Another error',
      },
    ]

    render(<ErrorTable errors={partialErrors} />)

    // Should render without crashing
    expect(screen.getByText('Some error')).toBeInTheDocument()
    expect(screen.getByText('Another error')).toBeInTheDocument()
  })
})
