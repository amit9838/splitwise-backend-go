package util

// BoolToInt converts a bool to 0 or 1 for SQLite storage.
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// IntToBool converts a SQLite 0/1 value to bool.
func IntToBool(i int) bool {
	return i != 0
}
