# Harness ↔ Inspect Interface Contract

Maps the AgentEscape harness contract (Week 2 target: a small **sync Go** harness with an
explicit lifecycle) onto Inspect's real `SandboxEnvironment` (async Python). Written during
the 05:30–07:00 recon block. Signatures below are copied verbatim from the source, not from
memory — see `/tmp/inspect_methods.txt` and
`~/code/inspect_ai/src/inspect_ai/util/_sandbox/environment.py`.

> ⚠️ The Inspect signatures differ from the naive "example" contract (tuple returns,
> `read_file -> bytes`, `close()`). The real ones are documented here so Week 2's adapter
> doesn't get built against a wrong assumption.

## Mapping Table

| Harness Method | Inspect Method | Inspect Signature (exact) | Notes |
|---|---|---|---|
| `Start()` | `sample_init()` (classmethod) | `async def sample_init(cls, task_name, config, metadata) -> dict[str, SandboxEnvironment]` | Inspect has **no per-instance start**; it lazy-creates envs per sample and returns the instances. Harness `Start()` is explicit; adapter calls `sample_init` and keeps the returned object. |
| `Exec()` | `exec()` | `async def exec(self, cmd: list[str], input=None, cwd=None, env=None, user=None, timeout=None, timeout_retry=True, concurrency=True) -> ExecResult[str]` | **Returns `ExecResult[str]` dataclass, NOT `(stdout, stderr, exitcode)` tuple.** Fields: `success, returncode, stdout, stderr`. Async. |
| `ReadFile()` | `read_file()` | `async def read_file(self, file: str, text: bool = True) -> str \| bytes` | Param is `file` not `path`. Returns `str` when `text=True` (default), `bytes` when `text=False`. Async. |
| `WriteFile()` | `write_file()` | `async def write_file(self, file: str, contents: str \| bytes) -> None` | Params `file` / `contents` (not `path` / `data`). Creates parent dirs. Async. |
| `Stop()` | `sample_cleanup()` / `task_cleanup()` (classmethods) | `async def sample_cleanup(cls, task_name, config, environments, interrupted: bool) -> None` | **No `close()` on the instance.** Cleanup is class-level and batch (takes the dict of envs). `interrupted` flag signals error/cancellation. `task_cleanup` respects `--no-sandbox-cleanup`. |
| `Name()` | (none) | — | Harness addition for logging/registry. Inspect identifies envs by dict key (`"default"` first) + `SandboxEnvironmentSpec.type`, not a `Name()` method. |
| (connect) | `connection()` | `async def connection(self, *, user=None) -> SandboxConnection` | Optional. Returns a shell `command` string + optional port mappings/container name. |
| (register backend) | `@sandboxenv("name")` + entry point | decorator in `registry.py` + `[project.entry-points.inspect_ai]` | How a new backend plugs in out-of-tree (e.g. `inspect-k8s-sandbox`, `inspect-podman`). |

### ExecResult (what `exec` returns)
```python
@dataclass
class ExecResult(Generic[T]):   # T = str
    success: bool       # process exited successfully
    returncode: int     # exit code
    stdout: T           # stdout contents
    stderr: T           # stderr contents
```
Python adapter unpacking for Week 2: `(result.stdout, result.stderr, result.returncode) -> BenchmarkResult`.

## Gaps vs. Inspect

- **Async everywhere.** All of `exec` / `read_file` / `write_file` / lifecycle hooks are
  `async def`. Our harness is **sync Go**. The Python adapter must bridge: run the inspect
  coroutines on an event loop (e.g. `asyncio.run` / a persistent loop) and expose sync
  results to Go. Method *names* line up; the async/sync boundary is the real translation work.
- **Lifecycle shape differs.** Inspect has no per-instance `Start`/`Stop`. Creation and
  teardown are **classmethods operating on all envs for a sample** (`sample_init` →
  `dict[str, SandboxEnvironment]`, `sample_cleanup(..., environments, interrupted)`), plus
  task-level `task_init`/`task_cleanup`. Our harness wants explicit per-sandbox Start/Stop;
  the adapter wraps the class-level calls to present that.
- **Exception handling vs. context.Context.** Inspect signals failure via **exceptions**
  (`TimeoutError`, `PermissionError`, `FileNotFoundError`, `OutputLimitExceededError`,
  `UnicodeDecodeError`) and an `interrupted: bool` on cleanup. Go side uses
  `context.Context` for cancellation/timeout. Adapter maps Go ctx-cancel → inspect
  cancellation, and inspect exceptions → Go `error` values.
- **No `close()`; no `__enter__`/`__exit__`.** `__init__` exists (sets injection locks)
  but there is no context-manager protocol. Don't design the adapter around `with sandbox:`.
- **Output limits are provider-dependent.** `exec` stdout/stderr default cap 10 MiB;
  behavior over the limit varies (may truncate or raise). For large output, write to a file
  and `read_file` (which reliably raises `OutputLimitExceededError`). Relevant to our
  filesystem-canary reads.
- **Observation layer is missing entirely.** Inspect gives exec + file I/O + connection. It
  has **no** syscall / network / resource observation. Our canary/`observe()` surface is
  ours to build as a sidecar — it is NOT part of this interface. (See `provider-interface.md`.)

## Week 2 Integration Notes

- Python adapter will call these methods directly with **no wrapping of the names** — but it
  **must** wrap the async→sync boundary (Go is sync). Names match; runtime model does not.
- Method names MUST match inspect exactly, or a translation layer is required. They do match
  for `exec` / `read_file` / `write_file`; `Start`/`Stop` have **no** 1:1 inspect method and
  map to `sample_init` / `sample_cleanup` (classmethods) — plan the adapter accordingly.
- `ExecResult` unpacking: Python converts `(stdout, stderr, returncode)` → `BenchmarkResult`.
  Remember it's a **dataclass with a `success` bool**, not a bare tuple — read fields by name.
- **No runtime differences between Docker and gVisor from the interface perspective.** gVisor
  is "Docker + a runtime flag" (`docker run --runtime=runsc ...`); the `SandboxEnvironment`
  contract is identical. Backend choice is a `SandboxEnvironmentSpec(type=...)` / compose
  detail, not an interface change.
- Errors: map inspect exceptions → Go errors; map Go `context.Context` cancel → inspect
  cancellation (which surfaces as `interrupted=True` in cleanup).

## gVisor Note (Block 2)

gVisor is **Docker + a runtime flag**. Register once, then select per-run:
```bash
sudo runsc install                 # adds a "runsc" runtime to /etc/docker/daemon.json
sudo systemctl restart docker
docker run --runtime=runsc --rm hello-world     # the only difference from a normal run
```
(Install of `runsc` itself is via gVisor's apt repo, e.g. `apt-get install -y runsc`.)

**Isolation model:** gVisor interposes a userspace guest kernel — the **Sentry** — between
the container and the host. Application syscalls are trapped and serviced by the Sentry
(implemented in Go), so the container never talks to the host Linux kernel directly; the
host only sees the `runsc` process making a small, restricted set of syscalls. Stronger
isolation than stock Docker/runc, at a performance cost.

**Why this matters for measurement fairness:** we must capture syscalls **at the guest
boundary** (what the agent's process issues, as seen by the Sentry), **not** at the host
level — host-level tracing under gVisor only sees `runsc`'s own syscalls, which would make
gVisor look artificially "quiet" and break cross-backend comparability. Same canary
definitions, backend-appropriate observation point.

> **One-sentence version:** gVisor is Docker plus a runtime flag — identical interface, a
> different (Sentry-mediated) isolation model — so we observe at the guest boundary to keep
> escape measurements comparable across backends.

### Local install status
Not installed here: **`runsc` is Linux-only** and this host is macOS (darwin), which has no
Linux host kernel for the Sentry to run on. Per the block's fallback, reading the docs is
sufficient; local install is deferred to a Linux CI/host in Week 2.

---
_TODO (Week 2): confirm the exact event-loop bridging strategy (persistent loop vs.
per-call `asyncio.run`) once the Go↔Python adapter shape is chosen._
