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
