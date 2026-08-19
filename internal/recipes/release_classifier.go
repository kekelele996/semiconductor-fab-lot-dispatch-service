package recipes

import (
	"errors"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
)

func ClassifyReleaseError(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, platform.ErrInvariant):
		return "invalid"
	case errors.Is(err, platform.ErrConflict):
		return "conflict"
	default:
		return "retry"
	}
}
