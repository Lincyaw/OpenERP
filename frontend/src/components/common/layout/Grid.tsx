import type { CSSProperties, ReactNode } from 'react'

/**
 * Grid Component
 *
 * CSS Grid-based layout component for creating
 * responsive multi-column layouts.
 *
 * @example
 * // Equal 3-column grid
 * <Grid cols={3} gap="md">
 *   <Card>Item 1</Card>
 *   <Card>Item 2</Card>
 *   <Card>Item 3</Card>
 * </Grid>
 *
 * @example
 * // Responsive grid: 1 col mobile, 2 col tablet, 4 col desktop
 * <Grid cols={{ mobile: 1, tablet: 2, desktop: 4 }} gap="lg">
 *   {items.map(item => <Card key={item.id}>{item.name}</Card>)}
 * </Grid>
 */

export interface ResponsiveValue<T> {
  mobile?: T
  tablet?: T
  desktop?: T
  wide?: T
}

export interface GridProps {
  /** Number of columns (number or responsive object) */
  cols?: number | ResponsiveValue<number>
  /** Gap between grid items */
  gap?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  /** Row gap (if different from column gap) */
  rowGap?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  /** Column gap (if different from row gap) */
  colGap?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  /** Alignment of grid items along the row axis */
  justify?: 'start' | 'center' | 'end' | 'stretch' | 'space-between' | 'space-around'
  /** Alignment of grid items along the column axis */
  align?: 'start' | 'center' | 'end' | 'stretch' | 'baseline'
  /** Flow direction */
  flow?: 'row' | 'column' | 'row-dense' | 'column-dense'
  /** Custom grid template columns */
  templateCols?: string
  /** Additional class name */
  className?: string
  /** Inline styles */
  style?: CSSProperties
  /** Children */
  children: ReactNode
}

const gapMap: Record<NonNullable<GridProps['gap']>, string> = {
  none: '0',
  xs: 'var(--spacing-1)',
  sm: 'var(--spacing-2)',
  md: 'var(--spacing-4)',
  lg: 'var(--spacing-6)',
  xl: 'var(--spacing-8)',
}

function getGridTemplateColumns(cols: GridProps['cols']): string {
  if (typeof cols === 'number') {
    return `repeat(${cols}, minmax(0, 1fr))`
  }
  return 'repeat(1, minmax(0, 1fr))'
}

export function Grid({
  cols = 1,
  gap = 'md',
  rowGap,
  colGap,
  justify,
  align,
  flow,
  templateCols,
  className = '',
  style,
  children,
}: GridProps) {
  const isResponsive = typeof cols === 'object'

  const gridStyle: CSSProperties = {
    display: 'grid',
    gridTemplateColumns: templateCols || getGridTemplateColumns(cols),
    gap: gapMap[gap],
    rowGap: rowGap ? gapMap[rowGap] : undefined,
    columnGap: colGap ? gapMap[colGap] : undefined,
    justifyContent: justify,
    alignItems: align,
    gridAutoFlow: flow,
    ...style,
  }

  // Generate responsive class name
  let responsiveClass = ''
  if (isResponsive) {
    const { mobile = 1, tablet, desktop, wide } = cols as ResponsiveValue<number>
    responsiveClass = `ds-grid-cols-${mobile}`
    if (tablet) responsiveClass += ` ds-grid-cols-tablet-${tablet}`
    if (desktop) responsiveClass += ` ds-grid-cols-desktop-${desktop}`
    if (wide) responsiveClass += ` ds-grid-cols-wide-${wide}`
  }

  return (
    <div className={`ds-grid ${responsiveClass} ${className}`.trim()} style={gridStyle}>
      {children}
    </div>
  )
}

/**
 * GridItem Component
 *
 * Individual item within a Grid for controlling span and placement.
 *
 * @example
 * <Grid cols={12}>
 *   <GridItem span={8}>Main Content</GridItem>
 *   <GridItem span={4}>Sidebar</GridItem>
 * </Grid>
 */
export interface GridItemProps {
  /** Number of columns to span */
  span?: number | ResponsiveValue<number>
  /** Column start position */
  colStart?: number
  /** Column end position */
  colEnd?: number
  /** Row start position */
  rowStart?: number
  /** Row end position */
  rowEnd?: number
  /** Number of rows to span */
  rowSpan?: number
  /** Additional class name */
  className?: string
  /** Inline styles */
  style?: CSSProperties
  /** Children */
  children: ReactNode
}

export function GridItem({
  span,
  colStart,
  colEnd,
  rowStart,
  rowEnd,
  rowSpan,
  className = '',
  style,
  children,
}: GridItemProps) {
  const itemStyle: CSSProperties = {
    gridColumn: typeof span === 'number' ? `span ${span}` : undefined,
    gridColumnStart: colStart,
    gridColumnEnd: colEnd,
    gridRowStart: rowStart,
    gridRowEnd: rowEnd,
    gridRow: rowSpan ? `span ${rowSpan}` : undefined,
    ...style,
  }

  return (
    <div className={`ds-grid-item ${className}`.trim()} style={itemStyle}>
      {children}
    </div>
  )
}
