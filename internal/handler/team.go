package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"reviewer/internal/domain"
)

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

func (h *Handler) DeactivateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamName string `json:"team_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if req.TeamName == "" {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	result, err := h.svc.DeactivateTeamAndRemoveReviews(r.Context(), req.TeamName)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
