package ot

// Transform takes two concurrent operations and returns transformed versions
// that can be applied in either order to achieve the same final state.
//
// Given: op1 and op2 were created against the same document state.
// Returns: op1' (op1 transformed against op2), op2' (op2 transformed against op1).
func Transform(op1, op2 Operation) (Operation, Operation) {
	switch {
	case op1.IsInsert() && op2.IsInsert():
		return transformInsertInsert(op1, op2)
	case op1.IsDelete() && op2.IsDelete():
		return transformDeleteDelete(op1, op2)
	case op1.IsInsert() && op2.IsDelete():
		return transformInsertDelete(op1, op2)
	default:
		// op1 is Delete, op2 is Insert
		op2Prime, op1Prime := transformInsertDelete(op2, op1)

		return op1Prime, op2Prime
	}
}

// transformInsertInsert handles two concurrent inserts.
func transformInsertInsert(op1, op2 Operation) (Operation, Operation) {
	op1Prime := op1
	op2Prime := op2

	switch {
	case op1.Position < op2.Position:
		// op1 is before op2, so op2 needs to shift right
		op2Prime.Position++
	case op1.Position > op2.Position:
		// op2 is before op1, so op1 needs to shift right
		op1Prime.Position++
	default:
		// Same position: use UserID as tie-breaker
		// Lower UserID "wins" and stays in place, other shifts right
		if op1.UserID < op2.UserID {
			op2Prime.Position++
		} else {
			op1Prime.Position++
		}
	}

	return op1Prime, op2Prime
}

// transformDeleteDelete handles two concurrent deletes.
func transformDeleteDelete(op1, op2 Operation) (Operation, Operation) {
	op1Prime := op1
	op2Prime := op2

	switch {
	case op1.Position < op2.Position:
		// op1 deleted before op2's target, shift op2 left
		op2Prime.Position--
	case op1.Position > op2.Position:
		// op2 deleted before op1's target, shift op1 left
		op1Prime.Position--
	default:
		// Both deleting the same character - one becomes a no-op
		op1Prime.Position = -1 // Mark as no-op
		op2Prime.Position = -1 // Mark as no-op
	}

	return op1Prime, op2Prime
}

// transformInsertDelete handles insert (op1) vs delete (op2).
func transformInsertDelete(ins, del Operation) (Operation, Operation) {
	insPrime := ins
	delPrime := del

	if ins.Position <= del.Position {
		// Insert is at or before delete position
		// Delete position shifts right because of the insert
		delPrime.Position++
	} else {
		// Insert is after delete position
		// Insert position shifts left because of the delete
		insPrime.Position--
	}

	return insPrime, delPrime
}
