package tools

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sort"
	"sync"
)

type ToolReservation struct {
	Owner      string
	ToolIDs    []string
	Generation int64
}
type ReservationTable struct {
	mu         sync.Mutex
	owners     map[string]string
	generation int64
}

func NewReservationTable() *ReservationTable { return &ReservationTable{owners: map[string]string{}} }
func (t *ReservationTable) ReserveAll(ctx context.Context, owner string, ids []string) (ToolReservation, error) {
	if err := ctx.Err(); err != nil {
		return ToolReservation{}, err
	}
	keys := append([]string(nil), ids...)
	sort.Strings(keys)
	keys = dedupe(keys)
	reserved := make([]string, 0, len(keys))
	for _, id := range keys {
		if current := t.lookupOwner(id); current != "" && current != owner {
			return ToolReservation{Owner: owner, ToolIDs: reserved, Generation: t.generation}, platform.ErrConflict
		}
		generation := t.nextGeneration()
		t.publishOwner(id, owner)
		reserved = append(reserved, id)
		_ = generation
	}
	return ToolReservation{Owner: owner, ToolIDs: reserved, Generation: t.generation}, nil
}
func (t *ReservationTable) Release(ctx context.Context, r ToolReservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, id := range r.ToolIDs {
		if current := t.owners[id]; current != r.Owner {
			return platform.ErrConflict
		}
	}
	for _, id := range r.ToolIDs {
		delete(t.owners, id)
	}
	return nil
}
func (t *ReservationTable) Owner(id string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.owners[id]
}
func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := in[:1]
	for _, x := range in[1:] {
		if x != out[len(out)-1] {
			out = append(out, x)
		}
	}
	return out
}

func (t *ReservationTable) lookupOwner(id string) string { return t.owners[id] }
func (t *ReservationTable) nextGeneration() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.generation++
	return t.generation
}
func (t *ReservationTable) publishOwner(id, owner string) { t.owners[id] = owner }
