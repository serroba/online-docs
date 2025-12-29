package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/serroba/online-docs/internal/api"
	"github.com/serroba/online-docs/internal/collab"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/serroba/online-docs/internal/ws"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	hub := ws.NewHub()
	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
		Hub:   hub,
	})

	server := api.NewServer(api.ServerConfig{
		Manager: manager,
		Store:   store,
		Hub:     hub,
	})

	if server == nil {
		t.Error("NewServer returned nil")
	}
}

func TestServerHandler(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	hub := ws.NewHub()
	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
		Hub:   hub,
	})

	server := api.NewServer(api.ServerConfig{
		Manager: manager,
		Store:   store,
		Hub:     hub,
	})

	handler := server.Handler()

	if handler == nil {
		t.Error("Handler returned nil")
	}

	t.Run("documents endpoint requires auth", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/documents", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for missing auth, got %d", rec.Code)
		}
	})

	t.Run("ws endpoint requires auth", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for missing auth, got %d", rec.Code)
		}
	})

	t.Run("routes PUT to method not allowed", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPut, "/documents/test", nil)
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", rec.Code)
		}
	})
}
