package collab

import (
	"errors"
	"sync"

	"github.com/serroba/online-docs/internal/acl"
	"github.com/serroba/online-docs/internal/ot"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/serroba/online-docs/internal/ws"
)

// Common errors.
var (
	ErrSessionClosed = errors.New("session is closed")
)

// Session coordinates collaborative editing for a single document.
// It wires together OT, storage, ACL, and WebSocket broadcasting.
type Session struct {
	docID string

	mu       sync.RWMutex
	document *ot.Document
	queue    *ot.Queue
	closed   bool

	// Dependencies
	store          storage.Store
	permChecker    *acl.Checker
	hub            *ws.Hub
	snapshotPolicy *storage.SnapshotPolicy
}

// SessionConfig holds configuration for creating a session.
type SessionConfig struct {
	DocID          string
	Store          storage.Store
	PermChecker    *acl.Checker
	Hub            *ws.Hub
	SnapshotPolicy *storage.SnapshotPolicy
	HistorySize    int
}

// NewSession creates a new collaborative editing session.
func NewSession(cfg SessionConfig) *Session {
	historySize := cfg.HistorySize
	if historySize == 0 {
		historySize = 100
	}

	return &Session{
		docID:          cfg.DocID,
		document:       ot.NewDocument(""),
		queue:          ot.NewQueue(historySize),
		store:          cfg.Store,
		permChecker:    cfg.PermChecker,
		hub:            cfg.Hub,
		snapshotPolicy: cfg.SnapshotPolicy,
	}
}

// Load initializes the session by loading document state from storage.
func (s *Session) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrSessionClosed
	}

	loader := storage.NewDocumentLoader(s.store)

	result, err := loader.Load(s.docID, s.applyOp)
	if err != nil {
		return err
	}

	s.document = ot.NewDocument(result.Content)
	s.queue = ot.NewQueue(s.queue.HistorySize())
	s.queue.SetRevision(result.Revision)

	return nil
}

// applyOp applies a storage operation to content (used by DocumentLoader).
func (s *Session) applyOp(content string, op storage.Operation) (string, error) {
	doc := ot.NewDocument(content)

	otOp := ot.Operation{
		Type:     ot.OpType(op.Type),
		Position: op.Position,
		Char:     op.Char,
	}

	if err := doc.Apply(otOp); err != nil {
		return "", err
	}

	return doc.Content(), nil
}

// ApplyOperation processes an operation from a client.
// It checks permissions, applies OT, persists, and broadcasts.
func (s *Session) ApplyOperation(clientID, userID string, op ot.Operation, baseRevision int) (int, error) {
	if err := s.checkWritePermission(userID); err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, ErrSessionClosed
	}

	seqOp, err := s.applyAndPersist(op, baseRevision)
	if err != nil {
		return 0, err
	}

	s.maybeSnapshot()
	s.broadcast(clientID, userID, seqOp)

	return seqOp.Revision, nil
}

// checkWritePermission verifies the user has write access.
func (s *Session) checkWritePermission(userID string) error {
	if s.permChecker == nil {
		return nil
	}

	return s.permChecker.RequirePermission(s.docID, userID, acl.ActionWrite)
}

// applyAndPersist applies OT transformation and persists the operation.
func (s *Session) applyAndPersist(op ot.Operation, baseRevision int) (ot.SequencedOperation, error) {
	seqOp, err := s.queue.Apply(op, baseRevision)
	if err != nil {
		return ot.SequencedOperation{}, err
	}

	if err := s.document.Apply(seqOp.Operation); err != nil {
		return ot.SequencedOperation{}, err
	}

	if err := s.store.AppendOperation(s.docID, seqOp); err != nil {
		return ot.SequencedOperation{}, err
	}

	return seqOp, nil
}

// maybeSnapshot checks if a snapshot should be created and does so.
func (s *Session) maybeSnapshot() {
	if s.snapshotPolicy == nil {
		return
	}

	if s.snapshotPolicy.RecordOperation(s.docID) {
		_ = s.saveSnapshot() // Log but don't fail
		s.snapshotPolicy.Reset(s.docID)
	}
}

// broadcast sends the operation to other connected clients.
func (s *Session) broadcast(clientID, userID string, seqOp ot.SequencedOperation) {
	if s.hub == nil {
		return
	}

	s.hub.BroadcastOperation(
		s.docID,
		seqOp.Revision,
		int(seqOp.Type),
		seqOp.Position,
		seqOp.Char,
		userID,
		clientID,
	)
}

// saveSnapshot persists a snapshot of the current document state.
func (s *Session) saveSnapshot() error {
	return s.store.SaveSnapshot(s.docID, s.queue.Revision(), s.document.Content())
}

// GetState returns the current document state.
// It checks read permission before returning.
func (s *Session) GetState(userID string) (string, int, error) {
	// Check read permission
	if s.permChecker != nil {
		if err := s.permChecker.RequirePermission(s.docID, userID, acl.ActionRead); err != nil {
			return "", 0, err
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return "", 0, ErrSessionClosed
	}

	return s.document.Content(), s.queue.Revision(), nil
}

// DocID returns the document ID for this session.
func (s *Session) DocID() string {
	return s.docID
}

// Revision returns the current revision number.
func (s *Session) Revision() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queue.Revision()
}

// Close closes the session and saves a final snapshot.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	// Save final snapshot
	return s.saveSnapshot()
}
