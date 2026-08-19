package chambers

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"sort"
	"sync"
)

type Repository struct {
	mu      sync.RWMutex
	items   map[string]Chamber
	version int64
}

func NewRepository() *Repository { return &Repository{items: make(map[string]Chamber)} }
func (r *Repository) Create(ctx context.Context, item Chamber) (Chamber, error) {
	if err := ctx.Err(); err != nil {
		return Chamber{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; ok {
		return Chamber{}, platform.Wrap("create", "chambers", item.ID, platform.ErrConflict)
	}
	r.version++
	item.Version = r.version
	r.items[item.ID] = item.Clone()
	return item.Clone(), nil
}
func (r *Repository) Get(ctx context.Context, id string) (Chamber, error) {
	if err := ctx.Err(); err != nil {
		return Chamber{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return Chamber{}, platform.Wrap("get", "chambers", id, platform.ErrNotFound)
	}
	return item.Clone(), nil
}
func (r *Repository) Update(ctx context.Context, id string, expected int64, fn func(*Chamber) error) (Chamber, error) {
	if err := ctx.Err(); err != nil {
		return Chamber{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.items[id]
	if !ok {
		return Chamber{}, platform.Wrap("update", "chambers", id, platform.ErrNotFound)
	}
	if expected > 0 && cur.Version != expected {
		return Chamber{}, platform.Wrap("update", "chambers", id, platform.ErrConflict)
	}
	next := cur.Clone()
	if err := fn(&next); err != nil {
		return Chamber{}, err
	}
	r.version++
	next.Version = r.version
	r.items[id] = next.Clone()
	return next.Clone(), nil
}
func (r *Repository) Restore(ctx context.Context, item Chamber, expected int64) (Chamber, error) {
	return r.Update(ctx, item.ID, expected, func(next *Chamber) error { *next = item.Clone(); return nil })
}
func (r *Repository) List(ctx context.Context, fab string) ([]Chamber, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Chamber, 0, len(r.items))
	for _, x := range r.items {
		if fab == "" || x.FabID == fab {
			out = append(out, x.Clone())
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].ID < out[j].ID
		}
		return out[i].Priority > out[j].Priority
	})
	return out, nil
}
func (r *Repository) Snapshot(ctx context.Context, fab string) (Snapshot, error) {
	items, err := r.List(ctx, fab)
	if err != nil {
		return Snapshot{}, err
	}
	r.mu.RLock()
	v := r.version
	r.mu.RUnlock()
	return Snapshot{Items: items, Version: v}, nil
}
func (r *Repository) Delete(ctx context.Context, id string, expected int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.items[id]
	if !ok {
		return platform.Wrap("delete", "chambers", id, platform.ErrNotFound)
	}
	if expected > 0 && x.Version != expected {
		return platform.Wrap("delete", "chambers", id, platform.ErrConflict)
	}
	delete(r.items, id)
	r.version++
	return nil
}
