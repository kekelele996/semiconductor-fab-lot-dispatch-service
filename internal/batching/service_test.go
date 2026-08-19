package batching

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"testing"
	"time"
)

func TestServiceLifecycle(t *testing.T) {
	clock := platform.NewManualClock(time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC))
	svc := NewService(NewRepository(), DefaultPolicy(), clock, &platform.MemoryEventSink{})
	x, err := svc.Register(context.Background(), Batch{ID: "x-1", FabID: "fab-a", State: StateForming, Priority: 7, Quantity: 25, Deadline: clock.Now().Add(time.Hour), Constraints: map[string]string{"area": "bay-1"}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Apply(context.Background(), Command{ID: x.ID, ExpectedVersion: x.Version, TargetState: StateSealed, Reason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateSealed {
		t.Fatalf("state=%s", got.State)
	}
	ranked, err := svc.Candidates(context.Background(), "fab-a")
	if err != nil || len(ranked) != 1 {
		t.Fatalf("rank=%v err=%v", ranked, err)
	}
}
func TestRepositoryRejectsStaleVersion(t *testing.T) {
	clock := platform.NewManualClock(time.Now())
	svc := NewService(NewRepository(), DefaultPolicy(), clock, &platform.MemoryEventSink{})
	x, err := svc.Register(context.Background(), Batch{ID: "x-2", FabID: "fab-a", State: StateForming, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Apply(context.Background(), Command{ID: x.ID, ExpectedVersion: x.Version + 10, TargetState: StateSealed}); err == nil {
		t.Fatal("expected conflict")
	}
}
