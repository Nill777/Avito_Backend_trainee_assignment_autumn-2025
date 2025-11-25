package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"reviewer/internal/domain"
)

func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PRID     string `json:"pull_request_id"`
		Title    string `json:"pull_request_name"`
		AuthorID string `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	pr, err := h.svc.CreatePR(r.Context(), req.PRID, req.Title, req.AuthorID)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			writeAPIError(w, http.StatusConflict, "PR_EXISTS", "pr already exists")
			return
		}
		h.handleError(w, err)
		return
	}
	resp := map[string]any{
		"pr": map[string]any{
			"pull_request_id":    pr.ID,
			"pull_request_name":  pr.Title,
			"author_id":          pr.AuthorID,
			"status":             pr.Status,
			"assigned_reviewers": pr.Reviewers,
		},
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PRID string `json:"pull_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	pr, err := h.svc.MergePR(r.Context(), req.PRID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	resp := map[string]any{
		"pr": map[string]any{
			"pull_request_id":    pr.ID,
			"pull_request_name":  pr.Title,
			"author_id":          pr.AuthorID,
			"status":             pr.Status,
			"assigned_reviewers": pr.Reviewers,
			"mergedAt":           pr.MergedAt,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PRID  string `json:"pull_request_id"`
		OldID string `json:"old_reviewer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	pr, newReviewer, err := h.svc.ReassignReviewer(r.Context(), req.PRID, req.OldID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	resp := map[string]any{
		"pr": map[string]any{
			"pull_request_id":    pr.ID,
			"pull_request_name":  pr.Title,
			"author_id":          pr.AuthorID,
			"status":             pr.Status,
			"assigned_reviewers": pr.Reviewers,
		},
		"replaced_by": newReviewer.ID,
	}
	writeJSON(w, http.StatusOK, resp)
}
