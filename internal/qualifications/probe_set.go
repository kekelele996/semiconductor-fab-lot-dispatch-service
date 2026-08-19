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

func NewProbeSet() *ProbeSet             { return &ProbeSet{results: map[string]ProbeResult{}} }
func (s *ProbeSet) Record(r ProbeResult) { s.results[r.Probe] = r }
func (s *ProbeSet) Snapshot() []ProbeResult {
	out := make([]ProbeResult, 0, len(s.results))
	for _, r := range s.results {
		out = append(out, r)
	}
	return out
}
