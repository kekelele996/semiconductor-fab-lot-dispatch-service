package qualifications

import "sync"

type ProbeReport struct {
	wg      sync.WaitGroup
	mu      sync.Mutex
	results []ProbeResult
}

func (r *ProbeReport) Start(n int) { r.wg.Add(n) }
func (r *ProbeReport) Complete(x ProbeResult) {
	r.mu.Lock()
	r.results = append(r.results, x)
	r.mu.Unlock()
	r.wg.Done()
}
func (r *ProbeReport) Wait() []ProbeResult {
	r.wg.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ProbeResult, len(r.results))
	copy(out, r.results)
	return out
}
