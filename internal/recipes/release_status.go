package recipes

import (
	"errors"
	"semiconductor-fab-lot-dispatch-service/internal/platform"
)

func ReleaseStatus(err error) int {
	switch {
	case err == nil:
		return 200
	case errors.Is(err, platform.ErrInvariant):
		return 422
	case errors.Is(err, platform.ErrConflict):
		return 409
	case errors.Is(err, platform.ErrNotFound):
		return 404
	default:
		return 500
	}
}
