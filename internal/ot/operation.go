package ot

// OpType represents the type of operation.
type OpType int

const (
	Insert OpType = iota
	Delete
)

// Operation represents a single edit operation in the document.
type Operation struct {
	Type     OpType
	Position int    // Character position in the document
	Char     string // Character to insert (empty for delete)
	UserID   string // Used for tie-breaking concurrent inserts at same position
}

// NewInsert creates an insert operation.
func NewInsert(char string, position int, userID string) Operation {
	return Operation{
		Type:     Insert,
		Position: position,
		Char:     char,
		UserID:   userID,
	}
}

// NewDelete creates a delete operation.
func NewDelete(position int, userID string) Operation {
	return Operation{
		Type:     Delete,
		Position: position,
		UserID:   userID,
	}
}

// IsInsert returns true if this is an insert operation.
func (o Operation) IsInsert() bool {
	return o.Type == Insert
}

// IsDelete returns true if this is a delete operation.
func (o Operation) IsDelete() bool {
	return o.Type == Delete
}

// IsNoop returns true if the operation has become a no-op (position -1).
func (o Operation) IsNoop() bool {
	return o.Position < 0
}
