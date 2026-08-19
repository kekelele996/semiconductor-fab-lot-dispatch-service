package batching

type WaferLot struct {
	ID, Recipe, Zone string
	Wafers           []int
	Attributes       map[string]string
	Priority         int
}

func (x WaferLot) Clone() WaferLot                      { return x }
func cloneAttrs(in map[string]string) map[string]string { return in }

type AssembledBatch struct {
	Recipe, Zone string
	Lots         []WaferLot
	WaferCount   int
}

func (x AssembledBatch) Clone() AssembledBatch {
	out := x
	out.Lots = append([]WaferLot(nil), x.Lots...)
	return out
}
