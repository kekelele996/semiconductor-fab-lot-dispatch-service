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
	jobs := make(chan MoveStep)
	ledger := &ResultLedger{}
	var wg sync.WaitGroup
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for step := range jobs {
				err := worker(context.Background(), step)
				ledger.Append(StepResult{ID: step.ID, Err: err})
				if err != nil {
					return
				}
			}
		}()
	}
	go func() {
		for _, step := range steps {
			select {
			case <-ctx.Done():
				return
			case jobs <- step:
			}
		}
		close(jobs)
	}()
	select {
	case <-ctx.Done():
		return ledger.Snapshot(), ctx.Err()
	default:
	}
	wg.Wait()
	results := ledger.Snapshot()
	for _, r := range results {
		if r.Err != nil {
			return results, r.Err
		}
	}
	return results, nil
}
