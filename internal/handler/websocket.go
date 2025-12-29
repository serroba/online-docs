package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/serroba/online-docs/internal/acl"
	"github.com/serroba/online-docs/internal/ot"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/serroba/online-docs/internal/ws"
)

// handleWebSocket handles GET /ws?docId={id}.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	docID := r.URL.Query().Get("docId")
	if docID == "" {
		http.Error(w, "docId query parameter is required", http.StatusBadRequest)

		return
	}

	userID := UserIDFromContext(r.Context())

	client, cleanup, err := s.setupWebSocketClient(w, r, docID, userID)
	if err != nil {
		return
	}

	defer cleanup()

	session, err := s.initializeSession(client, docID, userID)
	if err != nil {
		return
	}

	s.handleMessages(client, session, docID, userID)
}

// setupWebSocketClient upgrades the connection and creates a client.
func (s *Server) setupWebSocketClient(
	w http.ResponseWriter, r *http.Request, docID, userID string,
) (*ws.Client, func(), error) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)

		return nil, nil, err
	}

	clientID := uuid.New().String()
	client := ws.NewClient(clientID, userID, conn)
	s.hub.Register(client)
	s.hub.Subscribe(client, docID)

	cleanup := func() {
		s.hub.Unregister(client)
		_ = client.Close()
	}

	return client, cleanup, nil
}

// initializeSession gets or creates a session and sends initial state.
func (s *Server) initializeSession(client *ws.Client, docID, userID string) (sessionInterface, error) {
	session, err := s.manager.GetOrCreateSession(docID)
	if err != nil {
		if errors.Is(err, storage.ErrDocumentNotFound) {
			_ = client.SendError(ws.ErrorCodeInvalidMessage, "document not found")
		} else {
			_ = client.SendError(ws.ErrorCodeInternalError, "failed to load document")
		}

		return nil, err
	}

	content, revision, err := session.GetState(userID)
	if err != nil {
		if errors.Is(err, acl.ErrAccessDenied) {
			_ = client.SendError(ws.ErrorCodeAccessDenied, "access denied")
		} else {
			_ = client.SendError(ws.ErrorCodeInternalError, "failed to get document state")
		}

		return nil, err
	}

	if err := client.Send(ws.Message{
		Type: ws.MessageTypeState,
		Payload: ws.StatePayload{
			DocID:    docID,
			Content:  content,
			Revision: revision,
		},
	}); err != nil {
		return nil, err
	}

	return session, nil
}

// handleMessages processes incoming messages from a client.
func (s *Server) handleMessages(client *ws.Client, session sessionInterface, docID, userID string) {
	for {
		msg, err := client.Receive()
		if err != nil {
			return
		}

		switch msg.Type {
		case ws.MessageTypeOperation:
			s.handleOperation(client, session, userID, msg)
		case ws.MessageTypeSync:
			s.handleSync(client, session, docID, userID)
		case ws.MessageTypeAck, ws.MessageTypeBroadcast, ws.MessageTypeState, ws.MessageTypeError:
			// Server-to-client messages - ignore if received from client
			_ = client.SendError(ws.ErrorCodeInvalidMessage, "unexpected message type")
		}
	}
}

// handleOperation processes an operation message.
func (s *Server) handleOperation(client *ws.Client, session sessionInterface, userID string, msg ws.Message) {
	payload, ok := msg.Payload.(ws.OperationPayload)
	if !ok {
		_ = client.SendError(ws.ErrorCodeInvalidMessage, "invalid operation payload")

		return
	}

	var op ot.Operation

	switch payload.OpType {
	case int(ot.Insert):
		op = ot.NewInsert(payload.Char, payload.Position, userID)
	case int(ot.Delete):
		op = ot.NewDelete(payload.Position, userID)
	default:
		_ = client.SendError(ws.ErrorCodeInvalidMessage, "invalid operation type")

		return
	}

	revision, err := session.ApplyOperation(client.ID, userID, op, payload.BaseRevision)
	if err != nil {
		if errors.Is(err, acl.ErrAccessDenied) {
			_ = client.SendError(ws.ErrorCodeAccessDenied, "write access denied")
		} else {
			_ = client.SendError(ws.ErrorCodeInternalError, err.Error())
		}

		return
	}

	_ = client.Send(ws.Message{
		Type: ws.MessageTypeAck,
		Payload: ws.AckPayload{
			Revision: revision,
		},
	})
}

// handleSync sends the current document state to the client.
func (s *Server) handleSync(client *ws.Client, session sessionInterface, docID, userID string) {
	content, revision, err := session.GetState(userID)
	if err != nil {
		if errors.Is(err, acl.ErrAccessDenied) {
			_ = client.SendError(ws.ErrorCodeAccessDenied, "access denied")
		} else {
			_ = client.SendError(ws.ErrorCodeInternalError, "failed to get document state")
		}

		return
	}

	_ = client.Send(ws.Message{
		Type: ws.MessageTypeState,
		Payload: ws.StatePayload{
			DocID:    docID,
			Content:  content,
			Revision: revision,
		},
	})
}

// sessionInterface allows mocking the session for testing.
type sessionInterface interface {
	ApplyOperation(clientID, userID string, op ot.Operation, baseRevision int) (int, error)
	GetState(userID string) (string, int, error)
}
