package ot_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/serroba/online-docs/internal/ot"
)

func TestQueue_NewQueue(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)

	if q.Revision() != 0 {
		t.Errorf("expected initial revision 0, got %d", q.Revision())
	}
}

func TestQueue_Apply_SingleOperation(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)
	op := ot.NewInsert("a", 0, "alice")

	result, err := q.Apply(op, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Revision != 1 {
		t.Errorf("expected revision 1, got %d", result.Revision)
	}

	if result.Position != 0 {
		t.Errorf("expected position 0, got %d", result.Position)
	}

	if q.Revision() != 1 {
		t.Errorf("expected queue revision 1, got %d", q.Revision())
	}
}

func TestQueue_Apply_SequentialOperations(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)

	// Alice inserts at position 0, based on revision 0
	op1 := ot.NewInsert("a", 0, "alice")

	result1, err := q.Apply(op1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result1.Revision != 1 {
		t.Errorf("expected revision 1, got %d", result1.Revision)
	}

	// Bob inserts at position 1, based on revision 1 (sees Alice's insert)
	op2 := ot.NewInsert("b", 1, "bob")

	result2, err := q.Apply(op2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result2.Revision != 2 {
		t.Errorf("expected revision 2, got %d", result2.Revision)
	}
}

func TestQueue_Apply_BaseRevisionEqualsCurrentRevision(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)

	// Apply some operations first to get to a non-zero revision
	for i := range 3 {
		op := ot.NewInsert("x", i, "setup")
		_, _ = q.Apply(op, i)
	}

	// Now revision is 3
	if q.Revision() != 3 {
		t.Fatalf("expected revision 3, got %d", q.Revision())
	}

	// Apply an operation based on current revision (no transform needed)
	op := ot.NewInsert("y", 5, "alice")

	result, err := q.Apply(op, 3) // baseRevision == current revision
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Operation should be applied without transformation
	if result.Position != 5 {
		t.Errorf("expected position 5 (unchanged), got %d", result.Position)
	}

	if result.Revision != 4 {
		t.Errorf("expected revision 4, got %d", result.Revision)
	}
}

func TestQueue_Apply_ConcurrentOperations_NeedsTransform(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)

	// Alice inserts "a" at position 0, based on revision 0
	op1 := ot.NewInsert("a", 0, "alice")

	_, err := q.Apply(op1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bob also inserts at position 0, BUT based on revision 0 (hasn't seen Alice's op)
	// This simulates concurrent editing
	op2 := ot.NewInsert("b", 0, "bob")

	result2, err := q.Apply(op2, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bob's insert should be transformed: since alice < bob alphabetically,
	// Alice wins the tie-breaker and Bob shifts right to position 1
	if result2.Position != 1 {
		t.Errorf("expected Bob's position to shift to 1, got %d", result2.Position)
	}
}

func TestQueue_Apply_ConcurrentInsertAndDelete(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)

	// Start with a document "HELLO" conceptually
	// Alice inserts "X" at position 2, based on revision 0
	op1 := ot.NewInsert("X", 2, "alice")

	_, err := q.Apply(op1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bob deletes at position 3, based on revision 0 (hasn't seen Alice's insert)
	op2 := ot.NewDelete(3, "bob")

	result2, err := q.Apply(op2, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bob's delete should shift right because Alice inserted before it
	// Original position 3 → position 4 after Alice's insert at 2
	if result2.Position != 4 {
		t.Errorf("expected Bob's delete position to shift to 4, got %d", result2.Position)
	}
}

func TestQueue_Apply_MultipleTransforms(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)

	// Three sequential inserts at position 0, all based on revision 0
	// This simulates a slow client that hasn't received any updates
	op1 := ot.NewInsert("a", 0, "alice")
	_, _ = q.Apply(op1, 0)

	op2 := ot.NewInsert("b", 0, "bob")
	_, _ = q.Apply(op2, 0)

	// Carol's insert at position 0, still based on revision 0
	// Needs to be transformed against BOTH Alice's and Bob's operations
	op3 := ot.NewInsert("c", 0, "carol")

	result3, err := q.Apply(op3, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After transforms:
	// - Transform against Alice (alice < carol): Carol shifts to 1
	// - Transform against Bob (bob < carol): Carol shifts to 2
	if result3.Position != 2 {
		t.Errorf("expected Carol's position to be 2, got %d", result3.Position)
	}
}

func TestQueue_Apply_FutureRevision_Error(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)
	op := ot.NewInsert("a", 0, "alice")

	// Try to apply based on revision 5 when we're at revision 0
	_, err := q.Apply(op, 5)
	if err == nil {
		t.Error("expected error for future revision")
	}
}

func TestQueue_Apply_RevisionTooOld_Error(t *testing.T) {
	t.Parallel()

	// Small history size
	q := ot.NewQueue(2)

	// Fill up history beyond capacity
	for i := range 5 {
		op := ot.NewInsert("x", i, "user")
		_, _ = q.Apply(op, i)
	}

	// Try to apply based on revision 0 (which should be pruned from history)
	op := ot.NewInsert("y", 0, "late-user")
	_, err := q.Apply(op, 0)

	if !errors.Is(err, ot.ErrRevisionTooOld) {
		t.Errorf("expected ErrRevisionTooOld, got %v", err)
	}
}

func TestQueue_History(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)

	// Add some operations
	for i := range 5 {
		op := ot.NewInsert("x", i, "user")
		_, _ = q.Apply(op, i)
	}

	// Get history since revision 2
	history := q.History(2)

	if len(history) != 3 {
		t.Errorf("expected 3 operations in history, got %d", len(history))
	}

	// Should contain revisions 3, 4, 5
	expectedRevisions := []int{3, 4, 5}
	for i, op := range history {
		if op.Revision != expectedRevisions[i] {
			t.Errorf("expected revision %d, got %d", expectedRevisions[i], op.Revision)
		}
	}
}

func TestQueue_History_Empty(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(100)
	history := q.History(0)

	if len(history) != 0 {
		t.Errorf("expected empty history, got %d items", len(history))
	}
}

func TestQueue_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	q := ot.NewQueue(1000)

	var wg sync.WaitGroup

	// Simulate 10 concurrent clients each sending 100 operations
	for client := range 10 {
		wg.Add(1)

		go func(clientID int) {
			defer wg.Done()

			for range 100 {
				op := ot.NewInsert("x", 0, string(rune('a'+clientID)))
				rev := q.Revision()
				_, _ = q.Apply(op, rev)
			}
		}(client)
	}

	wg.Wait()

	// All 1000 operations should have been applied
	if q.Revision() != 1000 {
		t.Errorf("expected revision 1000, got %d", q.Revision())
	}
}

func TestQueue_HistoryPruning(t *testing.T) {
	t.Parallel()

	historySize := 5
	q := ot.NewQueue(historySize)

	// Add more operations than history size
	for i := range 10 {
		op := ot.NewInsert("x", i, "user")
		_, _ = q.Apply(op, i)
	}

	// History should only contain the last 5 operations
	history := q.History(0)

	if len(history) != historySize {
		t.Errorf("expected history size %d, got %d", historySize, len(history))
	}

	// Oldest should be revision 6 (revisions 1-5 pruned)
	if history[0].Revision != 6 {
		t.Errorf("expected oldest revision 6, got %d", history[0].Revision)
	}
}
