package reticles

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sync"
)

type Mount struct {
	ReticleID, SlotID, ToolID, Zone string
	Generation                      int64
}
type MountRegistry struct {
	mu         sync.Mutex
	byReticle  map[string]Mount
	bySlot     map[string]Mount
	generation int64
}

func NewMountRegistry() *MountRegistry {
	return &MountRegistry{byReticle: map[string]Mount{}, bySlot: map[string]Mount{}}
}
func (r *MountRegistry) Mount(ctx context.Context, m Mount) (Mount, error) {
	if err := ctx.Err(); err != nil {
		return Mount{}, err
	}
	if _, ok := r.byReticle[m.ReticleID]; ok {
		return Mount{}, platform.ErrConflict
	}
	if _, ok := r.bySlot[m.SlotID]; ok {
		return Mount{}, platform.ErrConflict
	}
	r.mu.Lock()
	r.generation++
	m.Generation = r.generation
	r.mu.Unlock()
	r.byReticle[m.ReticleID] = m
	r.bySlot[m.SlotID] = m
	return m, nil
}
func (r *MountRegistry) Unmount(ctx context.Context, m Mount) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.byReticle[m.ReticleID]
	if !ok {
		return nil
	}
	if current.Generation != m.Generation {
		return platform.ErrConflict
	}
	delete(r.byReticle, m.ReticleID)
	delete(r.bySlot, m.SlotID)
	return nil
}
func (r *MountRegistry) LookupSlot(id string) (Mount, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.bySlot[id]
	return x, ok
}
