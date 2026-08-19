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
func (s *ReservationSaga) compensate(leases []ResourceLease) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var joined error
	for i := len(leases) - 1; i >= 0; i-- {
		if err := s.store.Release(ctx, leases[i]); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}
