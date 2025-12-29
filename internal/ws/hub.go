package ws

import (
	"sync"
)

// Hub manages WebSocket clients and broadcasts operations.
type Hub struct {
	mu sync.RWMutex

	// clients maps client ID to client
	clients map[string]*Client

	// documents maps document ID to set of client IDs
	documents map[string]map[string]struct{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:   make(map[string]*Client),
		documents: make(map[string]map[string]struct{}),
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
}

// Unregister removes a client from the hub and any document subscriptions.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove from document subscription
	docID := client.DocID()
	if docID != "" {
		if clients, ok := h.documents[docID]; ok {
			delete(clients, client.ID)

			if len(clients) == 0 {
				delete(h.documents, docID)
			}
		}
	}

	delete(h.clients, client.ID)
}

// Subscribe adds a client to a document's broadcast list.
func (h *Hub) Subscribe(client *Client, docID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Unsubscribe from previous document
	oldDocID := client.DocID()
	if oldDocID != "" && oldDocID != docID {
		if clients, ok := h.documents[oldDocID]; ok {
			delete(clients, client.ID)

			if len(clients) == 0 {
				delete(h.documents, oldDocID)
			}
		}
	}

	// Subscribe to new document
	if h.documents[docID] == nil {
		h.documents[docID] = make(map[string]struct{})
	}

	h.documents[docID][client.ID] = struct{}{}
	client.SetDocID(docID)
}

// Unsubscribe removes a client from a document's broadcast list.
func (h *Hub) Unsubscribe(client *Client, docID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.documents[docID]; ok {
		delete(clients, client.ID)

		if len(clients) == 0 {
			delete(h.documents, docID)
		}
	}

	if client.DocID() == docID {
		client.SetDocID("")
	}
}

// Broadcast sends a message to all clients subscribed to a document,
// except the sender (identified by excludeClientID).
func (h *Hub) Broadcast(docID string, msg Message, excludeClientID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientIDs, ok := h.documents[docID]
	if !ok {
		return
	}

	for clientID := range clientIDs {
		if clientID == excludeClientID {
			continue
		}

		client, ok := h.clients[clientID]
		if !ok {
			continue
		}

		// Send in goroutine to avoid blocking on slow clients
		go func(c *Client) {
			_ = c.Send(msg)
		}(client)
	}
}

// BroadcastOperation is a convenience method for broadcasting an operation.
func (h *Hub) BroadcastOperation(docID string, revision, opType, position int, char, userID, excludeClientID string) {
	msg := Message{
		Type: MessageTypeBroadcast,
		Payload: BroadcastPayload{
			DocID:    docID,
			Revision: revision,
			OpType:   opType,
			Position: position,
			Char:     char,
			UserID:   userID,
		},
	}

	h.Broadcast(docID, msg, excludeClientID)
}

// ClientCount returns the number of clients subscribed to a document.
func (h *Hub) ClientCount(docID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.documents[docID]; ok {
		return len(clients)
	}

	return 0
}

// TotalClients returns the total number of connected clients.
func (h *Hub) TotalClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}
