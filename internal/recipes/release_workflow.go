package recipes

import (
	"context"
	"fmt"
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
func validationFailure(id string, err error) error {
	return fmt.Errorf("validate release recipe %s: %w", id, err)
}
func rollbackVersion(before, updated Recipe) int64 { return updated.Version }
func rollbackContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Second)
}
func (w *ReleaseWorkflow) Release(ctx context.Context, id string, expected int64) (Recipe, error) {
	before, err := w.repo.Get(ctx, id)
	if err != nil {
		return Recipe{}, err
	}
	if err = validateRelease(before); err != nil {
		return Recipe{}, validationFailure(id, err)
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
		rbCtx, cancel := rollbackContext(ctx)
		defer cancel()
		if _, rb := w.repo.Restore(rbCtx, before, rollbackVersion(before, updated)); rb != nil {
			return Recipe{}, fmt.Errorf("publish release: %w; rollback release: %v", err, rb)
		}
		return Recipe{}, platform.Wrap("publish-release", "recipe", id, err)
	}
	return updated, nil
}
