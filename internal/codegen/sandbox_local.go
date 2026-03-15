package codegen

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultSandboxTimeoutSeconds = 20 * 60

type localSandboxState struct {
	rootDir string
	config  SandboxConfig
}

// LocalSandboxManager manages sandbox execution on the local filesystem.
type LocalSandboxManager struct {
	basePath string

	mu       sync.RWMutex
	sandboxes map[string]*localSandboxState
}

// NewLocalSandboxManager constructs a local sandbox manager rooted at basePath.
func NewLocalSandboxManager(basePath string) *LocalSandboxManager {
	cleanBase := strings.TrimSpace(basePath)
	if cleanBase == "" {
		cleanBase = filepath.Join(os.TempDir(), "neuco-sandboxes")
	}

	return &LocalSandboxManager{
		basePath:  cleanBase,
		sandboxes: make(map[string]*localSandboxState),
	}
}

// Provision creates a local sandbox by cloning the configured repository.
func (m *LocalSandboxManager) Provision(ctx context.Context, cfg SandboxConfig) (*Sandbox, error) {
	if strings.TrimSpace(cfg.RepoURL) == "" {
		return nil, errors.New("sandbox provision: repo URL is required")
	}

	if err := os.MkdirAll(m.basePath, 0o755); err != nil {
		return nil, fmt.Errorf("sandbox provision: create base path %q: %w", m.basePath, err)
	}

	sandboxID := uuid.NewString()
	sandboxRoot := filepath.Join(m.basePath, "sandbox-"+sandboxID)
	repoDir := filepath.Join(sandboxRoot, "repo")

	if err := os.MkdirAll(sandboxRoot, 0o755); err != nil {
		return nil, fmt.Errorf("sandbox provision: create sandbox root %q: %w", sandboxRoot, err)
	}

	ref := strings.TrimSpace(cfg.RepoRef)
	if err := m.cloneRepo(ctx, cfg.RepoURL, ref, repoDir); err != nil {
		_ = os.RemoveAll(sandboxRoot)
		return nil, err
	}

	if err := os.MkdirAll(filepath.Join(repoDir, ".neuco"), 0o755); err != nil {
		_ = os.RemoveAll(sandboxRoot)
		return nil, fmt.Errorf("sandbox provision: create .neuco directory: %w", err)
	}

	workDir := repoDir
	if override := strings.TrimSpace(cfg.WorkingDir); override != "" {
		if filepath.IsAbs(override) {
			workDir = filepath.Clean(override)
		} else {
			workDir = filepath.Clean(filepath.Join(repoDir, override))
		}
		if err := os.MkdirAll(workDir, 0o755); err != nil {
			_ = os.RemoveAll(sandboxRoot)
			return nil, fmt.Errorf("sandbox provision: create working directory %q: %w", workDir, err)
		}
	}

	now := time.Now().UTC()
	timeoutSeconds := cfg.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultSandboxTimeoutSeconds
	}

	sandbox := &Sandbox{
		ID:        sandboxID,
		Provider:  "local",
		WorkDir:   workDir,
		Status:    "ready",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(timeoutSeconds) * time.Second),
		Metadata: map[string]string{
			"sandbox_root": sandboxRoot,
			"repo_ref":     ref,
		},
	}

	m.mu.Lock()
	m.sandboxes[sandboxID] = &localSandboxState{
		rootDir: sandboxRoot,
		config:  cfg,
	}
	m.mu.Unlock()

	return sandbox, nil
}

// WriteFiles writes the provided files into the sandbox working directory.
func (m *LocalSandboxManager) WriteFiles(ctx context.Context, sb *Sandbox, files map[string]string) error {
	if sb == nil {
		return errors.New("sandbox write files: sandbox is required")
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("sandbox write files: context cancelled: %w", err)
	}

	for relPath, content := range files {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("sandbox write files: context cancelled: %w", err)
		}

		absPath, err := resolveSandboxPath(sb.WorkDir, relPath)
		if err != nil {
			return fmt.Errorf("sandbox write files: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return fmt.Errorf("sandbox write files: create directory for %q: %w", relPath, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("sandbox write files: write %q: %w", relPath, err)
		}
	}

	return nil
}

// Execute runs a command in the sandbox working directory and captures output.
func (m *LocalSandboxManager) Execute(ctx context.Context, sb *Sandbox, cmd string, args ...string) (*ExecResult, error) {
	if sb == nil {
		return nil, errors.New("sandbox execute: sandbox is required")
	}
	if strings.TrimSpace(cmd) == "" {
		return nil, errors.New("sandbox execute: command is required")
	}

	execCtx, cancel := m.withSandboxTimeout(ctx, sb)
	defer cancel()

	command := exec.CommandContext(execCtx, cmd, args...)
	command.Dir = sb.WorkDir
	command.Env = m.buildEnv(sb.ID)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	started := time.Now()
	err := command.Run()
	duration := time.Since(started)

	result := &ExecResult{
		Command:  strings.TrimSpace(strings.Join(append([]string{cmd}, args...), " ")),
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	if execCtx.Err() != nil {
		result.ExitCode = -1
		return result, fmt.Errorf("sandbox execute: command interrupted: %w", execCtx.Err())
	}

	result.ExitCode = -1
	return result, fmt.Errorf("sandbox execute: run %q: %w", result.Command, err)
}

// StreamOutput runs a command and streams stdout/stderr lines as LogEntry values.
func (m *LocalSandboxManager) StreamOutput(ctx context.Context, sb *Sandbox, cmd string, args ...string) (<-chan LogEntry, error) {
	if sb == nil {
		return nil, errors.New("sandbox stream output: sandbox is required")
	}
	if strings.TrimSpace(cmd) == "" {
		return nil, errors.New("sandbox stream output: command is required")
	}

	execCtx, cancel := m.withSandboxTimeout(ctx, sb)
	command := exec.CommandContext(execCtx, cmd, args...)
	command.Dir = sb.WorkDir
	command.Env = m.buildEnv(sb.ID)

	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("sandbox stream output: stdout pipe: %w", err)
	}
	stderrPipe, err := command.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("sandbox stream output: stderr pipe: %w", err)
	}

	if err := command.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("sandbox stream output: start command: %w", err)
	}

	out := make(chan LogEntry, 128)

	go func() {
		defer close(out)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(2)

		go m.streamReader(execCtx, &wg, out, "stdout", stdoutPipe)
		go m.streamReader(execCtx, &wg, out, "stderr", stderrPipe)

		wg.Wait()

		err := command.Wait()
		now := time.Now().UTC()
		if err == nil {
			m.sendLog(execCtx, out, LogEntry{Source: "system", Message: "command completed successfully", Timestamp: now})
			return
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			m.sendLog(execCtx, out, LogEntry{
				Source:    "system",
				Message:   fmt.Sprintf("command exited with code %d", exitErr.ExitCode()),
				Timestamp: now,
			})
			return
		}

		if execCtx.Err() != nil {
			m.sendLog(context.Background(), out, LogEntry{
				Source:    "system",
				Message:   fmt.Sprintf("command interrupted: %v", execCtx.Err()),
				Timestamp: now,
			})
			return
		}

		m.sendLog(execCtx, out, LogEntry{
			Source:    "system",
			Message:   fmt.Sprintf("command failed: %v", err),
			Timestamp: now,
		})
	}()

	return out, nil
}

// CollectDiff stages all changes, then collects per-file metadata and patch content.
func (m *LocalSandboxManager) CollectDiff(ctx context.Context, sb *Sandbox) ([]FileChange, error) {
	if sb == nil {
		return nil, errors.New("sandbox collect diff: sandbox is required")
	}

	if _, err := m.gitExec(ctx, sb.WorkDir, "add", "-A"); err != nil {
		return nil, fmt.Errorf("sandbox collect diff: stage changes: %w", err)
	}

	nameStatusRaw, err := m.gitExec(ctx, sb.WorkDir, "diff", "--cached", "--name-status")
	if err != nil {
		return nil, fmt.Errorf("sandbox collect diff: list changed files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(nameStatusRaw), "\n")
	if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
		return nil, nil
	}

	changes := make([]FileChange, 0, len(lines))
	for _, line := range lines {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("sandbox collect diff: context cancelled: %w", err)
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		change, err := parseNameStatusLine(line)
		if err != nil {
			return nil, fmt.Errorf("sandbox collect diff: parse line %q: %w", line, err)
		}

		diffRaw, err := m.gitExec(ctx, sb.WorkDir, "diff", "--cached", "--", change.Path)
		if err != nil {
			return nil, fmt.Errorf("sandbox collect diff: diff for %q: %w", change.Path, err)
		}
		change.Diff = diffRaw

		if change.ChangeType != "deleted" {
			contentPath := filepath.Join(sb.WorkDir, change.Path)
			if data, readErr := os.ReadFile(contentPath); readErr == nil {
				change.Content = string(data)
			}
		}

		changes = append(changes, change)
	}

	return changes, nil
}

// Destroy removes a sandbox and all related local resources.
func (m *LocalSandboxManager) Destroy(_ context.Context, sandboxID string) error {
	trimmed := strings.TrimSpace(sandboxID)
	if trimmed == "" {
		return errors.New("sandbox destroy: sandbox ID is required")
	}

	m.mu.Lock()
	state := m.sandboxes[trimmed]
	delete(m.sandboxes, trimmed)
	m.mu.Unlock()

	target := ""
	if state != nil && strings.TrimSpace(state.rootDir) != "" {
		target = state.rootDir
	} else {
		target = filepath.Join(m.basePath, "sandbox-"+strings.TrimPrefix(trimmed, "sandbox-"))
	}

	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("sandbox destroy: remove %q: %w", target, err)
	}

	return nil
}

func (m *LocalSandboxManager) cloneRepo(ctx context.Context, repoURL, repoRef, destination string) error {
	branch := strings.TrimSpace(repoRef)
	if branch != "" {
		if _, err := m.runCommand(ctx, "", "git", "clone", "--depth=1", "--branch", branch, repoURL, destination); err == nil {
			return nil
		} else {
			if _, fallbackErr := m.runCommand(ctx, "", "git", "clone", "--depth=1", repoURL, destination); fallbackErr == nil {
				return nil
			}
			return fmt.Errorf("sandbox provision: git clone with ref %q failed: %w", branch, err)
		}
	}

	if _, err := m.runCommand(ctx, "", "git", "clone", "--depth=1", repoURL, destination); err != nil {
		return fmt.Errorf("sandbox provision: git clone failed: %w", err)
	}

	return nil
}

func (m *LocalSandboxManager) withSandboxTimeout(ctx context.Context, sb *Sandbox) (context.Context, context.CancelFunc) {
	if sb == nil || sb.ExpiresAt.IsZero() {
		return context.WithCancel(ctx)
	}

	remaining := time.Until(sb.ExpiresAt)
	if remaining <= 0 {
		return context.WithTimeout(ctx, time.Second)
	}
	return context.WithTimeout(ctx, remaining)
}

func (m *LocalSandboxManager) buildEnv(sandboxID string) []string {
	env := os.Environ()

	m.mu.RLock()
	state := m.sandboxes[sandboxID]
	m.mu.RUnlock()
	if state == nil || len(state.config.Environment) == 0 {
		return env
	}

	for key, value := range state.config.Environment {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		env = append(env, trimmedKey+"="+value)
	}

	return env
}

func (m *LocalSandboxManager) streamReader(ctx context.Context, wg *sync.WaitGroup, out chan<- LogEntry, source string, r io.Reader) {
	defer wg.Done()

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		entry := LogEntry{
			Source:    source,
			Message:   scanner.Text(),
			Timestamp: time.Now().UTC(),
		}

		if !m.sendLog(ctx, out, entry) {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		_ = m.sendLog(ctx, out, LogEntry{
			Source:    "system",
			Message:   fmt.Sprintf("%s stream read error: %v", source, err),
			Timestamp: time.Now().UTC(),
		})
	}
}

func (m *LocalSandboxManager) sendLog(ctx context.Context, out chan<- LogEntry, entry LogEntry) bool {
	select {
	case out <- entry:
		return true
	case <-ctx.Done():
		return false
	}
}

func (m *LocalSandboxManager) gitExec(ctx context.Context, workDir string, args ...string) (string, error) {
	return m.runCommand(ctx, workDir, "git", append([]string{"-C", workDir}, args...)...)
}

func (m *LocalSandboxManager) runCommand(ctx context.Context, workDir string, cmd string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, cmd, args...)
	if strings.TrimSpace(workDir) != "" {
		command.Dir = workDir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		combined := strings.TrimSpace(stderr.String())
		if combined == "" {
			combined = strings.TrimSpace(stdout.String())
		}
		if combined != "" {
			return "", fmt.Errorf("run %s %s: %w: %s", cmd, strings.Join(args, " "), err, combined)
		}
		return "", fmt.Errorf("run %s %s: %w", cmd, strings.Join(args, " "), err)
	}

	return stdout.String(), nil
}

func resolveSandboxPath(workDir, relPath string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(relPath))
	if clean == "." || clean == "" {
		return "", errors.New("invalid file path")
	}
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("path %q escapes sandbox working directory", relPath)
	}

	root := filepath.Clean(workDir)
	target := filepath.Join(root, clean)
	if !strings.HasPrefix(target, root+string(os.PathSeparator)) && target != root {
		return "", fmt.Errorf("path %q escapes sandbox working directory", relPath)
	}

	return target, nil
}

func parseNameStatusLine(line string) (FileChange, error) {
	parts := strings.Split(line, "\t")
	if len(parts) < 2 {
		return FileChange{}, errors.New("invalid git name-status line")
	}

	status := strings.TrimSpace(parts[0])
	path := strings.TrimSpace(parts[1])
	if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
		if len(parts) < 3 {
			return FileChange{}, errors.New("rename/copy line missing destination path")
		}
		path = strings.TrimSpace(parts[2])
	}

	changeType := "modified"
	switch {
	case strings.HasPrefix(status, "A"):
		changeType = "added"
	case strings.HasPrefix(status, "M"):
		changeType = "modified"
	case strings.HasPrefix(status, "D"):
		changeType = "deleted"
	case strings.HasPrefix(status, "R"):
		changeType = "renamed"
	case strings.HasPrefix(status, "C"):
		changeType = "copied"
	}

	if path == "" {
		return FileChange{}, errors.New("empty path")
	}

	return FileChange{Path: path, ChangeType: changeType}, nil
}
