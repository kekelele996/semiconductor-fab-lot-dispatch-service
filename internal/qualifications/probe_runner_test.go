package qualifications

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestProbeRunnerCancelsAndJoinsEveryProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var active atomic.Int32
	started := make(chan struct{}, 3)
	probe := func(ctx context.Context) (ProbeResult, error) {
		active.Add(1)
		defer active.Add(-1)
		started <- struct{}{}
		<-ctx.Done()
		return ProbeResult{}, ctx.Err()
	}
	done := make(chan error, 1)
	go func() {
		_, err := ProbeRunner{Parallelism: 3}.Run(ctx, map[string]Probe{"a": probe, "b": probe, "c": probe})
		done <- err
	}()
	for i := 0; i < 3; i++ {
		<-started
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runner stuck")
	}
	if active.Load() != 0 {
		t.Fatalf("active=%d", active.Load())
	}
}
