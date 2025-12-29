package api

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/serroba/online-docs/internal/acl"
	"github.com/serroba/online-docs/internal/collab"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/serroba/online-docs/internal/ws"
)

// Server handles HTTP requests for the collaboration API.
type Server struct {
	manager   *collab.Manager
	store     storage.Store
	permStore acl.Store
	hub       *ws.Hub
	upgrader  websocket.Upgrader
}

// ServerConfig holds configuration for creating a server.
type ServerConfig struct {
	Manager   *collab.Manager
	Store     storage.Store
	PermStore acl.Store
	Hub       *ws.Hub
}

// NewServer creates a new API server.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		manager:   cfg.Manager,
		store:     cfg.Store,
		permStore: cfg.PermStore,
		hub:       cfg.Hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}
}

// Handler returns an http.Handler with all routes configured.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Document endpoints (require auth)
	mux.Handle("/documents", s.authMiddleware(http.HandlerFunc(s.handleCreateDocument)))
	mux.Handle("/documents/", s.authMiddleware(http.HandlerFunc(s.handleDocumentByID)))

	// WebSocket endpoint (requires auth)
	mux.Handle("/ws", s.authMiddleware(http.HandlerFunc(s.handleWebSocket)))

	return mux
}

// handleDocumentByID routes GET and DELETE requests for /documents/{id}.
func (s *Server) handleDocumentByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetDocument(w, r)
	case http.MethodDelete:
		s.handleDeleteDocument(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
