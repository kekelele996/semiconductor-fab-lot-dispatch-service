package batching

type WaferLot struct {
	ID, Recipe, Zone string
	Wafers           []int
	Attributes       map[string]string
	Priority         int
}

func (x WaferLot) Clone() WaferLot {
	out := x
	out.Wafers = append([]int(nil), x.Wafers...)
	out.Attributes = cloneAttrs(x.Attributes)
	return out
}
func cloneAttrs(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
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
	for i := range x.Lots {
		out.Lots[i] = x.Lots[i].Clone()
	}
	return out
}
