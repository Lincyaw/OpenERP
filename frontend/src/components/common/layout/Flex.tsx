import type { CSSProperties, ReactNode } from 'react'

/**
 * Flex Component
 *
 * Flexbox-based layout component for flexible alignment
 * and distribution of space.
 *
 * @example
 * // Horizontal row with space between
 * <Flex justify="space-between" align="center">
 *   <Logo />
 *   <Navigation />
 * </Flex>
 *
 * @example
 * // Vertical stack with gap
 * <Flex direction="column" gap="md">
 *   <FormField />
 *   <FormField />
 *   <Button>Submit</Button>
 * </Flex>
 */

export interface FlexProps {
  /** Flex direction */
  direction?: 'row' | 'row-reverse' | 'column' | 'column-reverse'
  /** Wrap behavior */
  wrap?: 'nowrap' | 'wrap' | 'wrap-reverse'
  /** Main axis alignment */
  justify?: 'start' | 'center' | 'end' | 'space-between' | 'space-around' | 'space-evenly'
  /** Cross axis alignment */
  align?: 'start' | 'center' | 'end' | 'stretch' | 'baseline'
  /** Multi-line alignment */
  alignContent?: 'start' | 'center' | 'end' | 'stretch' | 'space-between' | 'space-around'
  /** Gap between items */
  gap?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  /** Row gap (for wrapped content) */
  rowGap?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  /** Column gap (for wrapped content) */
  colGap?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  /** Whether to render as inline-flex */
  inline?: boolean
  /** Additional class name */
  className?: string
  /** Inline styles */
  style?: CSSProperties
  /** Children */
  children: ReactNode
}

const gapMap: Record<NonNullable<FlexProps['gap']>, string> = {
  none: '0',
  xs: 'var(--spacing-1)',
  sm: 'var(--spacing-2)',
  md: 'var(--spacing-4)',
  lg: 'var(--spacing-6)',
  xl: 'var(--spacing-8)',
}

const justifyMap: Record<NonNullable<FlexProps['justify']>, string> = {
  start: 'flex-start',
  center: 'center',
  end: 'flex-end',
  'space-between': 'space-between',
  'space-around': 'space-around',
  'space-evenly': 'space-evenly',
}

const alignMap: Record<NonNullable<FlexProps['align']>, string> = {
  start: 'flex-start',
  center: 'center',
  end: 'flex-end',
  stretch: 'stretch',
  baseline: 'baseline',
}

export function Flex({
  direction = 'row',
  wrap = 'nowrap',
  justify = 'start',
  align = 'stretch',
  alignContent,
  gap = 'none',
  rowGap,
  colGap,
  inline = false,
  className = '',
  style,
  children,
}: FlexProps) {
  const flexStyle: CSSProperties = {
    display: inline ? 'inline-flex' : 'flex',
    flexDirection: direction,
    flexWrap: wrap,
    justifyContent: justifyMap[justify],
    alignItems: alignMap[align],
    alignContent: alignContent ? alignMap[alignContent as keyof typeof alignMap] : undefined,
    gap: gapMap[gap],
    rowGap: rowGap ? gapMap[rowGap] : undefined,
    columnGap: colGap ? gapMap[colGap] : undefined,
    ...style,
  }

  return (
    <div className={`ds-flex ${className}`.trim()} style={flexStyle}>
      {children}
    </div>
  )
}

/**
 * Stack Component
 *
 * Vertical flex layout (shorthand for Flex direction="column")
 *
 * @example
 * <Stack gap="md">
 *   <Card>Card 1</Card>
 *   <Card>Card 2</Card>
 * </Stack>
 */
export type StackProps = Omit<FlexProps, 'direction'>

export function Stack(props: StackProps) {
  return <Flex {...props} direction="column" />
}

/**
 * Row Component
 *
 * Horizontal flex layout (shorthand for Flex direction="row")
 *
 * @example
 * <Row gap="sm" align="center">
 *   <Avatar />
 *   <Text>Username</Text>
 * </Row>
 */
export type RowProps = Omit<FlexProps, 'direction'>

export function Row(props: RowProps) {
  return <Flex {...props} direction="row" />
}

/**
 * Spacer Component
 *
 * Flexible spacer that takes remaining space in a flex container.
 *
 * @example
 * <Row>
 *   <Logo />
 *   <Spacer />
 *   <Button>Login</Button>
 * </Row>
 */
export function Spacer() {
  return <div className="ds-spacer" style={{ flex: '1 1 auto' }} />
}

/**
 * Divider Component
 *
 * Visual separator between flex items.
 *
 * @example
 * <Row gap="md">
 *   <Text>Item 1</Text>
 *   <Divider vertical />
 *   <Text>Item 2</Text>
 * </Row>
 */
export interface DividerProps {
  /** Vertical divider (for use in Row) */
  vertical?: boolean
  /** Additional class name */
  className?: string
  /** Inline styles */
  style?: CSSProperties
}

export function Divider({ vertical = false, className = '', style }: DividerProps) {
  const dividerStyle: CSSProperties = vertical
    ? {
        width: '1px',
        alignSelf: 'stretch',
        backgroundColor: 'var(--color-border-secondary)',
        ...style,
      }
    : {
        height: '1px',
        width: '100%',
        backgroundColor: 'var(--color-border-secondary)',
        ...style,
      }

  return (
    <div
      className={`ds-divider ${vertical ? 'ds-divider--vertical' : ''} ${className}`.trim()}
      style={dividerStyle}
      role="separator"
      aria-orientation={vertical ? 'vertical' : 'horizontal'}
    />
  )
}
