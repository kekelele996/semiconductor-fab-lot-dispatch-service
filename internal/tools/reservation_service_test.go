package tools

import (
	"context"
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
