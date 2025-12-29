package ws

import (
	"encoding/json"
	"sync"
)

// Conn abstracts a WebSocket connection for testability.
type Conn interface {
	WriteJSON(v any) error
	ReadJSON(v any) error
	Close() error
}

// Client represents a connected user.
type Client struct {
	ID     string
	UserID string
	conn   Conn

	mu    sync.Mutex
	docID string // Currently subscribed document
}

// NewClient creates a new client wrapper.
func NewClient(id, userID string, conn Conn) *Client {
	return &Client{
		ID:     id,
		UserID: userID,
		conn:   conn,
	}
}

// Send sends a message to the client.
func (c *Client) Send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.WriteJSON(msg)
}

// SendError sends an error message to the client.
func (c *Client) SendError(code, message string) error {
	return c.Send(Message{
		Type: MessageTypeError,
		Payload: ErrorPayload{
			Code:    code,
			Message: message,
		},
	})
}

// Receive reads a message from the client.
func (c *Client) Receive() (Message, error) {
	var raw struct {
		Type    MessageType     `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := c.conn.ReadJSON(&raw); err != nil {
		return Message{}, err
	}

	msg := Message{Type: raw.Type}

	// Parse payload based on message type
	switch raw.Type {
	case MessageTypeOperation:
		var payload OperationPayload
		if err := json.Unmarshal(raw.Payload, &payload); err != nil {
			return Message{}, err
		}

		msg.Payload = payload
	case MessageTypeSync:
		// Sync has no payload, just the doc ID in a simple struct
		var payload struct {
			DocID string `json:"docId"`
		}
		if err := json.Unmarshal(raw.Payload, &payload); err != nil {
			return Message{}, err
		}

		msg.Payload = payload
	case MessageTypeAck, MessageTypeBroadcast, MessageTypeState, MessageTypeError:
		// Server-to-client messages - keep raw payload
		msg.Payload = raw.Payload
	}

	return msg, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// DocID returns the document the client is subscribed to.
func (c *Client) DocID() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.docID
}

// SetDocID sets the document the client is subscribed to.
func (c *Client) SetDocID(docID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.docID = docID
}
