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
	c.mu.Lock()
	c.plans[plan.FabID] = plan
	c.mu.Unlock()
}
func (c *PlanCache) Snapshot(fab string) (DispatchPlan, bool) {
	c.mu.RLock()
	plan, ok := c.plans[fab]
	c.mu.RUnlock()
	if !ok {
		return DispatchPlan{}, false
	}
	return plan, true
}
func clonePlan(x DispatchPlan) DispatchPlan {
	return x
}
func cloneAssignments(in map[string]string) map[string]string { return in }
func cloneScores(in map[string]int64) map[string]int64        { return in }
