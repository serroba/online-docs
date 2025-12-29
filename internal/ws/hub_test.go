package ws_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/serroba/online-docs/internal/ws"
)

// mockConn is a test double for ws.Conn.
type mockConn struct {
	mu       sync.Mutex
	messages []ws.Message
	closed   bool

	// For ReadJSON simulation
	incoming chan ws.Message
}

func newMockConn() *mockConn {
	return &mockConn{
		messages: make([]ws.Message, 0),
		incoming: make(chan ws.Message, 10),
	}
}

func (m *mockConn) WriteJSON(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert to Message
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	var msg ws.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	m.messages = append(m.messages, msg)

	return nil
}

func (m *mockConn) ReadJSON(v any) error {
	msg := <-m.incoming

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true

	return nil
}

func (m *mockConn) Messages() []ws.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]ws.Message, len(m.messages))
	copy(result, m.messages)

	return result
}

func (m *mockConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.closed
}

func TestHub_RegisterUnregister(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()
	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	hub.Register(client)

	if hub.TotalClients() != 1 {
		t.Errorf("expected 1 client, got %d", hub.TotalClients())
	}

	hub.Unregister(client)

	if hub.TotalClients() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.TotalClients())
	}
}

func TestHub_Subscribe(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()
	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	hub.Register(client)
	hub.Subscribe(client, "doc1")

	if hub.ClientCount("doc1") != 1 {
		t.Errorf("expected 1 client on doc1, got %d", hub.ClientCount("doc1"))
	}

	if client.DocID() != "doc1" {
		t.Errorf("expected client docID doc1, got %s", client.DocID())
	}
}

func TestHub_Subscribe_SwitchesDocument(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()
	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	hub.Register(client)
	hub.Subscribe(client, "doc1")
	hub.Subscribe(client, "doc2")

	if hub.ClientCount("doc1") != 0 {
		t.Errorf("expected 0 clients on doc1, got %d", hub.ClientCount("doc1"))
	}

	if hub.ClientCount("doc2") != 1 {
		t.Errorf("expected 1 client on doc2, got %d", hub.ClientCount("doc2"))
	}
}

func TestHub_Unsubscribe(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()
	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	hub.Register(client)
	hub.Subscribe(client, "doc1")
	hub.Unsubscribe(client, "doc1")

	if hub.ClientCount("doc1") != 0 {
		t.Errorf("expected 0 clients on doc1, got %d", hub.ClientCount("doc1"))
	}

	if client.DocID() != "" {
		t.Errorf("expected empty docID, got %s", client.DocID())
	}
}

func TestHub_Unregister_CleansUpSubscription(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()
	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	hub.Register(client)
	hub.Subscribe(client, "doc1")
	hub.Unregister(client)

	if hub.ClientCount("doc1") != 0 {
		t.Errorf("expected 0 clients on doc1 after unregister, got %d", hub.ClientCount("doc1"))
	}
}

func TestHub_Broadcast(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()

	conn1 := newMockConn()
	conn2 := newMockConn()
	conn3 := newMockConn()

	client1 := ws.NewClient("c1", "user1", conn1)
	client2 := ws.NewClient("c2", "user2", conn2)
	client3 := ws.NewClient("c3", "user3", conn3)

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)

	hub.Subscribe(client1, "doc1")
	hub.Subscribe(client2, "doc1")
	hub.Subscribe(client3, "doc2") // Different document

	msg := ws.Message{
		Type:    ws.MessageTypeBroadcast,
		Payload: "test",
	}

	// Broadcast to doc1, excluding client1 (the sender)
	hub.Broadcast("doc1", msg, "c1")

	// Give goroutines time to send
	time.Sleep(10 * time.Millisecond)

	// client1 should NOT receive (excluded)
	if len(conn1.Messages()) != 0 {
		t.Errorf("client1 should not receive broadcast, got %d messages", len(conn1.Messages()))
	}

	// client2 should receive
	if len(conn2.Messages()) != 1 {
		t.Errorf("client2 should receive 1 message, got %d", len(conn2.Messages()))
	}

	// client3 should NOT receive (different document)
	if len(conn3.Messages()) != 0 {
		t.Errorf("client3 should not receive (different doc), got %d messages", len(conn3.Messages()))
	}
}

func TestHub_BroadcastOperation(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	hub.Register(client)
	hub.Subscribe(client, "doc1")

	hub.BroadcastOperation("doc1", 5, 0, 10, "a", "user2", "other")

	time.Sleep(10 * time.Millisecond)

	messages := conn.Messages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Type != ws.MessageTypeBroadcast {
		t.Errorf("expected broadcast type, got %s", messages[0].Type)
	}
}

func TestHub_MultipleDocuments(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()

	conn1 := newMockConn()
	conn2 := newMockConn()

	client1 := ws.NewClient("c1", "user1", conn1)
	client2 := ws.NewClient("c2", "user2", conn2)

	hub.Register(client1)
	hub.Register(client2)

	hub.Subscribe(client1, "doc1")
	hub.Subscribe(client2, "doc2")

	if hub.ClientCount("doc1") != 1 {
		t.Errorf("expected 1 client on doc1, got %d", hub.ClientCount("doc1"))
	}

	if hub.ClientCount("doc2") != 1 {
		t.Errorf("expected 1 client on doc2, got %d", hub.ClientCount("doc2"))
	}

	if hub.TotalClients() != 2 {
		t.Errorf("expected 2 total clients, got %d", hub.TotalClients())
	}
}

func TestHub_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()

	var wg sync.WaitGroup

	// Register many clients concurrently
	for i := range 20 {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			conn := newMockConn()
			client := ws.NewClient(string(rune('a'+n)), "user", conn)

			hub.Register(client)
			hub.Subscribe(client, "doc1")
		}(i)
	}

	wg.Wait()

	if hub.ClientCount("doc1") != 20 {
		t.Errorf("expected 20 clients on doc1, got %d", hub.ClientCount("doc1"))
	}
}
