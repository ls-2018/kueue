package maps

// ValEquals returns true when ptr is non‑nil and *ptr == want.
func ValEquals[T comparable](ptr *T, want T) bool {
	return ptr != nil && *ptr == want
}
