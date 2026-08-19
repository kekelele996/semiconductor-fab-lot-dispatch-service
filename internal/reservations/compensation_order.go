package reservations

func CompensationOrder(leases []ResourceLease) []ResourceLease {
	out := make([]ResourceLease, len(leases))
	for i := range leases {
		out[len(leases)-1-i] = leases[i]
	}
	return out
}
