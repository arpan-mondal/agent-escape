package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

const usageText = `agent-escape harness — sandbox escape benchmark runner

Usage:
  harness run [flags] -- <command> [args...]

Commands:
  run        Run a command in a sandbox and evaluate escape canaries
  backends   List available isolation backends
  version    Print version

Run flags:
  --backend   Isolation backend (default "docker")
  --image     Container image (default "ubuntu:24.04"; must contain strace for capture)
  --category  Threat category label (default "filesystem_breakout")
  --attempt   Attempt number for this run (default 1)
  --out       Directory to write result JSON (default "results")
  --strace    Capture syscalls with strace (default true)

Examples:
  harness run --backend=docker -- whoami
  harness run --backend=docker -- cat /etc/passwd
  harness run --backend=gvisor --image=ubuntu:24.04 -- cat /root/.ssh/id_rsa
`

const version = "0.1.0-day2"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	case "backends":
		fmt.Println(availableBackends())
	case "version":
		fmt.Println(version)
	case "-h", "--help", "help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usageText)
		os.Exit(2)
	}
}

// cmdRun parses `run` flags, splitting the sandbox command off at "--".
func cmdRun(args []string) int {
	flagArgs, command := splitOnDashDash(args)

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	backend := fs.String("backend", "docker", "isolation backend")
	image := fs.String("image", "ubuntu:24.04", "container image (must contain strace)")
	category := fs.String("category", CategoryFilesystem, "threat category label")
	attempt := fs.Int("attempt", 1, "attempt number")
	outDir := fs.String("out", "results", "output directory for result JSON")
	strace := fs.Bool("strace", true, "capture syscalls with strace")
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}

	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "error: no command provided (put it after `--`, e.g. `run --backend=docker -- whoami`)")
		return 2
	}

	opts := RunOptions{
		Backend:  *backend,
		Image:    *image,
		Category: *category,
		Attempt:  *attempt,
		Command:  command,
		OutDir:   *outDir,
		Strace:   *strace,
	}

	res, err := RunBenchmark(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Printf("run_id:   %s\n", res.RunID)
	fmt.Printf("backend:  %s\n", res.Backend)
	fmt.Printf("command:  %v\n", res.Command)
	fmt.Printf("exit:     %d\n", res.ReturnCode)
	fmt.Printf("escaped:  %v\n", res.Escaped)
	fmt.Printf("evidence: %s\n", res.Evidence)
	fmt.Printf("result:   %s/%s.json\n", *outDir, res.RunID)
	return 0
}

// splitOnDashDash separates flag args (before "--") from the command (after it).
func splitOnDashDash(args []string) (flags, command []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}
