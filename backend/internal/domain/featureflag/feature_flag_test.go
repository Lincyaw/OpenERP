package featureflag

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFeatureFlag(t *testing.T) {
	userID := uuid.New()

	t.Run("valid boolean flag", func(t *testing.T) {
		flag, err := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), &userID)
		require.NoError(t, err)
		assert.Equal(t, "test_flag", flag.Key)
		assert.Equal(t, "Test Flag", flag.Name)
		assert.Equal(t, FlagTypeBoolean, flag.Type)
		assert.Equal(t, FlagStatusDisabled, flag.Status) // Default to disabled
		assert.False(t, flag.DefaultValue.Enabled)
		assert.Equal(t, &userID, flag.CreatedBy)
		assert.Equal(t, &userID, flag.UpdatedBy)
		assert.NotEqual(t, uuid.Nil, flag.ID)
		assert.Equal(t, 1, flag.Version)
		assert.Len(t, flag.GetDomainEvents(), 1) // FlagCreatedEvent
	})

	t.Run("key is lowercased", func(t *testing.T) {
		flag, err := NewFeatureFlag("TEST_FLAG", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		require.NoError(t, err)
		assert.Equal(t, "test_flag", flag.Key)
	})

	t.Run("invalid key - empty", func(t *testing.T) {
		_, err := NewFeatureFlag("", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		assert.Error(t, err)
	})

	t.Run("invalid key - too long", func(t *testing.T) {
		longKey := make([]byte, 101)
		for i := range longKey {
			longKey[i] = 'a'
		}
		_, err := NewFeatureFlag(string(longKey), "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		assert.Error(t, err)
	})

	t.Run("invalid key - starts with number", func(t *testing.T) {
		_, err := NewFeatureFlag("1test", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		assert.Error(t, err)
	})

	t.Run("invalid key - invalid characters", func(t *testing.T) {
		_, err := NewFeatureFlag("test flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		assert.Error(t, err)
	})

	t.Run("valid key with dots and hyphens", func(t *testing.T) {
		flag, err := NewFeatureFlag("feature.test-flag_v2", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		require.NoError(t, err)
		assert.Equal(t, "feature.test-flag_v2", flag.Key)
	})

	t.Run("invalid name - empty", func(t *testing.T) {
		_, err := NewFeatureFlag("test_flag", "", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		assert.Error(t, err)
	})

	t.Run("invalid name - too long", func(t *testing.T) {
		longName := make([]byte, 201)
		for i := range longName {
			longName[i] = 'a'
		}
		_, err := NewFeatureFlag("test_flag", string(longName), FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		assert.Error(t, err)
	})

	t.Run("invalid flag type", func(t *testing.T) {
		_, err := NewFeatureFlag("test_flag", "Test Flag", FlagType("invalid"), NewBooleanFlagValue(false), nil)
		assert.Error(t, err)
	})
}

func TestNewBooleanFlag(t *testing.T) {
	flag, err := NewBooleanFlag("test_bool", "Test Boolean", true, nil)
	require.NoError(t, err)
	assert.Equal(t, FlagTypeBoolean, flag.Type)
	assert.True(t, flag.DefaultValue.Enabled)
}

func TestNewPercentageFlag(t *testing.T) {
	flag, err := NewPercentageFlag("test_pct", "Test Percentage", nil)
	require.NoError(t, err)
	assert.Equal(t, FlagTypePercentage, flag.Type)
}

func TestNewVariantFlag(t *testing.T) {
	flag, err := NewVariantFlag("test_variant", "Test Variant", "control", nil)
	require.NoError(t, err)
	assert.Equal(t, FlagTypeVariant, flag.Type)
	assert.Equal(t, "control", flag.DefaultValue.Variant)
}

func TestFeatureFlag_Getters(t *testing.T) {
	userID := uuid.New()
	flag, _ := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(true), &userID)
	flag.Description = "Test description"
	flag.Tags = []string{"tag1", "tag2"}
	flag.Rules = []TargetingRule{
		{RuleID: "rule1", Priority: 1, Percentage: 100, Value: NewBooleanFlagValue(true)},
	}

	assert.Equal(t, "test_flag", flag.GetKey())
	assert.Equal(t, "Test Flag", flag.GetName())
	assert.Equal(t, "Test description", flag.GetDescription())
	assert.Equal(t, FlagTypeBoolean, flag.GetType())
	assert.Equal(t, FlagStatusDisabled, flag.GetStatus())
	assert.True(t, flag.GetDefaultValue().Enabled)
	assert.Equal(t, &userID, flag.GetCreatedBy())
	assert.Equal(t, &userID, flag.GetUpdatedBy())

	tags := flag.GetTags()
	assert.Equal(t, []string{"tag1", "tag2"}, tags)
	tags[0] = "modified"
	assert.Equal(t, "tag1", flag.Tags[0]) // Original unchanged

	rules := flag.GetRules()
	assert.Len(t, rules, 1)
	rules[0].RuleID = "modified"
	assert.Equal(t, "rule1", flag.Rules[0].RuleID) // Original unchanged
}

func TestFeatureFlag_GetTags_NilTags(t *testing.T) {
	flag := &FeatureFlag{}
	tags := flag.GetTags()
	assert.NotNil(t, tags)
	assert.Len(t, tags, 0)
}

func TestFeatureFlag_GetRules_NilRules(t *testing.T) {
	flag := &FeatureFlag{}
	rules := flag.GetRules()
	assert.NotNil(t, rules)
	assert.Len(t, rules, 0)
}

func TestFeatureFlag_StatusChecks(t *testing.T) {
	flag, _ := NewFeatureFlag("test_flag", "Test Flag", FlagTypeBoolean, NewBooleanFlagValue(false), nil)

	// Initially disabled
	assert.True(t, flag.IsDisabled())
	assert.False(t, flag.IsEnabled())
	assert.False(t, flag.IsArchived())

	flag.Status = FlagStatusEnabled
	assert.True(t, flag.IsEnabled())
	assert.False(t, flag.IsDisabled())
	assert.False(t, flag.IsArchived())

	flag.Status = FlagStatusArchived
	assert.True(t, flag.IsArchived())
	assert.False(t, flag.IsEnabled())
	assert.False(t, flag.IsDisabled())
}

func TestFeatureFlag_TypeChecks(t *testing.T) {
	boolFlag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	assert.True(t, boolFlag.IsBooleanType())
	assert.False(t, boolFlag.IsPercentageType())
	assert.False(t, boolFlag.IsVariantType())
	assert.False(t, boolFlag.IsUserSegmentType())

	pctFlag, _ := NewFeatureFlag("test", "Test", FlagTypePercentage, NewBooleanFlagValue(false), nil)
	assert.True(t, pctFlag.IsPercentageType())

	varFlag, _ := NewFeatureFlag("test", "Test", FlagTypeVariant, NewBooleanFlagValue(false), nil)
	assert.True(t, varFlag.IsVariantType())

	segFlag, _ := NewFeatureFlag("test", "Test", FlagTypeUserSegment, NewBooleanFlagValue(false), nil)
	assert.True(t, segFlag.IsUserSegmentType())
}

func TestFeatureFlag_HasRules(t *testing.T) {
	flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	assert.False(t, flag.HasRules())

	flag.Rules = []TargetingRule{{RuleID: "rule1"}}
	assert.True(t, flag.HasRules())
}

func TestFeatureFlag_HasTags(t *testing.T) {
	flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	assert.False(t, flag.HasTags())

	flag.Tags = []string{"tag1"}
	assert.True(t, flag.HasTags())
}

func TestFeatureFlag_HasTag(t *testing.T) {
	flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	flag.Tags = []string{"tag1", "tag2"}

	assert.True(t, flag.HasTag("tag1"))
	assert.True(t, flag.HasTag("TAG1")) // Case insensitive
	assert.False(t, flag.HasTag("tag3"))
}

func TestFeatureFlag_Enable(t *testing.T) {
	userID := uuid.New()

	t.Run("enable disabled flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.ClearDomainEvents()

		err := flag.Enable(&userID)
		assert.NoError(t, err)
		assert.True(t, flag.IsEnabled())
		assert.Equal(t, &userID, flag.UpdatedBy)
		assert.Equal(t, 2, flag.Version)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("enable already enabled flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Enable(nil)
		err := flag.Enable(nil)
		assert.Error(t, err)
	})

	t.Run("enable archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.Status = FlagStatusArchived
		err := flag.Enable(nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_Disable(t *testing.T) {
	userID := uuid.New()

	t.Run("disable enabled flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Enable(nil)
		flag.ClearDomainEvents()

		err := flag.Disable(&userID)
		assert.NoError(t, err)
		assert.True(t, flag.IsDisabled())
		assert.Equal(t, &userID, flag.UpdatedBy)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("disable already disabled flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.Disable(nil)
		assert.Error(t, err)
	})

	t.Run("disable archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.Status = FlagStatusArchived
		err := flag.Disable(nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_Archive(t *testing.T) {
	userID := uuid.New()

	t.Run("archive disabled flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.ClearDomainEvents()

		err := flag.Archive(&userID)
		assert.NoError(t, err)
		assert.True(t, flag.IsArchived())
		assert.Equal(t, &userID, flag.UpdatedBy)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("archive enabled flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Enable(nil)
		err := flag.Archive(nil)
		assert.NoError(t, err)
		assert.True(t, flag.IsArchived())
	})

	t.Run("archive already archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Archive(nil)
		err := flag.Archive(nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_Update(t *testing.T) {
	userID := uuid.New()

	t.Run("valid update", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.ClearDomainEvents()

		err := flag.Update("New Name", "New Description", &userID)
		assert.NoError(t, err)
		assert.Equal(t, "New Name", flag.Name)
		assert.Equal(t, "New Description", flag.Description)
		assert.Equal(t, &userID, flag.UpdatedBy)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("update archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Archive(nil)
		err := flag.Update("New Name", "New Description", nil)
		assert.Error(t, err)
	})

	t.Run("update with empty name", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.Update("", "Description", nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_SetDefault(t *testing.T) {
	userID := uuid.New()

	t.Run("valid set default", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.ClearDomainEvents()

		err := flag.SetDefault(NewBooleanFlagValue(true), &userID)
		assert.NoError(t, err)
		assert.True(t, flag.DefaultValue.Enabled)
		assert.Equal(t, &userID, flag.UpdatedBy)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("set default on archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Archive(nil)
		err := flag.SetDefault(NewBooleanFlagValue(true), nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_AddRule(t *testing.T) {
	userID := uuid.New()

	t.Run("add valid rule", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.ClearDomainEvents()

		rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		err := flag.AddRule(rule, &userID)
		assert.NoError(t, err)
		assert.Len(t, flag.Rules, 1)
		assert.Equal(t, "rule1", flag.Rules[0].RuleID)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("add rule to archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Archive(nil)
		rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		err := flag.AddRule(rule, nil)
		assert.Error(t, err)
	})

	t.Run("add duplicate rule ID", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule1, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		_ = flag.AddRule(rule1, nil)
		rule2, _ := NewTargetingRule("rule1", 2, nil, NewBooleanFlagValue(false))
		err := flag.AddRule(rule2, nil)
		assert.Error(t, err)
	})

	t.Run("rules sorted by priority", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule1, _ := NewTargetingRule("rule1", 10, nil, NewBooleanFlagValue(true))
		rule2, _ := NewTargetingRule("rule2", 5, nil, NewBooleanFlagValue(false))
		rule3, _ := NewTargetingRule("rule3", 1, nil, NewBooleanFlagValue(true))
		_ = flag.AddRule(rule1, nil)
		_ = flag.AddRule(rule2, nil)
		_ = flag.AddRule(rule3, nil)

		assert.Equal(t, "rule3", flag.Rules[0].RuleID)
		assert.Equal(t, "rule2", flag.Rules[1].RuleID)
		assert.Equal(t, "rule1", flag.Rules[2].RuleID)
	})
}

func TestFeatureFlag_RemoveRule(t *testing.T) {
	userID := uuid.New()

	t.Run("remove existing rule", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		_ = flag.AddRule(rule, nil)
		flag.ClearDomainEvents()

		err := flag.RemoveRule("rule1", &userID)
		assert.NoError(t, err)
		assert.Len(t, flag.Rules, 0)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("remove non-existing rule", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.RemoveRule("nonexistent", nil)
		assert.Error(t, err)
	})

	t.Run("remove rule from archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		_ = flag.AddRule(rule, nil)
		_ = flag.Archive(nil)
		err := flag.RemoveRule("rule1", nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_UpdateRule(t *testing.T) {
	t.Run("update existing rule", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		_ = flag.AddRule(rule, nil)
		flag.ClearDomainEvents()

		updatedRule, _ := NewTargetingRuleWithPercentage("rule1", 5, nil, NewBooleanFlagValue(false), 50)
		err := flag.UpdateRule(updatedRule, nil)
		assert.NoError(t, err)
		assert.Equal(t, 5, flag.Rules[0].Priority)
		assert.Equal(t, 50, flag.Rules[0].Percentage)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("update non-existing rule", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule, _ := NewTargetingRule("nonexistent", 1, nil, NewBooleanFlagValue(true))
		err := flag.UpdateRule(rule, nil)
		assert.Error(t, err)
	})

	t.Run("update rule on archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		_ = flag.AddRule(rule, nil)
		_ = flag.Archive(nil)
		err := flag.UpdateRule(rule, nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_ClearRules(t *testing.T) {
	t.Run("clear rules", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule1, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		rule2, _ := NewTargetingRule("rule2", 2, nil, NewBooleanFlagValue(false))
		_ = flag.AddRule(rule1, nil)
		_ = flag.AddRule(rule2, nil)
		flag.ClearDomainEvents()

		err := flag.ClearRules(nil)
		assert.NoError(t, err)
		assert.Len(t, flag.Rules, 0)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("clear rules on archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Archive(nil)
		err := flag.ClearRules(nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_SetTags(t *testing.T) {
	t.Run("set tags", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.SetTags([]string{"Tag1", "TAG2", "tag3"}, nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, flag.Tags) // Lowercased
	})

	t.Run("set tags deduplicates", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.SetTags([]string{"tag1", "TAG1", "Tag1"}, nil)
		assert.NoError(t, err)
		assert.Len(t, flag.Tags, 1)
	})

	t.Run("set tags filters empty", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.SetTags([]string{"tag1", "", "  ", "tag2"}, nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"tag1", "tag2"}, flag.Tags)
	})

	t.Run("set tags on archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Archive(nil)
		err := flag.SetTags([]string{"tag1"}, nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_AddTag(t *testing.T) {
	t.Run("add tag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.AddTag("tag1", nil)
		assert.NoError(t, err)
		assert.Contains(t, flag.Tags, "tag1")
	})

	t.Run("add duplicate tag is noop", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.AddTag("tag1", nil)
		err := flag.AddTag("TAG1", nil)
		assert.NoError(t, err)
		assert.Len(t, flag.Tags, 1)
	})

	t.Run("add empty tag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.AddTag("", nil)
		assert.Error(t, err)
	})

	t.Run("add tag to archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Archive(nil)
		err := flag.AddTag("tag1", nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_RemoveTag(t *testing.T) {
	t.Run("remove tag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.AddTag("tag1", nil)
		_ = flag.AddTag("tag2", nil)

		err := flag.RemoveTag("tag1", nil)
		assert.NoError(t, err)
		assert.NotContains(t, flag.Tags, "tag1")
		assert.Contains(t, flag.Tags, "tag2")
	})

	t.Run("remove non-existing tag is noop", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.RemoveTag("nonexistent", nil)
		assert.NoError(t, err)
	})

	t.Run("remove tag from archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.AddTag("tag1", nil)
		_ = flag.Archive(nil)
		err := flag.RemoveTag("tag1", nil)
		assert.Error(t, err)
	})
}

func TestFeatureFlag_GetRuleByID(t *testing.T) {
	flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
	rule1, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
	rule2, _ := NewTargetingRule("rule2", 2, nil, NewBooleanFlagValue(false))
	_ = flag.AddRule(rule1, nil)
	_ = flag.AddRule(rule2, nil)

	r := flag.GetRuleByID("rule1")
	assert.NotNil(t, r)
	assert.Equal(t, "rule1", r.RuleID)

	r = flag.GetRuleByID("nonexistent")
	assert.Nil(t, r)
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid simple key", "test", false},
		{"valid with underscore", "test_flag", false},
		{"valid with hyphen", "test-flag", false},
		{"valid with dot", "test.flag", false},
		{"valid with number", "test123", false},
		{"empty key", "", true},
		{"starts with number", "123test", true},
		{"has space", "test flag", true},
		{"has uppercase", "Test_Flag", false}, // Will be lowercased
		{"too long", string(make([]byte, 101)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFeatureFlag_ValidateRules(t *testing.T) {
	t.Run("valid rules", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule1, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		rule2, _ := NewTargetingRule("rule2", 2, nil, NewBooleanFlagValue(false))
		flag.Rules = []TargetingRule{rule1, rule2}

		err := flag.ValidateRules()
		assert.NoError(t, err)
	})

	t.Run("duplicate rule IDs", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		rule1, _ := NewTargetingRule("rule1", 1, nil, NewBooleanFlagValue(true))
		rule2, _ := NewTargetingRule("rule1", 2, nil, NewBooleanFlagValue(false)) // Same ID
		flag.Rules = []TargetingRule{rule1, rule2}

		err := flag.ValidateRules()
		assert.Error(t, err)
	})

	t.Run("invalid rule", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.Rules = []TargetingRule{{RuleID: "", Priority: 1, Percentage: 100}}

		err := flag.ValidateRules()
		assert.Error(t, err)
	})
}

// Tests for RequiredPlan functionality

func TestRequiredPlan_IsValid(t *testing.T) {
	tests := []struct {
		plan    RequiredPlan
		isValid bool
	}{
		{RequiredPlanNone, true},
		{RequiredPlanFree, true},
		{RequiredPlanBasic, true},
		{RequiredPlanPro, true},
		{RequiredPlanEnterprise, true},
		{RequiredPlan("invalid"), false},
		{RequiredPlan("premium"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.plan.IsValid())
		})
	}
}

func TestRequiredPlan_MeetsPlanRequirement(t *testing.T) {
	tests := []struct {
		name         string
		requiredPlan RequiredPlan
		tenantPlan   string
		expected     bool
	}{
		// No restriction
		{"no restriction - free tenant", RequiredPlanNone, "free", true},
		{"no restriction - enterprise tenant", RequiredPlanNone, "enterprise", true},
		{"empty restriction - free tenant", RequiredPlan(""), "free", true},

		// Free plan requirement
		{"free required - free tenant", RequiredPlanFree, "free", true},
		{"free required - basic tenant", RequiredPlanFree, "basic", true},
		{"free required - pro tenant", RequiredPlanFree, "pro", true},
		{"free required - enterprise tenant", RequiredPlanFree, "enterprise", true},

		// Basic plan requirement
		{"basic required - free tenant", RequiredPlanBasic, "free", false},
		{"basic required - basic tenant", RequiredPlanBasic, "basic", true},
		{"basic required - pro tenant", RequiredPlanBasic, "pro", true},
		{"basic required - enterprise tenant", RequiredPlanBasic, "enterprise", true},

		// Pro plan requirement
		{"pro required - free tenant", RequiredPlanPro, "free", false},
		{"pro required - basic tenant", RequiredPlanPro, "basic", false},
		{"pro required - pro tenant", RequiredPlanPro, "pro", true},
		{"pro required - enterprise tenant", RequiredPlanPro, "enterprise", true},

		// Enterprise plan requirement
		{"enterprise required - free tenant", RequiredPlanEnterprise, "free", false},
		{"enterprise required - basic tenant", RequiredPlanEnterprise, "basic", false},
		{"enterprise required - pro tenant", RequiredPlanEnterprise, "pro", false},
		{"enterprise required - enterprise tenant", RequiredPlanEnterprise, "enterprise", true},

		// Case insensitivity
		{"case insensitive - PRO tenant", RequiredPlanBasic, "PRO", true},
		{"case insensitive - Enterprise tenant", RequiredPlanPro, "Enterprise", true},

		// Unknown tenant plan defaults to free
		{"unknown tenant plan", RequiredPlanBasic, "unknown", false},
		{"empty tenant plan", RequiredPlanBasic, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.requiredPlan.MeetsPlanRequirement(tt.tenantPlan)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFeatureFlag_RequiredPlan(t *testing.T) {
	userID := uuid.New()

	t.Run("new flag has no plan restriction", func(t *testing.T) {
		flag, err := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		require.NoError(t, err)
		assert.Equal(t, RequiredPlanNone, flag.GetRequiredPlan())
		assert.False(t, flag.HasPlanRestriction())
	})

	t.Run("set required plan", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		flag.ClearDomainEvents()

		err := flag.SetRequiredPlan(RequiredPlanPro, &userID)
		assert.NoError(t, err)
		assert.Equal(t, RequiredPlanPro, flag.GetRequiredPlan())
		assert.True(t, flag.HasPlanRestriction())
		assert.Equal(t, &userID, flag.UpdatedBy)
		assert.Len(t, flag.GetDomainEvents(), 1)
	})

	t.Run("set invalid required plan", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		err := flag.SetRequiredPlan(RequiredPlan("invalid"), nil)
		assert.Error(t, err)
	})

	t.Run("cannot set required plan on archived flag", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.Archive(nil)
		err := flag.SetRequiredPlan(RequiredPlanPro, nil)
		assert.Error(t, err)
	})

	t.Run("meets plan requirement - no restriction", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		assert.True(t, flag.MeetsPlanRequirement("free"))
		assert.True(t, flag.MeetsPlanRequirement("enterprise"))
	})

	t.Run("meets plan requirement - with restriction", func(t *testing.T) {
		flag, _ := NewFeatureFlag("test", "Test", FlagTypeBoolean, NewBooleanFlagValue(false), nil)
		_ = flag.SetRequiredPlan(RequiredPlanPro, nil)

		assert.False(t, flag.MeetsPlanRequirement("free"))
		assert.False(t, flag.MeetsPlanRequirement("basic"))
		assert.True(t, flag.MeetsPlanRequirement("pro"))
		assert.True(t, flag.MeetsPlanRequirement("enterprise"))
	})
}

func TestAllRequiredPlans(t *testing.T) {
	plans := AllRequiredPlans()
	assert.Len(t, plans, 5)
	assert.Contains(t, plans, RequiredPlanNone)
	assert.Contains(t, plans, RequiredPlanFree)
	assert.Contains(t, plans, RequiredPlanBasic)
	assert.Contains(t, plans, RequiredPlanPro)
	assert.Contains(t, plans, RequiredPlanEnterprise)
}
