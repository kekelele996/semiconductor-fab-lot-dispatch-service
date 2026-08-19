package dispatch

import "sync"

type DispatchPlan struct {
	FabID       string
	Version     int64
	Assignments map[string]string
	Order       []string
	Scores      map[string]int64
}
type PlanCache struct {
	mu    sync.RWMutex
	plans map[string]DispatchPlan
}

func NewPlanCache() *PlanCache { return &PlanCache{plans: map[string]DispatchPlan{}} }
func (c *PlanCache) Publish(plan DispatchPlan) {
	copyPlan := clonePlan(plan)
	c.mu.Lock()
	c.plans[plan.FabID] = copyPlan
	c.mu.Unlock()
}
func (c *PlanCache) Snapshot(fab string) (DispatchPlan, bool) {
	c.mu.RLock()
	plan, ok := c.plans[fab]
	c.mu.RUnlock()
	if !ok {
		return DispatchPlan{}, false
	}
	return clonePlan(plan), true
}
func clonePlan(x DispatchPlan) DispatchPlan {
	x.Assignments = cloneAssignments(x.Assignments)
	x.Scores = cloneScores(x.Scores)
	x.Order = append([]string(nil), x.Order...)
	return x
}
func cloneAssignments(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func cloneScores(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
