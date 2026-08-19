package batching

type WaferLot struct {
	ID, Recipe, Zone string
	Wafers           []int
	Attributes       map[string]string
	Priority         int
}

func (x WaferLot) Clone() WaferLot {
	x.Wafers = append([]int(nil), x.Wafers...)
	x.Attributes = cloneAttrs(x.Attributes)
	return x
}
func cloneAttrs(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

type AssembledBatch struct {
	Recipe, Zone string
	Lots         []WaferLot
	WaferCount   int
}

func (x AssembledBatch) Clone() AssembledBatch {
	out := x
	out.Lots = make([]WaferLot, len(x.Lots))
	for i, lot := range x.Lots {
		out.Lots[i] = lot.Clone()
	}
	return out
}
