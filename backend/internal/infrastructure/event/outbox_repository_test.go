package event

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return db, mock
}

func TestGormOutboxRepository_Save(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewGormOutboxRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	event := newTestEvent("TestEvent", tenantID)
	payload := []byte(`{"test": true}`)
	entry := shared.NewOutboxEntry(tenantID, event, payload)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "outbox_events"`)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(entry.CreatedAt, entry.UpdatedAt))
	mock.ExpectCommit()

	err := repo.Save(ctx, entry)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormOutboxRepository_Save_Empty(t *testing.T) {
	db, _ := setupMockDB(t)
	repo := NewGormOutboxRepository(db)
	ctx := context.Background()

	err := repo.Save(ctx)

	require.NoError(t, err)
}

func TestGormOutboxRepository_FindPending(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewGormOutboxRepository(db)
	ctx := context.Background()

	entryID := uuid.New()
	tenantID := uuid.New()
	eventID := uuid.New()
	aggID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "event_id", "event_type", "aggregate_id",
		"aggregate_type", "payload", "status", "retry_count", "max_retries",
		"last_error", "next_retry_at", "processed_at", "created_at", "updated_at",
	}).AddRow(
		entryID, tenantID, eventID, "TestEvent", aggID,
		"TestAggregate", []byte(`{}`), "PENDING", 0, 5,
		"", nil, nil, now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "outbox_events" WHERE status = $1 ORDER BY created_at ASC LIMIT $2`)).
		WithArgs(shared.OutboxStatusPending, 10).
		WillReturnRows(rows)

	entries, err := repo.FindPending(ctx, 10)

	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entryID, entries[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormOutboxRepository_FindRetryable(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewGormOutboxRepository(db)
	ctx := context.Background()

	before := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "event_id", "event_type", "aggregate_id",
		"aggregate_type", "payload", "status", "retry_count", "max_retries",
		"last_error", "next_retry_at", "processed_at", "created_at", "updated_at",
	})

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "outbox_events" WHERE status = $1 AND next_retry_at <= $2 ORDER BY next_retry_at ASC LIMIT $3`)).
		WithArgs(shared.OutboxStatusFailed, before, 10).
		WillReturnRows(rows)

	entries, err := repo.FindRetryable(ctx, before, 10)

	require.NoError(t, err)
	assert.Len(t, entries, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormOutboxRepository_Update(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewGormOutboxRepository(db)
	ctx := context.Background()

	tenantID := uuid.New()
	event := newTestEvent("TestEvent", tenantID)
	entry := shared.NewOutboxEntry(tenantID, event, []byte(`{}`))
	entry.MarkSent()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outbox_events"`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Update(ctx, entry)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormOutboxRepository_DeleteOlderThan(t *testing.T) {
	db, mock := setupMockDB(t)
	repo := NewGormOutboxRepository(db)
	ctx := context.Background()

	before := time.Now().Add(-7 * 24 * time.Hour)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "outbox_events" WHERE status = $1 AND processed_at < $2`)).
		WithArgs(shared.OutboxStatusSent, before).
		WillReturnResult(sqlmock.NewResult(0, 5))
	mock.ExpectCommit()

	deleted, err := repo.DeleteOlderThan(ctx, before)

	require.NoError(t, err)
	assert.Equal(t, int64(5), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGormOutboxRepository_WithTx(t *testing.T) {
	db, _ := setupMockDB(t)
	repo := NewGormOutboxRepository(db)

	newRepo := repo.WithTx(db)

	assert.NotNil(t, newRepo)
	assert.NotSame(t, repo, newRepo)
}
