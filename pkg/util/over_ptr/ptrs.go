package maps

// ValEquals returns true when over_ptr is non‑nil and *over_ptr == want.
func ValEquals[T comparable](ptr *T, want T) bool {
	return ptr != nil && *ptr == want
}
