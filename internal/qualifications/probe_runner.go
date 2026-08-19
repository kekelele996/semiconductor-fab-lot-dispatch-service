package qualifications

import (
	"context"
	"sync"
)

type Probe func(context.Context) (ProbeResult, error)
type ProbeRunner struct{ Parallelism int }

func (r ProbeRunner) Run(ctx context.Context, probes map[string]Probe) ([]ProbeResult, error) {
	limit := r.Parallelism
	if limit < 1 {
		limit = 1
	}

	// Derive a cancellable context so that cancellation (parent or a probe
	// error) propagates to every in-flight probe, and defer cancel so Run never
	// leaves a probe running after it returns.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	names := make(chan string)
	set := NewProbeSet()
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range names {
				result, err := probes[name](ctx)
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					// Stop dispatching further probes and release the workers
					// still blocked on names.
					cancel()
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

	// Always join every worker before returning; never bail out early on
	// ctx.Done(), otherwise probes keep running in the background.
	wg.Wait()

	select {
	case err := <-errCh:
		return set.Snapshot(), err
	default:
		// If the run was cancelled but no probe surfaced an error itself
		// (e.g. a probe that ignores its context), still report cancellation.
		return set.Snapshot(), ctx.Err()
	}
}
