package batching

func FilterEligibleLots(input []WaferLot, recipe, zone string) []WaferLot {
	out := make([]WaferLot, 0, len(input))
	for _, lot := range input {
		if lot.Recipe == recipe && lot.Zone == zone {
			out = append(out, lot.Clone())
		}
	}
	return out
}
