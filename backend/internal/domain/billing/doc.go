// Package billing provides domain models for usage metering and billing in a multi-tenant SaaS application.
//
// This package implements the usage metering bounded context, which is responsible for:
//   - Recording usage events (API calls, storage, active users, orders created, etc.)
//   - Aggregating usage data by tenant and time period
//   - Defining and enforcing usage quotas per subscription plan
//
// Key Aggregates:
//   - UsageRecord: Immutable record of a single usage event
//   - UsageQuota: Defines usage limits for a specific usage type and plan
//
// Value Objects:
//   - UsageMeter: Aggregated usage statistics for a tenant over a time period
//   - UsageType: Enumeration of measurable usage types
//
// The billing domain integrates with:
//   - Identity domain: For tenant and subscription plan information
//   - All other domains: As sources of usage events
package billing
