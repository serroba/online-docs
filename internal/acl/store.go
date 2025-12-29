package acl

import "errors"

// Common errors.
var (
	ErrPermissionNotFound = errors.New("permission not found")
	ErrAccessDenied       = errors.New("access denied")
)

// Store defines the interface for persisting document permissions.
type Store interface {
	// Grant gives a user a specific role on a document.
	// If the user already has a permission, it is replaced.
	Grant(docID, userID string, role Role) error

	// Revoke removes a user's permission on a document.
	// Returns ErrPermissionNotFound if no permission exists.
	Revoke(docID, userID string) error

	// GetRole returns the user's role for a document.
	// Returns ErrPermissionNotFound if no permission exists.
	GetRole(docID, userID string) (Role, error)

	// ListPermissions returns all permissions for a document.
	ListPermissions(docID string) ([]Permission, error)
}
