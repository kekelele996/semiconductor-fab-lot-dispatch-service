package recipes

import "semiconductor-fab-lot-dispatch-service/internal/platform"

func ClassifyReleaseError(err error) string {
	switch {
	case err == nil:
		return "ok"
	case err == platform.ErrInvariant:
		return "invalid"
	case err == platform.ErrConflict:
		return "conflict"
	default:
		return "retry"
	}
}
