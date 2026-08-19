package capacity

import (
	"fmt"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sort"
	"time"
)

type Policy struct {
	MaxQuantity        int
	MinimumPriority    int
	AllowedTransitions map[string]map[string]bool
}

func DefaultPolicy() Policy {
	return Policy{MaxQuantity: 1000, AllowedTransitions: map[string]map[string]bool{StateOpen: {StateAllocated: true}, StateAllocated: {StateConsumed: true, StateOpen: true}, StateConsumed: {StateOpen: true}}}
}
func (p Policy) Validate(x CapacitySlot) error {
	if x.ID == "" || x.FabID == "" {
		return fmt.Errorf("%w: id and fab required", platform.ErrInvariant)
	}
	if x.Quantity < 0 || x.Quantity > p.MaxQuantity {
		return fmt.Errorf("%w: quantity %d", platform.ErrCapacity, x.Quantity)
	}
	if x.Priority < p.MinimumPriority {
		return fmt.Errorf("%w: priority floor", platform.ErrInvariant)
	}
	if !x.Deadline.IsZero() && !x.ReadyAt.IsZero() && x.Deadline.Before(x.ReadyAt) {
		return fmt.Errorf("%w: deadline before readiness", platform.ErrInvariant)
	}
	return nil
}
func (p Policy) CanTransition(from, to string) bool {
	return from == to || p.AllowedTransitions[from][to]
}
func (p Policy) Score(x CapacitySlot, now time.Time) Candidate {
	reasons := []string{}
	score := int64(x.Priority*1000 + x.Quantity)
	ok := true
	if !x.ReadyAt.IsZero() && x.ReadyAt.After(now) {
		ok = false
		reasons = append(reasons, "not-ready")
	}
	if !x.Deadline.IsZero() {
		m := int64(x.Deadline.Sub(now) / time.Minute)
		if m < 0 {
			score += 500000
			reasons = append(reasons, "late")
		} else if m < 120 {
			score += 100000 - m
			reasons = append(reasons, "due-soon")
		}
	}
	if x.State == StateConsumed {
		ok = false
		reasons = append(reasons, "terminal")
	}
	return Candidate{ID: x.ID, Score: score, Feasible: ok, Reasons: reasons, Version: x.Version}
}
func Rank(p Policy, items []CapacitySlot, now time.Time) []Candidate {
	out := make([]Candidate, 0, len(items))
	for _, x := range items {
		out = append(out, p.Score(x, now))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Feasible != out[j].Feasible {
			return out[i].Feasible
		}
		if out[i].Score == out[j].Score {
			return out[i].ID < out[j].ID
		}
		return out[i].Score > out[j].Score
	})
	return out
}
