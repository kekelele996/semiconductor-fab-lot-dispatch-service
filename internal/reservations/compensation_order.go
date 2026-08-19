package reservations

// CompensationOrder returns the leases in the reverse order they were acquired,
// so that compensation releases the most recently acquired resource first.
func CompensationOrder(leases []ResourceLease) []ResourceLease {
	out := make([]ResourceLease, len(leases))
	for i, l := range leases {
		out[len(leases)-1-i] = l
	}
	return out
}
