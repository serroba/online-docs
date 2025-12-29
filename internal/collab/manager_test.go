package collab_test

import (
	"sync"
	"testing"

	"github.com/serroba/online-docs/internal/collab"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestManager_GetOrCreateSession(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
	})

	session, err := manager.GetOrCreateSession("doc1")
	require.NoError(t, err)

	if session == nil {
		t.Fatal("expected session, got nil")
	}

	if session.DocID() != "doc1" {
		t.Errorf("expected docID doc1, got %s", session.DocID())
	}

	// Getting again should return the same session
	session2, err := manager.GetOrCreateSession("doc1")
	require.NoError(t, err)

	if session != session2 {
		t.Error("expected same session instance")
	}
}

func TestManager_GetSession(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
	})

	// Before creating - should return nil
	session := manager.GetSession("doc1")
	if session != nil {
		t.Error("expected nil before creating")
	}

	// Create session
	_, err := manager.GetOrCreateSession("doc1")
	require.NoError(t, err)

	// After creating - should return session
	session = manager.GetSession("doc1")
	if session == nil {
		t.Error("expected session after creating")
	}
}

func TestManager_CloseSession(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
	})

	_, err := manager.GetOrCreateSession("doc1")
	require.NoError(t, err)

	if manager.SessionCount() != 1 {
		t.Errorf("expected 1 session, got %d", manager.SessionCount())
	}

	require.NoError(t, manager.CloseSession("doc1"))

	if manager.SessionCount() != 0 {
		t.Errorf("expected 0 sessions after close, got %d", manager.SessionCount())
	}

	// Closing non-existent should not error
	require.NoError(t, manager.CloseSession("doc1"))
}

func TestManager_CloseAll(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))
	require.NoError(t, store.CreateDocument("doc2"))
	require.NoError(t, store.CreateDocument("doc3"))

	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
	})

	_, err := manager.GetOrCreateSession("doc1")
	require.NoError(t, err)

	_, err = manager.GetOrCreateSession("doc2")
	require.NoError(t, err)

	_, err = manager.GetOrCreateSession("doc3")
	require.NoError(t, err)

	if manager.SessionCount() != 3 {
		t.Errorf("expected 3 sessions, got %d", manager.SessionCount())
	}

	require.NoError(t, manager.CloseAll())

	if manager.SessionCount() != 0 {
		t.Errorf("expected 0 sessions after CloseAll, got %d", manager.SessionCount())
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()

	for i := range 10 {
		docID := string(rune('a' + i))
		require.NoError(t, store.CreateDocument(docID))
	}

	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
	})

	var wg sync.WaitGroup

	// Concurrently create sessions
	for i := range 10 {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			docID := string(rune('a' + n))

			_, err := manager.GetOrCreateSession(docID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	wg.Wait()

	if manager.SessionCount() != 10 {
		t.Errorf("expected 10 sessions, got %d", manager.SessionCount())
	}
}

func TestManager_SessionCount(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))
	require.NoError(t, store.CreateDocument("doc2"))

	manager := collab.NewManager(collab.ManagerConfig{
		Store: store,
	})

	if manager.SessionCount() != 0 {
		t.Errorf("expected 0 sessions initially, got %d", manager.SessionCount())
	}

	_, err := manager.GetOrCreateSession("doc1")
	require.NoError(t, err)

	if manager.SessionCount() != 1 {
		t.Errorf("expected 1 session, got %d", manager.SessionCount())
	}

	_, err = manager.GetOrCreateSession("doc2")
	require.NoError(t, err)

	if manager.SessionCount() != 2 {
		t.Errorf("expected 2 sessions, got %d", manager.SessionCount())
	}
}
