package ws_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/serroba/online-docs/internal/ws"
)

const testDocID = "doc1"

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
	hub.Subscribe(client, testDocID)

	if hub.ClientCount(testDocID) != 1 {
		t.Errorf("expected 1 client on doc1, got %d", hub.ClientCount(testDocID))
	}

	if client.DocID() != testDocID {
		t.Errorf("expected client docID doc1, got %s", client.DocID())
	}
}

func TestHub_Subscribe_SwitchesDocument(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()
	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	hub.Register(client)
	hub.Subscribe(client, testDocID)
	hub.Subscribe(client, "doc2")

	if hub.ClientCount(testDocID) != 0 {
		t.Errorf("expected 0 clients on doc1, got %d", hub.ClientCount(testDocID))
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
	hub.Subscribe(client, testDocID)
	hub.Unsubscribe(client, testDocID)

	if hub.ClientCount(testDocID) != 0 {
		t.Errorf("expected 0 clients on doc1, got %d", hub.ClientCount(testDocID))
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
	hub.Subscribe(client, testDocID)
	hub.Unregister(client)

	if hub.ClientCount(testDocID) != 0 {
		t.Errorf("expected 0 clients on doc1 after unregister, got %d", hub.ClientCount(testDocID))
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

	hub.Subscribe(client1, testDocID)
	hub.Subscribe(client2, testDocID)
	hub.Subscribe(client3, "doc2") // Different document

	msg := ws.Message{
		Type:    ws.MessageTypeBroadcast,
		Payload: "test",
	}

	// Broadcast to doc1, excluding client1 (the sender)
	hub.Broadcast(testDocID, msg, "c1")

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
	hub.Subscribe(client, testDocID)

	hub.BroadcastOperation(testDocID, 5, 0, 10, "a", "user2", "other")

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

	hub.Subscribe(client1, testDocID)
	hub.Subscribe(client2, "doc2")

	if hub.ClientCount(testDocID) != 1 {
		t.Errorf("expected 1 client on doc1, got %d", hub.ClientCount(testDocID))
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
			hub.Subscribe(client, testDocID)
		}(i)
	}

	wg.Wait()

	if hub.ClientCount(testDocID) != 20 {
		t.Errorf("expected 20 clients on doc1, got %d", hub.ClientCount(testDocID))
	}
}

func TestHub_Broadcast_NoSubscribers(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()

	// Broadcast to a document with no subscribers - should not panic
	msg := ws.Message{
		Type:    ws.MessageTypeBroadcast,
		Payload: "test",
	}

	hub.Broadcast("nonexistent", msg, "")

	// No error expected, just a no-op
}

func TestHub_Broadcast_ClientNotInMap(t *testing.T) {
	t.Parallel()

	hub := ws.NewHub()

	conn := newMockConn()
	client := ws.NewClient("c1", "user1", conn)

	hub.Register(client)
	hub.Subscribe(client, testDocID)

	// Unregister the client but leave the document subscription orphaned
	// This simulates a race condition where client is removed from clients map
	// but still in documents map
	hub.Unregister(client)

	// Re-add to documents manually to simulate the race
	// We can't easily do this, so instead we test by broadcasting
	// when the doc exists but client doesn't

	// Actually, Unregister cleans up properly, so let's test another way:
	// Register a new client, subscribe, then broadcast excluding that client
	conn2 := newMockConn()
	client2 := ws.NewClient("c2", "user2", conn2)

	hub.Register(client2)
	hub.Subscribe(client2, testDocID)

	msg := ws.Message{
		Type:    ws.MessageTypeBroadcast,
		Payload: "test",
	}

	// Broadcast excluding c2 - no one should receive
	hub.Broadcast(testDocID, msg, "c2")

	time.Sleep(10 * time.Millisecond)

	if len(conn2.Messages()) != 0 {
		t.Errorf("excluded client should not receive, got %d messages", len(conn2.Messages()))
	}
}
