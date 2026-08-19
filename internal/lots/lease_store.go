package lots

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sync"
	"time"
)

type Lease struct {
	LotID, Owner string
	Generation   int64
	ExpiresAt    time.Time
}
type LeaseStore struct {
	mu         sync.Mutex
	leases     map[string]Lease
	generation int64
}

func NewLeaseStore() *LeaseStore { return &LeaseStore{leases: map[string]Lease{}} }
func (s *LeaseStore) TryAcquire(ctx context.Context, lotID, owner string, now time.Time, ttl time.Duration) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.leases[lotID]; ok && current.ExpiresAt.After(now) {
		return Lease{}, platform.ErrConflict
	}
	s.generation++
	lease := Lease{LotID: lotID, Owner: owner, Generation: s.generation, ExpiresAt: now.Add(ttl)}
	s.leases[lotID] = lease
	return lease, nil
}
func (s *LeaseStore) Release(ctx context.Context, lease Lease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.leases[lease.LotID]; !ok {
		return nil
	}
	delete(s.leases, lease.LotID)
	return nil
}
func (s *LeaseStore) Active(lotID string, now time.Time) (Lease, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.leases[lotID]
	if ok && !x.ExpiresAt.After(now) {
		delete(s.leases, lotID)
		return Lease{}, false
	}
	return x, ok
}
