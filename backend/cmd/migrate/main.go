package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/erp/backend/internal/infrastructure/logger"
	"github.com/erp/backend/internal/infrastructure/migration"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const defaultMigrationsPath = "migrations"

func main() {
	// Parse flags
	var (
		migrationsPath string
		logLevel       string
	)

	flag.StringVar(&migrationsPath, "path", "", "Path to migrations directory (default: ./migrations)")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Get command and arguments
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}
	command := args[0]

	// Initialize logger
	log, err := logger.New(&logger.Config{
		Level:      logLevel,
		Format:     "console",
		Output:     "stdout",
		TimeFormat: "2006-01-02 15:04:05",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync(log)
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Determine migrations path
	if migrationsPath == "" {
		// Try to find migrations directory relative to executable or current dir
		if _, err := os.Stat(defaultMigrationsPath); err == nil {
			migrationsPath = defaultMigrationsPath
		} else {
			// Try relative to executable
			execPath, err := os.Executable()
			if err == nil {
				execDir := filepath.Dir(execPath)
				candidatePath := filepath.Join(execDir, "..", "..", defaultMigrationsPath)
				if _, err := os.Stat(candidatePath); err == nil {
					migrationsPath = candidatePath
				}
			}
		}
		if migrationsPath == "" {
			migrationsPath = defaultMigrationsPath
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		log.Fatal("Failed to get absolute path", zap.Error(err))
	}
	migrationsPath = absPath

	log.Info("Migration CLI started",
		zap.String("command", command),
		zap.String("migrations_path", migrationsPath),
	)

	// Handle create command separately (doesn't need DB)
	if command == "create" {
		if len(args) < 2 {
			log.Fatal("Migration name required. Usage: migrate create <name> [description]")
		}
		name := args[1]
		description := ""
		if len(args) > 2 {
			description = args[2]
		}

		mf, err := migration.CreateMigration(migrationsPath, name, description)
		if err != nil {
			log.Fatal("Failed to create migration", zap.Error(err))
		}

		log.Info("Migration created successfully",
			zap.String("version", mf.Version),
			zap.String("up_file", mf.UpPath),
			zap.String("down_file", mf.DownPath),
		)
		return
	}

	// Handle list command (doesn't need DB connection)
	if command == "list" {
		migrations, err := migration.ListMigrations(migrationsPath)
		if err != nil {
			log.Fatal("Failed to list migrations", zap.Error(err))
		}

		if len(migrations) == 0 {
			log.Info("No migrations found")
			return
		}

		log.Info("Available migrations", zap.Int("count", len(migrations)))
		for _, m := range migrations {
			fmt.Println("  -", m)
		}
		return
	}

	// Commands that need database connection
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database", zap.Error(err))
	}

	// Create migrator
	m, err := migration.New(db, migrationsPath, log)
	if err != nil {
		log.Fatal("Failed to create migrator", zap.Error(err))
	}
	defer m.Close()

	// Execute command
	switch command {
	case "up":
		if err := m.Up(); err != nil {
			log.Fatal("Migration up failed", zap.Error(err))
		}

	case "down":
		if err := m.Down(); err != nil {
			log.Fatal("Migration down failed", zap.Error(err))
		}

	case "step":
		if len(args) < 2 {
			log.Fatal("Step count required. Usage: migrate step <n>")
		}
		n, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatal("Invalid step count", zap.String("value", args[1]))
		}
		if err := m.Steps(n); err != nil {
			log.Fatal("Migration step failed", zap.Error(err))
		}

	case "goto":
		if len(args) < 2 {
			log.Fatal("Version required. Usage: migrate goto <version>")
		}
		version, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			log.Fatal("Invalid version number", zap.String("value", args[1]))
		}
		if err := m.GoTo(uint(version)); err != nil {
			log.Fatal("Migration goto failed", zap.Error(err))
		}

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatal("Failed to get version", zap.Error(err))
		}
		if version == 0 {
			log.Info("No migrations applied")
		} else {
			log.Info("Current migration version",
				zap.Uint("version", version),
				zap.Bool("dirty", dirty),
			)
		}

	case "force":
		if len(args) < 2 {
			log.Fatal("Version required. Usage: migrate force <version>")
		}
		version, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatal("Invalid version number", zap.String("value", args[1]))
		}
		log.Warn("Forcing migration version - use with caution!")
		if err := m.Force(version); err != nil {
			log.Fatal("Force version failed", zap.Error(err))
		}

	case "drop":
		log.Warn("This will DROP all database objects. Are you sure? (use -confirm flag)")
		// For safety, require explicit confirmation
		confirm := false
		for _, arg := range args[1:] {
			if arg == "-confirm" || arg == "--confirm" {
				confirm = true
				break
			}
		}
		if !confirm {
			log.Fatal("Drop cancelled. Use 'migrate drop -confirm' to confirm.")
		}
		if err := m.Drop(); err != nil {
			log.Fatal("Drop failed", zap.Error(err))
		}

	default:
		log.Error("Unknown command", zap.String("command", command))
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ERP Database Migration Tool

Usage:
  migrate [flags] <command> [arguments]

Commands:
  up                    Apply all pending migrations
  down                  Roll back all migrations
  step <n>              Apply n migrations (positive=up, negative=down)
  goto <version>        Migrate to a specific version
  version               Show current migration version
  force <version>       Force set migration version (use with caution)
  drop -confirm         Drop all database objects (DANGEROUS)
  create <name> [desc]  Create a new migration file pair
  list                  List available migrations

Flags:
  -path string          Path to migrations directory (default: ./migrations)
  -log-level string     Log level: debug, info, warn, error (default: info)

Environment Variables:
  DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSL_MODE

Examples:
  # Apply all pending migrations
  migrate up

  # Roll back the last migration
  migrate step -1

  # Create a new migration
  migrate create add_users_table "Create users table with basic fields"

  # Check current version
  migrate version`)
}
