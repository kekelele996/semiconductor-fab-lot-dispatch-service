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
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	names := make(chan string)
	set := NewProbeSet()
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-runCtx.Done():
					return
				case name, ok := <-names:
					if !ok {
						return
					}
					result, err := probes[name](runCtx)
					if err != nil {
						select {
						case errCh <- err:
						default:
						}
						cancel()
						return
					}
					result.Probe = name
					set.Record(result)
				}
			}
		}()
	}
	go func() {
		defer close(names)
		for name := range probes {
			select {
			case <-runCtx.Done():
				return
			case names <- name:
			}
		}
	}()
	wg.Wait()
	select {
	case err := <-errCh:
		return set.Snapshot(), err
	default:
	}
	if err := ctx.Err(); err != nil {
		return set.Snapshot(), err
	}
	return set.Snapshot(), nil
}
