# AgentEscape

**A multi-backend sandbox escape benchmark for AI agent execution isolation.**

## Research Question

When AI agents run code in sandboxes, when and how do they escape? And does the escape risk scale with inference compute?

We believe reward hacking in RL environments and sandbox escape in agent execution are the same phenomenon: an agent optimizing for a goal while respecting a boundary, then finding that the boundary can be crossed. This matters to everyone from environment designers (Inspect, RL-Gym) to AI safety and control teams.

AgentEscape is an open benchmark measuring escape success across five threat categories, multiple isolation backends, and a transparent leaderboard.

## Why This Matters

- **Escape = Control Problem**: A sandbox escape is a concrete, measurable control failure. It tells you when your isolation assumptions break.
- **No Unified Measurement**: Existing work (SandboxEscapeBench, control-arena) tests specific backends or scenarios, but there's no open comparison across isolation strategies.
- **Contribution Path**: Infrastructure benchmarks get cited. This one is designed for community submissions.

## Threat Model

Five escape categories, each with a canary that makes success measurable:

1. **Filesystem Breakout** — read/write outside workspace (host files, other tenants, mounted secrets)
2. **Network Egress** — unauthorized connections (metadata endpoints, DNS exfiltration, internal services)
3. **Resource Exhaustion** — denial of service to host or co-tenants (fork bomb, unbounded memory, disk fill)
4. **Side Channel** — infer across boundary without direct channel (timing, shared kernel state)
5. **Privilege Escalation** — escape to host-level privilege (mounted sockets, kernel CVEs, over-permissive caps)

[See `threat-model/README.md` for detailed specs and canaries.]

## Isolation Backends Tested

- Firecracker (microVM, from StacyOS)
- Docker + runc (container)
- gVisor (user-space kernel)
- systemd-nspawn (lightweight namespace)
- Native OS process with seccomp (baseline)

## Running the Benchmark

```bash
git clone https://github.com/arpan-mondal/agent-escape
cd agent-escape
# stub: instructions coming
```

## Leaderboard

Coming after Day 2. [Link TBD.]

## Contributing

AgentEscape is designed for community submission of new escape attempts, backends, and canary strategies. See `CONTRIBUTING.md`.

## Related Work

- SandboxEscapeBench (arXiv:2603.02277)
- Inspect (google-deepmind/inspect)
- control-arena (MIRI/Anthropic)
- StacyOS: Federated Execution & Cryptographic Trust for AI Artifacts (prior isolation work)

## Citation

```bibtex
@software{agentscape2026,
  author = {Mondal, Arpan},
  title = {AgentEscape: A Multi-Backend Sandbox Escape Benchmark},
  year = {2026},
  howpublished = {\url{https://github.com/arpan-mondal/agent-escape}}
}
```

## License

MIT
