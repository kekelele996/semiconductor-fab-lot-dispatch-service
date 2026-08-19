package recipes

import (
	"context"
	"errors"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
	"testing"
	"time"
)

type recipeFailSink struct{}

func (recipeFailSink) Publish(context.Context, platform.Event) error {
	return errors.New("registry down")
}
func TestReleaseWorkflowPreservesTypedValidation(t *testing.T) {
	repo := NewRepository()
	clock := platform.NewManualClock(time.Now())
	bad, _ := repo.Create(context.Background(), Recipe{ID: "r-bad", FabID: "fab", State: StateQualified, Constraints: map[string]string{"revision": "1"}})
	wf := NewReleaseWorkflow(repo, &platform.MemoryEventSink{}, clock)
	_, err := wf.Release(context.Background(), bad.ID, bad.Version)
	if !errors.Is(err, platform.ErrInvariant) {
		t.Fatalf("typed error lost: %v", err)
	}
}

func TestReleaseWorkflowRollsBackPublishFailure(t *testing.T) {
	repo := NewRepository()
	clock := platform.NewManualClock(time.Now())
	good, _ := repo.Create(context.Background(), Recipe{ID: "r-good", FabID: "fab", State: StateQualified, Constraints: map[string]string{"process_family": "etch", "tool_group": "g1", "revision": "2", "checksum": "abc"}})
	wf := NewReleaseWorkflow(repo, recipeFailSink{}, clock)
	_, err := wf.Release(context.Background(), good.ID, good.Version)
	if err == nil {
		t.Fatal("expected publish error")
	}
	after, _ := repo.Get(context.Background(), good.ID)
	if after.State != StateQualified {
		t.Fatalf("state leaked: %s", after.State)
	}
}

func TestReleaseClassifierRejectsWrappedValidation(t *testing.T) {
	err := platform.Wrap("release", "recipe", "r", &ReleaseValidationError{RecipeID: "r", Missing: []string{"checksum"}, Err: platform.ErrInvariant})
	if got := ClassifyReleaseError(err); got != "invalid" {
		t.Fatalf("class=%s", got)
	}
}
func TestReleaseStatusMapsWrappedInvariant(t *testing.T) {
	err := platform.Wrap("release", "recipe", "r", &ReleaseValidationError{RecipeID: "r", Missing: []string{"checksum"}, Err: platform.ErrInvariant})
	if got := ReleaseStatus(err); got != 422 {
		t.Fatalf("status=%d", got)
	}
}
