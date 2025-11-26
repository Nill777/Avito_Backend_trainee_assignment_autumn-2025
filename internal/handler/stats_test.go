package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_Stats(t *testing.T) {
	r, _, teardown := setupIntegration(t)
	defer teardown()

	t.Run("GetStats_Empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/stats/assignments", http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"stats":[]`)
	})
}
