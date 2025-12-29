package main

import (
	"log"
	"net/http"
	"time"

	"github.com/serroba/online-docs/internal/acl"
	"github.com/serroba/online-docs/internal/api"
	"github.com/serroba/online-docs/internal/collab"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/serroba/online-docs/internal/ws"
)

func main() {
	// Initialize stores
	store := storage.NewMemoryStore()
	permStore := acl.NewMemoryStore()

	// Initialize WebSocket hub
	hub := ws.NewHub()

	// Initialize session manager
	manager := collab.NewManager(collab.ManagerConfig{
		Store:     store,
		PermStore: permStore,
		Hub:       hub,
	})

	// Initialize API server
	server := api.NewServer(api.ServerConfig{
		Manager:   manager,
		Store:     store,
		PermStore: permStore,
		Hub:       hub,
	})

	// Configure HTTP server with timeouts
	addr := ":8080"
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("Starting server on %s", addr)

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
