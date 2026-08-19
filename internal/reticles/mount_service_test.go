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

func TestMountServiceAcceptsNilRegistryConfiguration(t *testing.T) {
	svc := NewMountService(nil, &platform.MemoryEventSink{}, platform.NewManualClock(time.Now()))
	mounted, err := svc.MountForExposure(context.Background(), "rt-zero", "litho-2", "slot-zero", "clean")
	if err != nil {
		t.Fatal(err)
	}
	if mounted.ReticleID != "rt-zero" {
		t.Fatalf("mount=%#v", mounted)
	}
}
