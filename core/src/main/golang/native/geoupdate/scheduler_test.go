package geoupdate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func receive(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for GEO update task")
	}
}

func TestFailureWaitsForConfiguredInterval(t *testing.T) {
	var scheduler Scheduler
	defer scheduler.Stop()
	attempts := make(chan time.Time, 2)
	interval := 40 * time.Millisecond
	scheduler.Start(interval, 0, func(context.Context) error {
		select {
		case attempts <- time.Now():
		default:
		}
		return errors.New("offline")
	})
	nextAttempt := func() time.Time {
		t.Helper()
		select {
		case at := <-attempts:
			return at
		case <-time.After(3 * time.Second):
			t.Fatal("failed update did not continue on the next cycle")
			return time.Time{}
		}
	}
	first := nextAttempt()
	if elapsed := nextAttempt().Sub(first); elapsed < interval {
		t.Fatalf("next attempt started after %s, before the configured %s", elapsed, interval)
	}
}

func TestStopCancelsTask(t *testing.T) {
	var scheduler Scheduler
	defer scheduler.Stop()
	started := make(chan struct{})
	stopped := make(chan struct{})
	scheduler.Start(time.Hour, 0, func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	})
	receive(t, started)
	scheduler.Stop()
	receive(t, stopped)
}

func TestReplacementDoesNotOverlapRunningUpdate(t *testing.T) {
	var scheduler Scheduler
	defer scheduler.Stop()
	started := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	scheduler.Start(time.Hour, 0, func(context.Context) error {
		close(started)
		<-release // The core's existing update API cannot cancel in-flight HTTP.
		return nil
	})
	receive(t, started)
	replaced := make(chan struct{})
	scheduler.Start(time.Hour, 0, func(context.Context) error {
		close(replaced)
		return nil
	})
	select {
	case <-replaced:
		t.Fatal("new task overlapped the previous update")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	receive(t, replaced)
}

func TestCancelledWaitingTaskNeverUpdates(t *testing.T) {
	var scheduler Scheduler
	defer scheduler.Stop()
	scheduler.updateMu.Lock()
	obsolete := make(chan struct{}, 1)
	scheduler.Start(time.Hour, 0, func(context.Context) error {
		obsolete <- struct{}{}
		return nil
	})
	current := make(chan struct{})
	scheduler.Start(time.Hour, 0, func(context.Context) error {
		close(current)
		return nil
	})
	scheduler.updateMu.Unlock()
	receive(t, current)
	select {
	case <-obsolete:
		t.Fatal("cancelled task performed an update after replacement")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestInvalidIntervalStopsExistingTask(t *testing.T) {
	var scheduler Scheduler
	defer scheduler.Stop()
	started := make(chan struct{})
	stopped := make(chan struct{})
	scheduler.Start(time.Hour, 0, func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	})
	receive(t, started)
	scheduler.Start(0, 0, func(context.Context) error {
		t.Error("invalid interval started an update")
		return nil
	})
	receive(t, stopped)
}

func TestInitialDelay(t *testing.T) {
	now := time.Unix(1700000000, 0)
	dir := t.TempDir()
	makeFile := func(name string, age time.Duration) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("geo"), 0600); err != nil {
			t.Fatal(err)
		}
		modified := now.Add(-age)
		if err := os.Chtimes(path, modified, modified); err != nil {
			t.Fatal(err)
		}
		return path
	}
	fresh := makeFile("geosite.dat", time.Hour)
	older := makeFile("geoip.dat", 6*time.Hour)
	stale := makeFile("stale.dat", 25*time.Hour)
	future := makeFile("future.dat", -time.Hour)
	for _, test := range []struct {
		name  string
		paths []string
		want  time.Duration
	}{
		{"fresh", []string{fresh}, 23 * time.Hour},
		{"oldest used file", []string{fresh, older}, 18 * time.Hour},
		{"stale", []string{fresh, stale}, 0},
		{"missing", []string{filepath.Join(dir, "missing.dat")}, 0},
		{"clock moved backwards", []string{future}, 24 * time.Hour},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := InitialDelay(test.paths, 24*time.Hour, now); got != test.want {
				t.Fatalf("InitialDelay = %s, want %s", got, test.want)
			}
		})
	}
}
