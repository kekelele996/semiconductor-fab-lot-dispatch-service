package transport

import (
	"context"
	"sync"
)

type Executor struct{ Parallelism int }

func (e Executor) Execute(ctx context.Context, steps []MoveStep, worker StepWorker) ([]StepResult, error) {
	limit := e.Parallelism
	if limit < 1 {
		limit = 1
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan MoveStep)
	ledger := &ResultLedger{}
	var wg sync.WaitGroup
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-runCtx.Done():
					return
				case step, ok := <-jobs:
					if !ok {
						return
					}
					err := worker(runCtx, step)
					ledger.Append(StepResult{ID: step.ID, Err: err})
					if err != nil {
						cancel()
						return
					}
				}
			}
		}()
	}
	sendDone := make(chan struct{})
	go func() {
		defer close(sendDone)
		defer close(jobs)
		for _, step := range steps {
			select {
			case <-runCtx.Done():
				return
			case jobs <- step:
			}
		}
	}()
	<-sendDone
	wg.Wait()
	results := ledger.Snapshot()
	if err := ctx.Err(); err != nil {
		return results, err
	}
	for _, r := range results {
		if r.Err != nil {
			return results, r.Err
		}
	}
	return results, nil
}
