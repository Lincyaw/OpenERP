import type { CSSProperties, ReactNode, ElementType } from 'react'
import { Spin } from '@douyinfe/semi-ui-19'
import './PageSummary.css'

/**
 * Gap size options for PageSummary grid
 */
export type PageSummaryGap = 'sm' | 'md' | 'lg'

/**
 * Title heading level options
 */
export type PageSummaryTitleAs = 'h2' | 'h3' | 'h4' | 'h5' | 'div'

/**
 * PageSummary component props
 */
export interface PageSummaryProps {
  /** Child KPICard components */
  children: ReactNode
  /** Whether the summary is in loading state */
  loading?: boolean
  /** Gap between cards */
  gap?: PageSummaryGap
  /** Optional title for the summary section */
  title?: string
  /** Heading level for the title (defaults to h3 for semantic hierarchy) */
  titleAs?: PageSummaryTitleAs
  /** Optional className for custom styling */
  className?: string
  /** Optional inline styles */
  style?: CSSProperties
}

/**
 * PageSummary - A container component for KPICard lists
 *
 * Provides responsive grid layout for displaying multiple KPI cards.
 * Automatically adjusts column count based on viewport width:
 * - Mobile (< 768px): 2 columns
 * - Tablet (768px - 1023px): 3 columns
 * - Desktop (>= 1024px): 4 columns
 *
 * Features:
 * - Responsive grid with CSS Grid
 * - Loading state with skeleton/spinner
 * - Configurable gap between cards
 * - Optional section title
 * - Full-width container
 *
 * @example
 * // Basic usage with KPICards
 * <PageSummary>
 *   <KPICard label="Total Orders" value={128} variant="primary" />
 *   <KPICard label="Pending" value={15} variant="warning" />
 *   <KPICard label="Completed" value={100} variant="success" />
 *   <KPICard label="Overdue" value={3} variant="danger" />
 * </PageSummary>
 *
 * @example
 * // With loading state and title
 * <PageSummary loading={summaryLoading} title="Sales Overview">
 *   <KPICard label="Revenue" value="¥125,000" />
 *   <KPICard label="Orders" value={45} />
 * </PageSummary>
 *
 * @example
 * // With custom gap
 * <PageSummary gap="lg">
 *   <KPICard label="Item 1" value={100} />
 *   <KPICard label="Item 2" value={200} />
 * </PageSummary>
 *
 * @example
 * // With custom heading level for proper document hierarchy
 * <PageSummary title="Summary" titleAs="h2">
 *   <KPICard label="Total" value={500} />
 * </PageSummary>
 */
export function PageSummary({
  children,
  loading = false,
  gap = 'md',
  title,
  titleAs = 'h3',
  className = '',
  style,
}: PageSummaryProps) {
  const TitleTag = titleAs as ElementType

  return (
    <div className={`page-summary page-summary--gap-${gap} ${className}`} style={style}>
      {title && <TitleTag className="page-summary__title">{title}</TitleTag>}
      <Spin spinning={loading}>
        <div className="page-summary__grid">{children}</div>
      </Spin>
    </div>
  )
}

export default PageSummary
