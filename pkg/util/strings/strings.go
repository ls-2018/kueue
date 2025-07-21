package strings

import "strings"

func StringContainsSubstrings(s string, substrings ...string) bool {
	for _, substring := range substrings {
		if !strings.Contains(s, substring) {
			return false
		}
	}

	return true
}

// Join function for string-like types.
func Join[T ~string](a []T, sep string) string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = string(v)
	}
	return strings.Join(strs, sep)
}
