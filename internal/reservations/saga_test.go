package reservations

import (
	"context"
	"errors"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"testing"
	"time"
)

type nthFailSink struct{ n int }

func (s *nthFailSink) Publish(context.Context, platform.Event) error {
	s.n--
	if s.n == 0 {
		return errors.New("event failure")
	}
	return nil
}
func TestReservationSagaCompensatesEveryAcquiredResource(t *testing.T) {
	store := NewSagaStore()
	saga := NewReservationSaga(store, &nthFailSink{n: 3}, platform.NewManualClock(time.Now()))
	_, err := saga.Execute(context.Background(), "lot-a", []string{"tool", "reticle", "carrier", "transport"})
	if err == nil {
		t.Fatal("expected failure")
	}
	for _, resource := range []string{"tool", "reticle", "carrier", "transport"} {
		if owner := store.Owner(resource); owner != "" {
			t.Fatalf("leaked %s to %s", resource, owner)
		}
	}
}
