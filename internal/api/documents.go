package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/serroba/online-docs/internal/acl"
	"github.com/serroba/online-docs/internal/storage"
)

// CreateDocumentRequest is the request body for creating a document.
type CreateDocumentRequest struct {
	ID string `json:"id"`
}

// CreateDocumentResponse is the response body for creating a document.
type CreateDocumentResponse struct {
	ID string `json:"id"`
}

// GetDocumentResponse is the response body for getting a document.
type GetDocumentResponse struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Revision int    `json:"revision"`
}

// handleCreateDocument handles POST /documents.
func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	var req CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	if req.ID == "" {
		http.Error(w, "document ID is required", http.StatusBadRequest)

		return
	}

	if err := s.store.CreateDocument(req.ID); err != nil {
		if errors.Is(err, storage.ErrDocumentExists) {
			http.Error(w, "document already exists", http.StatusConflict)

			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	// Grant the creator Owner role if ACL store is configured
	userID := UserIDFromContext(r.Context())
	if s.permStore != nil && userID != "" {
		_ = s.permStore.Grant(req.ID, userID, acl.Owner)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(CreateDocumentResponse(req)); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// handleGetDocument handles GET /documents/{id}.
func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	docID := extractDocID(r.URL.Path, "/documents/")
	if docID == "" {
		http.Error(w, "document ID is required", http.StatusBadRequest)

		return
	}

	userID := UserIDFromContext(r.Context())

	// Get or create a session to retrieve current state
	session, err := s.manager.GetOrCreateSession(docID)
	if err != nil {
		if errors.Is(err, storage.ErrDocumentNotFound) {
			http.Error(w, "document not found", http.StatusNotFound)

			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	content, revision, err := session.GetState(userID)
	if err != nil {
		if errors.Is(err, acl.ErrAccessDenied) {
			http.Error(w, "access denied", http.StatusForbidden)

			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(GetDocumentResponse{
		ID:       docID,
		Content:  content,
		Revision: revision,
	}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// handleDeleteDocument handles DELETE /documents/{id}.
func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	docID := extractDocID(r.URL.Path, "/documents/")
	if docID == "" {
		http.Error(w, "document ID is required", http.StatusBadRequest)

		return
	}

	userID := UserIDFromContext(r.Context())

	// Check delete permission if ACL is configured
	if s.permStore != nil {
		checker := acl.NewChecker(s.permStore)
		if err := checker.RequirePermission(docID, userID, acl.ActionDelete); err != nil {
			if errors.Is(err, acl.ErrAccessDenied) {
				http.Error(w, "access denied", http.StatusForbidden)

				return
			}

			http.Error(w, "internal server error", http.StatusInternalServerError)

			return
		}
	}

	// Close any active session first
	if err := s.manager.CloseSession(docID); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	if err := s.store.DeleteDocument(docID); err != nil {
		if errors.Is(err, storage.ErrDocumentNotFound) {
			http.Error(w, "document not found", http.StatusNotFound)

			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// extractDocID extracts the document ID from a URL path.
func extractDocID(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	return strings.TrimPrefix(path, prefix)
}
