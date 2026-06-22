package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations executes embedded SQL migration files against db.
// Each file is run only once; already-applied files are skipped.
func RunMigrations(db *sql.DB, _ string) error {
	// Ensure meta table exists
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, dirty INTEGER DEFAULT 0)`); err != nil {
		return fmt.Errorf("create meta table: %w", err)
	}

	entries, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Strings(entries)

	for _, name := range entries {
		// Extract version number from filename (e.g. 0001_init.sql → 1)
		base := filepath.Base(name)
		version := 0
		fmt.Sscanf(base, "%d", &version)

		// Check if already applied
		var count int
		db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ? AND dirty = 0", version).Scan(&count)
		if count > 0 {
			continue
		}

		b, err := migrationFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		sqls := strings.TrimSpace(string(b))
		if sqls == "" {
			continue
		}
		if _, err := db.Exec(sqls); err != nil {
			// Ignore "duplicate column" errors from ALTER TABLE ADD COLUMN
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("exec %s: %w", name, err)
		}
		// Mark as applied
		if _, err := db.Exec("INSERT OR REPLACE INTO schema_migrations (version, dirty) VALUES (?, 0)", version); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	return nil
}
