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
	r.Post("/team/deactivate", h.DeactivateTeam)
	r.Post("/users/setIsActive", h.SetUserActive)
	r.Get("/users/getReview", h.GetUserReviews)
	r.Post("/pullRequest/create", h.CreatePR)
	r.Post("/pullRequest/merge", h.MergePR)
	r.Post("/pullRequest/reassign", h.ReassignReviewer)
	r.Get("/healthz", h.HealthCheck)
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
