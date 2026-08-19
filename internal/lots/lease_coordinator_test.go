package lots

import (
	"context"
	"errors"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sync"
	"testing"
	"time"
)

type rejectingSink struct{}

func (rejectingSink) Publish(context.Context, platform.Event) error {
	return errors.New("broker unavailable")
}
func TestLeaseCoordinatorConcurrentSingleWinner(t *testing.T) {
	clock := platform.NewManualClock(time.Now())
	store := NewLeaseStore()
	c := NewLeaseCoordinator(store, clock, &platform.MemoryEventSink{}, time.Minute)
	cand := []Candidate{{ID: "lot-9", Feasible: true, Score: 99}}
	start := make(chan struct{})
	var wg sync.WaitGroup
	wins := 0
	var mu sync.Mutex
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := c.AcquireBest(context.Background(), "dispatcher", cand); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()
	if wins != 1 {
		t.Fatalf("wins=%d", wins)
	}
}
func TestLeaseCoordinatorPublishFailureRollsBack(t *testing.T) {
	clock := platform.NewManualClock(time.Now())
	store := NewLeaseStore()
	c := NewLeaseCoordinator(store, clock, rejectingSink{}, time.Minute)
	_, err := c.AcquireBest(context.Background(), "dispatcher", []Candidate{{ID: "lot-7", Feasible: true}})
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := store.Active("lot-7", clock.Now()); ok {
		t.Fatal("lease leaked after publish failure")
	}
}
