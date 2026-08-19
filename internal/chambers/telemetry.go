package chambers

import (
	"sync"
	"time"
)

type TelemetrySample struct {
	ChamberID string
	At        time.Time
	Values    map[string]float64
	Labels    []string
}
type TelemetryWindow struct {
	mu       sync.RWMutex
	capacity int
	samples  []TelemetrySample
}

func NewTelemetryWindow(capacity int) *TelemetryWindow { return &TelemetryWindow{capacity: capacity} }
func (w *TelemetryWindow) Add(sample TelemetrySample) {
	w.mu.Lock()
	defer w.mu.Unlock()
	sample = cloneSample(sample)
	w.samples = append(w.samples, sample)
	if len(w.samples) > w.capacity {
		drop := len(w.samples) - w.capacity
		next := make([]TelemetrySample, w.capacity)
		copy(next, w.samples[drop:])
		w.samples = next
	}
}
func (w *TelemetryWindow) Snapshot() []TelemetrySample {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]TelemetrySample, len(w.samples))
	for i, x := range w.samples {
		out[i] = cloneSample(x)
	}
	return out
}
func cloneSample(x TelemetrySample) TelemetrySample {
	x.Values = cloneValues(x.Values)
	x.Labels = append([]string(nil), x.Labels...)
	return x
}
func cloneValues(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
