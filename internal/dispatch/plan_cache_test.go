package dispatch

import (
	"sync"
	"testing"
)

func TestPlanCachePublishesImmutableSnapshots(t *testing.T) {
	cache := NewPlanCache()
	input := PlanInput{FabID: "fab", Version: 1, Lots: []string{"b", "a"}, EligibleTools: map[string][]string{"a": {"t1"}, "b": {"t2"}}, Scores: map[string]int64{"a": 9, "b": 5}}
	plan := BuildPlan(input)
	cache.Publish(plan)
	plan.Assignments["a"] = "corrupt"
	plan.Order[0] = "corrupt"
	got, _ := cache.Snapshot("fab")
	got.Assignments["a"] = "changed"
	got.Order[0] = "changed"
	again, _ := cache.Snapshot("fab")
	if again.Assignments["a"] != "t1" || again.Order[0] != "a" {
		t.Fatalf("cache polluted: %#v", again)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(v int64) {
			defer wg.Done()
			p := BuildPlan(input)
			p.Version = v
			cache.Publish(p)
			_, _ = cache.Snapshot("fab")
		}(int64(i))
	}
	wg.Wait()
}
