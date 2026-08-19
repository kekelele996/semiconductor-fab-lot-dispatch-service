package reservations

func CompensationOrder(leases []ResourceLease) []ResourceLease {
	out := append([]ResourceLease(nil), leases...)
	return out
}
