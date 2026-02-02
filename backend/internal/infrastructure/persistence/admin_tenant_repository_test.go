package persistence

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMockAdminTenantRepository creates a GormAdminTenantRepository with a mocked SQL connection
func newMockAdminTenantRepository(t *testing.T) (*GormAdminTenantRepository, sqlmock.Sqlmock, *sql.DB) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	repo := NewGormAdminTenantRepository(gormDB)
	return repo, mock, mockDB
}

// tenantColumns returns the columns for tenant queries
func tenantColumns() []string {
	return []string{
		"id", "created_at", "updated_at", "version",
		"code", "name", "short_name", "status", "plan",
		"contact_name", "contact_phone", "contact_email",
		"address", "logo_url", "domain", "expires_at", "trial_ends_at",
		"config_max_users", "config_max_warehouses", "config_max_products",
		"config_features", "config_settings", "config_cost_strategy",
		"config_currency", "config_timezone", "config_locale",
		"notes", "stripe_customer_id", "stripe_subscription_id",
	}
}

// addTenantRow adds a tenant row to the mock rows
func addTenantRow(rows *sqlmock.Rows, id uuid.UUID, code, name string, status identity.TenantStatus, plan identity.TenantPlan) *sqlmock.Rows {
	now := time.Now()
	return rows.AddRow(
		id, now, now, 1,
		code, name, "", status, plan,
		"Contact", "123456", "contact@example.com",
		"Address", "", "", nil, nil,
		5, 3, 1000,
		"{}", "{}", "weighted_average",
		"CNY", "Asia/Shanghai", "zh-CN",
		"", "", "",
	)
}

func TestGormAdminTenantRepository_FindAll(t *testing.T) {
	t.Run("returns all tenants with default filter", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "TENANT1", "Tenant One", identity.TenantStatusActive, identity.TenantPlanBasic)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" ORDER BY created_at DESC LIMIT $1`)).
			WithArgs(20).
			WillReturnRows(rows)

		filter := identity.DefaultAdminTenantFilter()
		tenants, err := repo.FindAll(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)
		assert.Equal(t, "TENANT1", tenants[0].Code)
		assert.Equal(t, "Tenant One", tenants[0].Name)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("applies status filter", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		status := identity.TenantStatusActive
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "ACTIVE1", "Active Tenant", identity.TenantStatusActive, identity.TenantPlanPro)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE status = $1 ORDER BY created_at DESC LIMIT $2`)).
			WithArgs(status, 20).
			WillReturnRows(rows)

		filter := identity.DefaultAdminTenantFilter()
		filter.Status = &status
		tenants, err := repo.FindAll(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)
		assert.Equal(t, identity.TenantStatusActive, tenants[0].Status)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("applies plan type filter", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		plan := identity.TenantPlanEnterprise
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "ENT1", "Enterprise Tenant", identity.TenantStatusActive, identity.TenantPlanEnterprise)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE plan = $1 ORDER BY created_at DESC LIMIT $2`)).
			WithArgs(plan, 20).
			WillReturnRows(rows)

		filter := identity.DefaultAdminTenantFilter()
		filter.PlanType = &plan
		tenants, err := repo.FindAll(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)
		assert.Equal(t, identity.TenantPlanEnterprise, tenants[0].Plan)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("applies search filter", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "TEST1", "Test Tenant", identity.TenantStatusActive, identity.TenantPlanBasic)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE name ILIKE $1 OR code ILIKE $2 OR short_name ILIKE $3 OR contact_email ILIKE $4 ORDER BY created_at DESC LIMIT $5`)).
			WithArgs("%test%", "%test%", "%test%", "%test%", 20).
			WillReturnRows(rows)

		filter := identity.DefaultAdminTenantFilter()
		filter.Search = "test"
		tenants, err := repo.FindAll(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("applies pagination", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "PAGE3", "Page 3 Tenant", identity.TenantStatusActive, identity.TenantPlanBasic)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" ORDER BY created_at DESC LIMIT $1 OFFSET $2`)).
			WithArgs(10, 20).
			WillReturnRows(rows)

		filter := identity.AdminTenantFilter{
			Page:     3,
			PageSize: 10,
			OrderBy:  "created_at",
			OrderDir: "desc",
		}
		tenants, err := repo.FindAll(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("applies sorting", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "ALPHA", "Alpha Tenant", identity.TenantStatusActive, identity.TenantPlanBasic)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" ORDER BY name ASC LIMIT $1`)).
			WithArgs(20).
			WillReturnRows(rows)

		filter := identity.DefaultAdminTenantFilter()
		filter.OrderBy = "name"
		filter.OrderDir = "asc"
		tenants, err := repo.FindAll(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("validates sort field against whitelist", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "TENANT1", "Tenant One", identity.TenantStatusActive, identity.TenantPlanBasic)

		// Invalid sort field should default to created_at
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" ORDER BY created_at DESC LIMIT $1`)).
			WithArgs(20).
			WillReturnRows(rows)

		filter := identity.DefaultAdminTenantFilter()
		filter.OrderBy = "invalid_field; DROP TABLE tenants; --"
		tenants, err := repo.FindAll(context.Background(), filter)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_FindByID(t *testing.T) {
	t.Run("returns tenant when found", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "FOUND", "Found Tenant", identity.TenantStatusActive, identity.TenantPlanPro)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE id = $1 ORDER BY "tenants"."id" LIMIT $2`)).
			WithArgs(tenantID, 1).
			WillReturnRows(rows)

		tenant, err := repo.FindByID(context.Background(), tenantID)

		require.NoError(t, err)
		assert.NotNil(t, tenant)
		assert.Equal(t, tenantID, tenant.ID)
		assert.Equal(t, "FOUND", tenant.Code)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("returns ErrNotFound when tenant not found", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE id = $1 ORDER BY "tenants"."id" LIMIT $2`)).
			WithArgs(tenantID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		tenant, err := repo.FindByID(context.Background(), tenantID)

		assert.ErrorIs(t, err, shared.ErrNotFound)
		assert.Nil(t, tenant)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_Count(t *testing.T) {
	t.Run("returns total count with default filter", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tenants"`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

		filter := identity.DefaultAdminTenantFilter()
		count, err := repo.Count(context.Background(), filter)

		require.NoError(t, err)
		assert.Equal(t, int64(100), count)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("returns count with status filter", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		status := identity.TenantStatusActive

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tenants" WHERE status = $1`)).
			WithArgs(status).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))

		filter := identity.DefaultAdminTenantFilter()
		filter.Status = &status
		count, err := repo.Count(context.Background(), filter)

		require.NoError(t, err)
		assert.Equal(t, int64(50), count)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_Search(t *testing.T) {
	t.Run("searches tenants by query", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "ACME", "Acme Corp", identity.TenantStatusActive, identity.TenantPlanEnterprise)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE name ILIKE $1 OR code ILIKE $2 OR short_name ILIKE $3 OR contact_email ILIKE $4 ORDER BY created_at DESC LIMIT $5`)).
			WithArgs("%acme%", "%acme%", "%acme%", "%acme%", 20).
			WillReturnRows(rows)

		filter := identity.DefaultAdminTenantFilter()
		tenants, err := repo.Search(context.Background(), "acme", filter)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)
		assert.Equal(t, "ACME", tenants[0].Code)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_CountByStatus(t *testing.T) {
	t.Run("returns counts grouped by status", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT status, COUNT(*) as count FROM "tenants" GROUP BY "status"`)).
			WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
				AddRow(identity.TenantStatusActive, 50).
				AddRow(identity.TenantStatusInactive, 10).
				AddRow(identity.TenantStatusSuspended, 5).
				AddRow(identity.TenantStatusTrial, 20))

		counts, err := repo.CountByStatus(context.Background())

		require.NoError(t, err)
		assert.Equal(t, int64(50), counts[identity.TenantStatusActive])
		assert.Equal(t, int64(10), counts[identity.TenantStatusInactive])
		assert.Equal(t, int64(5), counts[identity.TenantStatusSuspended])
		assert.Equal(t, int64(20), counts[identity.TenantStatusTrial])

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_CountByPlan(t *testing.T) {
	t.Run("returns counts grouped by plan", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT plan, COUNT(*) as count FROM "tenants" GROUP BY "plan"`)).
			WillReturnRows(sqlmock.NewRows([]string{"plan", "count"}).
				AddRow(identity.TenantPlanFree, 30).
				AddRow(identity.TenantPlanBasic, 25).
				AddRow(identity.TenantPlanPro, 15).
				AddRow(identity.TenantPlanEnterprise, 5))

		counts, err := repo.CountByPlan(context.Background())

		require.NoError(t, err)
		assert.Equal(t, int64(30), counts[identity.TenantPlanFree])
		assert.Equal(t, int64(25), counts[identity.TenantPlanBasic])
		assert.Equal(t, int64(15), counts[identity.TenantPlanPro])
		assert.Equal(t, int64(5), counts[identity.TenantPlanEnterprise])

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_FindTrialExpiring(t *testing.T) {
	t.Run("returns tenants with trial expiring within days", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "TRIAL1", "Trial Tenant", identity.TenantStatusTrial, identity.TenantPlanFree)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE status = $1 AND trial_ends_at IS NOT NULL AND trial_ends_at <= $2 AND trial_ends_at > $3 ORDER BY trial_ends_at ASC`)).
			WithArgs(identity.TenantStatusTrial, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		tenants, err := repo.FindTrialExpiring(context.Background(), 7)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)
		assert.Equal(t, identity.TenantStatusTrial, tenants[0].Status)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_FindSubscriptionExpiring(t *testing.T) {
	t.Run("returns tenants with subscription expiring within days", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "EXPIRING1", "Expiring Tenant", identity.TenantStatusActive, identity.TenantPlanPro)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE status = $1 AND expires_at IS NOT NULL AND expires_at <= $2 AND expires_at > $3 ORDER BY expires_at ASC`)).
			WithArgs(identity.TenantStatusActive, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		tenants, err := repo.FindSubscriptionExpiring(context.Background(), 30)

		require.NoError(t, err)
		assert.Len(t, tenants, 1)
		assert.Equal(t, identity.TenantStatusActive, tenants[0].Status)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_FindExpired(t *testing.T) {
	t.Run("returns expired tenants", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		rows := sqlmock.NewRows(tenantColumns())
		addTenantRow(rows, tenantID, "EXPIRED1", "Expired Tenant", identity.TenantStatusTrial, identity.TenantPlanFree)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE (status = $1 AND trial_ends_at IS NOT NULL AND trial_ends_at < $2) OR (status = $3 AND expires_at IS NOT NULL AND expires_at < $4) ORDER BY created_at DESC`)).
			WithArgs(identity.TenantStatusTrial, sqlmock.AnyArg(), identity.TenantStatusActive, sqlmock.AnyArg()).
			WillReturnRows(rows)

		tenants, err := repo.FindExpired(context.Background())

		require.NoError(t, err)
		assert.Len(t, tenants, 1)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGormAdminTenantRepository_GetStatistics(t *testing.T) {
	t.Run("returns aggregated statistics", func(t *testing.T) {
		repo, mock, mockDB := newMockAdminTenantRepository(t)
		defer mockDB.Close()

		// Total count
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tenants"`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

		// Count by status
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT status, COUNT(*) as count FROM "tenants" GROUP BY "status"`)).
			WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
				AddRow(identity.TenantStatusActive, 60).
				AddRow(identity.TenantStatusInactive, 15).
				AddRow(identity.TenantStatusSuspended, 5).
				AddRow(identity.TenantStatusTrial, 20))

		// Count by plan
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT plan, COUNT(*) as count FROM "tenants" GROUP BY "plan"`)).
			WillReturnRows(sqlmock.NewRows([]string{"plan", "count"}).
				AddRow(identity.TenantPlanFree, 40).
				AddRow(identity.TenantPlanBasic, 30).
				AddRow(identity.TenantPlanPro, 20).
				AddRow(identity.TenantPlanEnterprise, 10))

		// Trial expiring
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE status = $1 AND trial_ends_at IS NOT NULL AND trial_ends_at <= $2 AND trial_ends_at > $3 ORDER BY trial_ends_at ASC`)).
			WithArgs(identity.TenantStatusTrial, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows(tenantColumns()))

		// Subscription expiring
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE status = $1 AND expires_at IS NOT NULL AND expires_at <= $2 AND expires_at > $3 ORDER BY expires_at ASC`)).
			WithArgs(identity.TenantStatusActive, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows(tenantColumns()))

		// Expired
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE (status = $1 AND trial_ends_at IS NOT NULL AND trial_ends_at < $2) OR (status = $3 AND expires_at IS NOT NULL AND expires_at < $4) ORDER BY created_at DESC`)).
			WithArgs(identity.TenantStatusTrial, sqlmock.AnyArg(), identity.TenantStatusActive, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows(tenantColumns()))

		stats, err := repo.GetStatistics(context.Background())

		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(100), stats.TotalTenants)
		assert.Equal(t, int64(60), stats.ActiveTenants)
		assert.Equal(t, int64(15), stats.InactiveTenants)
		assert.Equal(t, int64(5), stats.SuspendedTenants)
		assert.Equal(t, int64(20), stats.TrialTenants)
		assert.Equal(t, int64(40), stats.ByPlan["free"])
		assert.Equal(t, int64(30), stats.ByPlan["basic"])
		assert.Equal(t, int64(20), stats.ByPlan["pro"])
		assert.Equal(t, int64(10), stats.ByPlan["enterprise"])

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestAdminTenantFilter(t *testing.T) {
	t.Run("DefaultAdminTenantFilter returns correct defaults", func(t *testing.T) {
		filter := identity.DefaultAdminTenantFilter()

		assert.Equal(t, 1, filter.Page)
		assert.Equal(t, 20, filter.PageSize)
		assert.Equal(t, "created_at", filter.OrderBy)
		assert.Equal(t, "desc", filter.OrderDir)
		assert.Empty(t, filter.Search)
		assert.Nil(t, filter.Status)
		assert.Nil(t, filter.PlanType)
	})

	t.Run("ToSharedFilter converts correctly", func(t *testing.T) {
		filter := identity.AdminTenantFilter{
			Page:     2,
			PageSize: 50,
			OrderBy:  "name",
			OrderDir: "asc",
			Search:   "test",
		}

		sharedFilter := filter.ToSharedFilter()

		assert.Equal(t, 2, sharedFilter.Page)
		assert.Equal(t, 50, sharedFilter.PageSize)
		assert.Equal(t, "name", sharedFilter.OrderBy)
		assert.Equal(t, "asc", sharedFilter.OrderDir)
		assert.Equal(t, "test", sharedFilter.Search)
	})
}

func TestAdminTenantSortFields(t *testing.T) {
	t.Run("contains expected sort fields", func(t *testing.T) {
		expectedFields := []string{
			"id", "created_at", "updated_at", "code", "name",
			"short_name", "status", "plan", "expires_at",
			"trial_ends_at", "contact_name", "contact_email",
		}

		for _, field := range expectedFields {
			assert.True(t, AdminTenantSortFields[field], "Expected field %s to be in AdminTenantSortFields", field)
		}
	})

	t.Run("does not contain dangerous fields", func(t *testing.T) {
		dangerousFields := []string{
			"password", "stripe_customer_id", "stripe_subscription_id",
			"config_features", "config_settings",
		}

		for _, field := range dangerousFields {
			assert.False(t, AdminTenantSortFields[field], "Field %s should not be in AdminTenantSortFields", field)
		}
	})
}

func TestTenantStatistics(t *testing.T) {
	t.Run("TenantStatistics struct initializes correctly", func(t *testing.T) {
		stats := &identity.TenantStatistics{
			TotalTenants:     100,
			ActiveTenants:    60,
			InactiveTenants:  15,
			SuspendedTenants: 5,
			TrialTenants:     20,
			ByPlan: map[string]int64{
				"free":       40,
				"basic":      30,
				"pro":        20,
				"enterprise": 10,
			},
			TrialExpiring7d: 3,
			SubExpiring30d:  5,
			ExpiredTenants:  2,
		}

		assert.Equal(t, int64(100), stats.TotalTenants)
		assert.Equal(t, int64(60), stats.ActiveTenants)
		assert.Equal(t, int64(40), stats.ByPlan["free"])
		assert.Equal(t, int64(3), stats.TrialExpiring7d)
	})
}
