// Package geoupdate manages the lifetime of CMFA's GEO update task.
package geoupdate

import (
	"context"
	"os"
	"sync"
	"time"
)

// Scheduler replaces its previous task on Start. Stop cancels future attempts;
// an update already inside the core is allowed to finish. Updates never overlap.
type Scheduler struct {
	mu       sync.Mutex
	updateMu sync.Mutex
	cancel   context.CancelFunc
}

func (s *Scheduler) Start(interval, delay time.Duration, update func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	if interval <= 0 {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.run(ctx, interval, delay, update)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
}

func (s *Scheduler) stop() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *Scheduler) run(ctx context.Context, interval, delay time.Duration, update func(context.Context) error) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		s.updateMu.Lock()
		if ctx.Err() != nil {
			s.updateMu.Unlock()
			return
		}
		_ = update(ctx)
		s.updateMu.Unlock()
		if ctx.Err() != nil {
			return
		}
		timer.Reset(interval)
	}
}

// InitialDelay checks the databases selected for the active configuration. The
// oldest file determines when to check; missing files are checked immediately.
func InitialDelay(paths []string, interval time.Duration, now time.Time) time.Duration {
	delay := interval
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return 0
		}
		age := now.Sub(info.ModTime())
		if age >= interval {
			return 0
		}
		if age > 0 && interval-age < delay {
			delay = interval - age
		}
	}
	return delay
}
