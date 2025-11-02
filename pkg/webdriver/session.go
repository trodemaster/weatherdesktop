// Copyright 2025 Safari Driver MCP. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"fmt"
	"sync"
)

// SessionManager manages WebDriver sessions
type SessionManager struct {
	client   *Client
	sessions map[string]*Session
	mutex    sync.RWMutex
}

// Session represents an active WebDriver session
type Session struct {
	ID           string
	Capabilities *SessionCapabilities
	client       *Client
	debugMode    bool
}

// NewSessionManager creates a new session manager
func NewSessionManager(client *Client) *SessionManager {
	return &SessionManager{
		client:   client,
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new WebDriver session with Safari
func (sm *SessionManager) CreateSession(debugMode bool) (*Session, error) {
	capabilities := map[string]interface{}{
		"browserName": "safari",
		// Disable automatic inspection - we want clean viewport for screenshots
		// User can manually open inspector in debug mode if needed (Develop menu)
		"safari:automaticInspection": false,
		"safari:automaticProfiling":  false,
	}

	resp, err := sm.client.CreateSession(capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebDriver session: %w", err)
	}

	session := &Session{
		ID:           resp.SessionID,
		Capabilities: resp.Capabilities,
		client:       sm.client,
		debugMode:    debugMode,
	}

	sm.mutex.Lock()
	sm.sessions[session.ID] = session
	sm.mutex.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mutex.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

// DeleteSession deletes a WebDriver session
func (sm *SessionManager) DeleteSession(sessionID string) error {
	sm.mutex.Lock()
	_, exists := sm.sessions[sessionID]
	if exists {
		delete(sm.sessions, sessionID)
	}
	sm.mutex.Unlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	return sm.client.DeleteSession(sessionID)
}

// ListSessions returns all active session IDs
func (sm *SessionManager) ListSessions() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	sessionIDs := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	return sessionIDs
}

// Session methods

// NavigateTo navigates the session to the specified URL
func (s *Session) NavigateTo(url string) error {
	return s.client.NavigateTo(s.ID, url)
}

// GetCurrentURL returns the current page URL
func (s *Session) GetCurrentURL() (string, error) {
	return s.client.GetCurrentURL(s.ID)
}

// GetPageSource returns the current page's HTML source
func (s *Session) GetPageSource() (string, error) {
	return s.client.GetPageSource(s.ID)
}

// GetTitle returns the current page's title
func (s *Session) GetTitle() (string, error) {
	return s.client.GetTitle(s.ID)
}
