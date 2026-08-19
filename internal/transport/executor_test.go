package transport

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutorWaitsForWorkersOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var active atomic.Int32
	started := make(chan struct{}, 8)
	barrier := make(chan struct{})
	done := make(chan error, 2)
	worker := func(ctx context.Context, step MoveStep) error {
		active.Add(1)
		defer active.Add(-1)
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}
	run := func(prefix string) {
		<-barrier
		_, err := Executor{Parallelism: 4}.Execute(ctx, []MoveStep{{ID: prefix + "1"}, {ID: prefix + "2"}, {ID: prefix + "3"}, {ID: prefix + "4"}}, worker)
		done <- err
	}
	go run("a")
	go run("b")
	close(barrier)
	for i := 0; i < 8; i++ {
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
			t.Fatal("executor did not return")
		}
	}
	if active.Load() != 0 {
		t.Fatalf("workers still active: %d", active.Load())
	}
}

func TestResultLedgerSnapshotIsIsolated(t *testing.T) {
	ledger := &ResultLedger{}
	start := make(chan struct{})
	done := make(chan struct{}, 2)
	go func() { <-start; ledger.Append(StepResult{ID: "a"}); done <- struct{}{} }()
	go func() { <-start; ledger.Append(StepResult{ID: "b"}); done <- struct{}{} }()
	close(start)
	<-done
	<-done
	first := ledger.Snapshot()
	if len(first) != 2 {
		t.Fatalf("len=%d", len(first))
	}
	first[0].ID = "changed"
	second := ledger.Snapshot()
	if second[0].ID == "changed" {
		t.Fatalf("ledger snapshot mutated: %#v", second)
	}
}
