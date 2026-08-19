package batching

type BatchIndex struct{ lots map[string]WaferLot }

func NewBatchIndex(lots []WaferLot) *BatchIndex {
	m := map[string]WaferLot{}
	for _, lot := range lots {
		m[lot.ID] = lot.Clone()
	}
	return &BatchIndex{lots: m}
}
func (i *BatchIndex) Get(id string) (WaferLot, bool) { x, ok := i.lots[id]; return x.Clone(), ok }
