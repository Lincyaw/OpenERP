/**
 * Feature Store
 *
 * Manages SaaS plan-based feature permissions for the current tenant.
 * Features are derived from the tenant's subscription plan (free, basic, pro, enterprise).
 *
 * This is different from feature flags (featureFlagStore) which are for A/B testing
 * and gradual rollouts. This store handles subscription-based feature gating.
 *
 * @module store/featureStore
 */

import { create } from 'zustand'
import { devtools, persist, createJSONStorage } from 'zustand/middleware'

// ============================================================================
// Types
// ============================================================================

/**
 * Subscription plan types matching backend identity.TenantPlan
 */
export type TenantPlan = 'free' | 'basic' | 'pro' | 'enterprise'

/**
 * Feature keys matching backend identity.FeatureKey
 * These are the SaaS features that can be enabled/disabled per plan
 */
export type FeatureKey =
  // Core features
  | 'multi_warehouse'
  | 'batch_management'
  | 'serial_tracking'
  | 'multi_currency'
  | 'advanced_reporting'
  | 'api_access'
  | 'custom_fields'
  | 'audit_log'
  | 'data_export'
  | 'data_import'
  // Trade features
  | 'sales_orders'
  | 'purchase_orders'
  | 'sales_returns'
  | 'purchase_returns'
  | 'quotations'
  | 'price_management'
  | 'discount_rules'
  | 'credit_management'
  // Finance features
  | 'receivables'
  | 'payables'
  | 'reconciliation'
  | 'expense_tracking'
  | 'financial_reports'
  // Advanced features
  | 'workflow_approval'
  | 'notifications'
  | 'integrations'
  | 'white_labeling'
  | 'priority_support'
  | 'dedicated_support'
  | 'sla'

/**
 * Feature definition with metadata
 */
export interface FeatureDefinition {
  key: FeatureKey
  enabled: boolean
  limit?: number | null
  description: string
  /** Which plan is required to unlock this feature */
  requiredPlan: TenantPlan
}

/**
 * Feature store state
 */
export interface FeatureState {
  /** Current tenant's subscription plan */
  plan: TenantPlan
  /** Map of feature keys to their definitions */
  features: Record<FeatureKey, FeatureDefinition>
  /** Whether the store has been initialized */
  isReady: boolean
  /** Tenant ID for cache invalidation */
  tenantId: string | null
}

/**
 * Feature store actions
 */
export interface FeatureActions {
  /** Initialize features for a tenant plan */
  initialize: (plan: TenantPlan, tenantId: string) => void
  /** Update the tenant's plan (e.g., after upgrade) */
  setPlan: (plan: TenantPlan) => void
  /** Check if a feature is enabled for the current plan */
  hasFeature: (key: FeatureKey) => boolean
  /** Get the limit for a feature (null = unlimited, undefined = not found) */
  getFeatureLimit: (key: FeatureKey) => number | null | undefined
  /** Get full feature definition */
  getFeature: (key: FeatureKey) => FeatureDefinition | undefined
  /** Get all enabled features */
  getEnabledFeatures: () => FeatureDefinition[]
  /** Get all disabled features (for upgrade prompts) */
  getDisabledFeatures: () => FeatureDefinition[]
  /** Get the minimum plan required for a feature */
  getRequiredPlan: (key: FeatureKey) => TenantPlan | undefined
  /** Reset store state */
  reset: () => void
}

// ============================================================================
// Constants
// ============================================================================

const STORAGE_KEY = 'erp-features'

/**
 * Plan hierarchy for comparison
 */
const PLAN_HIERARCHY: Record<TenantPlan, number> = {
  free: 0,
  basic: 1,
  pro: 2,
  enterprise: 3,
}

/**
 * Feature definitions per plan
 * Matches backend identity.DefaultPlanFeatures
 */
const PLAN_FEATURES: Record<
  TenantPlan,
  Partial<Record<FeatureKey, { enabled: boolean; limit?: number; description: string }>>
> = {
  free: {
    // Core features - limited
    multi_warehouse: { enabled: false, description: 'Multiple warehouse management' },
    batch_management: { enabled: false, description: 'Batch/lot tracking' },
    serial_tracking: { enabled: false, description: 'Serial number tracking' },
    multi_currency: { enabled: false, description: 'Multi-currency support' },
    advanced_reporting: { enabled: false, description: 'Advanced analytics and reports' },
    api_access: { enabled: false, description: 'API access for integrations' },
    custom_fields: { enabled: false, description: 'Custom fields on entities' },
    audit_log: { enabled: false, description: 'Audit log tracking' },
    data_export: { enabled: true, description: 'Export data to CSV/Excel' },
    data_import: {
      enabled: true,
      limit: 100,
      description: 'Import data from CSV (100 rows/import)',
    },
    // Trade features - basic only
    sales_orders: { enabled: true, description: 'Create and manage sales orders' },
    purchase_orders: { enabled: true, description: 'Create and manage purchase orders' },
    sales_returns: { enabled: true, description: 'Process sales returns' },
    purchase_returns: { enabled: true, description: 'Process purchase returns' },
    quotations: { enabled: false, description: 'Create quotations' },
    price_management: { enabled: false, description: 'Advanced price management' },
    discount_rules: { enabled: false, description: 'Discount rules engine' },
    credit_management: { enabled: false, description: 'Customer credit management' },
    // Finance features - basic only
    receivables: { enabled: true, description: 'Accounts receivable tracking' },
    payables: { enabled: true, description: 'Accounts payable tracking' },
    reconciliation: { enabled: false, description: 'Account reconciliation' },
    expense_tracking: { enabled: false, description: 'Expense tracking' },
    financial_reports: { enabled: false, description: 'Financial reports' },
    // Advanced features - none
    workflow_approval: { enabled: false, description: 'Workflow approval system' },
    notifications: { enabled: false, description: 'Email/SMS notifications' },
    integrations: { enabled: false, description: 'Third-party integrations' },
    white_labeling: { enabled: false, description: 'White-label branding' },
    priority_support: { enabled: false, description: 'Priority support' },
    dedicated_support: { enabled: false, description: 'Dedicated support manager' },
    sla: { enabled: false, description: 'Service level agreement' },
  },
  basic: {
    // Core features - some enabled
    multi_warehouse: { enabled: true, description: 'Multiple warehouse management' },
    batch_management: { enabled: true, description: 'Batch/lot tracking' },
    serial_tracking: { enabled: false, description: 'Serial number tracking' },
    multi_currency: { enabled: false, description: 'Multi-currency support' },
    advanced_reporting: { enabled: false, description: 'Advanced analytics and reports' },
    api_access: { enabled: false, description: 'API access for integrations' },
    custom_fields: { enabled: false, description: 'Custom fields on entities' },
    audit_log: { enabled: true, description: 'Audit log tracking' },
    data_export: { enabled: true, description: 'Export data to CSV/Excel' },
    data_import: {
      enabled: true,
      limit: 1000,
      description: 'Import data from CSV (1000 rows/import)',
    },
    // Trade features - most enabled
    sales_orders: { enabled: true, description: 'Create and manage sales orders' },
    purchase_orders: { enabled: true, description: 'Create and manage purchase orders' },
    sales_returns: { enabled: true, description: 'Process sales returns' },
    purchase_returns: { enabled: true, description: 'Process purchase returns' },
    quotations: { enabled: true, description: 'Create quotations' },
    price_management: { enabled: true, description: 'Advanced price management' },
    discount_rules: { enabled: false, description: 'Discount rules engine' },
    credit_management: { enabled: true, description: 'Customer credit management' },
    // Finance features - most enabled
    receivables: { enabled: true, description: 'Accounts receivable tracking' },
    payables: { enabled: true, description: 'Accounts payable tracking' },
    reconciliation: { enabled: true, description: 'Account reconciliation' },
    expense_tracking: { enabled: true, description: 'Expense tracking' },
    financial_reports: { enabled: false, description: 'Financial reports' },
    // Advanced features - limited
    workflow_approval: { enabled: false, description: 'Workflow approval system' },
    notifications: { enabled: true, description: 'Email/SMS notifications' },
    integrations: { enabled: false, description: 'Third-party integrations' },
    white_labeling: { enabled: false, description: 'White-label branding' },
    priority_support: { enabled: false, description: 'Priority support' },
    dedicated_support: { enabled: false, description: 'Dedicated support manager' },
    sla: { enabled: false, description: 'Service level agreement' },
  },
  pro: {
    // Core features - most enabled
    multi_warehouse: { enabled: true, description: 'Multiple warehouse management' },
    batch_management: { enabled: true, description: 'Batch/lot tracking' },
    serial_tracking: { enabled: true, description: 'Serial number tracking' },
    multi_currency: { enabled: true, description: 'Multi-currency support' },
    advanced_reporting: { enabled: true, description: 'Advanced analytics and reports' },
    api_access: { enabled: true, description: 'API access for integrations' },
    custom_fields: { enabled: true, description: 'Custom fields on entities' },
    audit_log: { enabled: true, description: 'Audit log tracking' },
    data_export: { enabled: true, description: 'Export data to CSV/Excel' },
    data_import: {
      enabled: true,
      limit: 10000,
      description: 'Import data from CSV (10000 rows/import)',
    },
    // Trade features - all enabled
    sales_orders: { enabled: true, description: 'Create and manage sales orders' },
    purchase_orders: { enabled: true, description: 'Create and manage purchase orders' },
    sales_returns: { enabled: true, description: 'Process sales returns' },
    purchase_returns: { enabled: true, description: 'Process purchase returns' },
    quotations: { enabled: true, description: 'Create quotations' },
    price_management: { enabled: true, description: 'Advanced price management' },
    discount_rules: { enabled: true, description: 'Discount rules engine' },
    credit_management: { enabled: true, description: 'Customer credit management' },
    // Finance features - all enabled
    receivables: { enabled: true, description: 'Accounts receivable tracking' },
    payables: { enabled: true, description: 'Accounts payable tracking' },
    reconciliation: { enabled: true, description: 'Account reconciliation' },
    expense_tracking: { enabled: true, description: 'Expense tracking' },
    financial_reports: { enabled: true, description: 'Financial reports' },
    // Advanced features - most enabled
    workflow_approval: { enabled: true, description: 'Workflow approval system' },
    notifications: { enabled: true, description: 'Email/SMS notifications' },
    integrations: { enabled: true, description: 'Third-party integrations' },
    white_labeling: { enabled: false, description: 'White-label branding' },
    priority_support: { enabled: true, description: 'Priority support' },
    dedicated_support: { enabled: false, description: 'Dedicated support manager' },
    sla: { enabled: false, description: 'Service level agreement' },
  },
  enterprise: {
    // All features enabled, unlimited
    multi_warehouse: { enabled: true, description: 'Multiple warehouse management' },
    batch_management: { enabled: true, description: 'Batch/lot tracking' },
    serial_tracking: { enabled: true, description: 'Serial number tracking' },
    multi_currency: { enabled: true, description: 'Multi-currency support' },
    advanced_reporting: { enabled: true, description: 'Advanced analytics and reports' },
    api_access: { enabled: true, description: 'API access for integrations' },
    custom_fields: { enabled: true, description: 'Custom fields on entities' },
    audit_log: { enabled: true, description: 'Audit log tracking' },
    data_export: { enabled: true, description: 'Export data to CSV/Excel' },
    data_import: { enabled: true, description: 'Import data from CSV (unlimited)' },
    // Trade features - all enabled
    sales_orders: { enabled: true, description: 'Create and manage sales orders' },
    purchase_orders: { enabled: true, description: 'Create and manage purchase orders' },
    sales_returns: { enabled: true, description: 'Process sales returns' },
    purchase_returns: { enabled: true, description: 'Process purchase returns' },
    quotations: { enabled: true, description: 'Create quotations' },
    price_management: { enabled: true, description: 'Advanced price management' },
    discount_rules: { enabled: true, description: 'Discount rules engine' },
    credit_management: { enabled: true, description: 'Customer credit management' },
    // Finance features - all enabled
    receivables: { enabled: true, description: 'Accounts receivable tracking' },
    payables: { enabled: true, description: 'Accounts payable tracking' },
    reconciliation: { enabled: true, description: 'Account reconciliation' },
    expense_tracking: { enabled: true, description: 'Expense tracking' },
    financial_reports: { enabled: true, description: 'Financial reports' },
    // Advanced features - all enabled
    workflow_approval: { enabled: true, description: 'Workflow approval system' },
    notifications: { enabled: true, description: 'Email/SMS notifications' },
    integrations: { enabled: true, description: 'Third-party integrations' },
    white_labeling: { enabled: true, description: 'White-label branding' },
    priority_support: { enabled: true, description: 'Priority support' },
    dedicated_support: { enabled: true, description: 'Dedicated support manager' },
    sla: { enabled: true, description: 'Service level agreement' },
  },
}

/**
 * Get the minimum plan required for a feature
 */
function getMinimumPlanForFeature(key: FeatureKey): TenantPlan {
  const plans: TenantPlan[] = ['free', 'basic', 'pro', 'enterprise']
  for (const plan of plans) {
    const feature = PLAN_FEATURES[plan][key]
    if (feature?.enabled) {
      return plan
    }
  }
  return 'enterprise' // Default to enterprise if not found
}

/**
 * Build features map for a given plan
 */
function buildFeaturesForPlan(plan: TenantPlan): Record<FeatureKey, FeatureDefinition> {
  const planFeatures = PLAN_FEATURES[plan]
  const features: Partial<Record<FeatureKey, FeatureDefinition>> = {}

  for (const [key, value] of Object.entries(planFeatures)) {
    const featureKey = key as FeatureKey
    features[featureKey] = {
      key: featureKey,
      enabled: value.enabled,
      limit: value.limit ?? null,
      description: value.description,
      requiredPlan: getMinimumPlanForFeature(featureKey),
    }
  }

  return features as Record<FeatureKey, FeatureDefinition>
}

// ============================================================================
// Initial State
// ============================================================================

const initialState: FeatureState = {
  plan: 'free',
  features: buildFeaturesForPlan('free'),
  isReady: false,
  tenantId: null,
}

// ============================================================================
// Store Implementation
// ============================================================================

/**
 * Feature Store
 *
 * Manages SaaS plan-based feature permissions.
 *
 * @example
 * ```tsx
 * import { useFeatureStore } from '@/store'
 *
 * function MyComponent() {
 *   const hasFeature = useFeatureStore((state) => state.hasFeature)
 *
 *   if (hasFeature('advanced_reporting')) {
 *     return <AdvancedReports />
 *   }
 *
 *   return <BasicReports />
 * }
 * ```
 */
export const useFeatureStore = create<FeatureState & FeatureActions>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialState,

        initialize: (plan: TenantPlan, tenantId: string) => {
          const features = buildFeaturesForPlan(plan)
          set(
            {
              plan,
              features,
              isReady: true,
              tenantId,
            },
            false,
            'features/initialize'
          )
        },

        setPlan: (plan: TenantPlan) => {
          const features = buildFeaturesForPlan(plan)
          set(
            {
              plan,
              features,
            },
            false,
            'features/setPlan'
          )
        },

        hasFeature: (key: FeatureKey) => {
          const feature = get().features[key]
          return feature?.enabled ?? false
        },

        getFeatureLimit: (key: FeatureKey) => {
          const feature = get().features[key]
          return feature?.limit
        },

        getFeature: (key: FeatureKey) => {
          return get().features[key]
        },

        getEnabledFeatures: () => {
          const { features } = get()
          return Object.values(features).filter((f) => f.enabled)
        },

        getDisabledFeatures: () => {
          const { features } = get()
          return Object.values(features).filter((f) => !f.enabled)
        },

        getRequiredPlan: (key: FeatureKey) => {
          const feature = get().features[key]
          return feature?.requiredPlan
        },

        reset: () => {
          set(initialState, false, 'features/reset')
        },
      }),
      {
        name: STORAGE_KEY,
        storage: createJSONStorage(() => sessionStorage),
        partialize: (state) => ({
          plan: state.plan,
          tenantId: state.tenantId,
        }),
      }
    ),
    { name: 'FeatureStore' }
  )
)

// ============================================================================
// Selector Hooks
// ============================================================================

/**
 * Hook to get the current tenant plan
 */
export const useTenantPlan = () => useFeatureStore((state) => state.plan)

/**
 * Hook to check if features are ready
 */
export const useFeaturesReady = () => useFeatureStore((state) => state.isReady)

/**
 * Hook to check if a specific feature is enabled
 *
 * @example
 * ```tsx
 * const hasAdvancedReporting = useHasFeature('advanced_reporting')
 * ```
 */
export const useHasFeature = (key: FeatureKey) =>
  useFeatureStore((state) => state.features[key]?.enabled ?? false)

/**
 * Hook to get a feature's limit
 *
 * @example
 * ```tsx
 * const importLimit = useFeatureLimit('data_import')
 * // importLimit might be 100, 1000, 10000, or null (unlimited)
 * ```
 */
export const useFeatureLimit = (key: FeatureKey) =>
  useFeatureStore((state) => state.features[key]?.limit)

/**
 * Hook to get the required plan for a feature
 *
 * @example
 * ```tsx
 * const requiredPlan = useRequiredPlan('api_access')
 * // requiredPlan would be 'pro'
 * ```
 */
export const useRequiredPlan = (key: FeatureKey) =>
  useFeatureStore((state) => state.features[key]?.requiredPlan)

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Compare two plans to determine if one is higher than another
 *
 * @example
 * ```ts
 * isPlanHigherOrEqual('pro', 'basic') // true
 * isPlanHigherOrEqual('free', 'basic') // false
 * ```
 */
export function isPlanHigherOrEqual(currentPlan: TenantPlan, requiredPlan: TenantPlan): boolean {
  return PLAN_HIERARCHY[currentPlan] >= PLAN_HIERARCHY[requiredPlan]
}

/**
 * Get the next upgrade plan
 *
 * @example
 * ```ts
 * getNextPlan('free') // 'basic'
 * getNextPlan('enterprise') // null
 * ```
 */
export function getNextPlan(currentPlan: TenantPlan): TenantPlan | null {
  const plans: TenantPlan[] = ['free', 'basic', 'pro', 'enterprise']
  const currentIndex = plans.indexOf(currentPlan)
  if (currentIndex < plans.length - 1) {
    return plans[currentIndex + 1]
  }
  return null
}

/**
 * Get human-readable plan name
 */
export function getPlanDisplayName(plan: TenantPlan): string {
  const names: Record<TenantPlan, string> = {
    free: 'Free',
    basic: 'Basic',
    pro: 'Professional',
    enterprise: 'Enterprise',
  }
  return names[plan]
}

/**
 * Get all feature keys
 */
export function getAllFeatureKeys(): FeatureKey[] {
  return Object.keys(PLAN_FEATURES.free) as FeatureKey[]
}
