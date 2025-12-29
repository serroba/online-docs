package storage

import (
	"errors"
	"time"

	"github.com/serroba/online-docs/internal/ot"
)

// Common errors.
var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrDocumentExists   = errors.New("document already exists")
	ErrSnapshotNotFound = errors.New("snapshot not found")
)

// Snapshot represents a point-in-time capture of a document's state.
type Snapshot struct {
	DocID     string
	Revision  int
	Content   string
	CreatedAt time.Time
}

// Store defines the interface for persisting document state.
// Implementations can use in-memory storage, databases, or other backends.
type Store interface {
	// CreateDocument creates a new document with the given ID.
	// Returns ErrDocumentExists if the document already exists.
	CreateDocument(docID string) error

	// DocumentExists checks if a document exists.
	DocumentExists(docID string) (bool, error)

	// SaveSnapshot persists a snapshot of the document at the given revision.
	// Returns ErrDocumentNotFound if the document doesn't exist.
	SaveSnapshot(docID string, revision int, content string) error

	// LoadSnapshot retrieves the latest snapshot for a document.
	// Returns ErrDocumentNotFound if the document doesn't exist.
	// Returns ErrSnapshotNotFound if document exists but has no snapshot.
	LoadSnapshot(docID string) (Snapshot, error)

	// AppendOperation adds an operation to the document's operation log.
	// Returns ErrDocumentNotFound if the document doesn't exist.
	AppendOperation(docID string, op ot.SequencedOperation) error

	// LoadOperations retrieves all operations after the given revision.
	// Returns ErrDocumentNotFound if the document doesn't exist.
	LoadOperations(docID string, sinceRevision int) ([]ot.SequencedOperation, error)

	// LatestRevision returns the highest revision number for a document.
	// Returns ErrDocumentNotFound if the document doesn't exist.
	LatestRevision(docID string) (int, error)
}
