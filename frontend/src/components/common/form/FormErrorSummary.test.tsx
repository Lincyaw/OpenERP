import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { FieldErrors } from 'react-hook-form'
import { FormErrorSummary } from './FormErrorSummary'

// Mock scrollIntoView which is not available in jsdom
Element.prototype.scrollIntoView = vi.fn()

// Mock useTranslation
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, options?: { count?: number; defaultValue?: string }) => {
      const translations: Record<string, string> = {
        'errorSummary.title': 'Please fix the following errors',
        'errorSummary.single': 'Form has 1 error',
        'errorSummary.multiple': `Form has ${options?.count || 0} errors`,
        'errorSummary.screenReaderAnnounce': `Form validation failed, ${options?.count || 0} errors need to be fixed`,
        'fieldLabels.name': 'Name',
        'fieldLabels.email': 'Email',
        'fieldLabels.price': 'Price',
      }
      return translations[key] || options?.defaultValue || key
    },
  }),
}))

describe('FormErrorSummary', () => {
  const mockErrors: FieldErrors = {
    name: { type: 'required', message: 'Name is required' },
    email: { type: 'pattern', message: 'Invalid email format' },
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Rendering', () => {
    it('renders nothing when there are no errors', () => {
      const { container } = render(<FormErrorSummary errors={{}} />)
      expect(container.firstChild).toBeNull()
    })

    it('renders nothing when show is false', () => {
      const { container } = render(<FormErrorSummary errors={mockErrors} show={false} />)
      expect(container.firstChild).toBeNull()
    })

    it('renders error summary when there are errors', () => {
      render(<FormErrorSummary errors={mockErrors} />)
      // Use querySelector to get our specific alert div (not Banner's nested alert)
      const summaryAlert = document.querySelector('.form-error-summary[role="alert"]')
      expect(summaryAlert).toBeInTheDocument()
      expect(screen.getByText('Please fix the following errors')).toBeInTheDocument()
    })

    it('displays all error messages', () => {
      render(<FormErrorSummary errors={mockErrors} />)
      expect(screen.getByText('Name is required')).toBeInTheDocument()
      expect(screen.getByText('Invalid email format')).toBeInTheDocument()
    })

    it('displays field labels from fieldLabels prop', () => {
      render(
        <FormErrorSummary
          errors={mockErrors}
          fieldLabels={{
            name: 'Product Name',
            email: 'Email Address',
          }}
        />
      )
      expect(screen.getByText('Product Name:')).toBeInTheDocument()
      expect(screen.getByText('Email Address:')).toBeInTheDocument()
    })

    it('uses i18n field labels as fallback', () => {
      render(<FormErrorSummary errors={mockErrors} />)
      expect(screen.getByText('Name:')).toBeInTheDocument()
      expect(screen.getByText('Email:')).toBeInTheDocument()
    })

    it('shows correct error count for single error', () => {
      const singleError: FieldErrors = {
        name: { type: 'required', message: 'Required' },
      }
      render(<FormErrorSummary errors={singleError} />)
      expect(screen.getByText('Form has 1 error')).toBeInTheDocument()
    })

    it('shows correct error count for multiple errors', () => {
      render(<FormErrorSummary errors={mockErrors} />)
      expect(screen.getByText('Form has 2 errors')).toBeInTheDocument()
    })
  })

  describe('MaxVisible prop', () => {
    const manyErrors: FieldErrors = {
      field1: { type: 'required', message: 'Error 1' },
      field2: { type: 'required', message: 'Error 2' },
      field3: { type: 'required', message: 'Error 3' },
      field4: { type: 'required', message: 'Error 4' },
      field5: { type: 'required', message: 'Error 5' },
      field6: { type: 'required', message: 'Error 6' },
    }

    it('limits visible errors to maxVisible', () => {
      render(<FormErrorSummary errors={manyErrors} maxVisible={3} />)
      expect(screen.getByText('Error 1')).toBeInTheDocument()
      expect(screen.getByText('Error 2')).toBeInTheDocument()
      expect(screen.getByText('Error 3')).toBeInTheDocument()
      expect(screen.queryByText('Error 4')).not.toBeInTheDocument()
    })

    it('shows all errors when maxVisible is 0', () => {
      render(<FormErrorSummary errors={manyErrors} maxVisible={0} />)
      expect(screen.getByText('Error 1')).toBeInTheDocument()
      expect(screen.getByText('Error 6')).toBeInTheDocument()
    })
  })

  describe('Nested errors', () => {
    it('flattens nested field errors', () => {
      const nestedErrors: FieldErrors = {
        address: {
          street: { type: 'required', message: 'Street is required' },
          city: { type: 'required', message: 'City is required' },
        },
      }
      render(
        <FormErrorSummary
          errors={nestedErrors}
          fieldLabels={{
            'address.street': 'Street Address',
            'address.city': 'City',
          }}
        />
      )
      expect(screen.getByText('Street is required')).toBeInTheDocument()
      expect(screen.getByText('City is required')).toBeInTheDocument()
    })
  })

  describe('Interaction', () => {
    it('calls onErrorClick when clicking an error link', async () => {
      const user = userEvent.setup()
      const onErrorClick = vi.fn()
      render(<FormErrorSummary errors={mockErrors} onErrorClick={onErrorClick} />)

      const nameErrorLink = screen.getByText('Name:').closest('a')
      await user.click(nameErrorLink!)

      expect(onErrorClick).toHaveBeenCalledWith('name')
    })

    it('handles keyboard navigation', async () => {
      const user = userEvent.setup()
      const onErrorClick = vi.fn()
      render(<FormErrorSummary errors={mockErrors} onErrorClick={onErrorClick} />)

      const nameErrorLink = screen.getByText('Name:').closest('a')
      nameErrorLink?.focus()
      await user.keyboard('{Enter}')

      expect(onErrorClick).toHaveBeenCalledWith('name')
    })

    it('focuses field when autoFocusOnClick is true', async () => {
      const user = userEvent.setup()
      const mockInput = document.createElement('input')
      mockInput.name = 'name'
      mockInput.id = 'name'
      document.body.appendChild(mockInput)

      render(<FormErrorSummary errors={mockErrors} autoFocusOnClick={true} />)

      const nameErrorLink = screen.getByText('Name:').closest('a')
      await user.click(nameErrorLink!)

      expect(document.activeElement).toBe(mockInput)

      document.body.removeChild(mockInput)
    })
  })

  describe('Accessibility', () => {
    it('has correct ARIA attributes', () => {
      render(<FormErrorSummary errors={mockErrors} />)

      // Use querySelector to get our specific alert div
      const alert = document.querySelector('.form-error-summary[role="alert"]')
      expect(alert).toBeInTheDocument()
      // Use attribute selector for dynamic id from useId()
      expect(alert).toHaveAttribute('aria-labelledby')
      const labelledById = alert?.getAttribute('aria-labelledby')
      expect(document.getElementById(labelledById!)).toBeInTheDocument()

      const liveRegion = document.querySelector('[aria-live="polite"]')
      expect(liveRegion).toBeInTheDocument()
      expect(liveRegion).toHaveAttribute('aria-atomic', 'true')
    })

    it('has list role for error items', () => {
      render(<FormErrorSummary errors={mockErrors} />)
      expect(screen.getByRole('list')).toBeInTheDocument()
    })

    it('error links are focusable', () => {
      render(<FormErrorSummary errors={mockErrors} />)
      const links = screen.getAllByRole('link')
      links.forEach((link) => {
        expect(link).toHaveAttribute('tabIndex', '0')
      })
    })
  })

  describe('Custom className', () => {
    it('applies custom className', () => {
      render(<FormErrorSummary errors={mockErrors} className="custom-class" />)
      const summary = document.querySelector('.form-error-summary')
      expect(summary).toHaveClass('custom-class')
    })
  })
})
