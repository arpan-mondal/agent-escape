# Day 2 — Acceptance Checklist

## Deep Block 1 — Backends + CLI
- [x] Go module set up (`go.mod`, no external deps)
- [x] `ExecResult` = `{Success bool, ReturnCode int, Stdout string, Stderr string}`
- [x] `Sandbox` interface: `Name / Start / Exec / ReadFile / WriteFile / Stop`
- [x] `Backend` interface: `Name / New(cfg SandboxConfig) (Sandbox, error)`
- [x] Docker backend: create container (workspace mount, `--cap-add=SYS_PTRACE`),
      exec with strace wrapping, read/write files, cleanup
- [x] gVisor backend: Docker path + `--runtime=runsc`
- [x] CLI wired (`run` / `backends` / `version`, `--backend/--image/--category/--attempt/--out/--strace`, `--` passthrough)
- [x] `go build` succeeds
- [ ] `bin/harness run --backend=docker -- whoami` works  ← **needs a running Docker daemon** (not available on this macOS box)

## Deep Block 2 — Capture layer + result schema
- [x] Syscall channel: parse strace for openat/open/stat/read/write/connect/bind → `["syscall:target", ...]`
- [x] Filesystem channel: match syscalls against canaries → `[]FSAccess` + `escaped`
- [x] Network channel: placeholder deriving attempts from connect/bind (Week 2: iptables/eBPF)
- [x] `BenchmarkResult` JSON: run_id, backend, attempt, category, command, syscalls[],
      fs_access[], network_attempts[], escaped, evidence, return_code, duration_ms, timestamp
- [x] Result written to `results/<run_id>.json`
- [x] Unit tests for the capture layer pass (`go test ./harness/`)
- [ ] `bin/harness run --backend=docker -- cat /etc/passwd` → `escaped: true` + JSON  ← **needs Docker**

## Verified locally (macOS, no Docker)
- `go build -o bin/harness ./harness` — OK
- `go vet ./...` — clean
- `go test ./harness/` — 5/5 pass (ParseStrace, DetectFilesystemEscapes ±, dir-canary match, network)
- `bin/harness version` / `backends` / help — OK

## Deferred to a Linux host with Docker (Blocks 5–8 / Week 2)
- End-to-end `docker` run: `whoami` (escaped:false) and `cat /etc/passwd` (escaped:true)
- gVisor run: install `runsc`, `runsc install`, restart docker, `--runtime=runsc`
- Confirm syscall capture happens at the guest boundary under gVisor
- Default image must contain `strace` (ubuntu: `apt-get install -y strace`)
