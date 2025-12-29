package storage_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/serroba/online-docs/internal/ot"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_CreateDocument(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	err := store.CreateDocument("doc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, err := store.DocumentExists("doc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !exists {
		t.Error("expected document to exist after creation")
	}
}

func TestMemoryStore_CreateDocument_AlreadyExists(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	require.NoError(t, store.CreateDocument("doc1"))

	err := store.CreateDocument("doc1")
	if !errors.Is(err, storage.ErrDocumentExists) {
		t.Errorf("expected ErrDocumentExists, got %v", err)
	}
}

func TestMemoryStore_DocumentExists_NotFound(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	exists, err := store.DocumentExists("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exists {
		t.Error("expected document to not exist")
	}
}

func TestMemoryStore_SaveAndLoadSnapshot(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	err := store.SaveSnapshot("doc1", 10, "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshot, err := store.LoadSnapshot("doc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.DocID != "doc1" {
		t.Errorf("expected docID doc1, got %s", snapshot.DocID)
	}

	if snapshot.Revision != 10 {
		t.Errorf("expected revision 10, got %d", snapshot.Revision)
	}

	if snapshot.Content != "hello world" {
		t.Errorf("expected content 'hello world', got %s", snapshot.Content)
	}

	if snapshot.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestMemoryStore_SaveSnapshot_DocumentNotFound(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	err := store.SaveSnapshot("nonexistent", 10, "content")
	if !errors.Is(err, storage.ErrDocumentNotFound) {
		t.Errorf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestMemoryStore_LoadSnapshot_DocumentNotFound(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	_, err := store.LoadSnapshot("nonexistent")
	if !errors.Is(err, storage.ErrDocumentNotFound) {
		t.Errorf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestMemoryStore_LoadSnapshot_NoSnapshot(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	_, err := store.LoadSnapshot("doc1")
	if !errors.Is(err, storage.ErrSnapshotNotFound) {
		t.Errorf("expected ErrSnapshotNotFound, got %v", err)
	}
}

func TestMemoryStore_AppendAndLoadOperations(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	ops := []ot.SequencedOperation{
		{Operation: ot.NewInsert("a", 0, "user"), Revision: 1},
		{Operation: ot.NewInsert("b", 1, "user"), Revision: 2},
		{Operation: ot.NewInsert("c", 2, "user"), Revision: 3},
	}

	for _, op := range ops {
		err := store.AppendOperation("doc1", op)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	loaded, err := store.LoadOperations("doc1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("expected 3 operations, got %d", len(loaded))
	}
}

func TestMemoryStore_AppendOperation_DocumentNotFound(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	op := ot.SequencedOperation{
		Operation: ot.NewInsert("a", 0, "user"),
		Revision:  1,
	}

	err := store.AppendOperation("nonexistent", op)
	if !errors.Is(err, storage.ErrDocumentNotFound) {
		t.Errorf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestMemoryStore_LoadOperations_SinceRevision(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	for i := 1; i <= 5; i++ {
		op := ot.SequencedOperation{
			Operation: ot.NewInsert("x", i-1, "user"),
			Revision:  i,
		}

		require.NoError(t, store.AppendOperation("doc1", op))
	}

	loaded, err := store.LoadOperations("doc1", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 operations (revisions 4, 5), got %d", len(loaded))
	}

	if loaded[0].Revision != 4 {
		t.Errorf("expected first op revision 4, got %d", loaded[0].Revision)
	}

	if loaded[1].Revision != 5 {
		t.Errorf("expected second op revision 5, got %d", loaded[1].Revision)
	}
}

func TestMemoryStore_LoadOperations_DocumentNotFound(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	_, err := store.LoadOperations("nonexistent", 0)
	if !errors.Is(err, storage.ErrDocumentNotFound) {
		t.Errorf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestMemoryStore_LatestRevision(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	// Initially 0 (document exists but no ops)
	rev, err := store.LatestRevision("doc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rev != 0 {
		t.Errorf("expected revision 0, got %d", rev)
	}

	// After operations
	for i := 1; i <= 3; i++ {
		op := ot.SequencedOperation{
			Operation: ot.NewInsert("x", 0, "user"),
			Revision:  i,
		}

		require.NoError(t, store.AppendOperation("doc1", op))
	}

	rev, err = store.LatestRevision("doc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rev != 3 {
		t.Errorf("expected revision 3, got %d", rev)
	}
}

func TestMemoryStore_LatestRevision_DocumentNotFound(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	_, err := store.LatestRevision("nonexistent")
	if !errors.Is(err, storage.ErrDocumentNotFound) {
		t.Errorf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestMemoryStore_LatestRevision_FromSnapshot(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))
	require.NoError(t, store.SaveSnapshot("doc1", 10, "content"))

	rev, err := store.LatestRevision("doc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rev != 10 {
		t.Errorf("expected revision 10, got %d", rev)
	}
}

func TestMemoryStore_SnapshotPrunesOperations(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	for i := 1; i <= 5; i++ {
		op := ot.SequencedOperation{
			Operation: ot.NewInsert("x", 0, "user"),
			Revision:  i,
		}

		require.NoError(t, store.AppendOperation("doc1", op))
	}

	require.NoError(t, store.SaveSnapshot("doc1", 3, "xxx"))

	ops, _ := store.LoadOperations("doc1", 0)

	if len(ops) != 2 {
		t.Errorf("expected 2 operations after prune, got %d", len(ops))
	}

	if ops[0].Revision != 4 {
		t.Errorf("expected first remaining op revision 4, got %d", ops[0].Revision)
	}
}

func TestMemoryStore_MultipleDocuments(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))
	require.NoError(t, store.CreateDocument("doc2"))

	require.NoError(t, store.AppendOperation("doc1", ot.SequencedOperation{
		Operation: ot.NewInsert("a", 0, "user"),
		Revision:  1,
	}))

	require.NoError(t, store.AppendOperation("doc2", ot.SequencedOperation{
		Operation: ot.NewInsert("b", 0, "user"),
		Revision:  1,
	}))

	ops1, _ := store.LoadOperations("doc1", 0)
	ops2, _ := store.LoadOperations("doc2", 0)

	if len(ops1) != 1 || len(ops2) != 1 {
		t.Errorf("expected 1 op each, got %d and %d", len(ops1), len(ops2))
	}

	if ops1[0].Char != "a" {
		t.Errorf("expected doc1 op char 'a', got %s", ops1[0].Char)
	}

	if ops2[0].Char != "b" {
		t.Errorf("expected doc2 op char 'b', got %s", ops2[0].Char)
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Add(1)

		go func(revision int) {
			defer wg.Done()

			op := ot.SequencedOperation{
				Operation: ot.NewInsert("x", 0, "user"),
				Revision:  revision,
			}

			// Note: Using _ here since require is not goroutine-safe
			_ = store.AppendOperation("doc1", op)
		}(i + 1)
	}

	wg.Wait()

	ops, _ := store.LoadOperations("doc1", 0)

	if len(ops) != 10 {
		t.Errorf("expected 10 operations, got %d", len(ops))
	}
}

func TestMemoryStore_SnapshotOverwrite(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	require.NoError(t, store.SaveSnapshot("doc1", 5, "first"))
	require.NoError(t, store.SaveSnapshot("doc1", 10, "second"))

	snapshot, _ := store.LoadSnapshot("doc1")

	if snapshot.Revision != 10 {
		t.Errorf("expected revision 10, got %d", snapshot.Revision)
	}

	if snapshot.Content != "second" {
		t.Errorf("expected content 'second', got %s", snapshot.Content)
	}
}
