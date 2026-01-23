package event

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupPublisherMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestOutboxPublisher_PublishWithTx(t *testing.T) {
	db, mock := setupPublisherMockDB(t)
	serializer := NewEventSerializer()
	serializer.Register("TestEvent", &testEvent{})
	publisher := NewOutboxPublisher(serializer)
	ctx := context.Background()

	tenantID := uuid.New()
	event := newTestEvent("TestEvent", tenantID)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "outbox_events"`)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(event.OccurredAt(), event.OccurredAt()))
	mock.ExpectCommit()

	err := db.Transaction(func(tx *gorm.DB) error {
		return publisher.PublishWithTx(ctx, tx, event)
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxPublisher_PublishWithTx_MultipleEvents(t *testing.T) {
	db, mock := setupPublisherMockDB(t)
	serializer := NewEventSerializer()
	serializer.Register("TestEvent", &testEvent{})
	publisher := NewOutboxPublisher(serializer)
	ctx := context.Background()

	tenantID := uuid.New()
	events := []shared.DomainEvent{
		newTestEvent("TestEvent", tenantID),
		newTestEvent("TestEvent", tenantID),
		newTestEvent("TestEvent", tenantID),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "outbox_events"`)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(events[0].OccurredAt(), events[0].OccurredAt()).
			AddRow(events[1].OccurredAt(), events[1].OccurredAt()).
			AddRow(events[2].OccurredAt(), events[2].OccurredAt()))
	mock.ExpectCommit()

	err := db.Transaction(func(tx *gorm.DB) error {
		return publisher.PublishWithTx(ctx, tx, events...)
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxPublisher_PublishWithTx_EmptyEvents(t *testing.T) {
	db, mock := setupPublisherMockDB(t)
	serializer := NewEventSerializer()
	publisher := NewOutboxPublisher(serializer)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectCommit()

	err := db.Transaction(func(tx *gorm.DB) error {
		return publisher.PublishWithTx(ctx, tx)
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxPublisher_PublishWithTx_TransactionRollback(t *testing.T) {
	db, mock := setupPublisherMockDB(t)
	serializer := NewEventSerializer()
	serializer.Register("TestEvent", &testEvent{})
	publisher := NewOutboxPublisher(serializer)
	ctx := context.Background()

	tenantID := uuid.New()
	event := newTestEvent("TestEvent", tenantID)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "outbox_events"`)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(event.OccurredAt(), event.OccurredAt()))
	mock.ExpectRollback()

	testErr := errors.New("simulated error")
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := publisher.PublishWithTx(ctx, tx, event); err != nil {
			return err
		}
		return testErr
	})

	require.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
