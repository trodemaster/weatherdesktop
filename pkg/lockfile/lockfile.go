package lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// LockFile represents a process lock file
type LockFile struct {
	path string
	pid  int
}

// New creates a new lock file in the system temp directory
func New() *LockFile {
	// Use macOS built-in tmpdir functionality
	tmpDir := os.TempDir() // Respects $TMPDIR on macOS
	lockPath := filepath.Join(tmpDir, "wd.lock")
	
	return &LockFile{
		path: lockPath,
		pid:  os.Getpid(),
	}
}

// TryLock attempts to acquire the lock file
// Returns error if another instance is already running
func (l *LockFile) TryLock() error {
	// Check if lock file exists
	if data, err := os.ReadFile(l.path); err == nil {
		// Lock file exists, check if process is still running
		pidStr := strings.TrimSpace(string(data))
		pid, err := strconv.Atoi(pidStr)
		if err == nil {
			// Check if process with this PID exists
			if isProcessRunning(pid) {
				return fmt.Errorf("another instance is already running (PID: %d)", pid)
			}
			// Stale lock file, remove it
			os.Remove(l.path)
		}
	}
	
	// Create lock file with current PID
	pidStr := fmt.Sprintf("%d\n", l.pid)
	if err := os.WriteFile(l.path, []byte(pidStr), 0644); err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	
	return nil
}

// Unlock removes the lock file
func (l *LockFile) Unlock() error {
	// Only remove if it's our lock file
	if data, err := os.ReadFile(l.path); err == nil {
		pidStr := strings.TrimSpace(string(data))
		pid, err := strconv.Atoi(pidStr)
		if err == nil && pid == l.pid {
			return os.Remove(l.path)
		}
	}
	return nil
}

// isProcessRunning checks if a process with given PID is running
func isProcessRunning(pid int) bool {
	// Use kill(pid, 0) to check if process exists
	// This doesn't actually send a signal, just checks permission
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	
	// Check if error is because process doesn't exist
	if err == syscall.ESRCH {
		return false
	}
	
	// If we get EPERM, process exists but we can't signal it
	if err == syscall.EPERM {
		return true
	}
	
	return false
}

