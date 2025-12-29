package storage

import (
	"errors"
	"sync"
)

// SnapshotPolicy determines when to create snapshots.
type SnapshotPolicy struct {
	mu               sync.Mutex
	threshold        int            // Create snapshot every N operations
	opsSinceSnapshot map[string]int // Track ops per document since last snapshot
}

// NewSnapshotPolicy creates a policy that triggers snapshots every N operations.
func NewSnapshotPolicy(threshold int) *SnapshotPolicy {
	return &SnapshotPolicy{
		threshold:        threshold,
		opsSinceSnapshot: make(map[string]int),
	}
}

// RecordOperation records that an operation was applied.
// Returns true if a snapshot should be created.
func (p *SnapshotPolicy) RecordOperation(docID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.opsSinceSnapshot[docID]++

	return p.opsSinceSnapshot[docID] >= p.threshold
}

// Reset resets the counter after a snapshot is created.
func (p *SnapshotPolicy) Reset(docID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.opsSinceSnapshot[docID] = 0
}

// OperationsSinceSnapshot returns the number of operations since the last snapshot.
func (p *SnapshotPolicy) OperationsSinceSnapshot(docID string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.opsSinceSnapshot[docID]
}

// DocumentLoader provides the ability to load a document from storage.
// It handles the snapshot + operation replay pattern.
type DocumentLoader struct {
	store Store
}

// NewDocumentLoader creates a new document loader.
func NewDocumentLoader(store Store) *DocumentLoader {
	return &DocumentLoader{store: store}
}

// LoadResult contains the result of loading a document.
type LoadResult struct {
	Content  string // Reconstructed document content
	Revision int    // Current revision
	IsNew    bool   // True if document didn't exist
}

// ApplyFunc is a function that applies an operation to content.
type ApplyFunc func(content string, op Operation) (string, error)

// Load reconstructs a document's state from storage.
// It loads the latest snapshot and replays any operations since.
func (l *DocumentLoader) Load(docID string, applyOp ApplyFunc) (LoadResult, error) {
	// Try to load snapshot
	snapshot, err := l.store.LoadSnapshot(docID)

	var content string

	var startRevision int

	switch {
	case errors.Is(err, ErrSnapshotNotFound):
		// No snapshot - start from empty
		content = ""
		startRevision = 0
	case err != nil:
		return LoadResult{}, err
	default:
		content = snapshot.Content
		startRevision = snapshot.Revision
	}

	// Load operations since snapshot
	ops, err := l.store.LoadOperations(docID, startRevision)
	if err != nil {
		return LoadResult{}, err
	}

	// Replay operations
	currentRevision := startRevision

	for _, op := range ops {
		content, err = applyOp(content, Operation{
			Type:     int(op.Type),
			Position: op.Position,
			Char:     op.Char,
		})
		if err != nil {
			return LoadResult{}, err
		}

		currentRevision = op.Revision
	}

	return LoadResult{
		Content:  content,
		Revision: currentRevision,
		IsNew:    startRevision == 0 && len(ops) == 0,
	}, nil
}

// Operation mirrors ot.Operation for the loader to avoid circular imports.
type Operation struct {
	Type     int
	Position int
	Char     string
}
