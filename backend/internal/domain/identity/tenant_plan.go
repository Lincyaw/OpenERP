package identity

// PlanLevel returns the numeric level of a plan for comparison
// Higher level means more features/higher tier
func (p TenantPlan) Level() int {
	switch p {
	case TenantPlanFree:
		return 0
	case TenantPlanBasic:
		return 1
	case TenantPlanPro:
		return 2
	case TenantPlanEnterprise:
		return 3
	default:
		return -1
	}
}

// IsUpgradeFrom returns true if changing to this plan from 'from' is an upgrade
func (p TenantPlan) IsUpgradeFrom(from TenantPlan) bool {
	return p.Level() > from.Level()
}

// IsDowngradeFrom returns true if changing to this plan from 'from' is a downgrade
func (p TenantPlan) IsDowngradeFrom(from TenantPlan) bool {
	return p.Level() < from.Level()
}

// IsSamePlan returns true if this plan is the same as the other plan
func (p TenantPlan) IsSamePlan(other TenantPlan) bool {
	return p == other
}

// String returns the string representation of the plan
func (p TenantPlan) String() string {
	return string(p)
}

// IsValid returns true if the plan is a valid tenant plan
func (p TenantPlan) IsValid() bool {
	switch p {
	case TenantPlanFree, TenantPlanBasic, TenantPlanPro, TenantPlanEnterprise:
		return true
	default:
		return false
	}
}

// AllPlans returns all available tenant plans in order from lowest to highest tier
func AllPlans() []TenantPlan {
	return []TenantPlan{
		TenantPlanFree,
		TenantPlanBasic,
		TenantPlanPro,
		TenantPlanEnterprise,
	}
}

// PlanDisplayName returns a human-readable display name for the plan
func (p TenantPlan) DisplayName() string {
	switch p {
	case TenantPlanFree:
		return "Free"
	case TenantPlanBasic:
		return "Basic"
	case TenantPlanPro:
		return "Professional"
	case TenantPlanEnterprise:
		return "Enterprise"
	default:
		return "Unknown"
	}
}
