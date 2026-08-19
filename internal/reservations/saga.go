package reservations

import (
	"context"
	"errors"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"time"
)

type ReservationSaga struct {
	store  *SagaStore
	events platform.EventSink
	clock  platform.Clock
}

func NewReservationSaga(s *SagaStore, e platform.EventSink, c platform.Clock) *ReservationSaga {
	return &ReservationSaga{store: s, events: e, clock: c}
}
func (s *ReservationSaga) Execute(ctx context.Context, owner string, resources []string) ([]ResourceLease, error) {
	leases := make([]ResourceLease, 0, len(resources))
	for _, resource := range resources {
		x, err := s.store.Acquire(ctx, resource, owner)
		if err != nil {
			return nil, errors.Join(platform.Wrap("acquire", "reservation", resource, err), s.compensate(leases))
		}
		leases = append(leases, x)
		if err = s.events.Publish(ctx, platform.Event{Topic: "reservations.resource-acquired", Key: resource, Version: x.Generation, At: s.clock.Now(), Attributes: map[string]string{"owner": owner}}); err != nil {
			return nil, errors.Join(platform.Wrap("publish", "reservation", resource, err), s.compensate(leases))
		}
	}
	return leases, nil
}

// compensate releases every already-acquired lease so that a partial saga
// leaves no resource held. Leases are released in reverse acquisition order,
// each release error is preserved via MergeCleanupErrors, and the generation
// guard in SagaStore.Release prevents a stale compensation from dropping a
// lease a later acquire legitimately re-obtained.
func (s *ReservationSaga) compensate(leases []ResourceLease) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if len(leases) == 0 {
		return nil
	}
	errs := make([]error, 0, len(leases))
	for _, lease := range CompensationOrder(leases) {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			return MergeCleanupErrors(errs)
		}
		errs = append(errs, s.store.Release(ctx, lease))
	}
	return MergeCleanupErrors(errs)
}
