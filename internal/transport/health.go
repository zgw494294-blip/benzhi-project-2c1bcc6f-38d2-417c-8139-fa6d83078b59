package transport

import (
	"archive-release/internal/persistence"
	"encoding/json"
	"net/http"
)

func HealthHandler(repo persistence.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := persistence.Check(repo)
		w.Header().Set("Content-Type", "application/json")
		if !h.Ready {
			w.WriteHeader(503)
		}
		_ = json.NewEncoder(w).Encode(h)
	}
}
