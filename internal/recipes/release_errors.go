package recipes

import (
	"fmt"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
)

type ReleaseValidationError struct {
	RecipeID string
	Missing  []string
	Err      error
}

func (e *ReleaseValidationError) Error() string {
	return fmt.Sprintf("recipe %s release rejected: missing %v: %v", e.RecipeID, e.Missing, e.Err)
}
func (e *ReleaseValidationError) Unwrap() error { return e.Err }
func validateRelease(x Recipe) error {
	required := []string{"process_family", "tool_group", "revision", "checksum"}
	missing := make([]string, 0)
	for _, key := range required {
		if x.Constraints[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return &ReleaseValidationError{RecipeID: x.ID, Missing: missing, Err: platform.ErrInvariant}
	}
	return nil
}
