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
