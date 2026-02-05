package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantPlan_Level(t *testing.T) {
	tests := []struct {
		name  string
		plan  TenantPlan
		level int
	}{
		{"free is level 0", TenantPlanFree, 0},
		{"basic is level 1", TenantPlanBasic, 1},
		{"pro is level 2", TenantPlanPro, 2},
		{"enterprise is level 3", TenantPlanEnterprise, 3},
		{"unknown plan is level -1", TenantPlan("unknown"), -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.level, tt.plan.Level())
		})
	}
}

func TestTenantPlan_IsUpgradeFrom(t *testing.T) {
	tests := []struct {
		name      string
		newPlan   TenantPlan
		fromPlan  TenantPlan
		isUpgrade bool
	}{
		{"free to basic is upgrade", TenantPlanBasic, TenantPlanFree, true},
		{"free to pro is upgrade", TenantPlanPro, TenantPlanFree, true},
		{"free to enterprise is upgrade", TenantPlanEnterprise, TenantPlanFree, true},
		{"basic to pro is upgrade", TenantPlanPro, TenantPlanBasic, true},
		{"basic to enterprise is upgrade", TenantPlanEnterprise, TenantPlanBasic, true},
		{"pro to enterprise is upgrade", TenantPlanEnterprise, TenantPlanPro, true},
		{"basic to free is not upgrade", TenantPlanFree, TenantPlanBasic, false},
		{"pro to basic is not upgrade", TenantPlanBasic, TenantPlanPro, false},
		{"enterprise to pro is not upgrade", TenantPlanPro, TenantPlanEnterprise, false},
		{"same plan is not upgrade", TenantPlanBasic, TenantPlanBasic, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isUpgrade, tt.newPlan.IsUpgradeFrom(tt.fromPlan))
		})
	}
}

func TestTenantPlan_IsDowngradeFrom(t *testing.T) {
	tests := []struct {
		name        string
		newPlan     TenantPlan
		fromPlan    TenantPlan
		isDowngrade bool
	}{
		{"basic to free is downgrade", TenantPlanFree, TenantPlanBasic, true},
		{"pro to free is downgrade", TenantPlanFree, TenantPlanPro, true},
		{"enterprise to free is downgrade", TenantPlanFree, TenantPlanEnterprise, true},
		{"pro to basic is downgrade", TenantPlanBasic, TenantPlanPro, true},
		{"enterprise to basic is downgrade", TenantPlanBasic, TenantPlanEnterprise, true},
		{"enterprise to pro is downgrade", TenantPlanPro, TenantPlanEnterprise, true},
		{"free to basic is not downgrade", TenantPlanBasic, TenantPlanFree, false},
		{"basic to pro is not downgrade", TenantPlanPro, TenantPlanBasic, false},
		{"pro to enterprise is not downgrade", TenantPlanEnterprise, TenantPlanPro, false},
		{"same plan is not downgrade", TenantPlanBasic, TenantPlanBasic, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isDowngrade, tt.newPlan.IsDowngradeFrom(tt.fromPlan))
		})
	}
}

func TestTenantPlan_IsSamePlan(t *testing.T) {
	tests := []struct {
		name   string
		plan1  TenantPlan
		plan2  TenantPlan
		isSame bool
	}{
		{"free equals free", TenantPlanFree, TenantPlanFree, true},
		{"basic equals basic", TenantPlanBasic, TenantPlanBasic, true},
		{"pro equals pro", TenantPlanPro, TenantPlanPro, true},
		{"enterprise equals enterprise", TenantPlanEnterprise, TenantPlanEnterprise, true},
		{"free not equals basic", TenantPlanFree, TenantPlanBasic, false},
		{"basic not equals pro", TenantPlanBasic, TenantPlanPro, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isSame, tt.plan1.IsSamePlan(tt.plan2))
		})
	}
}

func TestTenantPlan_String(t *testing.T) {
	assert.Equal(t, "free", TenantPlanFree.String())
	assert.Equal(t, "basic", TenantPlanBasic.String())
	assert.Equal(t, "pro", TenantPlanPro.String())
	assert.Equal(t, "enterprise", TenantPlanEnterprise.String())
}

func TestTenantPlan_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		plan    TenantPlan
		isValid bool
	}{
		{"free is valid", TenantPlanFree, true},
		{"basic is valid", TenantPlanBasic, true},
		{"pro is valid", TenantPlanPro, true},
		{"enterprise is valid", TenantPlanEnterprise, true},
		{"empty is invalid", TenantPlan(""), false},
		{"unknown is invalid", TenantPlan("unknown"), false},
		{"premium is invalid", TenantPlan("premium"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.plan.IsValid())
		})
	}
}

func TestAllPlans(t *testing.T) {
	plans := AllPlans()

	assert.Len(t, plans, 4)
	assert.Equal(t, TenantPlanFree, plans[0])
	assert.Equal(t, TenantPlanBasic, plans[1])
	assert.Equal(t, TenantPlanPro, plans[2])
	assert.Equal(t, TenantPlanEnterprise, plans[3])
}

func TestTenantPlan_DisplayName(t *testing.T) {
	tests := []struct {
		name        string
		plan        TenantPlan
		displayName string
	}{
		{"free display name", TenantPlanFree, "Free"},
		{"basic display name", TenantPlanBasic, "Basic"},
		{"pro display name", TenantPlanPro, "Professional"},
		{"enterprise display name", TenantPlanEnterprise, "Enterprise"},
		{"unknown display name", TenantPlan("unknown"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.displayName, tt.plan.DisplayName())
		})
	}
}
