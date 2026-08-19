package reservations

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sync"
)

type ResourceLease struct {
	Resource, Owner string
	Generation      int64
}
type SagaStore struct {
	mu         sync.Mutex
	owners     map[string]ResourceLease
	generation int64
}

func NewSagaStore() *SagaStore { return &SagaStore{owners: map[string]ResourceLease{}} }
func (s *SagaStore) Acquire(ctx context.Context, resource, owner string) (ResourceLease, error) {
	if err := ctx.Err(); err != nil {
		return ResourceLease{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.owners[resource]; ok {
		return ResourceLease{}, platform.ErrConflict
	}
	s.generation++
	x := ResourceLease{Resource: resource, Owner: owner, Generation: s.generation}
	s.owners[resource] = x
	return x, nil
}
func (s *SagaStore) Release(ctx context.Context, x ResourceLease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.owners[x.Resource]
	if !ok {
		return nil
	}
	if !sameLeaseHolder(current, x) {
		return platform.ErrConflict
	}
	delete(s.owners, x.Resource)
	return nil
}
func (s *SagaStore) Owner(resource string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.owners[resource].Owner
}

// sameLeaseHolder reports whether requested still holds the resource: the
// resource, owner, and generation must all match. The generation check stops a
// stale compensation (from an older acquire) from releasing a lease that a
// later acquire legitimately re-obtained.
func sameLeaseHolder(current, requested ResourceLease) bool {
	if current.Resource != requested.Resource {
		return false
	}
	if current.Owner != requested.Owner {
		return false
	}
	if current.Generation != requested.Generation {
		return false
	}
	return true
}
