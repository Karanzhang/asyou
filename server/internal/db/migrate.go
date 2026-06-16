package db

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations executes SQL files in the migrations directory against db
func RunMigrations(db *sql.DB, migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	sort.Strings(files)
	for _, f := range files {
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
	}
	return nil
}

// RunMigrationsForDemo is kept for backward compatibility (creates JSON files)
func RunMigrationsForDemo(migrationsDir, dataDir string) error {
	return nil
}
