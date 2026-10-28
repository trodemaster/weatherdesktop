package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
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

// waitForDaemon polls the Docker daemon until it responds or the timeout expires.
// Each probe uses a 5s context so a hung daemon doesn't stall the loop.
func (c *Client) waitForDaemon(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 10 * time.Second

	fmt.Printf("Waiting for Docker daemon (timeout: %s)...\n", timeout)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := exec.CommandContext(ctx, "docker", "info").Run()
		cancel()

		if err == nil {
			fmt.Println("Docker daemon ready")
			return nil
		}

		remaining := time.Until(deadline)
		if remaining <= interval {
			return fmt.Errorf("Docker daemon not available after %s", timeout)
		}

		elapsed := timeout - remaining
		fmt.Printf("Docker not ready, retrying in %s... (%s elapsed)\n", interval, elapsed.Round(time.Second))
		time.Sleep(interval)
	}
}

// EnsureRunning ensures the wd-worker container is running and healthy
func (c *Client) EnsureRunning() error {
	if err := c.waitForDaemon(4 * time.Minute); err != nil {
		return err
	}

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

