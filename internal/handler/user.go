package handler

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}

	user, err := h.svc.UpdateUser(r.Context(), req.UserID, &req.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("user_id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}
	prs, err := h.svc.ListPRsByReviewer(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":       id,
		"pull_requests": prs,
	})
}
