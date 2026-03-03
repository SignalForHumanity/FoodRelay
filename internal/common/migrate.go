package common

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// RunMigrations executes SQL files from fsys whose names start with prefix,
// in sorted order. Pass prefix="" to run all files.
// Files are idempotent via IF NOT EXISTS.
func RunMigrations(db *sql.DB, fsys fs.FS, prefix string) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			return fmt.Errorf("exec %s: %w", name, err)
		}
	}
	return nil
}
