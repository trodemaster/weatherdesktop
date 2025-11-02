package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client handles Docker Compose operations
type Client struct {
	projectDir string
}

// New creates a new Docker Compose client
func New(projectDir string) *Client {
	return &Client{
		projectDir: projectDir,
	}
}

// EnsureRunning ensures the wd-worker container is running and healthy
func (c *Client) EnsureRunning() error {
	// Check if container exists and is running
	cmd := exec.Command("docker", "compose", "ps", "--format", "json", "wd-worker")
	cmd.Dir = c.projectDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Container doesn't exist or compose file issue
		return c.start()
	}
	
	// Parse JSON output
	var services []map[string]interface{}
	if err := json.Unmarshal(output, &services); err != nil {
		// Try single object format
		var service map[string]interface{}
		if err := json.Unmarshal(output, &service); err != nil {
			return c.start()
		}
		services = []map[string]interface{}{service}
	}
	
	// Check if running and healthy
	for _, svc := range services {
		state, ok := svc["State"].(string)
		if !ok {
			continue
		}
		
		health, _ := svc["Health"].(string)
		
		if state == "running" && (health == "" || health == "healthy") {
			// Container is running and healthy
			return nil
		}
	}
	
	// Need to start
	return c.start()
}

// start starts the container and waits for health check
func (c *Client) start() error {
	fmt.Println("Starting wd-worker container...")
	
	cmd := exec.Command("docker", "compose", "up", "-d", "--wait", "wd-worker")
	cmd.Dir = c.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	
	fmt.Println("Container started successfully")
	return nil
}

// Exec executes a command inside the wd-worker container
func (c *Client) Exec(args ...string) error {
	// Build command: docker compose exec -T wd-worker <args>
	cmdArgs := []string{"compose", "exec", "-T", "wd-worker"}
	cmdArgs = append(cmdArgs, args...)
	
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = c.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	return cmd.Run()
}

// ExecQuiet executes a command and captures output
func (c *Client) ExecQuiet(args ...string) (string, error) {
	cmdArgs := []string{"compose", "exec", "-T", "wd-worker"}
	cmdArgs = append(cmdArgs, args...)
	
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = c.projectDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	
	return strings.TrimSpace(stdout.String()), nil
}

// Stop stops the wd-worker container
func (c *Client) Stop() error {
	cmd := exec.Command("docker", "compose", "stop", "wd-worker")
	cmd.Dir = c.projectDir
	return cmd.Run()
}

// Restart restarts the wd-worker container
func (c *Client) Restart() error {
	cmd := exec.Command("docker", "compose", "restart", "wd-worker")
	cmd.Dir = c.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Logs shows container logs
func (c *Client) Logs(follow bool) error {
	args := []string{"compose", "logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, "wd-worker")
	
	cmd := exec.Command("docker", args...)
	cmd.Dir = c.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

