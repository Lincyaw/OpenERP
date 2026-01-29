package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/erp/backend/internal/domain/catalog"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMockProductAttachmentRepository creates a GormProductAttachmentRepository with a mocked SQL connection
func newMockProductAttachmentRepository(t *testing.T) (*GormProductAttachmentRepository, sqlmock.Sqlmock, *sql.DB) {
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

	return NewGormProductAttachmentRepository(gormDB), mock, mockDB
}

func TestNewGormProductAttachmentRepository(t *testing.T) {
	t.Run("creates repository with valid DB", func(t *testing.T) {
		repo, _, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.db)
	})
}

func TestGormProductAttachmentRepository_FindByID(t *testing.T) {
	t.Run("finds existing attachment", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		attachmentID := uuid.New()
		tenantID := uuid.New()
		productID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			attachmentID, tenantID, productID, "main_image", "active", "image.jpg", int64(1024),
			"image/jpeg", "products/image.jpg", "products/thumbs/image.jpg", 0, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE id = \$1 ORDER BY .* LIMIT .*`).
			WithArgs(attachmentID, 1).
			WillReturnRows(rows)

		attachment, err := repo.FindByID(context.Background(), attachmentID)

		assert.NoError(t, err)
		assert.NotNil(t, attachment)
		assert.Equal(t, attachmentID, attachment.ID)
		assert.Equal(t, "image.jpg", attachment.FileName)
		assert.Equal(t, catalog.AttachmentTypeMainImage, attachment.Type)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent attachment", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		attachmentID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE id = \$1 ORDER BY .* LIMIT .*`).
			WithArgs(attachmentID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		attachment, err := repo.FindByID(context.Background(), attachmentID)

		assert.Error(t, err)
		assert.Nil(t, attachment)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_FindByIDForTenant(t *testing.T) {
	t.Run("finds attachment within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		attachmentID := uuid.New()
		tenantID := uuid.New()
		productID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			attachmentID, tenantID, productID, "gallery_image", "active", "gallery.jpg", int64(2048),
			"image/jpeg", "products/gallery.jpg", "", 1, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND id = \$2 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, attachmentID, 1).
			WillReturnRows(rows)

		attachment, err := repo.FindByIDForTenant(context.Background(), tenantID, attachmentID)

		assert.NoError(t, err)
		assert.NotNil(t, attachment)
		assert.Equal(t, tenantID, attachment.TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_FindByIDs(t *testing.T) {
	t.Run("finds multiple attachments by IDs", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			id1, tenantID, productID, "main_image", "active", "main.jpg", int64(1024),
			"image/jpeg", "products/main.jpg", "", 0, nil,
			now, now, 1,
		).AddRow(
			id2, tenantID, productID, "gallery_image", "active", "gallery.jpg", int64(2048),
			"image/jpeg", "products/gallery.jpg", "", 1, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND id IN \(\$2,\$3\) ORDER BY sort_order ASC`).
			WithArgs(tenantID, id1, id2).
			WillReturnRows(rows)

		attachments, err := repo.FindByIDs(context.Background(), tenantID, []uuid.UUID{id1, id2})

		assert.NoError(t, err)
		assert.Len(t, attachments, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice for empty IDs", func(t *testing.T) {
		repo, _, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		attachments, err := repo.FindByIDs(context.Background(), uuid.New(), []uuid.UUID{})

		assert.NoError(t, err)
		assert.Empty(t, attachments)
	})
}

func TestGormProductAttachmentRepository_FindByProduct(t *testing.T) {
	t.Run("finds attachments for product", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()
		id1 := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			id1, tenantID, productID, "main_image", "active", "main.jpg", int64(1024),
			"image/jpeg", "products/main.jpg", "", 0, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 ORDER BY sort_order ASC, created_at ASC`).
			WithArgs(tenantID, productID).
			WillReturnRows(rows)

		attachments, err := repo.FindByProduct(context.Background(), tenantID, productID, shared.Filter{})

		assert.NoError(t, err)
		assert.Len(t, attachments, 1)
		assert.Equal(t, productID, attachments[0].ProductID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_FindActiveByProduct(t *testing.T) {
	t.Run("finds active attachments for product", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()
		id1 := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			id1, tenantID, productID, "main_image", "active", "main.jpg", int64(1024),
			"image/jpeg", "products/main.jpg", "", 0, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND status = \$3 ORDER BY sort_order ASC, created_at ASC`).
			WithArgs(tenantID, productID, catalog.AttachmentStatusActive).
			WillReturnRows(rows)

		attachments, err := repo.FindActiveByProduct(context.Background(), tenantID, productID)

		assert.NoError(t, err)
		assert.Len(t, attachments, 1)
		assert.Equal(t, catalog.AttachmentStatusActive, attachments[0].Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_FindMainImage(t *testing.T) {
	t.Run("finds main image for product", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()
		attachmentID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			attachmentID, tenantID, productID, "main_image", "active", "main.jpg", int64(1024),
			"image/jpeg", "products/main.jpg", "products/thumbs/main.jpg", 0, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND type = \$3 AND status = \$4 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, productID, catalog.AttachmentTypeMainImage, catalog.AttachmentStatusActive, 1).
			WillReturnRows(rows)

		attachment, err := repo.FindMainImage(context.Background(), tenantID, productID)

		assert.NoError(t, err)
		assert.NotNil(t, attachment)
		assert.Equal(t, catalog.AttachmentTypeMainImage, attachment.Type)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when no main image exists", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND type = \$3 AND status = \$4 ORDER BY .* LIMIT .*`).
			WithArgs(tenantID, productID, catalog.AttachmentTypeMainImage, catalog.AttachmentStatusActive, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		attachment, err := repo.FindMainImage(context.Background(), tenantID, productID)

		assert.Error(t, err)
		assert.Nil(t, attachment)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_FindByType(t *testing.T) {
	t.Run("finds attachments by type", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			id1, tenantID, productID, "gallery_image", "active", "gallery1.jpg", int64(1024),
			"image/jpeg", "products/gallery1.jpg", "", 0, nil,
			now, now, 1,
		).AddRow(
			id2, tenantID, productID, "gallery_image", "active", "gallery2.jpg", int64(2048),
			"image/jpeg", "products/gallery2.jpg", "", 1, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND type = \$3 AND status = \$4 ORDER BY sort_order ASC, created_at ASC`).
			WithArgs(tenantID, productID, catalog.AttachmentTypeGalleryImage, catalog.AttachmentStatusActive).
			WillReturnRows(rows)

		attachments, err := repo.FindByType(context.Background(), tenantID, productID, catalog.AttachmentTypeGalleryImage)

		assert.NoError(t, err)
		assert.Len(t, attachments, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_CountByProduct(t *testing.T) {
	t.Run("counts all attachments for product", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2`).
			WithArgs(tenantID, productID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.CountByProduct(context.Background(), tenantID, productID)

		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_CountActiveByProduct(t *testing.T) {
	t.Run("counts active attachments for product", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT count\(\*\) FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND status = \$3`).
			WithArgs(tenantID, productID, catalog.AttachmentStatusActive).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		count, err := repo.CountActiveByProduct(context.Background(), tenantID, productID)

		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_ExistsByStorageKey(t *testing.T) {
	t.Run("returns true when storage key exists", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		storageKey := "products/image.jpg"

		mock.ExpectQuery(`SELECT count\(\*\) FROM "product_attachments" WHERE tenant_id = \$1 AND storage_key = \$2`).
			WithArgs(tenantID, storageKey).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := repo.ExistsByStorageKey(context.Background(), tenantID, storageKey)

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when storage key does not exist", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		storageKey := "products/nonexistent.jpg"

		mock.ExpectQuery(`SELECT count\(\*\) FROM "product_attachments" WHERE tenant_id = \$1 AND storage_key = \$2`).
			WithArgs(tenantID, storageKey).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := repo.ExistsByStorageKey(context.Background(), tenantID, storageKey)

		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_Save(t *testing.T) {
	t.Run("saves attachment", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()
		attachment, _ := catalog.NewProductAttachment(
			tenantID,
			productID,
			catalog.AttachmentTypeMainImage,
			"image.jpg",
			1024,
			"image/jpeg",
			"products/image.jpg",
			nil,
		)

		mock.ExpectExec(`UPDATE "product_attachments" SET`).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Save(context.Background(), attachment)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_SaveBatch(t *testing.T) {
	t.Run("returns nil for empty batch", func(t *testing.T) {
		repo, _, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		err := repo.SaveBatch(context.Background(), []*catalog.ProductAttachment{})

		assert.NoError(t, err)
	})
}

func TestGormProductAttachmentRepository_Delete(t *testing.T) {
	t.Run("deletes existing attachment", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		attachmentID := uuid.New()

		mock.ExpectExec(`DELETE FROM "product_attachments" WHERE id = \$1`).
			WithArgs(attachmentID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(context.Background(), attachmentID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for non-existent attachment", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		attachmentID := uuid.New()

		mock.ExpectExec(`DELETE FROM "product_attachments" WHERE id = \$1`).
			WithArgs(attachmentID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(context.Background(), attachmentID)

		assert.Error(t, err)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_DeleteForTenant(t *testing.T) {
	t.Run("deletes attachment within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		attachmentID := uuid.New()

		mock.ExpectExec(`DELETE FROM "product_attachments" WHERE tenant_id = \$1 AND id = \$2`).
			WithArgs(tenantID, attachmentID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteForTenant(context.Background(), tenantID, attachmentID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_DeleteByProduct(t *testing.T) {
	t.Run("deletes all attachments for product", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()

		mock.ExpectExec(`DELETE FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2`).
			WithArgs(tenantID, productID).
			WillReturnResult(sqlmock.NewResult(0, 3))

		err := repo.DeleteByProduct(context.Background(), tenantID, productID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_GetMaxSortOrder(t *testing.T) {
	t.Run("returns max sort order", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND status = \$3`).
			WithArgs(tenantID, productID, catalog.AttachmentStatusActive).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(5))

		maxOrder, err := repo.GetMaxSortOrder(context.Background(), tenantID, productID)

		assert.NoError(t, err)
		assert.Equal(t, 5, maxOrder)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns -1 when no attachments exist", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()

		mock.ExpectQuery(`SELECT MAX\(sort_order\) FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND status = \$3`).
			WithArgs(tenantID, productID, catalog.AttachmentStatusActive).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(nil))

		maxOrder, err := repo.GetMaxSortOrder(context.Background(), tenantID, productID)

		assert.NoError(t, err)
		assert.Equal(t, -1, maxOrder)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_InterfaceCompliance(t *testing.T) {
	t.Run("implements ProductAttachmentRepository interface", func(t *testing.T) {
		repo, _, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		var _ catalog.ProductAttachmentRepository = repo
		var _ catalog.ProductAttachmentReader = repo
		var _ catalog.ProductAttachmentFinder = repo
		var _ catalog.ProductAttachmentWriter = repo
	})
}

func TestGormProductAttachmentRepository_FindByProductAndStatus(t *testing.T) {
	t.Run("finds attachments by product and status", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()
		id1 := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			id1, tenantID, productID, "main_image", "pending", "main.jpg", int64(1024),
			"image/jpeg", "products/main.jpg", "", 0, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND status = \$3 ORDER BY sort_order ASC, created_at ASC`).
			WithArgs(tenantID, productID, catalog.AttachmentStatusPending).
			WillReturnRows(rows)

		attachments, err := repo.FindByProductAndStatus(context.Background(), tenantID, productID, catalog.AttachmentStatusPending, shared.Filter{})

		assert.NoError(t, err)
		assert.Len(t, attachments, 1)
		assert.Equal(t, catalog.AttachmentStatusPending, attachments[0].Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when no matches", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		})

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND product_id = \$2 AND status = \$3 ORDER BY sort_order ASC, created_at ASC`).
			WithArgs(tenantID, productID, catalog.AttachmentStatusDeleted).
			WillReturnRows(rows)

		attachments, err := repo.FindByProductAndStatus(context.Background(), tenantID, productID, catalog.AttachmentStatusDeleted, shared.Filter{})

		assert.NoError(t, err)
		assert.Empty(t, attachments)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_FindPendingOlderThan(t *testing.T) {
	t.Run("finds old pending attachments", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		productID := uuid.New()
		id1 := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		}).AddRow(
			id1, tenantID, productID, "main_image", "pending", "main.jpg", int64(1024),
			"image/jpeg", "products/main.jpg", "", 0, nil,
			now, now, 1,
		)

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND status = \$2 AND created_at < NOW\(\) - INTERVAL '1 second' \* \$3 ORDER BY created_at ASC`).
			WithArgs(tenantID, catalog.AttachmentStatusPending, int64(3600)).
			WillReturnRows(rows)

		attachments, err := repo.FindPendingOlderThan(context.Background(), tenantID, 3600)

		assert.NoError(t, err)
		assert.Len(t, attachments, 1)
		assert.Equal(t, catalog.AttachmentStatusPending, attachments[0].Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when no old pending attachments", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "type", "status", "file_name", "file_size",
			"content_type", "storage_key", "thumbnail_key", "sort_order", "uploaded_by",
			"created_at", "updated_at", "version",
		})

		mock.ExpectQuery(`SELECT \* FROM "product_attachments" WHERE tenant_id = \$1 AND status = \$2 AND created_at < NOW\(\) - INTERVAL '1 second' \* \$3 ORDER BY created_at ASC`).
			WithArgs(tenantID, catalog.AttachmentStatusPending, int64(3600)).
			WillReturnRows(rows)

		attachments, err := repo.FindPendingOlderThan(context.Background(), tenantID, 3600)

		assert.NoError(t, err)
		assert.Empty(t, attachments)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGormProductAttachmentRepository_DeleteForTenant_NotFound(t *testing.T) {
	t.Run("returns error for non-existent attachment within tenant", func(t *testing.T) {
		repo, mock, mockDB := newMockProductAttachmentRepository(t)
		defer mockDB.Close()

		tenantID := uuid.New()
		attachmentID := uuid.New()

		mock.ExpectExec(`DELETE FROM "product_attachments" WHERE tenant_id = \$1 AND id = \$2`).
			WithArgs(tenantID, attachmentID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.DeleteForTenant(context.Background(), tenantID, attachmentID)

		assert.Error(t, err)
		assert.Equal(t, shared.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProductAttachmentSortFields(t *testing.T) {
	t.Run("contains expected fields", func(t *testing.T) {
		expectedFields := []string{
			"id", "created_at", "updated_at", "product_id", "type",
			"status", "file_name", "file_size", "content_type", "sort_order",
		}

		for _, field := range expectedFields {
			assert.True(t, ProductAttachmentSortFields[field], "Expected field %s to be in ProductAttachmentSortFields", field)
		}
	})

	t.Run("does not contain dangerous fields", func(t *testing.T) {
		dangerousFields := []string{
			"tenant_id", "storage_key", "thumbnail_key", "uploaded_by",
		}

		for _, field := range dangerousFields {
			assert.False(t, ProductAttachmentSortFields[field], "Field %s should not be in ProductAttachmentSortFields for security", field)
		}
	})
}
