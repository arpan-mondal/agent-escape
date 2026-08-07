# Sandbox Provider Interface (from inspect_ai + control-arena)

Recon notes for the Day 2 harness. Goal: learn the *shape* a backend must expose
so AgentEscape's harness plugs into the same ecosystem instead of inventing its own.

**Key finding up front:** there is really only *one* interface to learn.
`control-arena` does **not** define its own sandbox provider abstraction — it
depends on `inspect-ai>=0.3.250` and consumes inspect's `SandboxEnvironment`.
It only adds a thin *config-selection* layer that picks a backend and hands
inspect a compose file. So: learn inspect's ABC; treat control-arena as a
worked example of how a downstream project selects backends.

Source files:
- `inspect_ai/src/inspect_ai/util/_sandbox/environment.py` — the ABC (the interface)
- `inspect_ai/src/inspect_ai/util/_sandbox/registry.py` — `@sandboxenv` registration
- `inspect_ai/src/inspect_ai/util/_sandbox/{local,docker/}` — reference impls
- `control_arena/settings/_sandbox_utils.py` — backend selection (`get_compose_sandbox_config`)
- `control_arena/utils/_sandbox_utils.py` — helpers built on `sandbox()` (copy dirs, clone repo)

---

## Required Methods

The contract is the abstract class `SandboxEnvironment(abc.ABC)`. Two kinds of
members: **per-instance async methods** (act on a running sandbox) and
**class-level lifecycle hooks** (create/tear down sandboxes around a run).

### Per-instance (abstract — every backend MUST implement)
- `exec(cmd: list[str], input=None, cwd=None, env=None, user=None, timeout=None, timeout_retry=True, concurrency=True) -> ExecResult[str]`
  — run a command; cwd defaults to the per-sample filesystem context. This is the
  workhorse (our attack scripts run through here).
- `write_file(file: str, contents: str | bytes) -> None` — create parent dirs as needed.
- `read_file(file: str, text: bool = True) -> str | bytes` — overloaded on `text`.

### Per-instance (concrete defaults — override if the backend supports them)
- `connection(*, user=None) -> SandboxConnection` — how to shell/vscode into the box
  (returns a `command` string, optional port mappings, container name). Raises
  `NotImplementedError` by default.
- `exec_remote(cmd, options=None, *, stream=True) -> ExecRemoteProcess | ExecResult[str]`
  — long-running / streaming process handle with `.kill()`. Built on top of `exec`.

### Class-level lifecycle (classmethods; `sample_cleanup` is abstract)
- `task_init(task_name, config) -> None` — once per task, allocate shared resources.
- `task_init_environment(config, metadata) -> dict[str,str]` — env vars that force a
  dedicated `task_init` for samples that need a distinct image (dynamic configs).
- `sample_init(task_name, config, metadata) -> dict[str, SandboxEnvironment]`
  — **this is the real "spawn"**: returns the named sandbox instances for one sample.
  First key is the default (`sandbox()` / `sandbox("default")`).
- `sample_cleanup(task_name, config, environments, interrupted) -> None` — **the "teardown"**.
- `task_cleanup(task_name, config, cleanup) -> None` — last-chance teardown (respects
  `--no-sandbox-cleanup`).
- `cli_cleanup(id) -> None` — for `inspect sandbox cleanup` from the CLI.

### Discovery / config helpers (classmethods)
- `config_files() -> list[str]` — filenames that auto-select this provider (e.g. `compose.yaml`).
- `is_docker_compatible() -> bool` — accepts Dockerfile/compose.
- `config_deserialize(config: dict) -> BaseModel` — parse a provider-specific config model.
- `default_concurrency() -> int | None` — max simultaneous sandboxes.

> Note: the tidy `spawn/execute/cleanup` triad in the CONTRIBUTING stub maps onto
> inspect as **`sample_init` / `exec` / `sample_cleanup`** — not identical names,
> but the same three responsibilities. Worth aligning our CONTRIBUTING wording to this.

---

## Registration (how a new backend plugs in)

Two steps, no core edits required:
1. Decorate the class: `@sandboxenv(name="firecracker")` (from
   `inspect_ai/util/_sandbox/registry.py`).
2. Ship it as a Python package that advertises the inspect entry point, e.g.
   ```toml
   [project.entry-points.inspect_ai]
   my_pkg = "my_pkg.providers"
   ```
   This is exactly how `inspect-k8s-sandbox` and `inspect-podman` register out of tree.

So AgentEscape backends can live in `experiments/backends/` as small packages that
register via entry point — no fork of inspect needed. This is the plug-in seam.

---

## Context Object the Harness Expects Back

There is **no single `SandboxContext` struct**. Instead:
- `sample_init(...)` returns `dict[str, SandboxEnvironment]` — the sandbox *instances*
  themselves are the "context"; you hold the object and call `exec`/`read_file` on it.
- `SandboxEnvironmentSpec(type: str, config: BaseModel | str | None)` is the *input*
  config that names a backend + points at its config file. (`"docker"` and
  `("docker", "compose.yaml")` are shorthands for it.)
- `SandboxConnection(type, command, ports?, container?)` is the connect-info returned
  by `connection()`.
- `ExecResult[str]` (returncode, stdout, stderr, success) is what `exec` returns.

For AgentEscape, our "evidence"/`observe()` idea from the threat model has **no
inspect equivalent** — inspect gives us `exec` + file I/O + a connection, but no
built-in syscall/network observation layer. That observation layer is *our* addition
and must sit alongside the inspect interface, not inside it.

---

## Multi-run Isolation

Not nested VMs at the inspect layer — inspect isolates **per sample**, one set of
named environments per sample, created by `sample_init` and destroyed by
`sample_cleanup`. Concretely per backend:
- **local** — runs tools directly on the host, *no* isolation (test only; loud warning).
- **docker** — one compose project per sample → isolated containers + network; teardown
  removes the project.
- **k8s** (`inspect-k8s-sandbox`, control-arena's heavy backend) — pods per sample.
- **podman** (`inspect-podman`) — same compose file, podman engine.

control-arena picks among these in `get_compose_sandbox_config(sandbox_type, compose_path, ...)`
returning a `SandboxEnvironmentSpec` — `docker` | `podman` | `local` share one compose
file; k8s is a separate provider. Adding a backend = "change one place."

**Contrast with SandboxEscapeBench** (see `notes/sandboxescapebench.md`): *it* uses an
outer Vagrant VM wrapping an inner Docker/k8s sandbox for safety. Neither inspect nor
control-arena bakes in that nested-VM safety wrapper. AgentEscape's threat model already
commits to "nested containment" (v2, Harness Design notes) — so **we supply the outer
safety layer ourselves**; inspect gives us the inner sandbox + exec/file plumbing.

---

## Where inspect and control-arena align vs. differ

**Align:**
- Same interface — control-arena has no separate abstraction; `SandboxEnvironment` is the standard.
- Same config object — both flow through `SandboxEnvironmentSpec(type, config)`.
- Backend-agnostic by design: swap `type` string, keep the compose file.

**Differ:**
- inspect owns the *provider ABC* and ships `local` + `docker` in core; other backends
  (k8s, podman) are out-of-tree plugins via entry points.
- control-arena owns *backend selection + eval settings* and leans on **Kubernetes**
  for its serious multi-tenant isolation, docker for lighter settings.

---

## Implications for the Day 2 harness

1. **Don't invent a provider interface — subclass `SandboxEnvironment`.** Our
   `spawn/observe/teardown` in CONTRIBUTING should be re-expressed as
   `sample_init` / (our observation layer) / `sample_cleanup`, or we accept we're
   wrapping inspect rather than reimplementing it.
2. **The observation/canary layer is ours to build.** inspect gives `exec` + file I/O
   + connection; it does NOT give syscall/network/resource observation. That's the
   novel harness surface — plan it as a sidecar around the inspect sandbox.
3. **Backends = entry-point packages** in `experiments/backends/`, decorated with
   `@sandboxenv(...)`. Firecracker/gVisor become their own small providers.
4. **We own the outer safety VM.** Adopt SandboxEscapeBench's nested containment; inspect
   only provides the inner sandbox.
5. **Naming to reconcile:** update `harness/provider_interface.py` (referenced in
   CONTRIBUTING) to mirror inspect's method names so contributors aren't confused by
   two vocabularies.
