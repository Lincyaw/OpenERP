package models

import (
	"encoding/json"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// logger for model conversion errors (silent failures are logged for debugging)
var modelLogger = zap.L().Named("featureflag.models")

// FeatureFlagModel is the persistence model for the FeatureFlag aggregate root.
// Feature flags are GLOBAL (not tenant-scoped) as they control application behavior
// across the entire system.
type FeatureFlagModel struct {
	AggregateModel
	Key              string                 `gorm:"type:varchar(100);not null;uniqueIndex"`
	Name             string                 `gorm:"type:varchar(200);not null"`
	Description      string                 `gorm:"type:text"`
	Type             featureflag.FlagType   `gorm:"type:varchar(20);not null"`
	Status           featureflag.FlagStatus `gorm:"type:varchar(20);not null;index"`
	DefaultValueJSON string                 `gorm:"column:default_value;type:jsonb;not null"`
	RulesJSON        string                 `gorm:"column:rules;type:jsonb;default:'[]'"`
	TagsJSON         string                 `gorm:"column:tags;type:jsonb;default:'[]'"`
	CreatedBy        *uuid.UUID             `gorm:"type:uuid;index"`
	UpdatedBy        *uuid.UUID             `gorm:"type:uuid"`
}

// TableName returns the table name for GORM
func (FeatureFlagModel) TableName() string {
	return "feature_flags"
}

// ToDomain converts the persistence model to a domain FeatureFlag entity.
func (m *FeatureFlagModel) ToDomain() *featureflag.FeatureFlag {
	flag := &featureflag.FeatureFlag{
		BaseAggregateRoot: shared.BaseAggregateRoot{
			BaseEntity: shared.BaseEntity{
				ID:        m.ID,
				CreatedAt: m.CreatedAt,
				UpdatedAt: m.UpdatedAt,
			},
			Version: m.Version,
		},
		Key:         m.Key,
		Name:        m.Name,
		Description: m.Description,
		Type:        m.Type,
		Status:      m.Status,
		Rules:       make([]featureflag.TargetingRule, 0),
		Tags:        make([]string, 0),
		CreatedBy:   m.CreatedBy,
		UpdatedBy:   m.UpdatedBy,
	}

	// Parse default value from JSON
	if m.DefaultValueJSON != "" {
		var defaultValue featureflag.FlagValue
		if err := json.Unmarshal([]byte(m.DefaultValueJSON), &defaultValue); err != nil {
			modelLogger.Warn("failed to parse default_value JSON",
				zap.String("flag_key", m.Key),
				zap.String("raw_json", m.DefaultValueJSON),
				zap.Error(err))
		} else {
			flag.DefaultValue = defaultValue
		}
	}

	// Parse rules from JSON
	if m.RulesJSON != "" && m.RulesJSON != "[]" {
		var rules []featureflag.TargetingRule
		if err := json.Unmarshal([]byte(m.RulesJSON), &rules); err != nil {
			modelLogger.Warn("failed to parse rules JSON",
				zap.String("flag_key", m.Key),
				zap.String("raw_json", m.RulesJSON),
				zap.Error(err))
		} else {
			flag.Rules = rules
		}
	}

	// Parse tags from JSON
	if m.TagsJSON != "" && m.TagsJSON != "[]" {
		var tags []string
		if err := json.Unmarshal([]byte(m.TagsJSON), &tags); err != nil {
			modelLogger.Warn("failed to parse tags JSON",
				zap.String("flag_key", m.Key),
				zap.String("raw_json", m.TagsJSON),
				zap.Error(err))
		} else {
			flag.Tags = tags
		}
	}

	return flag
}

// FromDomain populates the persistence model from a domain FeatureFlag entity.
func (m *FeatureFlagModel) FromDomain(f *featureflag.FeatureFlag) {
	m.FromDomainAggregateRoot(f.BaseAggregateRoot)
	m.Key = f.Key
	m.Name = f.Name
	m.Description = f.Description
	m.Type = f.Type
	m.Status = f.Status
	m.CreatedBy = f.CreatedBy
	m.UpdatedBy = f.UpdatedBy

	// Serialize default value to JSON
	if jsonBytes, err := json.Marshal(f.DefaultValue); err == nil {
		m.DefaultValueJSON = string(jsonBytes)
	} else {
		m.DefaultValueJSON = `{"enabled":false}`
	}

	// Serialize rules to JSON
	if len(f.Rules) > 0 {
		if jsonBytes, err := json.Marshal(f.Rules); err == nil {
			m.RulesJSON = string(jsonBytes)
		} else {
			m.RulesJSON = "[]"
		}
	} else {
		m.RulesJSON = "[]"
	}

	// Serialize tags to JSON
	if len(f.Tags) > 0 {
		if jsonBytes, err := json.Marshal(f.Tags); err == nil {
			m.TagsJSON = string(jsonBytes)
		} else {
			m.TagsJSON = "[]"
		}
	} else {
		m.TagsJSON = "[]"
	}
}

// FeatureFlagModelFromDomain creates a new persistence model from a domain FeatureFlag entity.
func FeatureFlagModelFromDomain(f *featureflag.FeatureFlag) *FeatureFlagModel {
	m := &FeatureFlagModel{}
	m.FromDomain(f)
	return m
}

// FlagOverrideModel is the persistence model for the FlagOverride entity.
type FlagOverrideModel struct {
	BaseModel
	FlagKey    string                         `gorm:"type:varchar(100);not null;index:idx_flag_override_key;index:idx_flag_override_target,priority:1"`
	TargetType featureflag.OverrideTargetType `gorm:"type:varchar(20);not null;index:idx_flag_override_target,priority:2"`
	TargetID   uuid.UUID                      `gorm:"type:uuid;not null;index:idx_flag_override_target,priority:3"`
	ValueJSON  string                         `gorm:"column:value;type:jsonb;not null"`
	Reason     string                         `gorm:"type:text"`
	ExpiresAt  *time.Time                     `gorm:"index"`
	CreatedBy  *uuid.UUID                     `gorm:"type:uuid;index"`
}

// TableName returns the table name for GORM
func (FlagOverrideModel) TableName() string {
	return "flag_overrides"
}

// ToDomain converts the persistence model to a domain FlagOverride entity.
func (m *FlagOverrideModel) ToDomain() *featureflag.FlagOverride {
	override := &featureflag.FlagOverride{
		BaseEntity: shared.BaseEntity{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		FlagKey:    m.FlagKey,
		TargetType: m.TargetType,
		TargetID:   m.TargetID,
		Reason:     m.Reason,
		ExpiresAt:  m.ExpiresAt,
		CreatedBy:  m.CreatedBy,
	}

	// Parse value from JSON
	if m.ValueJSON != "" {
		var value featureflag.FlagValue
		if err := json.Unmarshal([]byte(m.ValueJSON), &value); err != nil {
			modelLogger.Warn("failed to parse override value JSON",
				zap.String("flag_key", m.FlagKey),
				zap.String("target_type", string(m.TargetType)),
				zap.String("raw_json", m.ValueJSON),
				zap.Error(err))
		} else {
			override.Value = value
		}
	}

	return override
}

// FromDomain populates the persistence model from a domain FlagOverride entity.
func (m *FlagOverrideModel) FromDomain(o *featureflag.FlagOverride) {
	m.FromDomainBaseEntity(o.BaseEntity)
	m.FlagKey = o.FlagKey
	m.TargetType = o.TargetType
	m.TargetID = o.TargetID
	m.Reason = o.Reason
	m.ExpiresAt = o.ExpiresAt
	m.CreatedBy = o.CreatedBy

	// Serialize value to JSON
	if jsonBytes, err := json.Marshal(o.Value); err == nil {
		m.ValueJSON = string(jsonBytes)
	} else {
		m.ValueJSON = `{"enabled":false}`
	}
}

// FlagOverrideModelFromDomain creates a new persistence model from a domain FlagOverride entity.
func FlagOverrideModelFromDomain(o *featureflag.FlagOverride) *FlagOverrideModel {
	m := &FlagOverrideModel{}
	m.FromDomain(o)
	return m
}

// FlagAuditLogModel is the persistence model for the FlagAuditLog entity.
// Audit logs are append-only and should not be modified after creation.
type FlagAuditLogModel struct {
	BaseModel
	FlagKey      string                  `gorm:"type:varchar(100);not null;index"`
	Action       featureflag.AuditAction `gorm:"type:varchar(30);not null;index"`
	OldValueJSON string                  `gorm:"column:old_value;type:jsonb"`
	NewValueJSON string                  `gorm:"column:new_value;type:jsonb"`
	UserID       *uuid.UUID              `gorm:"type:uuid;index"`
	IPAddress    string                  `gorm:"type:varchar(45)"`
	UserAgent    string                  `gorm:"type:text"`
}

// TableName returns the table name for GORM
func (FlagAuditLogModel) TableName() string {
	return "flag_audit_logs"
}

// ToDomain converts the persistence model to a domain FlagAuditLog entity.
func (m *FlagAuditLogModel) ToDomain() *featureflag.FlagAuditLog {
	log := &featureflag.FlagAuditLog{
		BaseEntity: shared.BaseEntity{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		FlagKey:   m.FlagKey,
		Action:    m.Action,
		OldValue:  make(map[string]any),
		NewValue:  make(map[string]any),
		UserID:    m.UserID,
		IPAddress: m.IPAddress,
		UserAgent: m.UserAgent,
	}

	// Parse old value from JSON
	if m.OldValueJSON != "" && m.OldValueJSON != "{}" {
		var oldValue map[string]any
		if err := json.Unmarshal([]byte(m.OldValueJSON), &oldValue); err != nil {
			modelLogger.Warn("failed to parse audit log old_value JSON",
				zap.String("flag_key", m.FlagKey),
				zap.String("raw_json", m.OldValueJSON),
				zap.Error(err))
		} else {
			log.OldValue = oldValue
		}
	}

	// Parse new value from JSON
	if m.NewValueJSON != "" && m.NewValueJSON != "{}" {
		var newValue map[string]any
		if err := json.Unmarshal([]byte(m.NewValueJSON), &newValue); err != nil {
			modelLogger.Warn("failed to parse audit log new_value JSON",
				zap.String("flag_key", m.FlagKey),
				zap.String("raw_json", m.NewValueJSON),
				zap.Error(err))
		} else {
			log.NewValue = newValue
		}
	}

	return log
}

// FromDomain populates the persistence model from a domain FlagAuditLog entity.
func (m *FlagAuditLogModel) FromDomain(l *featureflag.FlagAuditLog) {
	m.FromDomainBaseEntity(l.BaseEntity)
	m.FlagKey = l.FlagKey
	m.Action = l.Action
	m.UserID = l.UserID
	m.IPAddress = l.IPAddress
	m.UserAgent = l.UserAgent

	// Serialize old value to JSON
	if len(l.OldValue) > 0 {
		if jsonBytes, err := json.Marshal(l.OldValue); err == nil {
			m.OldValueJSON = string(jsonBytes)
		} else {
			m.OldValueJSON = "{}"
		}
	} else {
		m.OldValueJSON = "{}"
	}

	// Serialize new value to JSON
	if len(l.NewValue) > 0 {
		if jsonBytes, err := json.Marshal(l.NewValue); err == nil {
			m.NewValueJSON = string(jsonBytes)
		} else {
			m.NewValueJSON = "{}"
		}
	} else {
		m.NewValueJSON = "{}"
	}
}

// FlagAuditLogModelFromDomain creates a new persistence model from a domain FlagAuditLog entity.
func FlagAuditLogModelFromDomain(l *featureflag.FlagAuditLog) *FlagAuditLogModel {
	m := &FlagAuditLogModel{}
	m.FromDomain(l)
	return m
}
