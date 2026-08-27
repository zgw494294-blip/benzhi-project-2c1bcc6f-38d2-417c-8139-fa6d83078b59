package transport

import (
	"encoding/json"
	"net/http"
	"time"
)

type envelope struct {
	Data any       `json:"data"`
	At   time.Time `json:"at"`
}

func writeEnvelope(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(envelope{Data: v, At: time.Now().UTC()})
}
func requestID(r *http.Request) string {
	if v := r.Header.Get("X-Request-ID"); v != "" {
		return v
	}
	return time.Now().UTC().Format("20060102150405.000000000")
}
func noCache(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }
