package recipes

import (
	"context"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"time"
)

type ReleaseWorkflow struct {
	repo   *Repository
	events platform.EventSink
	clock  platform.Clock
}

func NewReleaseWorkflow(r *Repository, e platform.EventSink, c platform.Clock) *ReleaseWorkflow {
	return &ReleaseWorkflow{repo: r, events: e, clock: c}
}
func (w *ReleaseWorkflow) Release(ctx context.Context, id string, expected int64) (Recipe, error) {
	before, err := w.repo.Get(ctx, id)
	if err != nil {
		return Recipe{}, err
	}
	if err = validateRelease(before); err != nil {
		return Recipe{}, platform.Wrap("validate-release", "recipe", id, err)
	}
	updated, err := w.repo.Update(ctx, id, expected, func(next *Recipe) error {
		if next.State != StateQualified {
			return platform.ErrInvalidTransition
		}
		next.State = StateReleased
		next.UpdatedAt = w.clock.Now()
		return nil
	})
	if err != nil {
		return Recipe{}, err
	}
	event := platform.Event{Topic: "recipes.released", Key: id, Version: updated.Version, At: w.clock.Now(), Attributes: map[string]string{"revision": updated.Constraints["revision"]}}
	if err = w.events.Publish(ctx, event); err != nil {
		rbCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, rb := w.repo.Restore(rbCtx, before, updated.Version); rb != nil {
			return Recipe{}, platform.Wrap("rollback-release", "recipe", id, rb)
		}
		return Recipe{}, platform.Wrap("publish-release", "recipe", id, err)
	}
	return updated, nil
}
