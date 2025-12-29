package ot_test

import (
	"testing"

	"github.com/serroba/online-docs/internal/ot"
)

const testDocHello = "HELLO"

func TestTransform_InsertVsInsert_DifferentPositions(t *testing.T) {
	t.Parallel()

	// Insert at position 2, Insert at position 5
	// op1 is before op2, so op2 should shift right
	op1 := ot.NewInsert("a", 2, "alice")
	op2 := ot.NewInsert("b", 5, "bob")

	op1Prime, op2Prime := ot.Transform(op1, op2)

	if op1Prime.Position != 2 {
		t.Errorf("op1 position should stay at 2, got %d", op1Prime.Position)
	}

	if op2Prime.Position != 6 {
		t.Errorf("op2 position should shift to 6, got %d", op2Prime.Position)
	}
}

func TestTransform_InsertVsInsert_SamePosition_TieBreaker(t *testing.T) {
	t.Parallel()

	// Both insert at position 2, alice < bob alphabetically
	op1 := ot.NewInsert("a", 2, "alice")
	op2 := ot.NewInsert("b", 2, "bob")

	op1Prime, op2Prime := ot.Transform(op1, op2)

	// alice wins (lower UserID), bob shifts right
	if op1Prime.Position != 2 {
		t.Errorf("alice should stay at 2, got %d", op1Prime.Position)
	}

	if op2Prime.Position != 3 {
		t.Errorf("bob should shift to 3, got %d", op2Prime.Position)
	}
}

func TestTransform_DeleteVsDelete_DifferentPositions(t *testing.T) {
	t.Parallel()

	// Delete at position 2, Delete at position 5
	op1 := ot.NewDelete(2, "alice")
	op2 := ot.NewDelete(5, "bob")

	op1Prime, op2Prime := ot.Transform(op1, op2)

	// op1 deletes before op2, so op2 shifts left
	if op1Prime.Position != 2 {
		t.Errorf("op1 position should stay at 2, got %d", op1Prime.Position)
	}

	if op2Prime.Position != 4 {
		t.Errorf("op2 position should shift to 4, got %d", op2Prime.Position)
	}
}

func TestTransform_DeleteVsDelete_Op1AfterOp2(t *testing.T) {
	t.Parallel()

	// Delete at position 5, Delete at position 2
	// op1 is AFTER op2, so op1 shifts left
	op1 := ot.NewDelete(5, "alice")
	op2 := ot.NewDelete(2, "bob")

	op1Prime, op2Prime := ot.Transform(op1, op2)

	// op2 deletes before op1, so op1 shifts left
	if op1Prime.Position != 4 {
		t.Errorf("op1 position should shift to 4, got %d", op1Prime.Position)
	}

	if op2Prime.Position != 2 {
		t.Errorf("op2 position should stay at 2, got %d", op2Prime.Position)
	}
}

func TestTransform_DeleteVsDelete_SamePosition(t *testing.T) {
	t.Parallel()

	// Both trying to delete the same character
	op1 := ot.NewDelete(3, "alice")
	op2 := ot.NewDelete(3, "bob")

	op1Prime, op2Prime := ot.Transform(op1, op2)

	// Both become no-ops (character already deleted by the other)
	if !op1Prime.IsNoop() {
		t.Errorf("op1 should be no-op, got position %d", op1Prime.Position)
	}

	if !op2Prime.IsNoop() {
		t.Errorf("op2 should be no-op, got position %d", op2Prime.Position)
	}
}

func TestTransform_InsertVsDelete_InsertBefore(t *testing.T) {
	t.Parallel()

	// Insert at 2, Delete at 5
	// Insert is before delete, so delete shifts right
	op1 := ot.NewInsert("x", 2, "alice")
	op2 := ot.NewDelete(5, "bob")

	op1Prime, op2Prime := ot.Transform(op1, op2)

	if op1Prime.Position != 2 {
		t.Errorf("insert should stay at 2, got %d", op1Prime.Position)
	}

	if op2Prime.Position != 6 {
		t.Errorf("delete should shift to 6, got %d", op2Prime.Position)
	}
}

func TestTransform_InsertVsDelete_InsertAfter(t *testing.T) {
	t.Parallel()

	// Insert at 5, Delete at 2
	// Delete is before insert, so insert shifts left
	op1 := ot.NewInsert("x", 5, "alice")
	op2 := ot.NewDelete(2, "bob")

	op1Prime, op2Prime := ot.Transform(op1, op2)

	if op1Prime.Position != 4 {
		t.Errorf("insert should shift to 4, got %d", op1Prime.Position)
	}

	if op2Prime.Position != 2 {
		t.Errorf("delete should stay at 2, got %d", op2Prime.Position)
	}
}

func TestTransform_DeleteVsInsert(t *testing.T) {
	t.Parallel()

	// Delete at 5, Insert at 2
	// Insert is before delete, so delete shifts right
	op1 := ot.NewDelete(5, "alice")
	op2 := ot.NewInsert("x", 2, "bob")

	op1Prime, op2Prime := ot.Transform(op1, op2)

	if op1Prime.Position != 6 {
		t.Errorf("delete should shift to 6, got %d", op1Prime.Position)
	}

	if op2Prime.Position != 2 {
		t.Errorf("insert should stay at 2, got %d", op2Prime.Position)
	}
}

// Integration test: The HELLO example from our walkthrough.
func TestTransform_HelloExample(t *testing.T) {
	t.Parallel()

	// Document: "HELLO" (positions 0-4)
	// Alice: Insert "X" at position 2 (after "E")
	// Bob: Delete at position 2 (first "L")

	alice := ot.NewInsert("X", 2, "alice")
	bob := ot.NewDelete(2, "bob")

	alicePrime, bobPrime := ot.Transform(alice, bob)

	// After transform:
	// - Alice's insert: position 2 stays (insert <= delete position)
	// - Bob's delete: shifts to position 3 (because alice inserted before)

	if alicePrime.Position != 2 {
		t.Errorf("alice insert should stay at 2, got %d", alicePrime.Position)
	}

	if bobPrime.Position != 3 {
		t.Errorf("bob delete should shift to 3, got %d", bobPrime.Position)
	}

	// Verify convergence by simulating both orderings
	doc := testDocHello

	// Path 1: Apply alice first, then transformed bob
	path1 := applyInsert(doc, alicePrime.Position, alicePrime.Char)
	path1 = applyDelete(path1, bobPrime.Position)

	// Path 2: Apply bob first, then transformed alice
	path2 := applyDelete(doc, bob.Position)
	path2 = applyInsert(path2, alicePrime.Position, alicePrime.Char)

	if path1 != path2 {
		t.Errorf("documents diverged!\nPath1: %s\nPath2: %s", path1, path2)
	}

	expected := "HEXLO"
	if path1 != expected {
		t.Errorf("expected %s, got %s", expected, path1)
	}
}

// Helper functions to simulate document operations.
func applyInsert(doc string, pos int, char string) string {
	if pos < 0 || pos > len(doc) {
		return doc
	}

	return doc[:pos] + char + doc[pos:]
}

func applyDelete(doc string, pos int) string {
	if pos < 0 || pos >= len(doc) {
		return doc
	}

	return doc[:pos] + doc[pos+1:]
}
