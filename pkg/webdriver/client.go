// Copyright 2025 Safari Driver MCP. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a WebDriver HTTP client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new WebDriver client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateSession creates a new WebDriver session
func (c *Client) CreateSession(capabilities map[string]interface{}) (*NewSessionResponse, error) {
	req := NewSessionRequest{}
	req.Capabilities.AlwaysMatch = capabilities

	respBody, err := c.post("/session", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	var response struct {
		Value NewSessionResponse `json:"value"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse session response: %w", err)
	}

	return &response.Value, nil
}

// DeleteSession deletes a WebDriver session
func (c *Client) DeleteSession(sessionID string) error {
	_, err := c.delete(fmt.Sprintf("/session/%s", sessionID))
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// NavigateTo navigates to the specified URL
func (c *Client) NavigateTo(sessionID, url string) error {
	req := NavigateRequest{URL: url}
	_, err := c.post(fmt.Sprintf("/session/%s/url", sessionID), req)
	if err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}
	return nil
}

// GetCurrentURL returns the current page URL
func (c *Client) GetCurrentURL(sessionID string) (string, error) {
	respBody, err := c.get(fmt.Sprintf("/session/%s/url", sessionID))
	if err != nil {
		return "", fmt.Errorf("failed to get current URL: %w", err)
	}

	var response WebDriverResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse URL response: %w", err)
	}

	var url string
	if err := json.Unmarshal(response.Value, &url); err != nil {
		return "", fmt.Errorf("failed to parse URL value: %w", err)
	}

	return url, nil
}

// GetPageSource returns the current page's HTML source
func (c *Client) GetPageSource(sessionID string) (string, error) {
	respBody, err := c.get(fmt.Sprintf("/session/%s/source", sessionID))
	if err != nil {
		return "", fmt.Errorf("failed to get page source: %w", err)
	}

	var response WebDriverResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse source response: %w", err)
	}

	var source string
	if err := json.Unmarshal(response.Value, &source); err != nil {
		return "", fmt.Errorf("failed to parse source value: %w", err)
	}

	return source, nil
}

// GetTitle returns the current page's title
func (c *Client) GetTitle(sessionID string) (string, error) {
	respBody, err := c.get(fmt.Sprintf("/session/%s/title", sessionID))
	if err != nil {
		return "", fmt.Errorf("failed to get page title: %w", err)
	}

	var response WebDriverResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse title response: %w", err)
	}

	var title string
	if err := json.Unmarshal(response.Value, &title); err != nil {
		return "", fmt.Errorf("failed to parse title value: %w", err)
	}

	return title, nil
}

// GetScreenshot takes a screenshot and returns base64 encoded PNG data
func (c *Client) GetScreenshot(sessionID string) (string, error) {
	respBody, err := c.get(fmt.Sprintf("/session/%s/screenshot", sessionID))
	if err != nil {
		return "", fmt.Errorf("failed to get screenshot: %w", err)
	}

	var response WebDriverResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse screenshot response: %w", err)
	}

	var screenshot string
	if err := json.Unmarshal(response.Value, &screenshot); err != nil {
		return "", fmt.Errorf("failed to parse screenshot value: %w", err)
	}

	return screenshot, nil
}

// FindElement finds an element using CSS selector
// Per W3C WebDriver spec: POST /session/{sessionId}/element
func (c *Client) FindElement(sessionID, selector string) (string, error) {
	req := map[string]interface{}{
		"using": "css selector",
		"value": selector,
	}
	
	respBody, err := c.post(fmt.Sprintf("/session/%s/element", sessionID), req)
	if err != nil {
		return "", fmt.Errorf("failed to find element: %w", err)
	}
	
	var response struct {
		Value map[string]string `json:"value"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse element response: %w", err)
	}
	
	// WebDriver returns element ID in different formats, try common ones
	if elementID, ok := response.Value["element-6066-11e4-a52e-4f735466cecf"]; ok {
		return elementID, nil
	}
	if elementID, ok := response.Value["ELEMENT"]; ok {
		return elementID, nil
	}
	
	return "", fmt.Errorf("element ID not found in response")
}

// GetElementScreenshot takes a screenshot of a specific element
// Per W3C WebDriver spec: GET /session/{sessionId}/element/{elementId}/screenshot
func (c *Client) GetElementScreenshot(sessionID, elementID string) (string, error) {
	respBody, err := c.get(fmt.Sprintf("/session/%s/element/%s/screenshot", sessionID, elementID))
	if err != nil {
		return "", fmt.Errorf("failed to get element screenshot: %w", err)
	}
	
	var response WebDriverResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse element screenshot response: %w", err)
	}
	
	var screenshot string
	if err := json.Unmarshal(response.Value, &screenshot); err != nil {
		return "", fmt.Errorf("failed to parse element screenshot value: %w", err)
	}
	
	return screenshot, nil
}

// ExecuteScript executes JavaScript synchronously and returns the result
// Per W3C WebDriver spec: POST /session/{sessionId}/execute/sync
func (c *Client) ExecuteScript(sessionID, script string, args []interface{}) (json.RawMessage, error) {
	req := map[string]interface{}{
		"script": script,
		"args":   args,
	}
	
	respBody, err := c.post(fmt.Sprintf("/session/%s/execute/sync", sessionID), req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	var response WebDriverResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse script response: %w", err)
	}

	return response.Value, nil
}

// SetWindowRect sets the window position and size
// Per W3C WebDriver spec: POST /session/{sessionId}/window/rect
func (c *Client) SetWindowRect(sessionID string, x, y, width, height int) error {
	req := map[string]interface{}{
		"x":      x,
		"y":      y,
		"width":  width,
		"height": height,
	}
	
	_, err := c.post(fmt.Sprintf("/session/%s/window/rect", sessionID), req)
	if err != nil {
		return fmt.Errorf("failed to set window rect: %w", err)
	}
	
	return nil
}

// MaximizeWindow maximizes the window
// Per W3C WebDriver spec: POST /session/{sessionId}/window/maximize
func (c *Client) MaximizeWindow(sessionID string) error {
	_, err := c.post(fmt.Sprintf("/session/%s/window/maximize", sessionID), nil)
	if err != nil {
		return fmt.Errorf("failed to maximize window: %w", err)
	}
	
	return nil
}

// MinimizeWindow minimizes the window
// Per W3C WebDriver spec: POST /session/{sessionId}/window/minimize
func (c *Client) MinimizeWindow(sessionID string) error {
	_, err := c.post(fmt.Sprintf("/session/%s/window/minimize", sessionID), nil)
	if err != nil {
		return fmt.Errorf("failed to minimize window: %w", err)
	}
	
	return nil
}

// Get performs a public GET request (for direct use by other packages)
func (c *Client) Get(path string) ([]byte, error) {
	return c.get(path)
}

// Helper methods for HTTP operations
func (c *Client) post(path string, data interface{}) ([]byte, error) {
	return c.request("POST", path, data)
}

func (c *Client) get(path string) ([]byte, error) {
	return c.request("GET", path, nil)
}

func (c *Client) delete(path string) ([]byte, error) {
	return c.request("DELETE", path, nil)
}

func (c *Client) request(method, path string, data interface{}) ([]byte, error) {
	url := c.baseURL + path

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for WebDriver errors
	if resp.StatusCode >= 400 {
		var wdError WebDriverError
		if json.Unmarshal(respBody, &wdError) == nil && wdError.Error != "" {
			return nil, fmt.Errorf("WebDriver error: %s - %s", wdError.Error, wdError.Message)
		}
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
