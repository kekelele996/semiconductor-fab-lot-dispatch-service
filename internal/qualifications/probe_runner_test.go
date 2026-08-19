package qualifications

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProbeRunnerCancelsAndJoinsEveryProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var active atomic.Int32
	started := make(chan struct{}, 6)
	barrier := make(chan struct{})
	done := make(chan error, 2)
	probe := func(ctx context.Context) (ProbeResult, error) {
		active.Add(1)
		defer active.Add(-1)
		started <- struct{}{}
		<-ctx.Done()
		return ProbeResult{}, ctx.Err()
	}
	run := func(prefix string) {
		<-barrier
		_, err := ProbeRunner{Parallelism: 3}.Run(ctx, map[string]Probe{prefix + "a": probe, prefix + "b": probe, prefix + "c": probe})
		done <- err
	}
	go run("x")
	go run("y")
	close(barrier)
	for i := 0; i < 6; i++ {
		<-started
	}
	cancel()
	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("err=%v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("runner stuck")
		}
	}
	if active.Load() != 0 {
		t.Fatalf("active=%d", active.Load())
	}
}
func TestProbeSetConcurrentRecordAndSnapshot(t *testing.T) {
	set := NewProbeSet()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; set.Record(ProbeResult{Probe: fmt.Sprintf("p-%d", i)}) }(i)
	}
	wg.Add(1)
	go func() { defer wg.Done(); <-start; _ = set.Snapshot() }()
	close(start)
	wg.Wait()
	if len(set.Snapshot()) != 8 {
		t.Fatalf("results=%d", len(set.Snapshot()))
	}
}
func TestProbeGateObservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	gate := NewProbeGate(ctx)
	cancel()
	if !errors.Is(gate.Allow(), context.Canceled) {
		t.Fatalf("gate err=%v", gate.Allow())
	}
}
func TestProbeReportWaitsForEveryCompletion(t *testing.T) {
	report := &ProbeReport{}
	report.Start(2)
	start := make(chan struct{})
	go func() { <-start; report.Complete(ProbeResult{Probe: "a"}) }()
	go func() { <-start; time.Sleep(20 * time.Millisecond); report.Complete(ProbeResult{Probe: "b"}) }()
	close(start)
	got := report.Wait()
	if len(got) != 2 {
		t.Fatalf("results=%d", len(got))
	}
}
