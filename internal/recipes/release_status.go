package recipes

import "semiconductor-fab-lot-dispatch-service/internal/platform"

func ReleaseStatus(err error) int {
	switch {
	case err == nil:
		return 200
	case err == platform.ErrInvariant:
		return 422
	case err == platform.ErrConflict:
		return 409
	case err == platform.ErrNotFound:
		return 404
	default:
		return 500
	}
}
