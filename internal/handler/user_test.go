package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_User(t *testing.T) {
	r, _, teardown := setupIntegration(t)
	defer teardown()

	body := `{"team_name": "user-api", "members": [{"user_id": "u100", "username": "TestUser", "is_active": true}]}`
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewBufferString(body)))

	t.Run("SetUserActive_Success", func(t *testing.T) {
		reqBody := `{"user_id": "u100", "is_active": false}`
		req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		user := resp["user"].(map[string]any)
		assert.Equal(t, false, user["is_active"])
	})

	t.Run("GetUserReviews_Empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=u100", http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "u100", resp["user_id"])
		assert.NotNil(t, resp["pull_requests"])
	})
}
