package qualifications

import "sync"

type ProbeResult struct {
	Probe  string
	Passed bool
	Detail string
}
type ProbeSet struct {
	mu      sync.RWMutex
	results map[string]ProbeResult
}

func NewProbeSet() *ProbeSet { return &ProbeSet{results: map[string]ProbeResult{}} }
func (s *ProbeSet) Record(r ProbeResult) {
	s.mu.Lock()
	s.results[r.Probe] = r
	s.mu.Unlock()
}
func (s *ProbeSet) Snapshot() []ProbeResult {
	s.mu.RLock()
	out := make([]ProbeResult, 0, len(s.results))
	for _, r := range s.results {
		out = append(out, r)
	}
	s.mu.RUnlock()
	return out
}
