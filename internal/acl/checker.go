package acl

import "errors"

// Action represents an operation a user wants to perform.
type Action int

const (
	ActionRead Action = iota
	ActionWrite
	ActionShare
	ActionDelete
)

// String returns the string representation of the action.
func (a Action) String() string {
	switch a {
	case ActionRead:
		return "read"
	case ActionWrite:
		return "write"
	case ActionShare:
		return "share"
	case ActionDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Checker validates user permissions for document operations.
type Checker struct {
	store Store
}

// NewChecker creates a new permission checker.
func NewChecker(store Store) *Checker {
	return &Checker{store: store}
}

// CanPerform checks if a user can perform an action on a document.
func (c *Checker) CanPerform(docID, userID string, action Action) (bool, error) {
	role, err := c.store.GetRole(docID, userID)
	if err != nil {
		if errors.Is(err, ErrPermissionNotFound) {
			return false, nil
		}

		return false, err
	}

	switch action {
	case ActionRead:
		return role.CanRead(), nil
	case ActionWrite:
		return role.CanWrite(), nil
	case ActionShare:
		return role.CanShare(), nil
	case ActionDelete:
		return role.CanDelete(), nil
	default:
		return false, nil
	}
}

// RequirePermission checks permission and returns an error if denied.
func (c *Checker) RequirePermission(docID, userID string, action Action) error {
	allowed, err := c.CanPerform(docID, userID, action)
	if err != nil {
		return err
	}

	if !allowed {
		return ErrAccessDenied
	}

	return nil
}
