package batching

func FilterEligibleLots(input []WaferLot, recipe, zone string) []WaferLot {
	out := input[:0]
	for _, lot := range input {
		if lot.Recipe == recipe && lot.Zone == zone {
			out = append(out, lot)
		}
	}
	return out
}
