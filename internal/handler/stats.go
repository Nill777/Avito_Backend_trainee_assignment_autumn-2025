package handler

import "net/http"

func (h *Handler) ReviewerStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.ReviewerStats(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}
