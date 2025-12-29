package collab

import (
	"sync"

	"github.com/serroba/online-docs/internal/acl"
	"github.com/serroba/online-docs/internal/storage"
	"github.com/serroba/online-docs/internal/ws"
)

// Manager manages multiple document sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session

	// Shared dependencies
	store          storage.Store
	permStore      acl.Store
	hub            *ws.Hub
	snapshotPolicy *storage.SnapshotPolicy
	historySize    int
}

// ManagerConfig holds configuration for creating a manager.
type ManagerConfig struct {
	Store          storage.Store
	PermStore      acl.Store
	Hub            *ws.Hub
	SnapshotPolicy *storage.SnapshotPolicy
	HistorySize    int
}

// NewManager creates a new session manager.
func NewManager(cfg ManagerConfig) *Manager {
	historySize := cfg.HistorySize
	if historySize == 0 {
		historySize = 100
	}

	return &Manager{
		sessions:       make(map[string]*Session),
		store:          cfg.Store,
		permStore:      cfg.PermStore,
		hub:            cfg.Hub,
		snapshotPolicy: cfg.SnapshotPolicy,
		historySize:    historySize,
	}
}

// GetOrCreateSession returns an existing session or creates a new one.
func (m *Manager) GetOrCreateSession(docID string) (*Session, error) {
	// Try read lock first
	m.mu.RLock()
	session, exists := m.sessions[docID]
	m.mu.RUnlock()

	if exists {
		return session, nil
	}

	// Need to create - acquire write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if session, exists = m.sessions[docID]; exists {
		return session, nil
	}

	// Create new session
	var permChecker *acl.Checker
	if m.permStore != nil {
		permChecker = acl.NewChecker(m.permStore)
	}

	session = NewSession(SessionConfig{
		DocID:          docID,
		Store:          m.store,
		PermChecker:    permChecker,
		Hub:            m.hub,
		SnapshotPolicy: m.snapshotPolicy,
		HistorySize:    m.historySize,
	})

	// Load from storage
	if err := session.Load(); err != nil {
		return nil, err
	}

	m.sessions[docID] = session

	return session, nil
}

// GetSession returns an existing session or nil if not found.
func (m *Manager) GetSession(docID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.sessions[docID]
}

// CloseSession closes and removes a session.
func (m *Manager) CloseSession(docID string) error {
	m.mu.Lock()
	session, exists := m.sessions[docID]

	if !exists {
		m.mu.Unlock()

		return nil
	}

	delete(m.sessions, docID)
	m.mu.Unlock()

	return session.Close()
}

// CloseAll closes all sessions.
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	sessions := make([]*Session, 0, len(m.sessions))

	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}

	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	var lastErr error

	for _, s := range sessions {
		if err := s.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// SessionCount returns the number of active sessions.
func (m *Manager) SessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}
