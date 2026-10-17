package geoupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMissingMMDBDoesNotPreventGeoSiteUpdate(t *testing.T) {
	dir := t.TempDir()
	site := filepath.Join(dir, "geosite.dat")
	if err := os.WriteFile(site, []byte("geo"), 0600); err != nil {
		t.Fatal(err)
	}
	updated := false
	databases := ExistingDatabases([]Database{
		{Path: filepath.Join(dir, "country.mmdb"), Update: func() error {
			t.Fatal("attempted to open missing MMDB after switching GEOIP mode")
			return nil
		}},
		{Path: site, Update: func() error { updated = true; return nil }},
	})
	if len(databases) != 1 || databases[0].Path != site {
		t.Fatalf("selected unexpected databases: %+v", databases)
	}
	if err := databases[0].Refresh(); err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("GeoSite was not updated")
	}
}

func TestRemovedDatabaseIsNotPassedToCoreUpdater(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "country.mmdb")
	if err := os.WriteFile(path, []byte("geo"), 0600); err != nil {
		t.Fatal(err)
	}
	databases := ExistingDatabases([]Database{{Path: path, Update: func() error {
		t.Fatal("core updater was called for a removed file")
		return nil
	}}})
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := databases[0].Refresh(); err == nil {
		t.Fatal("removed database should report an error")
	}
}
