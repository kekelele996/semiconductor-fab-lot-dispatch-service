package lots

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"time"
)

type LeaseCoordinator struct {
	store  *LeaseStore
	clock  platform.Clock
	events platform.EventSink
	ttl    time.Duration
}

func NewLeaseCoordinator(store *LeaseStore, clock platform.Clock, events platform.EventSink, ttl time.Duration) *LeaseCoordinator {
	return &LeaseCoordinator{store: store, clock: clock, events: events, ttl: ttl}
}
func (c *LeaseCoordinator) AcquireBest(ctx context.Context, owner string, candidates []Candidate) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return Lease{}, err
	}
	for _, candidate := range candidates {
		if !candidate.Feasible {
			continue
		}
		lease, err := c.store.TryAcquire(ctx, candidate.ID, owner, c.clock.Now(), c.ttl)
		if err == platform.ErrConflict {
			continue
		}
		if err != nil {
			return Lease{}, err
		}
		event := platform.Event{Topic: "lots.lease-acquired", Key: lease.LotID, Version: lease.Generation, At: c.clock.Now(), Attributes: map[string]string{"owner": owner}}
		if err = c.events.Publish(ctx, event); err != nil {
			c.rollbackFailedAcquire(lease)
			return Lease{}, platform.Wrap("publish", "lot-lease", lease.LotID, err)
		}
		return lease, nil
	}
	return Lease{}, platform.ErrCapacity
}
func (c *LeaseCoordinator) Release(ctx context.Context, lease Lease) error {
	if err := c.store.Release(ctx, lease); err != nil {
		return err
	}
	return c.events.Publish(ctx, platform.Event{Topic: "lots.lease-released", Key: lease.LotID, Version: lease.Generation, At: c.clock.Now(), Attributes: map[string]string{"owner": lease.Owner}})
}

func (c *LeaseCoordinator) rollbackFailedAcquire(lease Lease) {
	// Compensating cleanup must run even when the request that triggered it was
	// canceled: derive from context.Background so a canceled parent does not
	// short-circuit Release and leak the lease.
	rollbackCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, active := c.store.Active(lease.LotID, c.clock.Now()); !active {
		return
	}
	_ = c.store.Release(rollbackCtx, lease)
}
