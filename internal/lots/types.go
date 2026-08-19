package lots

import "time"

type Lot struct {
	ID          string            `json:"id"`
	FabID       string            `json:"fab_id"`
	State       string            `json:"state"`
	Priority    int               `json:"priority"`
	Quantity    int               `json:"quantity"`
	Version     int64             `json:"version"`
	ReadyAt     time.Time         `json:"ready_at"`
	Deadline    time.Time         `json:"deadline"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Constraints map[string]string `json:"constraints"`
	Tags        []string          `json:"tags"`
}

const (
	StateQueued    = "queued"
	StateRunning   = "running"
	StateCompleted = "completed"
)

type Command struct {
	ID              string            `json:"id"`
	ExpectedVersion int64             `json:"expected_version"`
	TargetState     string            `json:"target_state"`
	Priority        int               `json:"priority"`
	Quantity        int               `json:"quantity"`
	Deadline        time.Time         `json:"deadline"`
	Constraints     map[string]string `json:"constraints"`
	Reason          string            `json:"reason"`
}
type Candidate struct {
	ID       string   `json:"id"`
	Score    int64    `json:"score"`
	Feasible bool     `json:"feasible"`
	Reasons  []string `json:"reasons"`
	Version  int64    `json:"version"`
}
type Snapshot struct {
	Items       []Lot     `json:"items"`
	Version     int64     `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
}

func (x Lot) Clone() Lot {
	x.Constraints = cloneMap(x.Constraints)
	x.Tags = append([]string(nil), x.Tags...)
	return x
}
func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
