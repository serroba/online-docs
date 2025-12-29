package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/serroba/online-docs/internal/acl"
	"github.com/serroba/online-docs/internal/collab"
	"github.com/serroba/online-docs/internal/handler"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/serroba/online-docs/internal/ws"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateDocument(t *testing.T) {
	t.Parallel()

	t.Run("creates document successfully", func(t *testing.T) {
		t.Parallel()

		store := storage.NewMemoryStore()
		permStore := acl.NewMemoryStore()
		hub := ws.NewHub()
		manager := collab.NewManager(collab.ManagerConfig{
			Store:     store,
			PermStore: permStore,
			Hub:       hub,
		})

		server := handler.NewServer(handler.ServerConfig{
			Manager:   manager,
			Store:     store,
			PermStore: permStore,
			Hub:       hub,
		})

		body, _ := json.Marshal(map[string]string{"id": "doc1"})
		req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}

		var resp map[string]string
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

		if resp["id"] != "doc1" {
			t.Errorf("expected ID 'doc1', got %q", resp["id"])
		}

		// Verify document exists
		exists, _ := store.DocumentExists("doc1")
		if !exists {
			t.Error("expected document to exist")
		}

		// Verify owner permission was granted
		role, err := permStore.GetRole("doc1", "user1")
		require.NoError(t, err)

		if role != acl.Owner {
			t.Errorf("expected Owner role, got %v", role)
		}
	})

	t.Run("returns 409 for duplicate document", func(t *testing.T) {
		t.Parallel()

		store := storage.NewMemoryStore()
		require.NoError(t, store.CreateDocument("doc1"))

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

		body, _ := json.Marshal(map[string]string{"id": "doc1"})
		req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for empty ID", func(t *testing.T) {
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

		body, _ := json.Marshal(map[string]string{"id": ""})
		req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(body))
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 405 for wrong method", func(t *testing.T) {
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

		req := httptest.NewRequest(http.MethodGet, "/documents", nil)
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid JSON body", func(t *testing.T) {
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

		req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestHandleGetDocument(t *testing.T) {
	t.Parallel()

	t.Run("gets document successfully", func(t *testing.T) {
		t.Parallel()

		store := storage.NewMemoryStore()
		require.NoError(t, store.CreateDocument("doc1"))

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

		req := httptest.NewRequest(http.MethodGet, "/documents/doc1", nil)
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp map[string]any
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

		if resp["id"] != "doc1" {
			t.Errorf("expected ID 'doc1', got %v", resp["id"])
		}
	})

	t.Run("returns 404 for non-existent document", func(t *testing.T) {
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

		req := httptest.NewRequest(http.MethodGet, "/documents/nonexistent", nil)
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 403 for access denied", func(t *testing.T) {
		t.Parallel()

		store := storage.NewMemoryStore()
		require.NoError(t, store.CreateDocument("doc1"))

		permStore := acl.NewMemoryStore()
		hub := ws.NewHub()
		manager := collab.NewManager(collab.ManagerConfig{
			Store:     store,
			PermStore: permStore,
			Hub:       hub,
		})

		server := handler.NewServer(handler.ServerConfig{
			Manager:   manager,
			Store:     store,
			PermStore: permStore,
			Hub:       hub,
		})

		req := httptest.NewRequest(http.MethodGet, "/documents/doc1", nil)
		req.Header.Set("X-User-Id", "unauthorized")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for empty document ID", func(t *testing.T) {
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

		req := httptest.NewRequest(http.MethodGet, "/documents/", nil)
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestHandleDeleteDocument(t *testing.T) {
	t.Parallel()

	t.Run("deletes document successfully", func(t *testing.T) {
		t.Parallel()

		store := storage.NewMemoryStore()
		require.NoError(t, store.CreateDocument("doc1"))

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

		req := httptest.NewRequest(http.MethodDelete, "/documents/doc1", nil)
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rec.Code)
		}

		// Verify document was deleted
		exists, _ := store.DocumentExists("doc1")
		if exists {
			t.Error("expected document to be deleted")
		}
	})

	t.Run("returns 404 for non-existent document", func(t *testing.T) {
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

		req := httptest.NewRequest(http.MethodDelete, "/documents/nonexistent", nil)
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 403 for access denied", func(t *testing.T) {
		t.Parallel()

		store := storage.NewMemoryStore()
		require.NoError(t, store.CreateDocument("doc1"))

		permStore := acl.NewMemoryStore()
		require.NoError(t, permStore.Grant("doc1", "owner", acl.Owner))

		hub := ws.NewHub()
		manager := collab.NewManager(collab.ManagerConfig{
			Store:     store,
			PermStore: permStore,
			Hub:       hub,
		})

		server := handler.NewServer(handler.ServerConfig{
			Manager:   manager,
			Store:     store,
			PermStore: permStore,
			Hub:       hub,
		})

		req := httptest.NewRequest(http.MethodDelete, "/documents/doc1", nil)
		req.Header.Set("X-User-Id", "notowner")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for empty document ID", func(t *testing.T) {
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

		req := httptest.NewRequest(http.MethodDelete, "/documents/", nil)
		req.Header.Set("X-User-Id", "user1")

		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}
