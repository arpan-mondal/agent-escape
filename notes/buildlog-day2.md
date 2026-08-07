# Build Log — Day 2

> Reflective log. Fill in your own findings, decisions, and a screenshot of the first
> successful `escaped: true` run once you have Docker/a Linux host. Placeholders below.

## What got built
- Go harness scaffold: `Sandbox` + `Backend` interfaces, Docker + gVisor backends,
  strace-based capture layer, `BenchmarkResult` JSON schema, stdlib CLI.
- Capture layer is fully unit-tested (no Docker required); 5/5 tests pass.

## Key decisions (and why)
- **Zero external dependencies.** Backends shell out to the `docker` CLI via `os/exec`
  rather than the official Docker Go SDK. Rationale: guaranteed offline-clean build,
  lean repo, transparent commands, easy `--runtime=runsc` swap for gVisor. Trade-off:
  less type-safe than the SDK; revisit in Week 2 if we need streaming/events.
- **gVisor = Docker + a runtime flag.** `gvisorBackend` reuses `dockerSandbox` with
  `Runtime="runsc"`. Same interface, different isolation model.
- **Capture at the syscall log, canary match as ground truth.** strace wraps the agent
  command; filesystem escape = a path syscall touching a canary path.

## Open questions / TODO
- [ ] Screenshot: first `cat /etc/passwd` → `escaped: true` (needs Docker).
- [ ] Confirm strace works under gVisor and that we observe at the guest boundary.
- [ ] Decide default image (ubuntu + strace vs. a prebuilt image bundling strace).
- [ ] Week 2: real network enforcement (iptables/eBPF) + resource/side-channel/privilege canaries.

## Reflection
_(your notes here)_
