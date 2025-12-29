package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/serroba/online-docs/internal/collab"
	"github.com/serroba/online-docs/internal/handler"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/serroba/online-docs/internal/ws"
)

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	hub := ws.NewHub()
	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
		Hub:   hub,
	})

	server := handler.NewServer(handler.ServerConfig{
		Manager: manager,
		Store:   store,
		Hub:     hub,
	})

	handler := server.Handler()

	t.Run("returns 401 when X-User-Id header is missing", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/documents", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("passes request when X-User-Id header is present", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/documents/nonexistent", nil)
		req.Header.Set("X-User-Id", "user123")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Should get past auth middleware (404 means request was processed)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}
