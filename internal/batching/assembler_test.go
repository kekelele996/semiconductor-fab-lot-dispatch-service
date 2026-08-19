package batching

import (
	"reflect"
	"testing"
)

func TestAssemblerDoesNotMutateOrAliasInputLots(t *testing.T) {
	input := []WaferLot{{ID: "low", Recipe: "r", Zone: "z", Priority: 1, Wafers: []int{1, 2}, Attributes: map[string]string{"owner": "a"}}, {ID: "high", Recipe: "r", Zone: "z", Priority: 9, Wafers: []int{3, 4}, Attributes: map[string]string{"owner": "b"}}}
	before := []string{input[0].ID, input[1].ID}
	batch := Assembler{MaxWafers: 10, MaxLots: 2}.Assemble(input, "r", "z")
	if !reflect.DeepEqual(before, []string{input[0].ID, input[1].ID}) {
		t.Fatal("input reordered")
	}
	batch.Lots[0].Wafers[0] = 99
	batch.Lots[0].Attributes["owner"] = "changed"
	if input[1].Wafers[0] != 3 || input[1].Attributes["owner"] != "b" {
		t.Fatalf("input aliased: %#v", input[1])
	}
}

func TestAssembledBatchCloneOwnsNestedLotStorage(t *testing.T) {
	original := AssembledBatch{Recipe: "r", Zone: "z", Lots: []WaferLot{{ID: "x", Wafers: []int{1, 2}, Attributes: map[string]string{"owner": "fab"}}}}
	copyBatch := original.Clone()
	copyBatch.Lots[0].Wafers[0] = 99
	copyBatch.Lots[0].Attributes["owner"] = "changed"
	if original.Lots[0].Wafers[0] != 1 || original.Lots[0].Attributes["owner"] != "fab" {
		t.Fatalf("clone shares storage: %#v", original)
	}
}

func TestFilterEligibleLotsDoesNotCompactInput(t *testing.T) {
	input := []WaferLot{{ID: "skip", Recipe: "x", Zone: "z"}, {ID: "keep", Recipe: "r", Zone: "z", Wafers: []int{1}}}
	out := FilterEligibleLots(input, "r", "z")
	out[0].Wafers[0] = 9
	if input[0].ID != "skip" || input[1].Wafers[0] != 1 {
		t.Fatalf("input changed: %#v", input)
	}
}
func TestBatchIndexOwnsLotSnapshots(t *testing.T) {
	lots := []WaferLot{{ID: "a", Wafers: []int{1}, Attributes: map[string]string{"owner": "fab"}}}
	index := NewBatchIndex(lots)
	lots[0].Wafers[0] = 9
	first, _ := index.Get("a")
	first.Attributes["owner"] = "changed"
	second, _ := index.Get("a")
	if second.Wafers[0] != 1 || second.Attributes["owner"] != "fab" {
		t.Fatalf("index polluted: %#v", second)
	}
}
