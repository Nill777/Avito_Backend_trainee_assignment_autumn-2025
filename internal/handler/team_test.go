package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reviewer/internal/domain"
)

func TestHandler_Team(t *testing.T) {
	r, _, teardown := setupIntegration(t)
	defer teardown()

	t.Run("CreateTeam_Success", func(t *testing.T) {
		body := `{"team_name": "api-team", "members": [{"user_id": "u1", "username": "A", "is_active": true}]}`
		req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		team := resp["team"].(map[string]any)
		assert.Equal(t, "api-team", team["team_name"])
	})

	t.Run("CreateTeam_Duplicate", func(t *testing.T) {
		body := `{"team_name": "dup-team", "members": []}`
		req1 := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewBufferString(body))
		r.ServeHTTP(httptest.NewRecorder(), req1)
		req2 := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req2)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp APIErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "TEAM_EXISTS", resp.Error.Code)
	})

	t.Run("GetTeam_Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/team/get?team_name=api-team", http.NoBody)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var team domain.Team
		err := json.Unmarshal(w.Body.Bytes(), &team)
		require.NoError(t, err)
		assert.Equal(t, "api-team", team.Name)
		assert.Len(t, team.Members, 1)
	})

	t.Run("DeactivateTeam_Success", func(t *testing.T) {
		createBody := `{"team_name": "deact-api", "members": [{"user_id": "d1", "username": "D", "is_active": true}]}`
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewBufferString(createBody)))

		body := `{"team_name": "deact-api"}`
		req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp domain.DeactivationResult
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.DeactivatedUsers, 1)
		assert.Equal(t, "d1", resp.DeactivatedUsers[0].ID)
	})
}
