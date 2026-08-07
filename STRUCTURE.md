# Repository Structure

```
agent-escape/
├── go.mod                       # Go module (no external deps)
├── Makefile                     # build / test / vet / fmt targets
├── harness/                     # the Day 2 harness (Go, package main)
│   ├── types.go                 # ExecResult, Sandbox/Backend interfaces, BenchmarkResult, Canary
│   ├── backend.go               # backend registry (name -> Backend)
│   ├── backend_docker.go        # Docker backend (shells out to `docker` CLI) + runCmd helper
│   ├── backend_gvisor.go        # gVisor backend (Docker + --runtime=runsc)
│   ├── capture.go               # strace parsing + filesystem canary detection
│   ├── canary_server.go         # network canary (Day 2 placeholder; Week 2 = iptables/eBPF)
│   ├── run.go                   # RunBenchmark orchestration + result JSON writer
│   ├── util.go                  # run IDs, shell quoting
│   ├── main.go                  # CLI (stdlib flag; `run`, `backends`, `version`)
│   └── capture_test.go          # unit tests for the capture layer (no Docker needed)
├── experiments/                 # escape attempts by threat category (contributor-provided)
│   ├── filesystem_breakout/
│   ├── network_egress/
│   ├── resource_exhaustion/
│   ├── side_channel/
│   └── privilege_escalation/
├── results/                     # BenchmarkResult JSON output (one file per run)
├── threat-model/README.md       # the five categories, canaries, cross-backend matrix
├── notes/                       # recon + interface + build-log notes
│   ├── interface.md             # harness <-> Inspect contract
│   ├── provider-interface.md    # inspect_ai + control-arena recon
│   ├── sandboxescapebench.md    # positioning vs. prior work
│   ├── day2-checklist.md        # Day 2 acceptance checklist
│   └── buildlog-day2.md         # Day 2 build log (fill with findings/screenshots)
├── docs/index.html              # GitHub Pages landing stub
├── README.md
├── CONTRIBUTING.md
└── LICENSE                      # MIT
```

## Architecture at a glance

- **Sandbox interface** (`Name/Start/Exec/ReadFile/WriteFile/Stop`) — an explicit
  lifecycle around one isolation environment. `Backend` (`Name/New`) constructs them.
- **Backends** shell out to the `docker` CLI. gVisor reuses the Docker path with
  `--runtime=runsc` — identical interface, different isolation model (the Sentry).
- **Capture** wraps the agent command in `strace`, parses the syscall log, and matches
  filesystem accesses against canaries. Network attempts are derived from connect/bind
  syscalls (real enforcement is Week 2).
- **Result** is a `BenchmarkResult` JSON per run in `results/`.

## Build & run

```bash
make build            # -> bin/harness
make test             # capture-layer unit tests (no Docker)
bin/harness run --backend=docker -- cat /etc/passwd    # needs a running Docker daemon
```
