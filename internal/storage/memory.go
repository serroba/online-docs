package storage

import (
	"sync"
	"time"

	"github.com/serroba/online-docs/internal/ot"
)

// documentData holds all persisted data for a single document.
type documentData struct {
	snapshot   *Snapshot
	operations []ot.SequencedOperation
}

// MemoryStore is an in-memory implementation of the Store interface.
// Useful for testing and development.
type MemoryStore struct {
	mu   sync.RWMutex
	docs map[string]*documentData
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		docs: make(map[string]*documentData),
	}
}

// CreateDocument creates a new document with the given ID.
func (m *MemoryStore) CreateDocument(docID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.docs[docID]; exists {
		return ErrDocumentExists
	}

	m.docs[docID] = &documentData{
		operations: make([]ot.SequencedOperation, 0),
	}

	return nil
}

// DocumentExists checks if a document exists.
func (m *MemoryStore) DocumentExists(docID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.docs[docID]

	return exists, nil
}

// SaveSnapshot persists a snapshot of the document at the given revision.
func (m *MemoryStore) SaveSnapshot(docID string, revision int, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	doc, exists := m.docs[docID]
	if !exists {
		return ErrDocumentNotFound
	}

	doc.snapshot = &Snapshot{
		DocID:     docID,
		Revision:  revision,
		Content:   content,
		CreatedAt: time.Now(),
	}

	// Prune operations that are now covered by the snapshot
	m.pruneOperations(doc, revision)

	return nil
}

// pruneOperations removes operations that are at or before the snapshot revision.
func (m *MemoryStore) pruneOperations(doc *documentData, snapshotRevision int) {
	var kept []ot.SequencedOperation

	for _, op := range doc.operations {
		if op.Revision > snapshotRevision {
			kept = append(kept, op)
		}
	}

	doc.operations = kept
}

// LoadSnapshot retrieves the latest snapshot for a document.
func (m *MemoryStore) LoadSnapshot(docID string) (Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	doc, exists := m.docs[docID]
	if !exists {
		return Snapshot{}, ErrDocumentNotFound
	}

	if doc.snapshot == nil {
		return Snapshot{}, ErrSnapshotNotFound
	}

	return *doc.snapshot, nil
}

// AppendOperation adds an operation to the document's operation log.
func (m *MemoryStore) AppendOperation(docID string, op ot.SequencedOperation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	doc, exists := m.docs[docID]
	if !exists {
		return ErrDocumentNotFound
	}

	doc.operations = append(doc.operations, op)

	return nil
}

// LoadOperations retrieves all operations after the given revision.
func (m *MemoryStore) LoadOperations(docID string, sinceRevision int) ([]ot.SequencedOperation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	doc, exists := m.docs[docID]
	if !exists {
		return nil, ErrDocumentNotFound
	}

	var result []ot.SequencedOperation

	for _, op := range doc.operations {
		if op.Revision > sinceRevision {
			result = append(result, op)
		}
	}

	return result, nil
}

// LatestRevision returns the highest revision number for a document.
func (m *MemoryStore) LatestRevision(docID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	doc, exists := m.docs[docID]
	if !exists {
		return 0, ErrDocumentNotFound
	}

	// Check operations first (they're newer than snapshot)
	if len(doc.operations) > 0 {
		return doc.operations[len(doc.operations)-1].Revision, nil
	}

	// Fall back to snapshot revision
	if doc.snapshot != nil {
		return doc.snapshot.Revision, nil
	}

	return 0, nil
}

// Ensure MemoryStore implements Store.
var _ Store = (*MemoryStore)(nil)
