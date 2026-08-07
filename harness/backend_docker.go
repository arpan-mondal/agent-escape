package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

func init() {
	registerBackend(dockerBackend{})
}

// dockerBackend builds sandboxes backed by Docker (default runc runtime).
type dockerBackend struct{}

func (dockerBackend) Name() string { return "docker" }

func (dockerBackend) New(cfg SandboxConfig) (Sandbox, error) {
	return newDockerSandbox(cfg), nil
}

// dockerSandbox drives a single container via the `docker` CLI. The gVisor
// backend reuses this type with cfg.Runtime = "runsc".
type dockerSandbox struct {
	cfg SandboxConfig
}

func newDockerSandbox(cfg SandboxConfig) *dockerSandbox {
	if cfg.StraceLog == "" {
		cfg.StraceLog = "/tmp/agentescape-strace.log"
	}
	return &dockerSandbox{cfg: cfg}
}

func (s *dockerSandbox) Name() string { return s.cfg.Name }

// Start creates and launches the container, then plants any canaries that carry
// content. The container runs `sleep infinity` so we can exec into it repeatedly.
func (s *dockerSandbox) Start(ctx context.Context) error {
	args := []string{"run", "-d", "--name", s.cfg.Name, "--cap-add=SYS_PTRACE"}
	if s.cfg.Runtime != "" {
		args = append(args, "--runtime="+s.cfg.Runtime)
	}
	if s.cfg.Workspace != "" {
		args = append(args, "-v", s.cfg.Workspace+":/workspace")
	}
	args = append(args, s.cfg.Image, "sleep", "infinity")

	_, stderr, code, err := runCmd(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("docker run: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("docker run failed (exit %d): %s", code, stderr)
	}

	for _, c := range s.cfg.Canaries {
		if c.Content == "" {
			continue // pre-existing paths (e.g. /etc/passwd) are not planted
		}
		if err := s.WriteFile(ctx, c.Path, []byte(c.Content)); err != nil {
			return fmt.Errorf("plant canary %s: %w", c.Path, err)
		}
	}
	return nil
}

// Exec runs a command inside the container, optionally wrapped in strace so the
// capture layer can read the syscall log afterwards.
func (s *dockerSandbox) Exec(ctx context.Context, cmd []string) (ExecResult, error) {
	full := []string{"exec", s.cfg.Name}
	if s.cfg.Strace {
		full = append(full, "strace", "-f", "-qq", "-e", "trace="+straceSyscalls, "-o", s.cfg.StraceLog)
	}
	full = append(full, cmd...)

	stdout, stderr, code, err := runCmd(ctx, "docker", full...)
	if err != nil {
		return ExecResult{}, fmt.Errorf("docker exec: %w", err)
	}
	return ExecResult{
		Success:    code == 0,
		ReturnCode: code,
		Stdout:     string(stdout),
		Stderr:     stderr,
	}, nil
}

// ReadFile reads a file from inside the container via `docker exec cat`.
func (s *dockerSandbox) ReadFile(ctx context.Context, path string) ([]byte, error) {
	stdout, stderr, code, err := runCmd(ctx, "docker", "exec", s.cfg.Name, "cat", path)
	if err != nil {
		return nil, fmt.Errorf("docker exec cat: %w", err)
	}
	if code != 0 {
		return nil, fmt.Errorf("read %s failed (exit %d): %s", path, code, stderr)
	}
	return stdout, nil
}

// WriteFile writes data to a file inside the container, creating parent dirs.
func (s *dockerSandbox) WriteFile(ctx context.Context, path string, data []byte) error {
	script := fmt.Sprintf("mkdir -p %s && cat > %s", shellQuote(dirOf(path)), shellQuote(path))
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", s.cfg.Name, "sh", "-c", script)
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("write %s failed (exit %d): %s", path, ee.ExitCode(), stderr.String())
		}
		return fmt.Errorf("docker exec write: %w", err)
	}
	return nil
}

// Stop force-removes the container. Safe to call even if Start partially failed.
func (s *dockerSandbox) Stop(ctx context.Context) error {
	_, stderr, code, err := runCmd(ctx, "docker", "rm", "-f", s.cfg.Name)
	if err != nil {
		return fmt.Errorf("docker rm: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("docker rm failed (exit %d): %s", code, stderr)
	}
	return nil
}

// runCmd runs a command and returns stdout, stderr, and the exit code. A non-zero
// exit is NOT an error (code is returned); only a launch failure sets err.
func runCmd(ctx context.Context, name string, args ...string) (stdout []byte, stderr string, code int, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err = cmd.Run()
	stdout = so.Bytes()
	stderr = se.String()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return stdout, stderr, ee.ExitCode(), nil
		}
		return stdout, stderr, -1, err
	}
	return stdout, stderr, 0, nil
}
