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
	out := append([]ProbeResult(nil), r.results...)
	r.mu.Unlock()
	return out
}
