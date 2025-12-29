package ws

// MessageType identifies the kind of WebSocket message.
type MessageType string

const (
	// Client to Server messages.
	MessageTypeOperation MessageType = "operation" // Client submits an edit
	MessageTypeSync      MessageType = "sync"      // Client requests current state

	// Server to Client messages.
	MessageTypeAck       MessageType = "ack"       // Server confirms operation applied
	MessageTypeBroadcast MessageType = "broadcast" // Server pushes operation to clients
	MessageTypeState     MessageType = "state"     // Server sends full document state
	MessageTypeError     MessageType = "error"     // Server reports an error
)

// Message is the envelope for all WebSocket communication.
type Message struct {
	Type    MessageType `json:"type"`
	Payload any         `json:"payload,omitempty"`
}

// OperationPayload is sent when a client submits an edit.
type OperationPayload struct {
	DocID        string `json:"docId"`
	BaseRevision int    `json:"baseRevision"`
	OpType       int    `json:"opType"` // 0 = insert, 1 = delete
	Position     int    `json:"position"`
	Char         string `json:"char,omitempty"`
}

// AckPayload confirms an operation was applied.
type AckPayload struct {
	Revision int `json:"revision"` // The assigned revision number
}

// BroadcastPayload pushes an operation to other clients.
type BroadcastPayload struct {
	DocID    string `json:"docId"`
	Revision int    `json:"revision"`
	OpType   int    `json:"opType"`
	Position int    `json:"position"`
	Char     string `json:"char,omitempty"`
	UserID   string `json:"userId"`
}

// StatePayload sends the full document state.
type StatePayload struct {
	DocID    string `json:"docId"`
	Content  string `json:"content"`
	Revision int    `json:"revision"`
}

// ErrorPayload reports an error to the client.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error codes.
const (
	ErrorCodeAccessDenied   = "access_denied"
	ErrorCodeInvalidMessage = "invalid_message"
	ErrorCodeInternalError  = "internal_error"
)
