package category

import (
	"encoding/json"
	"errors"
	"net/http"
)

var ErrNotFound = errors.New("category not found")

// write json
func writeJSON(w http.ResponseWriter, status int, val any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(val)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// ========== HELPERS ===========
// Helper functions to convert between bool and int (SQLite stores booleans as 0/1)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}
