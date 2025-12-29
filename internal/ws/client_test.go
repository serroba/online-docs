package ws_test

import (
	"testing"

	"github.com/serroba/online-docs/internal/ws"
)

func TestClient_Send(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	msg := ws.Message{
		Type: ws.MessageTypeAck,
		Payload: ws.AckPayload{
			Revision: 5,
		},
	}

	err := client.Send(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := conn.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Type != ws.MessageTypeAck {
		t.Errorf("expected ack type, got %s", messages[0].Type)
	}
}

func TestClient_SendError(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	err := client.SendError(ws.ErrorCodeAccessDenied, "not allowed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := conn.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Type != ws.MessageTypeError {
		t.Errorf("expected error type, got %s", messages[0].Type)
	}
}

func TestClient_Close(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	err := client.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !conn.IsClosed() {
		t.Error("expected connection to be closed")
	}
}

func TestClient_DocID(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	if client.DocID() != "" {
		t.Errorf("expected empty docID, got %s", client.DocID())
	}

	client.SetDocID("doc1")

	if client.DocID() != "doc1" {
		t.Errorf("expected doc1, got %s", client.DocID())
	}
}

func TestClient_Receive_Operation(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	// Send a message to the incoming channel
	conn.incoming <- ws.Message{
		Type: ws.MessageTypeOperation,
		Payload: ws.OperationPayload{
			DocID:        "doc1",
			BaseRevision: 5,
			OpType:       0,
			Position:     10,
			Char:         "a",
		},
	}

	msg, err := client.Receive()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Type != ws.MessageTypeOperation {
		t.Errorf("expected operation type, got %s", msg.Type)
	}

	payload, ok := msg.Payload.(ws.OperationPayload)
	if !ok {
		t.Fatal("expected OperationPayload")
	}

	if payload.DocID != "doc1" {
		t.Errorf("expected docId doc1, got %s", payload.DocID)
	}

	if payload.Position != 10 {
		t.Errorf("expected position 10, got %d", payload.Position)
	}
}

func TestClient_Receive_Sync(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	// Simulate sync message with raw JSON
	conn.incoming <- ws.Message{
		Type: ws.MessageTypeSync,
		Payload: map[string]string{
			"docId": "doc1",
		},
	}

	msg, err := client.Receive()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Type != ws.MessageTypeSync {
		t.Errorf("expected sync type, got %s", msg.Type)
	}
}

func TestClient_Receive_ServerMessage(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	// Server-to-client messages keep raw payload
	conn.incoming <- ws.Message{
		Type:    ws.MessageTypeAck,
		Payload: ws.AckPayload{Revision: 5},
	}

	msg, err := client.Receive()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Type != ws.MessageTypeAck {
		t.Errorf("expected ack type, got %s", msg.Type)
	}
}
