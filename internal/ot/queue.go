package ot

import (
	"errors"
	"sync"
)

// ErrRevisionTooOld is returned when the client's base revision is too far behind.
var ErrRevisionTooOld = errors.New("base revision too old, history unavailable")

// SequencedOperation wraps an operation with its assigned revision.
type SequencedOperation struct {
	Operation
	Revision int
}

// Queue manages the sequencing and transformation of concurrent operations.
// It maintains a history of recent operations to transform incoming ops
// that are based on older revisions.
type Queue struct {
	mu          sync.RWMutex
	revision    int                  // Current document revision
	history     []SequencedOperation // Recent operations for transformation
	historySize int                  // Maximum history size to keep
}

// NewQueue creates a new operation queue.
// historySize determines how many past operations to retain for transformation.
func NewQueue(historySize int) *Queue {
	return &Queue{
		revision:    0,
		history:     make([]SequencedOperation, 0, historySize),
		historySize: historySize,
	}
}

// Revision returns the current document revision.
func (q *Queue) Revision() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.revision
}

// Apply takes an operation and its base revision, transforms it against
// any operations that have occurred since that revision, and returns
// the transformed operation with its new sequence number.
func (q *Queue) Apply(op Operation, baseRevision int) (SequencedOperation, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Validate base revision
	if baseRevision > q.revision {
		return SequencedOperation{}, errors.New("base revision is in the future")
	}

	// Check if we have enough history to transform
	// We need all operations after baseRevision to be in history
	if baseRevision < q.revision && len(q.history) > 0 {
		oldestAvailable := q.history[0].Revision

		// If client is based on revision older than our oldest history entry - 1,
		// we can't properly transform
		if baseRevision < oldestAvailable-1 {
			return SequencedOperation{}, ErrRevisionTooOld
		}
	}

	// Transform against all operations since baseRevision
	transformed := op

	for _, histOp := range q.history {
		if histOp.Revision > baseRevision {
			// Transform our operation against this historical operation
			transformed, _ = Transform(transformed, histOp.Operation)
		}
	}

	// Assign new revision
	q.revision++

	result := SequencedOperation{
		Operation: transformed,
		Revision:  q.revision,
	}

	// Add to history
	q.addToHistory(result)

	return result, nil
}

// addToHistory adds an operation to history, pruning old entries if needed.
func (q *Queue) addToHistory(op SequencedOperation) {
	q.history = append(q.history, op)

	// Prune if exceeding history size
	if len(q.history) > q.historySize {
		q.history = q.history[1:]
	}
}

// History returns a copy of the current operation history.
// Useful for clients that need to catch up.
func (q *Queue) History(sinceRevision int) []SequencedOperation {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []SequencedOperation

	for _, op := range q.history {
		if op.Revision > sinceRevision {
			result = append(result, op)
		}
	}

	return result
}
