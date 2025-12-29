package acl_test

import (
	"testing"

	"github.com/serroba/online-docs/internal/acl"
)

func TestRole_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		role     acl.Role
		expected string
	}{
		{acl.Viewer, "viewer"},
		{acl.Editor, "editor"},
		{acl.Owner, "owner"},
		{acl.Role(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.role.String() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.role.String())
		}
	}
}

func TestRole_Permissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		role      acl.Role
		canRead   bool
		canWrite  bool
		canShare  bool
		canDelete bool
	}{
		{acl.Viewer, true, false, false, false},
		{acl.Editor, true, true, false, false},
		{acl.Owner, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.role.String(), func(t *testing.T) {
			t.Parallel()

			if tt.role.CanRead() != tt.canRead {
				t.Errorf("CanRead: expected %v, got %v", tt.canRead, tt.role.CanRead())
			}

			if tt.role.CanWrite() != tt.canWrite {
				t.Errorf("CanWrite: expected %v, got %v", tt.canWrite, tt.role.CanWrite())
			}

			if tt.role.CanShare() != tt.canShare {
				t.Errorf("CanShare: expected %v, got %v", tt.canShare, tt.role.CanShare())
			}

			if tt.role.CanDelete() != tt.canDelete {
				t.Errorf("CanDelete: expected %v, got %v", tt.canDelete, tt.role.CanDelete())
			}
		})
	}
}
