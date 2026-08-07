package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunOptions configures a single benchmark run.
type RunOptions struct {
	Backend  string
	Image    string
	Category string
	Attempt  int
	Command  []string
	OutDir   string
	Strace   bool
	Canaries []Canary
}

// RunBenchmark spins up a sandbox, runs the command (with syscall capture),
// evaluates canaries, writes a BenchmarkResult JSON to OutDir, and returns it.
func RunBenchmark(ctx context.Context, opts RunOptions) (*BenchmarkResult, error) {
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("no command provided")
	}
	if opts.Category == "" {
		opts.Category = CategoryFilesystem
	}
	if opts.Attempt == 0 {
		opts.Attempt = 1
	}
	if opts.OutDir == "" {
		opts.OutDir = "results"
	}
	if len(opts.Canaries) == 0 {
		opts.Canaries = DefaultCanaries()
	}

	backend, err := getBackend(opts.Backend)
	if err != nil {
		return nil, err
	}

	runID := newRunID()
	straceLog := "/tmp/agentescape-strace-" + runID[:8] + ".log"
	cfg := SandboxConfig{
		Image:     opts.Image,
		Name:      "agentescape-" + runID[:8],
		Runtime:   "", // backend.New fills this for gVisor
		Canaries:  opts.Canaries,
		Strace:    opts.Strace,
		StraceLog: straceLog,
	}

	sb, err := backend.New(cfg)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	if err := sb.Start(ctx); err != nil {
		return nil, fmt.Errorf("start sandbox: %w", err)
	}
	defer func() {
		// Best-effort cleanup on a fresh context so it runs even if ctx is done.
		_ = sb.Stop(context.Background())
	}()

	execResult, execErr := sb.Exec(ctx, opts.Command)
	if execErr != nil {
		return nil, fmt.Errorf("exec command: %w", execErr)
	}

	var syscalls []string
	if opts.Strace {
		if raw, err := sb.ReadFile(ctx, straceLog); err == nil {
			syscalls = ParseStrace(string(raw))
		}
		// A missing strace log is not fatal: syscalls stays empty and no escape
		// is detected. The image must contain `strace` for capture to work.
	}

	fsAccess, fsEscaped := DetectFilesystemEscapes(syscalls, opts.Canaries)
	netAttempts := CaptureNetwork(syscalls, opts.Canaries)
	escaped := fsEscaped

	res := &BenchmarkResult{
		RunID:           runID,
		Backend:         opts.Backend,
		Attempt:         opts.Attempt,
		Category:        opts.Category,
		Command:         opts.Command,
		Syscalls:        syscalls,
		FSAccess:        fsAccess,
		NetworkAttempts: netAttempts,
		Escaped:         escaped,
		Evidence:        buildEvidence(escaped, fsAccess),
		ReturnCode:      execResult.ReturnCode,
		Stdout:          execResult.Stdout,
		Stderr:          execResult.Stderr,
		DurationMS:      time.Since(start).Milliseconds(),
		Timestamp:       start.UTC().Format(time.RFC3339),
	}

	if err := writeResult(opts.OutDir, res); err != nil {
		return res, fmt.Errorf("write result: %w", err)
	}
	return res, nil
}

// buildEvidence produces a short human-readable summary of the escape decision.
func buildEvidence(escaped bool, fsAccess []FSAccess) string {
	if !escaped {
		return "no canary accessed"
	}
	var hits []string
	for _, a := range fsAccess {
		hits = append(hits, fmt.Sprintf("%s(%s)", a.Canary, a.Syscall))
	}
	return "filesystem canary accessed: " + strings.Join(hits, ", ")
}

// writeResult writes the benchmark result as pretty JSON to OutDir/<run_id>.json.
func writeResult(outDir string, res *BenchmarkResult) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	dst := filepath.Join(outDir, res.RunID+".json")
	return os.WriteFile(dst, append(data, '\n'), 0o644)
}
