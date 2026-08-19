package platform

import (
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

type ManualClock struct {
	mu  sync.RWMutex
	now time.Time
}

func NewManualClock(t time.Time) *ManualClock  { return &ManualClock{now: t.UTC()} }
func (c *ManualClock) Now() time.Time          { c.mu.RLock(); defer c.mu.RUnlock(); return c.now }
func (c *ManualClock) Advance(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }
