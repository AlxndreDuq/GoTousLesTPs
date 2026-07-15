package middleware

import (
	"encoding/json"
	"net/http"
)

// writeJSONError writes a minimal error envelope matching the shape produced
// by internal/http/handlers.writeError, so responses stay consistent even
// when a request never reaches the handlers layer (panic, timeout).
func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
