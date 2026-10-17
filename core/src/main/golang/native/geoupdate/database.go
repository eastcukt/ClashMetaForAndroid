package geoupdate

import (
	"fmt"
	"os"
)

// Database describes one local database and its core refresh operation.
type Database struct {
	Path   string
	Update func() error
}

// ExistingDatabases excludes uninstalled formats. In particular, switching
// GEOIP modes must not call an MMDB refresh that lazily opens a missing old file.
func ExistingDatabases(candidates []Database) []Database {
	var databases []Database
	for _, database := range candidates {
		if info, err := os.Stat(database.Path); err == nil && info.Mode().IsRegular() {
			databases = append(databases, database)
		}
	}
	return databases
}

func (d Database) Refresh() error {
	// The file can disappear after scheduling (for example, an external import).
	if info, err := os.Stat(d.Path); err != nil {
		return fmt.Errorf("GEO database unavailable: %w", err)
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("GEO database is not a regular file: %s", d.Path)
	}
	return d.Update()
}
