package handlers

import "net/http"

// Health serves GET /health for container/orchestrator liveness checks.
func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
