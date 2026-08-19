package qualifications

import (
	"context"
	"sync"
)

type Probe func(context.Context) (ProbeResult, error)
type ProbeRunner struct{ Parallelism int }

func workerContext(ctx context.Context) context.Context { return context.Background() }
func waitWorkers(ctx context.Context, wg *sync.WaitGroup) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		wg.Wait()
		return nil
	}
}
func (r ProbeRunner) Run(ctx context.Context, probes map[string]Probe) ([]ProbeResult, error) {
	names := make(chan string)
	set := NewProbeSet()
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	limit := r.Parallelism
	if limit < 1 {
		limit = 1
	}
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range names {
				result, err := probes[name](workerContext(ctx))
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				result.Probe = name
				set.Record(result)
			}
		}()
	}
	go func() {
		defer close(names)
		for name := range probes {
			select {
			case names <- name:
			case <-ctx.Done():
				return
			}
		}
	}()
	if err := waitWorkers(ctx, &wg); err != nil {
		return set.Snapshot(), err
	}
	select {
	case err := <-errCh:
		return set.Snapshot(), err
	default:
		return set.Snapshot(), nil
	}
}
