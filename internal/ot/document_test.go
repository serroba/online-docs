package ot_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/serroba/online-docs/internal/ot"
)

func TestDocument_NewDocument_Empty(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("")

	if doc.Content() != "" {
		t.Errorf("expected empty content, got %q", doc.Content())
	}

	if doc.Len() != 0 {
		t.Errorf("expected length 0, got %d", doc.Len())
	}
}

func TestDocument_NewDocument_WithContent(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument(testDocHello)

	if doc.Content() != testDocHello {
		t.Errorf("expected HELLO, got %q", doc.Content())
	}

	if doc.Len() != 5 {
		t.Errorf("expected length 5, got %d", doc.Len())
	}
}

func TestDocument_NewDocument_Unicode(t *testing.T) {
	t.Parallel()

	// Test with unicode characters (emoji, accents)
	// "héllo 🌍" = h + é + l + l + o + space + 🌍 = 7 characters
	doc := ot.NewDocument("héllo 🌍")

	// Should be 7 characters, not bytes
	if doc.Len() != 7 {
		t.Errorf("expected length 7, got %d", doc.Len())
	}

	if doc.Content() != "héllo 🌍" {
		t.Errorf("expected 'héllo 🌍', got %q", doc.Content())
	}
}

func TestDocument_Apply_InsertAtBeginning(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("ELLO")
	op := ot.NewInsert("H", 0, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != testDocHello {
		t.Errorf("expected HELLO, got %q", doc.Content())
	}
}

func TestDocument_Apply_InsertAtEnd(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("HELL")
	op := ot.NewInsert("O", 4, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != testDocHello {
		t.Errorf("expected HELLO, got %q", doc.Content())
	}
}

func TestDocument_Apply_InsertInMiddle(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("HLLO")
	op := ot.NewInsert("E", 1, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != testDocHello {
		t.Errorf("expected HELLO, got %q", doc.Content())
	}
}

func TestDocument_Apply_InsertIntoEmpty(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("")
	op := ot.NewInsert("A", 0, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != "A" {
		t.Errorf("expected A, got %q", doc.Content())
	}
}

func TestDocument_Apply_InsertInvalidPosition(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("ABC")

	// Position beyond length
	op := ot.NewInsert("X", 10, "user")
	err := doc.Apply(op)

	if !errors.Is(err, ot.ErrInvalidPosition) {
		t.Errorf("expected ErrInvalidPosition, got %v", err)
	}

	// Content should be unchanged
	if doc.Content() != "ABC" {
		t.Errorf("expected ABC, got %q", doc.Content())
	}
}

func TestDocument_Apply_DeleteAtBeginning(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument(testDocHello)
	op := ot.NewDelete(0, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != "ELLO" {
		t.Errorf("expected ELLO, got %q", doc.Content())
	}
}

func TestDocument_Apply_DeleteAtEnd(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument(testDocHello)
	op := ot.NewDelete(4, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != "HELL" {
		t.Errorf("expected HELL, got %q", doc.Content())
	}
}

func TestDocument_Apply_DeleteInMiddle(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument(testDocHello)
	op := ot.NewDelete(2, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != "HELO" {
		t.Errorf("expected HELO, got %q", doc.Content())
	}
}

func TestDocument_Apply_DeleteInvalidPosition(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("ABC")

	// Position beyond length
	op := ot.NewDelete(10, "user")
	err := doc.Apply(op)

	if !errors.Is(err, ot.ErrInvalidPosition) {
		t.Errorf("expected ErrInvalidPosition, got %v", err)
	}

	// Content should be unchanged
	if doc.Content() != "ABC" {
		t.Errorf("expected ABC, got %q", doc.Content())
	}
}

func TestDocument_Apply_DeleteFromEmpty(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("")
	op := ot.NewDelete(0, "user")

	err := doc.Apply(op)
	if !errors.Is(err, ot.ErrInvalidPosition) {
		t.Errorf("expected ErrInvalidPosition, got %v", err)
	}
}

func TestDocument_Apply_Noop(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument(testDocHello)

	// Create a no-op (position -1)
	op := ot.Operation{
		Type:     ot.Delete,
		Position: -1,
		UserID:   "user",
	}

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error for noop: %v", err)
	}

	// Content should be unchanged
	if doc.Content() != testDocHello {
		t.Errorf("expected HELLO, got %q", doc.Content())
	}
}

func TestDocument_Apply_UnknownOperationType(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument(testDocHello)

	// Create an operation with an invalid type
	op := ot.Operation{
		Type:     ot.OpType(99), // Invalid type
		Position: 0,
		UserID:   "user",
	}

	err := doc.Apply(op)
	if err == nil {
		t.Error("expected error for unknown operation type")
	}

	// Content should be unchanged
	if doc.Content() != testDocHello {
		t.Errorf("expected HELLO, got %q", doc.Content())
	}
}

func TestDocument_Apply_UnicodeInsert(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("hello")
	op := ot.NewInsert("🌍", 5, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != "hello🌍" {
		t.Errorf("expected 'hello🌍', got %q", doc.Content())
	}

	if doc.Len() != 6 {
		t.Errorf("expected length 6, got %d", doc.Len())
	}
}

func TestDocument_Apply_UnicodeDelete(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("héllo")

	// Delete the é (at position 1)
	op := ot.NewDelete(1, "user")

	err := doc.Apply(op)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Content() != "hllo" {
		t.Errorf("expected 'hllo', got %q", doc.Content())
	}
}

func TestDocument_Apply_MultipleOperations(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("")

	// Build "HELLO" character by character
	ops := []ot.Operation{
		ot.NewInsert("H", 0, "user"),
		ot.NewInsert("E", 1, "user"),
		ot.NewInsert("L", 2, "user"),
		ot.NewInsert("L", 3, "user"),
		ot.NewInsert("O", 4, "user"),
	}

	for _, op := range ops {
		if err := doc.Apply(op); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if doc.Content() != "HELLO" {
		t.Errorf("expected HELLO, got %q", doc.Content())
	}
}

func TestDocument_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument("")

	var wg sync.WaitGroup

	// 10 goroutines each inserting 100 characters
	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range 100 {
				op := ot.NewInsert("x", 0, "user")
				if err := doc.Apply(op); err != nil {
					t.Errorf("unexpected error applying op: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// Should have 1000 characters
	if doc.Len() != 1000 {
		t.Errorf("expected length 1000, got %d", doc.Len())
	}
}

// Integration test: Apply transformed operations from Queue.
func TestDocument_IntegrationWithQueue(t *testing.T) {
	t.Parallel()

	doc := ot.NewDocument(testDocHello)
	queue := ot.NewQueue(100)

	// Alice inserts "X" at position 2
	aliceOp := ot.NewInsert("X", 2, "alice")
	aliceResult, _ := queue.Apply(aliceOp, 0)

	// Bob deletes at position 2, based on revision 0 (concurrent with Alice)
	bobOp := ot.NewDelete(2, "bob")
	bobResult, _ := queue.Apply(bobOp, 0)

	// Apply both transformed operations to the document
	if err := doc.Apply(aliceResult.Operation); err != nil {
		t.Fatalf("failed to apply alice's op: %v", err)
	}

	if err := doc.Apply(bobResult.Operation); err != nil {
		t.Fatalf("failed to apply bob's op: %v", err)
	}

	// Expected result: "HEXLO"
	// - Alice inserted X at 2: HEXLLO
	// - Bob's delete transformed to position 3: HEXLO
	if doc.Content() != "HEXLO" {
		t.Errorf("expected HEXLO, got %q", doc.Content())
	}
}
