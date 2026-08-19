package reticles

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sync"
	"testing"
	"time"
)

func TestMountServiceInitializesAndSerializesSlotOwnership(t *testing.T) {
	registry := &MountRegistry{}
	svc := NewMountService(registry, &platform.MemoryEventSink{}, platform.NewManualClock(time.Now()))
	start := make(chan struct{})
	var wg sync.WaitGroup
	wins := 0
	var mu sync.Mutex
	for _, id := range []string{"rt-a", "rt-b"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			<-start
			if _, err := svc.MountForExposure(context.Background(), id, "litho-1", "slot-3", "clean"); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}(id)
	}
	close(start)
	wg.Wait()
	if wins != 1 {
		t.Fatalf("wins=%d", wins)
	}
	if _, ok := registry.LookupSlot("slot-3"); !ok {
		t.Fatal("slot not recorded")
	}
}
