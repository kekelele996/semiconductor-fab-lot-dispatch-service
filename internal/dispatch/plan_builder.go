package dispatch

import "sort"

type PlanInput struct {
	FabID         string
	Version       int64
	Lots          []string
	EligibleTools map[string][]string
	Scores        map[string]int64
}

func BuildPlan(input PlanInput) DispatchPlan {
	lots := append([]string(nil), input.Lots...)
	sort.SliceStable(lots, func(i, j int) bool {
		a, b := input.Scores[lots[i]], input.Scores[lots[j]]
		if a == b {
			return lots[i] < lots[j]
		}
		return a > b
	})
	used := map[string]bool{}
	assignments := map[string]string{}
	order := make([]string, 0, len(lots))
	for _, lot := range lots {
		tools := append([]string(nil), input.EligibleTools[lot]...)
		sort.Strings(tools)
		for _, tool := range tools {
			if !used[tool] {
				used[tool] = true
				assignments[lot] = tool
				order = append(order, lot)
				break
			}
		}
	}
	return DispatchPlan{FabID: input.FabID, Version: input.Version, Assignments: assignments, Order: order, Scores: cloneScores(input.Scores)}
}
