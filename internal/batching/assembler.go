package batching

import "sort"

type Assembler struct {
	MaxWafers int
	MaxLots   int
}

func (a Assembler) Assemble(input []WaferLot, recipe, zone string) AssembledBatch {
	candidates := make([]WaferLot, 0, len(input))
	for _, lot := range input {
		if lot.Recipe == recipe && lot.Zone == zone {
			candidates = append(candidates, lot.Clone())
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Priority == candidates[j].Priority {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].Priority > candidates[j].Priority
	})
	out := AssembledBatch{Recipe: recipe, Zone: zone, Lots: make([]WaferLot, 0, len(candidates))}
	for _, lot := range candidates {
		if len(out.Lots) >= a.MaxLots {
			break
		}
		if out.WaferCount+len(lot.Wafers) > a.MaxWafers {
			continue
		}
		out.Lots = append(out.Lots, lot)
		out.WaferCount += len(lot.Wafers)
	}
	return out
}
