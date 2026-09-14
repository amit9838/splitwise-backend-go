package util

import "math"

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

// Round2 rounds a monetary value to two decimal places.
func Round2(x float64) float64 {
	return math.Round(x*100) / 100
}
