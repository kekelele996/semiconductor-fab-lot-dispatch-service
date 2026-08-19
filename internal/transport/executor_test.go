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
	var active atomic.Int32
	started := make(chan struct{}, 4)
	worker := func(ctx context.Context, step MoveStep) error {
		active.Add(1)
		defer active.Add(-1)
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}
	done := make(chan error, 1)
	go func() {
		_, err := Executor{Parallelism: 4}.Execute(ctx, []MoveStep{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}}, worker)
		done <- err
	}()
	for i := 0; i < 4; i++ {
		<-started
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("executor did not return")
	}
	if active.Load() != 0 {
		t.Fatalf("workers still active: %d", active.Load())
	}
}
