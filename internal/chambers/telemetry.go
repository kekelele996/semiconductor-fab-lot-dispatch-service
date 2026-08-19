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
	w.samples = append(w.samples, sample)
	if len(w.samples) > w.capacity {
		drop := len(w.samples) - w.capacity
		w.samples = w.samples[drop:]
	}
}
func (w *TelemetryWindow) Snapshot() []TelemetrySample {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]TelemetrySample, len(w.samples))
	copy(out, w.samples)
	return out
}
func cloneSample(x TelemetrySample) TelemetrySample        { return x }
func cloneValues(in map[string]float64) map[string]float64 { return in }
