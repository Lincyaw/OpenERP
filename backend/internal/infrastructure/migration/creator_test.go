package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"add users table", "add_users_table"},
		{"Add-Users-Table", "add_users_table"},
		{"ADD_USERS_TABLE", "add_users_table"},
		{"add__users__table", "add_users_table"},
		{"Add Users 123", "add_users_123"},
		{"create-product-category", "create_product_category"},
		{"   spaces   ", "spaces"},
		{"special!@#$chars", "specialchars"},
		{"trailing_", "trailing"},
		{"_leading", "leading"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateMigration(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migrations_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test creating a migration
	mf, err := CreateMigration(tmpDir, "add users table", "Create users table with basic fields")
	require.NoError(t, err)
	assert.NotNil(t, mf)

	// Verify version format (YYYYMMDDHHMMSS - 14 digits)
	assert.Len(t, mf.Version, 14)

	// Verify file names
	assert.True(t, strings.HasSuffix(mf.UpPath, ".up.sql"))
	assert.True(t, strings.HasSuffix(mf.DownPath, ".down.sql"))

	// Verify base names match
	upBase := strings.TrimSuffix(filepath.Base(mf.UpPath), ".up.sql")
	downBase := strings.TrimSuffix(filepath.Base(mf.DownPath), ".down.sql")
	assert.Equal(t, upBase, downBase)

	// Verify files exist
	_, err = os.Stat(mf.UpPath)
	assert.NoError(t, err)
	_, err = os.Stat(mf.DownPath)
	assert.NoError(t, err)

	// Verify up file content
	upContent, err := os.ReadFile(mf.UpPath)
	require.NoError(t, err)
	assert.Contains(t, string(upContent), "add users table")
	assert.Contains(t, string(upContent), "Create users table with basic fields")
	assert.Contains(t, string(upContent), "Write your UP migration SQL here")

	// Verify down file content
	downContent, err := os.ReadFile(mf.DownPath)
	require.NoError(t, err)
	assert.Contains(t, string(downContent), "Rollback")
	assert.Contains(t, string(downContent), "Write your DOWN migration SQL here")
}

func TestCreateMigration_CreatesDirectory(t *testing.T) {
	// Create a path that doesn't exist
	tmpDir, err := os.MkdirTemp("", "migrations_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	nestedPath := filepath.Join(tmpDir, "nested", "migrations")

	mf, err := CreateMigration(nestedPath, "test", "test migration")
	require.NoError(t, err)
	assert.NotNil(t, mf)

	// Verify directory was created
	info, err := os.Stat(nestedPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestListMigrations(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migrations_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create some migration files
	files := []string{
		"000001_init_schema.up.sql",
		"000001_init_schema.down.sql",
		"000002_add_users.up.sql",
		"000002_add_users.down.sql",
		"000003_add_products.up.sql",
		"000003_add_products.down.sql",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		err := os.WriteFile(path, []byte("-- test"), 0644)
		require.NoError(t, err)
	}

	// List migrations
	migrations, err := ListMigrations(tmpDir)
	require.NoError(t, err)
	assert.Len(t, migrations, 3)

	// Verify migration names
	expected := []string{
		"000001_init_schema",
		"000002_add_users",
		"000003_add_products",
	}
	for _, exp := range expected {
		assert.Contains(t, migrations, exp)
	}
}

func TestListMigrations_EmptyDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migrations_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	migrations, err := ListMigrations(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, migrations)
}

func TestListMigrations_NonexistentDirectory(t *testing.T) {
	migrations, err := ListMigrations("/nonexistent/path/to/migrations")
	require.NoError(t, err)
	assert.Empty(t, migrations)
}

func TestListMigrations_IgnoresNonMigrationFiles(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migrations_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create migration and non-migration files
	files := []string{
		"000001_init.up.sql",
		"000001_init.down.sql",
		"README.md",
		"config.yaml",
		".gitkeep",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		err := os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	migrations, err := ListMigrations(tmpDir)
	require.NoError(t, err)
	assert.Len(t, migrations, 1)
	assert.Contains(t, migrations, "000001_init")
}

func TestListMigrations_IgnoresDirectories(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migrations_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a migration file and a subdirectory
	err = os.WriteFile(filepath.Join(tmpDir, "000001_init.up.sql"), []byte("test"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "000001_init.down.sql"), []byte("test"), 0644)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(tmpDir, "subdir.up.sql"), 0755)
	require.NoError(t, err)

	migrations, err := ListMigrations(tmpDir)
	require.NoError(t, err)
	assert.Len(t, migrations, 1)
}
