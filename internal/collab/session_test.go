package collab_test

import (
	"errors"
	"testing"

	"github.com/serroba/online-docs/internal/acl"
	"github.com/serroba/online-docs/internal/collab"
	"github.com/serroba/online-docs/internal/ot"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestSession_ApplyOperation(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	session := collab.NewSession(collab.SessionConfig{
		DocID: "doc1",
		Store: store,
	})

	require.NoError(t, session.Load())

	// Apply an insert operation
	rev, err := session.ApplyOperation("client1", "user1", ot.NewInsert("H", 0, "user1"), 0)
	require.NoError(t, err)

	if rev != 1 {
		t.Errorf("expected revision 1, got %d", rev)
	}

	content, revision, err := session.GetState("user1")
	require.NoError(t, err)

	if content != "H" {
		t.Errorf("expected content 'H', got %q", content)
	}

	if revision != 1 {
		t.Errorf("expected revision 1, got %d", revision)
	}
}

func TestSession_ApplyOperation_MultipleOps(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	session := collab.NewSession(collab.SessionConfig{
		DocID: "doc1",
		Store: store,
	})

	require.NoError(t, session.Load())

	// Build "HI"
	_, err := session.ApplyOperation("c1", "u1", ot.NewInsert("H", 0, "u1"), 0)
	require.NoError(t, err)

	_, err = session.ApplyOperation("c1", "u1", ot.NewInsert("I", 1, "u1"), 1)
	require.NoError(t, err)

	content, revision, err := session.GetState("u1")
	require.NoError(t, err)

	if content != "HI" {
		t.Errorf("expected 'HI', got %q", content)
	}

	if revision != 2 {
		t.Errorf("expected revision 2, got %d", revision)
	}
}

func TestSession_ApplyOperation_WithPermissions(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	permStore := acl.NewMemoryStore()
	require.NoError(t, permStore.Grant("doc1", "editor", acl.Editor))
	require.NoError(t, permStore.Grant("doc1", "viewer", acl.Viewer))

	session := collab.NewSession(collab.SessionConfig{
		DocID:       "doc1",
		Store:       store,
		PermChecker: acl.NewChecker(permStore),
	})

	require.NoError(t, session.Load())

	// Editor should succeed
	_, err := session.ApplyOperation("c1", "editor", ot.NewInsert("A", 0, "editor"), 0)
	require.NoError(t, err)

	// Viewer should fail
	_, err = session.ApplyOperation("c2", "viewer", ot.NewInsert("B", 1, "viewer"), 1)
	if !errors.Is(err, acl.ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestSession_GetState_WithPermissions(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	permStore := acl.NewMemoryStore()
	require.NoError(t, permStore.Grant("doc1", "viewer", acl.Viewer))

	session := collab.NewSession(collab.SessionConfig{
		DocID:       "doc1",
		Store:       store,
		PermChecker: acl.NewChecker(permStore),
	})

	require.NoError(t, session.Load())

	// Viewer should be able to read
	_, _, err := session.GetState("viewer")
	require.NoError(t, err)

	// Unknown user should fail
	_, _, err = session.GetState("unknown")
	if !errors.Is(err, acl.ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestSession_Load_WithExistingData(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))
	require.NoError(t, store.SaveSnapshot("doc1", 5, "hello"))

	session := collab.NewSession(collab.SessionConfig{
		DocID: "doc1",
		Store: store,
	})

	require.NoError(t, session.Load())

	content, revision, err := session.GetState("user")
	require.NoError(t, err)

	if content != "hello" {
		t.Errorf("expected 'hello', got %q", content)
	}

	if revision != 5 {
		t.Errorf("expected revision 5, got %d", revision)
	}
}

func TestSession_Close(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	session := collab.NewSession(collab.SessionConfig{
		DocID: "doc1",
		Store: store,
	})

	require.NoError(t, session.Load())

	_, err := session.ApplyOperation("c1", "u1", ot.NewInsert("X", 0, "u1"), 0)
	require.NoError(t, err)

	require.NoError(t, session.Close())

	// Operations after close should fail
	_, err = session.ApplyOperation("c1", "u1", ot.NewInsert("Y", 1, "u1"), 1)
	if !errors.Is(err, collab.ErrSessionClosed) {
		t.Errorf("expected ErrSessionClosed, got %v", err)
	}

	// GetState after close should fail
	_, _, err = session.GetState("u1")
	if !errors.Is(err, collab.ErrSessionClosed) {
		t.Errorf("expected ErrSessionClosed, got %v", err)
	}
}

func TestSession_DocID(t *testing.T) {
	t.Parallel()

	session := collab.NewSession(collab.SessionConfig{
		DocID: "my-doc",
		Store: storage.NewMemoryStore(),
	})

	if session.DocID() != "my-doc" {
		t.Errorf("expected 'my-doc', got %q", session.DocID())
	}
}

func TestSession_Revision(t *testing.T) {
	t.Parallel()

	store := storage.NewMemoryStore()
	require.NoError(t, store.CreateDocument("doc1"))

	session := collab.NewSession(collab.SessionConfig{
		DocID: "doc1",
		Store: store,
	})

	require.NoError(t, session.Load())

	if session.Revision() != 0 {
		t.Errorf("expected revision 0, got %d", session.Revision())
	}

	_, err := session.ApplyOperation("c1", "u1", ot.NewInsert("A", 0, "u1"), 0)
	require.NoError(t, err)

	if session.Revision() != 1 {
		t.Errorf("expected revision 1, got %d", session.Revision())
	}
}
