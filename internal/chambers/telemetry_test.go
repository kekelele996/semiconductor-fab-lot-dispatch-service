package chambers

import (
	"testing"
	"time"
)

func TestTelemetrySnapshotsDoNotShareMutableState(t *testing.T) {
	w := NewTelemetryWindow(3)
	input := TelemetrySample{ChamberID: "c1", At: time.Now(), Values: map[string]float64{"pressure": 1.2}, Labels: []string{"stable"}}
	w.Add(input)
	input.Values["pressure"] = 99
	input.Labels[0] = "bad"
	a := w.Snapshot()
	a[0].Values["pressure"] = 77
	a[0].Labels[0] = "changed"
	b := w.Snapshot()
	if b[0].Values["pressure"] != 1.2 || b[0].Labels[0] != "stable" {
		t.Fatalf("snapshot polluted: %#v", b[0])
	}
	summary := AggregateTelemetry(b)
	summary[0].Maximums["pressure"] = 55
	if AggregateTelemetry(w.Snapshot())[0].Maximums["pressure"] != 1.2 {
		t.Fatal("aggregate shares storage")
	}
}
