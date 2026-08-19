package reservations

import "errors"

func MergeCleanupErrors(errs []error) error {
	var joined error
	for _, err := range errs {
		joined = errors.Join(joined, err)
	}
	return joined
}
