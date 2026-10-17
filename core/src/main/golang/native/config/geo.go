package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cfa/native/geoupdate"

	"github.com/metacubex/mihomo/component/mmdb"
	"github.com/metacubex/mihomo/component/updater"
	"github.com/metacubex/mihomo/config"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/log"
	"golang.org/x/sync/errgroup"
)

// Parsing temporarily changes the core's GEO settings, so profile validation,
// config application and GEO downloads must not run at the same time.
var configMu sync.Mutex
var geoUpdater geoupdate.Scheduler

// Called with configMu held, after the effective configuration is applied.
func syncGeoUpdater(cfg *config.Config) {
	geoUpdater.Stop()
	if !cfg.General.GeoAutoUpdate {
		return
	}
	hours := int64(cfg.General.GeoUpdateInterval)
	if hours <= 0 || hours > int64((1<<63-1)/time.Hour) {
		log.Warnln("[GEO] Invalid update interval: %d hours", hours)
		return
	}
	// Core Geo*Enable flags survive profile validation and config switches, so
	// select installed files explicitly rather than using UpdateGeoDatabases.
	candidates := []geoupdate.Database{
		{Path: C.Path.GeoSite(), Update: updater.UpdateGeoSite},
		{Path: C.Path.ASN(), Update: checkedMMDBUpdate(C.Path.ASN(), updater.UpdateASN)},
	}
	if cfg.General.GeodataMode {
		candidates = append(candidates, geoupdate.Database{Path: C.Path.GeoIP(), Update: updater.UpdateGeoIp})
	} else {
		candidates = append(candidates, geoupdate.Database{Path: C.Path.MMDB(), Update: checkedMMDBUpdate(C.Path.MMDB(), updater.UpdateMMDB)})
	}
	databases := geoupdate.ExistingDatabases(candidates)
	var paths []string
	for _, database := range databases {
		paths = append(paths, database.Path)
	}
	if len(paths) == 0 {
		return
	}
	interval := time.Duration(hours) * time.Hour
	delay := geoupdate.InitialDelay(paths, interval, time.Now())
	log.Infoln("[GEO] Automatic updates enabled: interval %s, next check in %s", interval, delay)
	geoUpdater.Start(interval, delay, func(ctx context.Context) error {
		configMu.Lock()
		defer configMu.Unlock()
		// A config switch may cancel this task while it waits for the lock.
		if err := ctx.Err(); err != nil {
			return err
		}
		var updates errgroup.Group
		for _, database := range databases {
			database := database
			updates.Go(database.Refresh)
		}
		if err := updates.Wait(); err != nil {
			log.Warnln("[GEO] Automatic update failed; next check in %s: %s", interval, err)
			return fmt.Errorf("update GEO databases: %w", err)
		}
		log.Infoln("[GEO] Automatic update completed")
		return nil
	})
}

func checkedMMDBUpdate(path string, update func() error) func() error {
	return func() error {
		// The core updater closes a lazily opened old reader before writing.
		// Reject an unreadable old file before that fatal initialization path.
		if !mmdb.Verify(path) {
			return fmt.Errorf("cannot refresh invalid MMDB database: %s", path)
		}
		return update()
	}
}
