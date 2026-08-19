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

func TestSagaStoreRejectsStaleCompensationGeneration(t *testing.T) {
	store := NewSagaStore()
	old, err := store.Acquire(context.Background(), "reticle", "lot-a")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Release(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	newLease, err := store.Acquire(context.Background(), "reticle", "lot-a")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Release(context.Background(), old); !errors.Is(err, platform.ErrConflict) {
		t.Fatalf("stale release err=%v", err)
	}
	if owner := store.Owner("reticle"); owner != newLease.Owner {
		t.Fatalf("new lease removed: %q", owner)
	}
}

func TestCompensationOrderIsReverseAcquisition(t *testing.T) {
	leases := []ResourceLease{{Resource: "tool"}, {Resource: "reticle"}, {Resource: "carrier"}}
	got := CompensationOrder(leases)
	want := []string{"carrier", "reticle", "tool"}
	for i, x := range want {
		if got[i].Resource != x {
			t.Fatalf("order=%v", got)
		}
	}
}
func TestMergeCleanupErrorsPreservesEveryFailure(t *testing.T) {
	a := errors.New("tool cleanup")
	b := errors.New("reticle cleanup")
	err := MergeCleanupErrors([]error{a, nil, b})
	if !errors.Is(err, a) || !errors.Is(err, b) {
		t.Fatalf("merged=%v", err)
	}
}
