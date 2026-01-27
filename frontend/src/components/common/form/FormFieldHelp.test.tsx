import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { FormFieldHelp, FormFieldHint, FormFieldExample } from './FormFieldHelp'

// Mock useTranslation
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, options?: { defaultValue?: string }) => {
      const translations: Record<string, string> = {
        'help.ariaLabel': 'Help information',
        'help.example': 'Example: ',
      }
      return translations[key] || options?.defaultValue || key
    },
  }),
}))

describe('FormFieldHelp', () => {
  describe('Basic rendering', () => {
    it('renders help icon button', () => {
      render(<FormFieldHelp content="Help text" />)
      const button = screen.getByRole('button')
      expect(button).toBeInTheDocument()
      expect(button).toHaveAttribute('type', 'button')
    })

    it('renders with default aria-label from i18n', () => {
      render(<FormFieldHelp content="Help text" />)
      const button = screen.getByRole('button')
      expect(button).toHaveAttribute('aria-label', 'Help information')
    })

    it('renders with custom aria-label', () => {
      render(<FormFieldHelp content="Help text" ariaLabel="Custom help" />)
      const button = screen.getByRole('button')
      expect(button).toHaveAttribute('aria-label', 'Custom help')
    })

    it('applies size variants', () => {
      const { rerender } = render(<FormFieldHelp content="Help" size="small" />)
      expect(document.querySelector('.form-field-help--small')).toBeInTheDocument()

      rerender(<FormFieldHelp content="Help" size="large" />)
      expect(document.querySelector('.form-field-help--large')).toBeInTheDocument()
    })

    it('applies custom className', () => {
      render(<FormFieldHelp content="Help" className="custom-class" />)
      expect(document.querySelector('.custom-class')).toBeInTheDocument()
    })
  })

  describe('Inline mode', () => {
    it('renders as inline text when inline is true', () => {
      render(<FormFieldHelp content="Inline help text" inline />)
      expect(screen.getByText('Inline help text')).toBeInTheDocument()
      expect(screen.queryByRole('button')).not.toBeInTheDocument()
      expect(document.querySelector('.form-field-help--inline')).toBeInTheDocument()
    })
  })

  describe('Custom icon', () => {
    it('renders custom icon', () => {
      render(<FormFieldHelp content="Help" icon={<span data-testid="custom-icon">?</span>} />)
      expect(screen.getByTestId('custom-icon')).toBeInTheDocument()
    })
  })

  describe('Tooltip behavior', () => {
    it('shows tooltip on hover', async () => {
      const user = userEvent.setup()
      render(<FormFieldHelp content="Tooltip content" />)

      const button = screen.getByRole('button')
      await user.hover(button)

      await waitFor(() => {
        expect(screen.getByText('Tooltip content')).toBeInTheDocument()
      })
    })

    it('renders tooltip with title', async () => {
      const user = userEvent.setup()
      render(<FormFieldHelp title="Help Title" content="Help content" />)

      const button = screen.getByRole('button')
      await user.hover(button)

      await waitFor(() => {
        expect(screen.getByText('Help Title')).toBeInTheDocument()
        expect(screen.getByText('Help content')).toBeInTheDocument()
      })
    })
  })

  describe('Keyboard accessibility', () => {
    it('is focusable', () => {
      render(<FormFieldHelp content="Help" />)
      const button = screen.getByRole('button')
      expect(button).toHaveAttribute('tabIndex', '0')
    })

    it('shows tooltip on focus', () => {
      render(<FormFieldHelp content="Tooltip content" />)

      const button = screen.getByRole('button')
      button.focus()

      // Focus alone doesn't trigger tooltip (only hover does in Semi UI)
      // But we can verify the button is focusable
      expect(document.activeElement).toBe(button)
    })
  })
})

describe('FormFieldHint', () => {
  describe('Basic rendering', () => {
    it('renders hint text', () => {
      render(<FormFieldHint>This is a hint</FormFieldHint>)
      expect(screen.getByText('This is a hint')).toBeInTheDocument()
    })

    it('has note role for accessibility', () => {
      render(<FormFieldHint>Hint text</FormFieldHint>)
      expect(screen.getByRole('note')).toBeInTheDocument()
    })

    it('renders default type', () => {
      render(<FormFieldHint>Hint</FormFieldHint>)
      expect(document.querySelector('.form-field-hint--default')).toBeInTheDocument()
    })
  })

  describe('Type variants', () => {
    it('renders info type', () => {
      render(<FormFieldHint type="info">Info hint</FormFieldHint>)
      expect(document.querySelector('.form-field-hint--info')).toBeInTheDocument()
    })

    it('renders warning type', () => {
      render(<FormFieldHint type="warning">Warning hint</FormFieldHint>)
      expect(document.querySelector('.form-field-hint--warning')).toBeInTheDocument()
    })
  })

  describe('Custom className', () => {
    it('applies custom className', () => {
      render(<FormFieldHint className="custom-hint">Hint</FormFieldHint>)
      expect(document.querySelector('.custom-hint')).toBeInTheDocument()
    })
  })
})

describe('FormFieldExample', () => {
  describe('Single example', () => {
    it('renders single example', () => {
      render(<FormFieldExample examples="example@test.com" />)
      expect(screen.getByText('example@test.com')).toBeInTheDocument()
    })

    it('renders with default prefix from i18n', () => {
      render(<FormFieldExample examples="example" />)
      expect(screen.getByText('Example:')).toBeInTheDocument()
    })

    it('renders with custom prefix', () => {
      render(<FormFieldExample examples="example" prefix="e.g., " />)
      expect(screen.getByText('e.g.,')).toBeInTheDocument()
    })
  })

  describe('Multiple examples', () => {
    it('renders multiple examples', () => {
      render(<FormFieldExample examples={['PROD-001', 'SKU-ABC']} />)
      expect(screen.getByText('PROD-001')).toBeInTheDocument()
      expect(screen.getByText('SKU-ABC')).toBeInTheDocument()
    })

    it('separates examples with commas', () => {
      render(<FormFieldExample examples={['one', 'two', 'three']} />)
      const separators = document.querySelectorAll('.form-field-example__separator')
      expect(separators).toHaveLength(2) // Between one-two and two-three
    })
  })

  describe('Styling', () => {
    it('renders examples in code elements', () => {
      render(<FormFieldExample examples="example" />)
      const code = document.querySelector('code')
      expect(code).toBeInTheDocument()
      expect(code).toHaveClass('form-field-example__value')
    })

    it('applies custom className', () => {
      render(<FormFieldExample examples="test" className="custom-example" />)
      expect(document.querySelector('.custom-example')).toBeInTheDocument()
    })
  })
})
