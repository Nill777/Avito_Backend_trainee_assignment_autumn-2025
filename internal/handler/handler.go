package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"reviewer/internal/domain"
	"reviewer/internal/service"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func New(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/team/add", h.CreateTeam)
	r.Get("/team/get", h.GetTeam)
	r.Post("/users/setIsActive", h.SetUserActive)
	r.Get("/users/getReview", h.GetUserReviews)
	r.Post("/pullRequest/create", h.CreatePR)
	r.Post("/pullRequest/merge", h.MergePR)
	r.Post("/pullRequest/reassign", h.ReassignReviewer)
	r.Get("/stats/assignments", h.ReviewerStats)
}

type APIErrorResponse struct {
	Error APIErrorDetail `json:"error"`
}

type APIErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIErrorResponse{Error: APIErrorDetail{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamName string `json:"team_name"`
		Members  []struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			IsActive bool   `json:"is_active"`
		} `json:"members"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	if req.TeamName == "" {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}
	seenUsers := make(map[string]bool)
	for _, m := range req.Members {
		if seenUsers[m.UserID] {
			writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "duplicate user_id found in request: "+m.UserID)
			return
		}
		seenUsers[m.UserID] = true
	}

	team, err := h.svc.CreateTeam(r.Context(), req.TeamName)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			writeAPIError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
			return
		}
		h.handleError(w, err)
		return
	}

	for _, m := range req.Members {
		_, err := h.svc.CreateUser(r.Context(), m.UserID, m.Username, team.Name, m.IsActive)
		if err != nil {
			h.log.Warn("failed to create user", "user_id", m.UserID, "error", err)
		}
	}

	respMembers := make([]domain.TeamMember, len(req.Members))
	for i, m := range req.Members {
		respMembers[i] = domain.TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}
	team.Members = respMembers

	writeJSON(w, http.StatusCreated, map[string]any{"team": team})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("team_name")
	if name == "" {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}
	team, err := h.svc.GetTeamByName(r.Context(), name)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, team)
}

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

func (h *Handler) ReviewerStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.ReviewerStats(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConflict):
		writeAPIError(w, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, domain.ErrReviewerExist), errors.Is(err, domain.ErrPRMerged):
		writeAPIError(w, http.StatusConflict, "PR_MERGED", err.Error())
	case errors.Is(err, domain.ErrNoCandidates):
		writeAPIError(w, http.StatusConflict, "NO_CANDIDATE", err.Error())
	case errors.Is(err, domain.ErrNotAssigned):
		writeAPIError(w, http.StatusConflict, "NOT_ASSIGNED", err.Error())
	default:
		h.log.Error("internal server error", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
