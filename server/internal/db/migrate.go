package db

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

// migrationMeta stores applied migration versions.
type migrationMeta struct {
	Version int
	Dirty   bool
}

// RunMigrations executes SQL files in the migrations directory against db.
// Each file is run only once; already-applied files are skipped.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	// Ensure meta table exists
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, dirty INTEGER DEFAULT 0)`); err != nil {
		return fmt.Errorf("create meta table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	sort.Strings(files)

	for _, f := range files {
		// Extract version number from filename (e.g. 0001_init.sql → 1)
		base := filepath.Base(f)
		version := 0
		fmt.Sscanf(base, "%d", &version)

		// Check if already applied
		var count int
		db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ? AND dirty = 0", version).Scan(&count)
		if count > 0 {
			continue
		}

		b, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		sqls := strings.TrimSpace(string(b))
		if sqls == "" {
			continue
		}
		if _, err := db.Exec(sqls); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
		// Mark as applied
		if _, err := db.Exec("INSERT OR REPLACE INTO schema_migrations (version, dirty) VALUES (?, 0)", version); err != nil {
			return fmt.Errorf("record migration %s: %w", f, err)
		}
	}
	return nil
}
