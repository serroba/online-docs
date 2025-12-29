package acl

import "sync"

// permissionKey uniquely identifies a user-document permission.
type permissionKey struct {
	docID  string
	userID string
}

// MemoryStore is an in-memory implementation of the Store interface.
type MemoryStore struct {
	mu          sync.RWMutex
	permissions map[permissionKey]Role
}

// NewMemoryStore creates a new in-memory permission store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		permissions: make(map[permissionKey]Role),
	}
}

// Grant gives a user a specific role on a document.
func (m *MemoryStore) Grant(docID, userID string, role Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := permissionKey{docID: docID, userID: userID}
	m.permissions[key] = role

	return nil
}

// Revoke removes a user's permission on a document.
func (m *MemoryStore) Revoke(docID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := permissionKey{docID: docID, userID: userID}

	if _, exists := m.permissions[key]; !exists {
		return ErrPermissionNotFound
	}

	delete(m.permissions, key)

	return nil
}

// GetRole returns the user's role for a document.
func (m *MemoryStore) GetRole(docID, userID string) (Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := permissionKey{docID: docID, userID: userID}

	role, exists := m.permissions[key]
	if !exists {
		return 0, ErrPermissionNotFound
	}

	return role, nil
}

// ListPermissions returns all permissions for a document.
func (m *MemoryStore) ListPermissions(docID string) ([]Permission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Permission

	for key, role := range m.permissions {
		if key.docID == docID {
			result = append(result, Permission{
				DocID:  key.docID,
				UserID: key.userID,
				Role:   role,
			})
		}
	}

	return result, nil
}

// Ensure MemoryStore implements Store.
var _ Store = (*MemoryStore)(nil)
