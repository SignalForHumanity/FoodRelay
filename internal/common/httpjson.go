package common

import (
	"encoding/json"
	"net/http"
)

// WriteJSON encodes v as JSON with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// ReadJSON decodes the request body into v.
// Bodies larger than 64 KB are rejected to prevent memory exhaustion.
func ReadJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	return json.NewDecoder(r.Body).Decode(v)
}
