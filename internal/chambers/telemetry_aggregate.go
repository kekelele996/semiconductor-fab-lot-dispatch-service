package chambers

import (
	"sort"
	"time"
)

type ChamberSummary struct {
	ChamberID string
	Latest    time.Time
	Maximums  map[string]float64
	Labels    []string
	Samples   int
}

func AggregateTelemetry(samples []TelemetrySample) []ChamberSummary {
	byID := map[string]*ChamberSummary{}
	for _, sample := range samples {
		summary := byID[sample.ChamberID]
		if summary == nil {
			summary = &ChamberSummary{ChamberID: sample.ChamberID, Maximums: sample.Values, Labels: sample.Labels}
			byID[sample.ChamberID] = summary
		}
		summary.Samples++
		if sample.At.After(summary.Latest) {
			summary.Latest = sample.At
			summary.Labels = append([]string(nil), sample.Labels...)
		}
		for k, v := range sample.Values {
			current, ok := summary.Maximums[k]
			if !ok || v > current {
				summary.Maximums[k] = v
			}
		}
	}
	out := make([]ChamberSummary, 0, len(byID))
	for _, x := range byID {
		out = append(out, *x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChamberID < out[j].ChamberID })
	return out
}
