package acl_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/serroba/online-docs/internal/acl"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_Grant(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	require.NoError(t, store.Grant("doc1", "user1", acl.Editor))

	role, err := store.GetRole("doc1", "user1")
	require.NoError(t, err)

	if role != acl.Editor {
		t.Errorf("expected Editor, got %v", role)
	}
}

func TestMemoryStore_Grant_OverwritesExisting(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	require.NoError(t, store.Grant("doc1", "user1", acl.Viewer))
	require.NoError(t, store.Grant("doc1", "user1", acl.Owner))

	role, err := store.GetRole("doc1", "user1")
	require.NoError(t, err)

	if role != acl.Owner {
		t.Errorf("expected Owner after overwrite, got %v", role)
	}
}

func TestMemoryStore_Revoke(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	require.NoError(t, store.Grant("doc1", "user1", acl.Editor))
	require.NoError(t, store.Revoke("doc1", "user1"))

	_, err := store.GetRole("doc1", "user1")
	if !errors.Is(err, acl.ErrPermissionNotFound) {
		t.Errorf("expected ErrPermissionNotFound, got %v", err)
	}
}

func TestMemoryStore_Revoke_NotFound(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	err := store.Revoke("doc1", "user1")
	if !errors.Is(err, acl.ErrPermissionNotFound) {
		t.Errorf("expected ErrPermissionNotFound, got %v", err)
	}
}

func TestMemoryStore_GetRole_NotFound(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	_, err := store.GetRole("doc1", "user1")
	if !errors.Is(err, acl.ErrPermissionNotFound) {
		t.Errorf("expected ErrPermissionNotFound, got %v", err)
	}
}

func TestMemoryStore_ListPermissions(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	require.NoError(t, store.Grant("doc1", "user1", acl.Owner))
	require.NoError(t, store.Grant("doc1", "user2", acl.Editor))
	require.NoError(t, store.Grant("doc1", "user3", acl.Viewer))
	require.NoError(t, store.Grant("doc2", "user1", acl.Owner)) // Different doc

	perms, err := store.ListPermissions("doc1")
	require.NoError(t, err)

	if len(perms) != 3 {
		t.Errorf("expected 3 permissions, got %d", len(perms))
	}

	// Verify all are for doc1
	for _, p := range perms {
		if p.DocID != "doc1" {
			t.Errorf("expected docID doc1, got %s", p.DocID)
		}
	}
}

func TestMemoryStore_ListPermissions_Empty(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	perms, err := store.ListPermissions("doc1")
	require.NoError(t, err)

	if len(perms) != 0 {
		t.Errorf("expected 0 permissions, got %d", len(perms))
	}
}

func TestMemoryStore_MultipleDocuments(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	require.NoError(t, store.Grant("doc1", "user1", acl.Owner))
	require.NoError(t, store.Grant("doc2", "user1", acl.Viewer))

	role1, err := store.GetRole("doc1", "user1")
	require.NoError(t, err)

	role2, err := store.GetRole("doc2", "user1")
	require.NoError(t, err)

	if role1 != acl.Owner {
		t.Errorf("expected Owner for doc1, got %v", role1)
	}

	if role2 != acl.Viewer {
		t.Errorf("expected Viewer for doc2, got %v", role2)
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Add(1)

		go func(userNum int) {
			defer wg.Done()

			userID := "user" + string(rune('0'+userNum))
			// Note: Using _ here since require is not goroutine-safe
			_ = store.Grant("doc1", userID, acl.Editor)
		}(i)
	}

	wg.Wait()

	perms, err := store.ListPermissions("doc1")
	require.NoError(t, err)

	if len(perms) != 10 {
		t.Errorf("expected 10 permissions, got %d", len(perms))
	}
}
