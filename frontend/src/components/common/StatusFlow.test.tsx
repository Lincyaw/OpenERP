import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import {
  StatusFlow,
  generateOrderFlowSteps,
  ORDER_FLOW_PRESETS,
  type StatusFlowStep,
} from './StatusFlow'

describe('StatusFlow Component', () => {
  describe('rendering', () => {
    it('renders nothing when steps array is empty', () => {
      const { container } = render(<StatusFlow steps={[]} />)
      expect(container.firstChild).toBeNull()
    })

    it('renders nothing when steps is undefined', () => {
      // @ts-expect-error testing undefined input
      const { container } = render(<StatusFlow steps={undefined} />)
      expect(container.firstChild).toBeNull()
    })

    it('renders all provided steps', () => {
      const steps: StatusFlowStep[] = [
        { key: 'step1', label: 'Step 1', state: 'completed' },
        { key: 'step2', label: 'Step 2', state: 'current' },
        { key: 'step3', label: 'Step 3', state: 'pending' },
      ]
      render(<StatusFlow steps={steps} />)

      expect(screen.getByText('Step 1')).toBeInTheDocument()
      expect(screen.getByText('Step 2')).toBeInTheDocument()
      expect(screen.getByText('Step 3')).toBeInTheDocument()
    })

    it('renders timestamps when showTimestamp is true', () => {
      const steps: StatusFlowStep[] = [
        {
          key: 'step1',
          label: 'Step 1',
          state: 'completed',
          timestamp: '2024-01-01 10:00',
        },
      ]
      render(<StatusFlow steps={steps} showTimestamp />)

      expect(screen.getByText('2024-01-01 10:00')).toBeInTheDocument()
    })

    it('does not render timestamps when showTimestamp is false', () => {
      const steps: StatusFlowStep[] = [
        {
          key: 'step1',
          label: 'Step 1',
          state: 'completed',
          timestamp: '2024-01-01 10:00',
        },
      ]
      render(<StatusFlow steps={steps} showTimestamp={false} />)

      expect(screen.queryByText('2024-01-01 10:00')).not.toBeInTheDocument()
    })

    it('renders description when provided', () => {
      const steps: StatusFlowStep[] = [
        {
          key: 'step1',
          label: 'Step 1',
          state: 'error',
          description: 'Payment failed',
        },
      ]
      render(<StatusFlow steps={steps} />)

      expect(screen.getByText('Payment failed')).toBeInTheDocument()
    })
  })

  describe('states', () => {
    it('renders completed state with check icon', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'completed' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const stepElement = container.querySelector('.status-flow__step--completed')
      expect(stepElement).toBeInTheDocument()
    })

    it('renders current state with highlight', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'current' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const stepElement = container.querySelector('.status-flow__step--current')
      expect(stepElement).toBeInTheDocument()
      expect(stepElement).toHaveAttribute('aria-current', 'step')
    })

    it('renders pending state', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const stepElement = container.querySelector('.status-flow__step--pending')
      expect(stepElement).toBeInTheDocument()
    })

    it('renders error state', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Error Step', state: 'error' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const stepElement = container.querySelector('.status-flow__step--error')
      expect(stepElement).toBeInTheDocument()
      expect(screen.getByText('Error Step')).toBeInTheDocument()
    })

    it('renders cancelled state with strikethrough label', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'cancelled' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const stepElement = container.querySelector('.status-flow__step--cancelled')
      expect(stepElement).toBeInTheDocument()
    })

    it('renders rejected state', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'rejected' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const stepElement = container.querySelector('.status-flow__step--rejected')
      expect(stepElement).toBeInTheDocument()
    })
  })

  describe('direction prop', () => {
    it('defaults to horizontal direction', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).not.toHaveClass('status-flow--vertical')
    })

    it('applies vertical class when direction is vertical', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} direction="vertical" />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).toHaveClass('status-flow--vertical')
    })

    it('does not apply vertical class when direction is horizontal', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} direction="horizontal" />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).not.toHaveClass('status-flow--vertical')
    })
  })

  describe('size prop', () => {
    it('defaults to default size', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).not.toHaveClass('status-flow--small')
    })

    it('applies small class when size is small', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} size="small" />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).toHaveClass('status-flow--small')
    })
  })

  describe('accessibility', () => {
    it('has correct role="list" attribute', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).toHaveAttribute('role', 'list')
    })

    it('has correct aria-label', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} ariaLabel="Order status" />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).toHaveAttribute('aria-label', 'Order status')
    })

    it('has role="listitem" on each step', () => {
      const steps: StatusFlowStep[] = [
        { key: 'step1', label: 'Step 1', state: 'completed' },
        { key: 'step2', label: 'Step 2', state: 'pending' },
      ]
      const { container } = render(<StatusFlow steps={steps} />)

      const listItems = container.querySelectorAll('[role="listitem"]')
      expect(listItems.length).toBe(2)
    })

    it('marks current step with aria-current="step"', () => {
      const steps: StatusFlowStep[] = [
        { key: 'step1', label: 'Step 1', state: 'completed' },
        { key: 'step2', label: 'Step 2', state: 'current' },
      ]
      const { container } = render(<StatusFlow steps={steps} />)

      const currentStep = container.querySelector('[aria-current="step"]')
      expect(currentStep).toBeInTheDocument()
      expect(currentStep?.textContent).toContain('Step 2')
    })

    it('connector lines have aria-hidden="true"', () => {
      const steps: StatusFlowStep[] = [
        { key: 'step1', label: 'Step 1', state: 'completed' },
        { key: 'step2', label: 'Step 2', state: 'pending' },
      ]
      const { container } = render(<StatusFlow steps={steps} />)

      const connectors = container.querySelectorAll('.status-flow__connector')
      connectors.forEach((connector) => {
        expect(connector).toHaveAttribute('aria-hidden', 'true')
      })
    })
  })

  describe('connectors', () => {
    it('does not render connector after last step', () => {
      const steps: StatusFlowStep[] = [
        { key: 'step1', label: 'Step 1', state: 'completed' },
        { key: 'step2', label: 'Step 2', state: 'pending' },
      ]
      const { container } = render(<StatusFlow steps={steps} />)

      // Should only have 1 connector for 2 steps
      const connectors = container.querySelectorAll('.status-flow__connector')
      expect(connectors.length).toBe(1)
    })

    it('renders no connector for single step', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'completed' }]
      const { container } = render(<StatusFlow steps={steps} />)

      const connectors = container.querySelectorAll('.status-flow__connector')
      expect(connectors.length).toBe(0)
    })
  })

  describe('custom icon', () => {
    it('renders custom icon when provided', () => {
      const steps: StatusFlowStep[] = [
        {
          key: 'step1',
          label: 'Step 1',
          state: 'pending',
          icon: <span data-testid="custom-icon">★</span>,
        },
      ]
      render(<StatusFlow steps={steps} />)

      expect(screen.getByTestId('custom-icon')).toBeInTheDocument()
    })
  })

  describe('className and style props', () => {
    it('applies custom className', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} className="custom-class" />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).toHaveClass('custom-class')
    })

    it('applies custom style', () => {
      const steps: StatusFlowStep[] = [{ key: 'step1', label: 'Step 1', state: 'pending' }]
      const { container } = render(<StatusFlow steps={steps} style={{ margin: '10px' }} />)

      const flowElement = container.querySelector('.status-flow')
      expect(flowElement).toHaveStyle({ margin: '10px' })
    })
  })
})

describe('generateOrderFlowSteps', () => {
  describe('sales_order flow', () => {
    it('marks steps before current as completed', () => {
      const steps = generateOrderFlowSteps('sales_order', 'shipped')

      expect(steps[0].state).toBe('completed') // draft
      expect(steps[1].state).toBe('completed') // confirmed
      expect(steps[2].state).toBe('current') // shipped
      expect(steps[3].state).toBe('pending') // completed
    })

    it('marks first step as current when at draft', () => {
      const steps = generateOrderFlowSteps('sales_order', 'draft')

      expect(steps[0].state).toBe('current') // draft
      expect(steps[1].state).toBe('pending') // confirmed
      expect(steps[2].state).toBe('pending') // shipped
      expect(steps[3].state).toBe('pending') // completed
    })

    it('marks all as completed when at final step', () => {
      const steps = generateOrderFlowSteps('sales_order', 'completed')

      expect(steps[0].state).toBe('completed') // draft
      expect(steps[1].state).toBe('completed') // confirmed
      expect(steps[2].state).toBe('completed') // shipped
      expect(steps[3].state).toBe('current') // completed
    })
  })

  describe('purchase_order flow', () => {
    it('generates correct steps for purchase order', () => {
      const steps = generateOrderFlowSteps('purchase_order', 'received')

      expect(steps.length).toBe(4)
      expect(steps[0].key).toBe('draft')
      expect(steps[1].key).toBe('confirmed')
      expect(steps[2].key).toBe('received')
      expect(steps[3].key).toBe('completed')

      expect(steps[0].state).toBe('completed')
      expect(steps[1].state).toBe('completed')
      expect(steps[2].state).toBe('current')
      expect(steps[3].state).toBe('pending')
    })
  })

  describe('return_order flow', () => {
    it('generates correct steps for return order', () => {
      const steps = generateOrderFlowSteps('return_order', 'approved')

      expect(steps.length).toBe(4)
      expect(steps[0].key).toBe('pending_review')
      expect(steps[1].key).toBe('approved')
      expect(steps[2].key).toBe('stored')
      expect(steps[3].key).toBe('completed')

      expect(steps[0].state).toBe('completed')
      expect(steps[1].state).toBe('current')
      expect(steps[2].state).toBe('pending')
      expect(steps[3].state).toBe('pending')
    })
  })

  describe('edge cases', () => {
    it('marks all as pending when currentStatusKey is not found', () => {
      const steps = generateOrderFlowSteps('sales_order', 'invalid_key')

      expect(steps.every((s) => s.state === 'pending')).toBe(true)
    })

    it('returns empty array for invalid flowType', () => {
      // @ts-expect-error testing invalid input
      const steps = generateOrderFlowSteps('invalid_type', 'draft')
      expect(steps).toEqual([])
    })

    it('includes timestamps when provided', () => {
      const timestamps = {
        draft: '2024-01-01 10:00',
        confirmed: '2024-01-02 14:30',
      }
      const steps = generateOrderFlowSteps('sales_order', 'confirmed', timestamps)

      expect(steps[0].timestamp).toBe('2024-01-01 10:00')
      expect(steps[1].timestamp).toBe('2024-01-02 14:30')
      expect(steps[2].timestamp).toBeUndefined()
      expect(steps[3].timestamp).toBeUndefined()
    })
  })
})

describe('ORDER_FLOW_PRESETS', () => {
  it('has all required flow types', () => {
    expect(ORDER_FLOW_PRESETS).toHaveProperty('sales_order')
    expect(ORDER_FLOW_PRESETS).toHaveProperty('purchase_order')
    expect(ORDER_FLOW_PRESETS).toHaveProperty('return_order')
  })

  it('sales_order has correct steps', () => {
    const preset = ORDER_FLOW_PRESETS.sales_order
    expect(preset.name).toBe('销售订单流程')
    expect(preset.steps.length).toBe(4)
    expect(preset.steps.map((s) => s.key)).toEqual(['draft', 'confirmed', 'shipped', 'completed'])
  })

  it('purchase_order has correct steps', () => {
    const preset = ORDER_FLOW_PRESETS.purchase_order
    expect(preset.name).toBe('采购订单流程')
    expect(preset.steps.length).toBe(4)
    expect(preset.steps.map((s) => s.key)).toEqual(['draft', 'confirmed', 'received', 'completed'])
  })

  it('return_order has correct steps', () => {
    const preset = ORDER_FLOW_PRESETS.return_order
    expect(preset.name).toBe('退货单流程')
    expect(preset.steps.length).toBe(4)
    expect(preset.steps.map((s) => s.key)).toEqual([
      'pending_review',
      'approved',
      'stored',
      'completed',
    ])
  })
})
