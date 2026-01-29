import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { IconMail, IconUser, IconSetting } from '@douyinfe/semi-icons'
import { FormSection } from './Form'

describe('FormSection Component', () => {
  describe('basic rendering', () => {
    it('renders children content', () => {
      render(
        <FormSection>
          <div data-testid="child-content">Form content</div>
        </FormSection>
      )

      expect(screen.getByTestId('child-content')).toBeInTheDocument()
      expect(screen.getByText('Form content')).toBeInTheDocument()
    })

    it('renders with title', () => {
      render(
        <FormSection title="Basic Information">
          <div>Content</div>
        </FormSection>
      )

      expect(screen.getByText('Basic Information')).toBeInTheDocument()
      expect(screen.getByRole('heading', { level: 3 })).toHaveTextContent('Basic Information')
    })

    it('renders with subtitle', () => {
      render(
        <FormSection title="Contact" subtitle="Enter your contact information">
          <div>Content</div>
        </FormSection>
      )

      expect(screen.getByText('Enter your contact information')).toBeInTheDocument()
    })

    it('renders with description (backward compatibility)', () => {
      render(
        <FormSection title="Contact" description="Description text">
          <div>Content</div>
        </FormSection>
      )

      expect(screen.getByText('Description text')).toBeInTheDocument()
    })

    it('prefers subtitle over description when both provided', () => {
      render(
        <FormSection title="Contact" subtitle="Subtitle text" description="Description text">
          <div>Content</div>
        </FormSection>
      )

      expect(screen.getByText('Subtitle text')).toBeInTheDocument()
      expect(screen.queryByText('Description text')).not.toBeInTheDocument()
    })

    it('renders without header when no title or subtitle', () => {
      const { container } = render(
        <FormSection>
          <div>Content only</div>
        </FormSection>
      )

      expect(container.querySelector('.form-section-header')).not.toBeInTheDocument()
    })

    it('applies custom className', () => {
      const { container } = render(
        <FormSection className="custom-section">
          <div>Content</div>
        </FormSection>
      )

      expect(container.querySelector('.form-section')).toHaveClass('custom-section')
    })

    it('renders with data-testid', () => {
      render(
        <FormSection data-testid="test-section">
          <div>Content</div>
        </FormSection>
      )

      expect(screen.getByTestId('test-section')).toBeInTheDocument()
    })
  })

  describe('icon prop', () => {
    it('renders icon component when provided', () => {
      const { container } = render(
        <FormSection title="Contact" icon={IconMail}>
          <div>Content</div>
        </FormSection>
      )

      const iconElement = container.querySelector('.form-section-icon')
      expect(iconElement).toBeInTheDocument()
    })

    it('renders icon ReactNode when provided', () => {
      render(
        <FormSection title="Contact" icon={<span data-testid="custom-icon">★</span>}>
          <div>Content</div>
        </FormSection>
      )

      expect(screen.getByTestId('custom-icon')).toBeInTheDocument()
    })

    it('renders different icon components correctly', () => {
      const { container, rerender } = render(
        <FormSection title="User" icon={IconUser}>
          <div>Content</div>
        </FormSection>
      )

      let iconElement = container.querySelector('.form-section-icon')
      expect(iconElement).toBeInTheDocument()

      rerender(
        <FormSection title="Settings" icon={IconSetting}>
          <div>Content</div>
        </FormSection>
      )

      iconElement = container.querySelector('.form-section-icon')
      expect(iconElement).toBeInTheDocument()
    })

    it('renders header with icon only (no title)', () => {
      const { container } = render(
        <FormSection icon={IconMail}>
          <div>Content</div>
        </FormSection>
      )

      const iconElement = container.querySelector('.form-section-icon')
      expect(iconElement).toBeInTheDocument()
      expect(container.querySelector('.form-section-header')).toBeInTheDocument()
    })
  })

  describe('required prop', () => {
    it('shows required indicator when required is true', () => {
      render(
        <FormSection title="Required Section" required>
          <div>Content</div>
        </FormSection>
      )

      const requiredIndicator = document.querySelector('.form-section-required')
      expect(requiredIndicator).toBeInTheDocument()
      expect(requiredIndicator).toHaveTextContent('*')
    })

    it('does not show required indicator when required is false', () => {
      render(
        <FormSection title="Optional Section" required={false}>
          <div>Content</div>
        </FormSection>
      )

      expect(document.querySelector('.form-section-required')).not.toBeInTheDocument()
    })

    it('does not show required indicator by default', () => {
      render(
        <FormSection title="Default Section">
          <div>Content</div>
        </FormSection>
      )

      expect(document.querySelector('.form-section-required')).not.toBeInTheDocument()
    })
  })

  describe('collapsible behavior', () => {
    it('is expanded by default when collapsible', () => {
      render(
        <FormSection title="Collapsible Section" collapsible>
          <div data-testid="content">Content</div>
        </FormSection>
      )

      expect(screen.getByTestId('content')).toBeVisible()
    })

    it('starts collapsed when defaultExpanded is false', () => {
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible defaultExpanded={false}>
          <div data-testid="content">Content</div>
        </FormSection>
      )

      expect(container.querySelector('.form-section--collapsed')).toBeInTheDocument()
    })

    it('respects defaultCollapsed prop (backward compatibility)', () => {
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible defaultCollapsed={true}>
          <div data-testid="content">Content</div>
        </FormSection>
      )

      expect(container.querySelector('.form-section--collapsed')).toBeInTheDocument()
    })

    it('defaultCollapsed takes priority over defaultExpanded', () => {
      const { container } = render(
        <FormSection
          title="Collapsible Section"
          collapsible
          defaultExpanded={true}
          defaultCollapsed={true}
        >
          <div data-testid="content">Content</div>
        </FormSection>
      )

      // defaultCollapsed=true should override defaultExpanded=true
      expect(container.querySelector('.form-section--collapsed')).toBeInTheDocument()
    })

    it('toggles expanded state on click', async () => {
      const user = userEvent.setup()
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div data-testid="content">Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')
      expect(header).toBeInTheDocument()

      // Initially expanded
      expect(container.querySelector('.form-section--collapsed')).not.toBeInTheDocument()

      // Click to collapse
      await user.click(header!)
      expect(container.querySelector('.form-section--collapsed')).toBeInTheDocument()

      // Click to expand again
      await user.click(header!)
      expect(container.querySelector('.form-section--collapsed')).not.toBeInTheDocument()
    })

    it('toggles on Enter key press', async () => {
      const user = userEvent.setup()
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div data-testid="content">Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')

      // Focus and press Enter
      header?.focus()
      await user.keyboard('{Enter}')

      expect(container.querySelector('.form-section--collapsed')).toBeInTheDocument()

      // Press Enter again
      await user.keyboard('{Enter}')
      expect(container.querySelector('.form-section--collapsed')).not.toBeInTheDocument()
    })

    it('toggles on Space key press', async () => {
      const user = userEvent.setup()
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div data-testid="content">Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')

      // Focus and press Space
      header?.focus()
      await user.keyboard(' ')

      expect(container.querySelector('.form-section--collapsed')).toBeInTheDocument()
    })

    it('does not toggle when collapsible is false', async () => {
      const { container } = render(
        <FormSection title="Non-Collapsible Section">
          <div data-testid="content">Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header')
      expect(header).not.toHaveClass('form-section-header--collapsible')

      // Click should not do anything (no toggle)
      fireEvent.click(header!)

      // No collapsed class should appear
      expect(container.querySelector('.form-section--collapsed')).not.toBeInTheDocument()
    })

    it('shows chevron icon when collapsible', () => {
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div>Content</div>
        </FormSection>
      )

      const chevron = container.querySelector('.form-section-chevron')
      expect(chevron).toBeInTheDocument()
    })

    it('does not show chevron icon when not collapsible', () => {
      const { container } = render(
        <FormSection title="Non-Collapsible Section">
          <div>Content</div>
        </FormSection>
      )

      const chevron = container.querySelector('.form-section-chevron')
      expect(chevron).not.toBeInTheDocument()
    })
  })

  describe('accessibility', () => {
    it('has role="group" attribute', () => {
      render(
        <FormSection title="Section">
          <div>Content</div>
        </FormSection>
      )

      expect(screen.getByRole('group')).toBeInTheDocument()
    })

    it('has aria-labelledby when title is present', () => {
      render(
        <FormSection title="Accessible Section" data-testid="section">
          <div>Content</div>
        </FormSection>
      )

      const section = screen.getByTestId('section')
      expect(section).toHaveAttribute('aria-labelledby')
    })

    it('does not have aria-labelledby when no title', () => {
      render(
        <FormSection data-testid="section">
          <div>Content</div>
        </FormSection>
      )

      const section = screen.getByTestId('section')
      expect(section).not.toHaveAttribute('aria-labelledby')
    })

    it('collapsible header has role="button"', () => {
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div>Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')
      expect(header).toHaveAttribute('role', 'button')
    })

    it('collapsible header has tabIndex="0"', () => {
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div>Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')
      expect(header).toHaveAttribute('tabIndex', '0')
    })

    it('collapsible header has aria-expanded attribute', () => {
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div>Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')
      expect(header).toHaveAttribute('aria-expanded', 'true')
    })

    it('aria-expanded updates when toggled', async () => {
      const user = userEvent.setup()
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div>Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')
      expect(header).toHaveAttribute('aria-expanded', 'true')

      await user.click(header!)
      expect(header).toHaveAttribute('aria-expanded', 'false')
    })

    it('collapsible header has aria-controls attribute', () => {
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div>Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')
      expect(header).toHaveAttribute('aria-controls')
    })

    it('content has aria-hidden when collapsed', async () => {
      const user = userEvent.setup()
      const { container } = render(
        <FormSection title="Collapsible Section" collapsible>
          <div>Content</div>
        </FormSection>
      )

      const header = container.querySelector('.form-section-header--collapsible')

      // Collapse
      await user.click(header!)

      const content = container.querySelector('.form-section-content')
      expect(content).toHaveAttribute('aria-hidden', 'true')
    })
  })

  describe('CSS classes', () => {
    it('has form-section base class', () => {
      const { container } = render(
        <FormSection>
          <div>Content</div>
        </FormSection>
      )

      expect(container.querySelector('.form-section')).toBeInTheDocument()
    })

    it('has form-section--collapsible class when collapsible', () => {
      const { container } = render(
        <FormSection title="Collapsible" collapsible>
          <div>Content</div>
        </FormSection>
      )

      expect(container.querySelector('.form-section--collapsible')).toBeInTheDocument()
    })

    it('has form-section--collapsed class when collapsed', () => {
      const { container } = render(
        <FormSection title="Collapsed" collapsible defaultExpanded={false}>
          <div>Content</div>
        </FormSection>
      )

      expect(container.querySelector('.form-section--collapsed')).toBeInTheDocument()
    })

    it('header has form-section-header--collapsible when collapsible', () => {
      const { container } = render(
        <FormSection title="Collapsible" collapsible>
          <div>Content</div>
        </FormSection>
      )

      expect(container.querySelector('.form-section-header--collapsible')).toBeInTheDocument()
    })
  })

  describe('combined props', () => {
    it('renders with all props combined', () => {
      const { container } = render(
        <FormSection
          title="Complete Section"
          subtitle="With all features"
          icon={IconUser}
          required
          collapsible
          defaultExpanded
          className="custom-class"
          data-testid="complete-section"
        >
          <div data-testid="child">Form fields</div>
        </FormSection>
      )

      // Title
      expect(screen.getByText('Complete Section')).toBeInTheDocument()

      // Subtitle
      expect(screen.getByText('With all features')).toBeInTheDocument()

      // Icon
      expect(container.querySelector('.form-section-icon')).toBeInTheDocument()

      // Required
      expect(container.querySelector('.form-section-required')).toBeInTheDocument()

      // Collapsible
      expect(container.querySelector('.form-section--collapsible')).toBeInTheDocument()

      // Custom class
      expect(container.querySelector('.form-section')).toHaveClass('custom-class')

      // Test ID
      expect(screen.getByTestId('complete-section')).toBeInTheDocument()

      // Children
      expect(screen.getByTestId('child')).toBeInTheDocument()
    })
  })
})
