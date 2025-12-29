package storage_test

import (
	"errors"
	"testing"

	"github.com/serroba/online-docs/internal/ot"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestSnapshotPolicy_TriggersAtThreshold(t *testing.T) {
	t.Parallel()

	policy := storage.NewSnapshotPolicy(5)

	// First 4 operations should not trigger
	for i := range 4 {
		shouldSnapshot := policy.RecordOperation("doc1")
		if shouldSnapshot {
			t.Errorf("should not trigger snapshot at operation %d", i+1)
		}
	}

	// 5th operation should trigger
	shouldSnapshot := policy.RecordOperation("doc1")
	if !shouldSnapshot {
		t.Error("should trigger snapshot at threshold")
	}
}

func TestSnapshotPolicy_Reset(t *testing.T) {
	t.Parallel()

	policy := storage.NewSnapshotPolicy(3)

	// Record operations until threshold
	for range 3 {
		_ = policy.RecordOperation("doc1")
	}

	// Reset
	policy.Reset("doc1")

	// Counter should be back to 0
	count := policy.OperationsSinceSnapshot("doc1")
	if count != 0 {
		t.Errorf("expected count 0 after reset, got %d", count)
	}

	// Should need 3 more operations to trigger
	for i := range 2 {
		shouldSnapshot := policy.RecordOperation("doc1")
		if shouldSnapshot {
			t.Errorf("should not trigger at operation %d after reset", i+1)
		}
	}

	shouldSnapshot := policy.RecordOperation("doc1")
	if !shouldSnapshot {
		t.Error("should trigger at threshold after reset")
	}
}

func TestSnapshotPolicy_MultipleDocuments(t *testing.T) {
	t.Parallel()

	policy := storage.NewSnapshotPolicy(3)

	// Record 2 operations for doc1
	_ = policy.RecordOperation("doc1")
	_ = policy.RecordOperation("doc1")

	// Record 2 operations for doc2
	_ = policy.RecordOperation("doc2")
	_ = policy.RecordOperation("doc2")

	// Neither should be at threshold yet
	if policy.OperationsSinceSnapshot("doc1") != 2 {
		t.Errorf("expected doc1 count 2, got %d", policy.OperationsSinceSnapshot("doc1"))
	}

	if policy.OperationsSinceSnapshot("doc2") != 2 {
		t.Errorf("expected doc2 count 2, got %d", policy.OperationsSinceSnapshot("doc2"))
	}

	// One more for doc1 should trigger
	shouldSnapshot := policy.RecordOperation("doc1")
	if !shouldSnapshot {
		t.Error("doc1 should trigger snapshot")
	}

	// doc2 should still not trigger
	if policy.OperationsSinceSnapshot("doc2") != 2 {
		t.Errorf("doc2 should still be at 2, got %d", policy.OperationsSinceSnapshot("doc2"))
	}
}

func TestSnapshotPolicy_OperationsSinceSnapshot(t *testing.T) {
	t.Parallel()

	policy := storage.NewSnapshotPolicy(10)

	// Initially 0
	if policy.OperationsSinceSnapshot("doc1") != 0 {
		t.Errorf("expected 0, got %d", policy.OperationsSinceSnapshot("doc1"))
	}

	// After 5 operations
	for range 5 {
		_ = policy.RecordOperation("doc1")
	}

	if policy.OperationsSinceSnapshot("doc1") != 5 {
		t.Errorf("expected 5, got %d", policy.OperationsSinceSnapshot("doc1"))
	}
}

func TestDocumentLoader_LoadEmpty(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	loader := storage.NewDocumentLoader(store)

	result, err := loader.Load("doc1", mockApplyOp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsNew {
		t.Error("expected IsNew to be true")
	}

	if result.Content != "" {
		t.Errorf("expected empty content, got %q", result.Content)
	}

	if result.Revision != 0 {
		t.Errorf("expected revision 0, got %d", result.Revision)
	}
}

func TestDocumentLoader_LoadFromSnapshot(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))
	require.NoError(t, store.SaveSnapshot("doc1", 10, "hello"))

	loader := storage.NewDocumentLoader(store)

	result, err := loader.Load("doc1", mockApplyOp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsNew {
		t.Error("expected IsNew to be false")
	}

	if result.Content != "hello" {
		t.Errorf("expected content 'hello', got %q", result.Content)
	}

	if result.Revision != 10 {
		t.Errorf("expected revision 10, got %d", result.Revision)
	}
}

func TestDocumentLoader_LoadWithReplay(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	// Snapshot at revision 2 with content "ab"
	require.NoError(t, store.SaveSnapshot("doc1", 2, "ab"))

	// Operations since snapshot
	require.NoError(t, store.AppendOperation("doc1", ot.SequencedOperation{
		Operation: ot.NewInsert("c", 2, "user"),
		Revision:  3,
	}))
	require.NoError(t, store.AppendOperation("doc1", ot.SequencedOperation{
		Operation: ot.NewInsert("d", 3, "user"),
		Revision:  4,
	}))

	loader := storage.NewDocumentLoader(store)

	result, err := loader.Load("doc1", mockApplyOp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be "ab" + "c" + "d" = "abcd"
	if result.Content != "abcd" {
		t.Errorf("expected content 'abcd', got %q", result.Content)
	}

	if result.Revision != 4 {
		t.Errorf("expected revision 4, got %d", result.Revision)
	}
}

func TestDocumentLoader_LoadOperationsOnly(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	// No snapshot, just operations
	require.NoError(t, store.AppendOperation("doc1", ot.SequencedOperation{
		Operation: ot.NewInsert("a", 0, "user"),
		Revision:  1,
	}))
	require.NoError(t, store.AppendOperation("doc1", ot.SequencedOperation{
		Operation: ot.NewInsert("b", 1, "user"),
		Revision:  2,
	}))

	loader := storage.NewDocumentLoader(store)

	result, err := loader.Load("doc1", mockApplyOp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content != "ab" {
		t.Errorf("expected content 'ab', got %q", result.Content)
	}

	if result.Revision != 2 {
		t.Errorf("expected revision 2, got %d", result.Revision)
	}

	if result.IsNew {
		t.Error("expected IsNew to be false when operations exist")
	}
}

func TestDocumentLoader_LoadOperationsError(t *testing.T) {
	t.Parallel()

	store := &errorStore{
		loadOpsErr: errors.New("load ops failed"),
	}
	loader := storage.NewDocumentLoader(store)

	_, err := loader.Load("doc1", mockApplyOp)
	if err == nil {
		t.Error("expected error from LoadOperations")
	}
}

func TestDocumentLoader_ApplyOpError(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))
	require.NoError(t, store.AppendOperation("doc1", ot.SequencedOperation{
		Operation: ot.NewInsert("a", 0, "user"),
		Revision:  1,
	}))

	loader := storage.NewDocumentLoader(store)

	failingApply := func(_ string, _ storage.Operation) (string, error) {
		return "", errors.New("apply failed")
	}

	_, err := loader.Load("doc1", failingApply)
	if err == nil {
		t.Error("expected error from applyOp")
	}
}

func TestDocumentLoader_LoadSnapshotError(t *testing.T) {
	t.Parallel()

	store := &errorStore{
		loadSnapshotErr: errors.New("snapshot error"),
	}
	loader := storage.NewDocumentLoader(store)

	_, err := loader.Load("doc1", mockApplyOp)
	if err == nil {
		t.Error("expected error from LoadSnapshot")
	}
}

// errorStore is a mock store that returns errors for testing.
type errorStore struct {
	loadSnapshotErr error
	loadOpsErr      error
}

func (e *errorStore) CreateDocument(_ string) error {
	return nil
}

func (e *errorStore) DocumentExists(_ string) (bool, error) {
	return true, nil
}

func (e *errorStore) SaveSnapshot(_ string, _ int, _ string) error {
	return nil
}

func (e *errorStore) LoadSnapshot(_ string) (storage.Snapshot, error) {
	if e.loadSnapshotErr != nil {
		return storage.Snapshot{}, e.loadSnapshotErr
	}

	return storage.Snapshot{}, storage.ErrSnapshotNotFound
}

func (e *errorStore) AppendOperation(_ string, _ ot.SequencedOperation) error {
	return nil
}

func (e *errorStore) LoadOperations(_ string, _ int) ([]ot.SequencedOperation, error) {
	return nil, e.loadOpsErr
}

func (e *errorStore) LatestRevision(_ string) (int, error) {
	return 0, nil
}

func (e *errorStore) DeleteDocument(_ string) error {
	return nil
}

// mockApplyOp simulates applying an operation to content.
func mockApplyOp(content string, op storage.Operation) (string, error) {
	runes := []rune(content)

	if op.Type == int(ot.Insert) {
		// Insert
		newRunes := make([]rune, 0, len(runes)+len(op.Char))
		newRunes = append(newRunes, runes[:op.Position]...)
		newRunes = append(newRunes, []rune(op.Char)...)
		newRunes = append(newRunes, runes[op.Position:]...)

		return string(newRunes), nil
	}

	// Delete
	newRunes := make([]rune, 0, len(runes)-1)
	newRunes = append(newRunes, runes[:op.Position]...)
	newRunes = append(newRunes, runes[op.Position+1:]...)

	return string(newRunes), nil
}
