package acl

// Role represents a user's access level for a document.
type Role int

const (
	// Viewer can only read document content.
	Viewer Role = iota
	// Editor can read and write document content.
	Editor
	// Owner has full access: read, write, share, and delete.
	Owner
)

// String returns the string representation of the role.
func (r Role) String() string {
	switch r {
	case Viewer:
		return "viewer"
	case Editor:
		return "editor"
	case Owner:
		return "owner"
	default:
		return "unknown"
	}
}

// CanRead returns true if the role allows reading.
func (r Role) CanRead() bool {
	return r >= Viewer
}

// CanWrite returns true if the role allows writing.
func (r Role) CanWrite() bool {
	return r >= Editor
}

// CanShare returns true if the role allows sharing.
func (r Role) CanShare() bool {
	return r >= Owner
}

// CanDelete returns true if the role allows deletion.
func (r Role) CanDelete() bool {
	return r >= Owner
}

// Permission represents a user's access to a specific document.
type Permission struct {
	DocID  string
	UserID string
	Role   Role
}
