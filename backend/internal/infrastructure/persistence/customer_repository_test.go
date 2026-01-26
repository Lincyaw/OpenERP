package persistence

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/partner"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMockCustomerRepository creates a GormCustomerRepository with a mocked SQL connection
func newMockCustomerRepository(t *testing.T) (*GormCustomerRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormCustomerRepository(gormDB), mock, mockDB
}

func TestNewGormCustomerRepository(t *testing.T) {
	t.Run("creates repository with valid DB", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.db)
	})
}

func TestGormCustomerRepository_FindByID(t *testing.T) {
	t.Run("finds existing customer", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customerID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "level", "status", "balance", "credit_limit"}).
			AddRow(customerID, tenantID, "CUST001", "Test Customer", "individual", "normal", "active", decimal.Zero, decimal.Zero)

		mock.ExpectQuery(`SELECT \* FROM "customers" WHERE id = \$1 ORDER BY .* LIMIT .*`).
			WithArgs(customerID, 1).
			WillReturnRows(rows)

		customer, err := repo.FindByID(context.Background(), customerID)

		assert.NoError(t, err)
		assert.NotNil(t, customer)
		assert.Equal(t, customerID, customer.ID)
		assert.Equal(t, "CUST001", customer.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent customer", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customerID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "customers" WHERE id = \$1 ORDER BY .* LIMIT .*`).
			WithArgs(customerID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		customer, err := repo.FindByID(context.Background(), customerID)

		assert.Error(t, err)
		assert.Nil(t, customer)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_FindByIDForTenant(t *testing.T) {
	t.Run("finds customer within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customerID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "level", "status", "balance", "credit_limit"}).
			AddRow(customerID, tenantID, "CUST001", "Test Customer", "individual", "normal", "active", decimal.Zero, decimal.Zero)

		mock.ExpectQuery(`SELECT \* FROM "customers" WHERE tenant_id = \$1 AND id = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, customerID, 1).
			WillReturnRows(rows)

		customer, err := repo.FindByIDForTenant(context.Background(), tenantID, customerID)

		assert.NoError(t, err)
		assert.NotNil(t, customer)
		assert.Equal(t, tenantID, customer.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_FindByCode(t *testing.T) {
	t.Run("finds customer by code", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customerID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "level", "status", "balance", "credit_limit"}).
			AddRow(customerID, tenantID, "CUST001", "Test Customer", "individual", "normal", "active", decimal.Zero, decimal.Zero)

		mock.ExpectQuery(`SELECT \* FROM "customers" WHERE tenant_id = \$1 AND code = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, "CUST001", 1).
			WillReturnRows(rows)

		customer, err := repo.FindByCode(context.Background(), tenantID, "cust001") // lowercase to test uppercasing

		assert.NoError(t, err)
		assert.NotNil(t, customer)
		assert.Equal(t, "CUST001", customer.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_FindByPhone(t *testing.T) {
	t.Run("finds customer by phone", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customerID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "phone", "type", "level", "status", "balance", "credit_limit"}).
			AddRow(customerID, tenantID, "CUST001", "Test Customer", "13800138000", "individual", "normal", "active", decimal.Zero, decimal.Zero)

		mock.ExpectQuery(`SELECT \* FROM "customers" WHERE tenant_id = \$1 AND phone = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, "13800138000", 1).
			WillReturnRows(rows)

		customer, err := repo.FindByPhone(context.Background(), tenantID, "13800138000")

		assert.NoError(t, err)
		assert.NotNil(t, customer)
		assert.Equal(t, "13800138000", customer.Phone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for empty phone", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		_, err := repo.FindByPhone(context.Background(), uuid.New(), "")

		assert.Error(t, err)
	})
}

func TestGormCustomerRepository_FindByEmail(t *testing.T) {
	t.Run("finds customer by email", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customerID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "email", "type", "level", "status", "balance", "credit_limit"}).
			AddRow(customerID, tenantID, "CUST001", "Test Customer", "test@example.com", "individual", "normal", "active", decimal.Zero, decimal.Zero)

		mock.ExpectQuery(`SELECT \* FROM "customers" WHERE tenant_id = \$1 AND email = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, "test@example.com", 1).
			WillReturnRows(rows)

		customer, err := repo.FindByEmail(context.Background(), tenantID, "test@example.com")

		assert.NoError(t, err)
		assert.NotNil(t, customer)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for empty email", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		_, err := repo.FindByEmail(context.Background(), uuid.New(), "")

		assert.Error(t, err)
	})
}

func TestGormCustomerRepository_FindByIDs(t *testing.T) {
	t.Run("finds multiple customers by IDs", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenant_id", "code", "name", "type", "level", "status", "balance", "credit_limit"}).
			AddRow(id1, tenantID, "CUST001", "Customer 1", "individual", "normal", "active", decimal.Zero, decimal.Zero).
			AddRow(id2, tenantID, "CUST002", "Customer 2", "organization", "gold", "active", decimal.Zero, decimal.Zero)

		mock.ExpectQuery(`SELECT \* FROM "customers" WHERE tenant_id = \$1 AND id IN \(\$2,\$3\)`).
			WithArgs(tenantID, id1, id2).
			WillReturnRows(rows)

		customers, err := repo.FindByIDs(context.Background(), tenantID, []uuid.UUID{id1, id2})

		assert.NoError(t, err)
		assert.Len(t, customers, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice for empty IDs", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customers, err := repo.FindByIDs(context.Background(), uuid.New(), []uuid.UUID{})

		assert.NoError(t, err)
		assert.Empty(t, customers)
	})
}

func TestGormCustomerRepository_FindByCodes(t *testing.T) {
	t.Run("returns empty slice for empty codes", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customers, err := repo.FindByCodes(context.Background(), uuid.New(), []string{})

		assert.NoError(t, err)
		assert.Empty(t, customers)
	})
}

func TestGormCustomerRepository_Save(t *testing.T) {
	t.Run("saves customer", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		customer, _ := partner.NewIndividualCustomer(tenantID, "CUST001", "Test Customer")

		mock.ExpectExec(`UPDATE "customers" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(context.Background(), customer)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_SaveBatch(t *testing.T) {
	t.Run("returns nil for empty batch", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		err := repo.SaveBatch(context.Background(), []*partner.Customer{})

		assert.NoError(t, err)
	})
}

func TestGormCustomerRepository_Delete(t *testing.T) {
	t.Run("deletes existing customer", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customerID := uuid.New()

		mock.ExpectExec(`DELETE FROM "customers" WHERE id = \$1`).
			WithArgs(customerID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), customerID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent customer", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		customerID := uuid.New()

		mock.ExpectExec(`DELETE FROM "customers" WHERE id = \$1`).
			WithArgs(customerID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(context.Background(), customerID)

		assert.Error(t, err)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_DeleteForTenant(t *testing.T) {
	t.Run("deletes customer within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		customerID := uuid.New()

		mock.ExpectExec(`DELETE FROM "customers" WHERE tenant_id = \$1 AND id = \$2`).
			WithArgs(tenantID, customerID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteForTenant(context.Background(), tenantID, customerID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_Count(t *testing.T) {
	t.Run("counts customers", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "customers"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

		count, err := repo.Count(context.Background(), shared.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(10), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_CountForTenant(t *testing.T) {
	t.Run("counts customers for tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "customers" WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.CountForTenant(context.Background(), tenantID, shared.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_CountByType(t *testing.T) {
	t.Run("counts customers by type", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "customers" WHERE tenant_id = \$1 AND type = \$2`).
			WithArgs(tenantID, partner.CustomerTypeIndividual).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		count, err := repo.CountByType(context.Background(), tenantID, partner.CustomerTypeIndividual)

		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_CountByLevel(t *testing.T) {
	t.Run("counts customers by level", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		goldLevel := partner.GoldLevel()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "customers" WHERE tenant_id = \$1 AND level = \$2`).
			WithArgs(tenantID, goldLevel.Code()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		count, err := repo.CountByLevel(context.Background(), tenantID, goldLevel)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_CountByStatus(t *testing.T) {
	t.Run("counts customers by status", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "customers" WHERE tenant_id = \$1 AND status = \$2`).
			WithArgs(tenantID, partner.CustomerStatusActive).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))

		count, err := repo.CountByStatus(context.Background(), tenantID, partner.CustomerStatusActive)

		assert.NoError(t, err)
		assert.Equal(t, int64(8), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_ExistsByCode(t *testing.T) {
	t.Run("returns true when customer exists", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "customers" WHERE tenant_id = \$1 AND code = \$2`).
			WithArgs(tenantID, "CUST001").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := repo.ExistsByCode(context.Background(), tenantID, "cust001")

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when customer does not exist", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "customers" WHERE tenant_id = \$1 AND code = \$2`).
			WithArgs(tenantID, "NONEXISTENT").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := repo.ExistsByCode(context.Background(), tenantID, "nonexistent")

		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_ExistsByPhone(t *testing.T) {
	t.Run("returns false for empty phone", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		exists, err := repo.ExistsByPhone(context.Background(), uuid.New(), "")

		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns true when phone exists", func(t *testing.T) {
		repo, mock, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "customers" WHERE tenant_id = \$1 AND phone = \$2`).
			WithArgs(tenantID, "13800138000").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := repo.ExistsByPhone(context.Background(), tenantID, "13800138000")

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormCustomerRepository_ExistsByEmail(t *testing.T) {
	t.Run("returns false for empty email", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		exists, err := repo.ExistsByEmail(context.Background(), uuid.New(), "")

		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestGormCustomerRepository_InterfaceCompliance(t *testing.T) {
	t.Run("implements CustomerRepository interface", func(t *testing.T) {
		repo, _, mockDB := newMockCustomerRepository(t)
		defer mockDB.Close()

		var _ partner.CustomerRepository = repo
	})
}
