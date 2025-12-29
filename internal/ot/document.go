package ot

import (
	"errors"
	"sync"
)

// ErrInvalidPosition is returned when an operation targets an invalid position.
var ErrInvalidPosition = errors.New("invalid position")

// Document represents the current state of a collaborative document.
// It is safe for concurrent use.
type Document struct {
	mu      sync.RWMutex
	content []rune
}

// NewDocument creates a new document with the given initial content.
func NewDocument(initial string) *Document {
	return &Document{
		content: []rune(initial),
	}
}

// Apply executes an operation on the document.
// No-op operations (position < 0) are silently ignored.
func (d *Document) Apply(op Operation) error {
	// Skip no-op operations
	if op.IsNoop() {
		return nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	switch op.Type {
	case Insert:
		return d.applyInsert(op)
	case Delete:
		return d.applyDelete(op)
	default:
		return errors.New("unknown operation type")
	}
}

// applyInsert inserts a character at the specified position.
func (d *Document) applyInsert(op Operation) error {
	if op.Position < 0 || op.Position > len(d.content) {
		return ErrInvalidPosition
	}

	chars := []rune(op.Char)

	// Insert at position
	newContent := make([]rune, 0, len(d.content)+len(chars))
	newContent = append(newContent, d.content[:op.Position]...)
	newContent = append(newContent, chars...)
	newContent = append(newContent, d.content[op.Position:]...)
	d.content = newContent

	return nil
}

// applyDelete removes a character at the specified position.
func (d *Document) applyDelete(op Operation) error {
	if op.Position < 0 || op.Position >= len(d.content) {
		return ErrInvalidPosition
	}

	// Delete at position
	newContent := make([]rune, 0, len(d.content)-1)
	newContent = append(newContent, d.content[:op.Position]...)
	newContent = append(newContent, d.content[op.Position+1:]...)
	d.content = newContent

	return nil
}

// Content returns the current document content as a string.
func (d *Document) Content() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return string(d.content)
}

// Len returns the number of characters in the document.
func (d *Document) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.content)
}
