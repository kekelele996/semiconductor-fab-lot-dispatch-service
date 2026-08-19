package reservations

import "errors"

// MergeCleanupErrors joins every non-nil cleanup failure so that callers can
// detect each individual error via errors.Is, and nil is returned only when
// every cleanup step succeeded.
func MergeCleanupErrors(errs []error) error {
	joined := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			joined = append(joined, err)
		}
	}
	return errors.Join(joined...)
}
