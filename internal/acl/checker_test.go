package acl_test

import (
	"errors"
	"testing"

	"github.com/serroba/online-docs/internal/acl"
	"github.com/stretchr/testify/require"
)

func TestAction_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		action   acl.Action
		expected string
	}{
		{acl.ActionRead, "read"},
		{acl.ActionWrite, "write"},
		{acl.ActionShare, "share"},
		{acl.ActionDelete, "delete"},
		{acl.Action(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.action.String() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.action.String())
		}
	}
}

func TestChecker_CanPerform_Viewer(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()
	require.NoError(t, store.Grant("doc1", "user1", acl.Viewer))

	checker := acl.NewChecker(store)

	tests := []struct {
		action   acl.Action
		expected bool
	}{
		{acl.ActionRead, true},
		{acl.ActionWrite, false},
		{acl.ActionShare, false},
		{acl.ActionDelete, false},
	}

	for _, tt := range tests {
		allowed, err := checker.CanPerform("doc1", "user1", tt.action)
		require.NoError(t, err)

		if allowed != tt.expected {
			t.Errorf("action %s: expected %v, got %v", tt.action, tt.expected, allowed)
		}
	}
}

func TestChecker_CanPerform_Editor(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()
	require.NoError(t, store.Grant("doc1", "user1", acl.Editor))

	checker := acl.NewChecker(store)

	tests := []struct {
		action   acl.Action
		expected bool
	}{
		{acl.ActionRead, true},
		{acl.ActionWrite, true},
		{acl.ActionShare, false},
		{acl.ActionDelete, false},
	}

	for _, tt := range tests {
		allowed, err := checker.CanPerform("doc1", "user1", tt.action)
		require.NoError(t, err)

		if allowed != tt.expected {
			t.Errorf("action %s: expected %v, got %v", tt.action, tt.expected, allowed)
		}
	}
}

func TestChecker_CanPerform_Owner(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()
	require.NoError(t, store.Grant("doc1", "user1", acl.Owner))

	checker := acl.NewChecker(store)

	tests := []struct {
		action   acl.Action
		expected bool
	}{
		{acl.ActionRead, true},
		{acl.ActionWrite, true},
		{acl.ActionShare, true},
		{acl.ActionDelete, true},
	}

	for _, tt := range tests {
		allowed, err := checker.CanPerform("doc1", "user1", tt.action)
		require.NoError(t, err)

		if allowed != tt.expected {
			t.Errorf("action %s: expected %v, got %v", tt.action, tt.expected, allowed)
		}
	}
}

func TestChecker_CanPerform_NoPermission(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()
	checker := acl.NewChecker(store)

	allowed, err := checker.CanPerform("doc1", "user1", acl.ActionRead)
	require.NoError(t, err)

	if allowed {
		t.Error("expected false for user with no permission")
	}
}

func TestChecker_CanPerform_UnknownAction(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()
	require.NoError(t, store.Grant("doc1", "user1", acl.Owner))

	checker := acl.NewChecker(store)

	allowed, err := checker.CanPerform("doc1", "user1", acl.Action(99))
	require.NoError(t, err)

	if allowed {
		t.Error("expected false for unknown action")
	}
}

func TestChecker_RequirePermission_Allowed(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()
	require.NoError(t, store.Grant("doc1", "user1", acl.Editor))

	checker := acl.NewChecker(store)

	err := checker.RequirePermission("doc1", "user1", acl.ActionWrite)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestChecker_RequirePermission_Denied(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()
	require.NoError(t, store.Grant("doc1", "user1", acl.Viewer))

	checker := acl.NewChecker(store)

	err := checker.RequirePermission("doc1", "user1", acl.ActionWrite)
	if !errors.Is(err, acl.ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestChecker_RequirePermission_NoPermission(t *testing.T) {
	t.Parallel()

	store := acl.NewMemoryStore()
	checker := acl.NewChecker(store)

	err := checker.RequirePermission("doc1", "user1", acl.ActionRead)
	if !errors.Is(err, acl.ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

// errorStore is a mock store that returns errors for testing.
type errorStore struct {
	err error
}

func (e *errorStore) Grant(_, _ string, _ acl.Role) error {
	return e.err
}

func (e *errorStore) Revoke(_, _ string) error {
	return e.err
}

func (e *errorStore) GetRole(_, _ string) (acl.Role, error) {
	return 0, e.err
}

func (e *errorStore) ListPermissions(_ string) ([]acl.Permission, error) {
	return nil, e.err
}

func TestChecker_CanPerform_StoreError(t *testing.T) {
	t.Parallel()

	storeErr := errors.New("store error")
	store := &errorStore{err: storeErr}
	checker := acl.NewChecker(store)

	_, err := checker.CanPerform("doc1", "user1", acl.ActionRead)
	if !errors.Is(err, storeErr) {
		t.Errorf("expected store error, got %v", err)
	}
}

func TestChecker_RequirePermission_StoreError(t *testing.T) {
	t.Parallel()

	storeErr := errors.New("store error")
	store := &errorStore{err: storeErr}
	checker := acl.NewChecker(store)

	err := checker.RequirePermission("doc1", "user1", acl.ActionRead)
	if !errors.Is(err, storeErr) {
		t.Errorf("expected store error, got %v", err)
	}
}
