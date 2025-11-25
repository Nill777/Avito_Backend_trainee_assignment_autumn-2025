package handler

import "net/http"

func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}
