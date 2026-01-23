package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// migrationTemplate is the default template for new migrations
const migrationUpTemplate = `-- Migration: {{.Name}}
-- Created: {{.Timestamp}}
-- Description: {{.Description}}

-- Write your UP migration SQL here

`

const migrationDownTemplate = `-- Migration: {{.Name}} (Rollback)
-- Created: {{.Timestamp}}
-- Description: Rollback for {{.Description}}

-- Write your DOWN migration SQL here

`

// MigrationFile represents a migration file pair
type MigrationFile struct {
	Version     string
	Name        string
	Description string
	Timestamp   string
	UpPath      string
	DownPath    string
}

// CreateMigration creates a new migration file pair
func CreateMigration(migrationsDir, name, description string) (*MigrationFile, error) {
	// Ensure migrations directory exists
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Generate version timestamp (YYYYMMDDHHMMSS format for sorting)
	now := time.Now()
	version := now.Format("20060102150405")
	timestamp := now.Format(time.RFC3339)

	// Create file names
	baseName := fmt.Sprintf("%s_%s", version, sanitizeName(name))
	upFileName := baseName + ".up.sql"
	downFileName := baseName + ".down.sql"

	upPath := filepath.Join(migrationsDir, upFileName)
	downPath := filepath.Join(migrationsDir, downFileName)

	mf := &MigrationFile{
		Version:     version,
		Name:        name,
		Description: description,
		Timestamp:   timestamp,
		UpPath:      upPath,
		DownPath:    downPath,
	}

	// Create up migration file
	if err := createMigrationFile(upPath, migrationUpTemplate, mf); err != nil {
		return nil, fmt.Errorf("failed to create up migration: %w", err)
	}

	// Create down migration file
	if err := createMigrationFile(downPath, migrationDownTemplate, mf); err != nil {
		// Clean up up file if down file creation fails
		_ = os.Remove(upPath)
		return nil, fmt.Errorf("failed to create down migration: %w", err)
	}

	return mf, nil
}

// createMigrationFile creates a single migration file from template
func createMigrationFile(path, tmplContent string, data *MigrationFile) error {
	tmpl, err := template.New("migration").Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// sanitizeName converts a migration name to a safe file name format
func sanitizeName(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z':
			result = append(result, c)
		case c >= 'A' && c <= 'Z':
			result = append(result, c+'a'-'A')
		case c >= '0' && c <= '9':
			result = append(result, c)
		case c == ' ' || c == '-' || c == '_':
			if len(result) > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
		}
	}
	// Trim trailing underscore
	if len(result) > 0 && result[len(result)-1] == '_' {
		result = result[:len(result)-1]
	}
	return string(result)
}

// ListMigrations returns a list of all migration files in a directory
func ListMigrations(migrationsDir string) ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrations := make([]string, 0)
	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Extract base name (without .up.sql or .down.sql)
		if len(name) > 7 && name[len(name)-7:] == ".up.sql" {
			baseName := name[:len(name)-7]
			if !seen[baseName] {
				seen[baseName] = true
				migrations = append(migrations, baseName)
			}
		}
	}

	return migrations, nil
}
