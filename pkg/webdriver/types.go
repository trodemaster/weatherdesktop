// Copyright 2025 Safari Driver MCP. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package webdriver

import "encoding/json"

// SessionCapabilities represents WebDriver session capabilities
type SessionCapabilities struct {
	BrowserName    string `json:"browserName"`
	BrowserVersion string `json:"browserVersion"`
	PlatformName   string `json:"platformName"`
}

// NewSessionRequest represents a request to create a new WebDriver session
type NewSessionRequest struct {
	Capabilities struct {
		AlwaysMatch map[string]interface{} `json:"alwaysMatch,omitempty"`
	} `json:"capabilities"`
}

// NewSessionResponse represents the response from creating a new WebDriver session
type NewSessionResponse struct {
	SessionID    string               `json:"sessionId"`
	Capabilities *SessionCapabilities `json:"capabilities"`
}

// WebDriverResponse represents a standard WebDriver JSON response
type WebDriverResponse struct {
	Value json.RawMessage `json:"value"`
}

// WebDriverError represents a WebDriver error response
type WebDriverError struct {
	Error        string `json:"error"`
	Message      string `json:"message"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Stacktrace   string `json:"stacktrace,omitempty"`
}

// Element represents a WebDriver element reference
type Element struct {
	ElementID string `json:"element-6066-11e4-a52e-4f735466cecf"`
}

// ElementFindRequest represents a request to find elements
type ElementFindRequest struct {
	Using string `json:"using"` // "css selector", "xpath", "id", "name", etc.
	Value string `json:"value"` // the selector value
}

// NavigateRequest represents a request to navigate to a URL
type NavigateRequest struct {
	URL string `json:"url"`
}

// ExecuteScriptRequest represents a request to execute JavaScript
type ExecuteScriptRequest struct {
	Script string        `json:"script"`
	Args   []interface{} `json:"args"`
}
