package tools

import (
	"context"
	"errors"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sync"
	"testing"
	"time"
)

func TestReservationServiceOverlappingRoutesAreAtomic(t *testing.T) {
	table := NewReservationTable()
	svc := NewReservationService(table, &platform.MemoryEventSink{}, platform.NewManualClock(time.Now()))
	start := make(chan struct{})
	var wg sync.WaitGroup
	wins := 0
	var mu sync.Mutex
	for _, owner := range []string{"alpha", "beta"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			<-start
			if _, err := svc.ReserveRoute(context.Background(), owner, []string{"etch-1", "clean-1"}, []string{"metrology-1"}); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}(owner)
	}
	close(start)
	wg.Wait()
	if wins != 1 {
		t.Fatalf("wins=%d", wins)
	}
	owner := table.Owner("etch-1")
	if owner == "" || table.Owner("clean-1") != owner || table.Owner("metrology-1") != owner {
		t.Fatal("partial route reservation")
	}
}

type cancelingToolSink struct{ cancel context.CancelFunc }

func (s cancelingToolSink) Publish(context.Context, platform.Event) error {
	s.cancel()
	return errors.New("event canceled request")
}
func TestReservationServiceCanceledPublishReleasesWholeRoute(t *testing.T) {
	table := NewReservationTable()
	ctx, cancel := context.WithCancel(context.Background())
	svc := NewReservationService(table, cancelingToolSink{cancel: cancel}, platform.NewManualClock(time.Now()))
	_, err := svc.ReserveRoute(ctx, "lot-c", []string{"etch-2", "clean-2"}, []string{"metro-2"})
	if err == nil {
		t.Fatal("expected publish failure")
	}
	for _, id := range []string{"etch-2", "clean-2", "metro-2"} {
		if table.Owner(id) != "" {
			t.Fatalf("reservation leaked for %s", id)
		}
	}
}
