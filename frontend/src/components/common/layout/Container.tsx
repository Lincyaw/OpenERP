import type { CSSProperties, ReactNode } from 'react'
import { useResponsive } from '@/hooks/useResponsive'

/**
 * Container Component
 *
 * Responsive container that centers content and provides
 * consistent max-width based on breakpoints.
 *
 * @example
 * <Container>
 *   <h1>Page Content</h1>
 * </Container>
 *
 * @example
 * <Container size="wide" padding="lg">
 *   <Dashboard />
 * </Container>
 */

export interface ContainerProps {
  /** Container max-width size */
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  /** Horizontal padding size */
  padding?: 'none' | 'sm' | 'md' | 'lg'
  /** Center the container */
  center?: boolean
  /** Additional class name */
  className?: string
  /** Inline styles */
  style?: CSSProperties
  /** Children */
  children: ReactNode
}

const sizeMap: Record<NonNullable<ContainerProps['size']>, string> = {
  sm: '640px',
  md: '768px',
  lg: '1024px',
  xl: '1280px',
  full: '100%',
}

// Desktop padding values
const paddingMap: Record<NonNullable<ContainerProps['padding']>, string> = {
  none: '0',
  sm: 'var(--content-padding-mobile)',
  md: 'var(--content-padding-tablet)',
  lg: 'var(--content-padding-desktop)',
}

// Mobile padding values (smaller for better screen utilization)
const mobilePaddingMap: Record<NonNullable<ContainerProps['padding']>, string> = {
  none: '0',
  sm: 'var(--content-padding-mobile)',
  md: 'var(--content-padding-mobile)',
  lg: 'var(--content-padding-mobile)',
}

export function Container({
  size = 'xl',
  padding = 'md',
  center = true,
  className = '',
  style,
  children,
}: ContainerProps) {
  const { isMobile } = useResponsive()
  const activePaddingMap = isMobile ? mobilePaddingMap : paddingMap

  const containerStyle: CSSProperties = {
    width: '100%',
    maxWidth: sizeMap[size],
    paddingLeft: activePaddingMap[padding],
    paddingRight: activePaddingMap[padding],
    marginLeft: center ? 'auto' : undefined,
    marginRight: center ? 'auto' : undefined,
    ...style,
  }

  return (
    <div className={`ds-container ${className}`.trim()} style={containerStyle}>
      {children}
    </div>
  )
}
