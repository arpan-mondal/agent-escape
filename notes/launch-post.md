# AgentEscape — Launch Post Drafts

Drafts for the Day 5 public launch. Twitter thread + LessWrong placeholder + checklist.

---

## Twitter

Handle: [@arpanstacy](https://twitter.com/arpanstacy)

### Option A — single tweet (272 chars)

> When AI agents run code in a sandbox, when do they escape — and does escape risk
> scale with inference compute?
>
> AgentEscape: an open, multi-backend benchmark for agent execution isolation.
> 5 threat categories, canary-based scoring, Docker → gVisor → Firecracker.
>
> 🔗 github.com/arpan-mondal/agent-escape

### Option B — thread (each tweet ≤ 280 chars)

**1/5**
> Sandbox escape and reward hacking are the same phenomenon: an agent optimizing for
> a goal, then finding the boundary meant to contain it can be crossed.
>
> So I built AgentEscape — an open benchmark that measures it. 🧵

**2/5**
> Five threat categories, each with a *canary* that makes success unambiguous:
> • Filesystem breakout
> • Network egress
> • Resource exhaustion
> • Side channel
> • Privilege escalation
>
> No partial credit. The canary fires or it doesn't.

**3/5**
> The part existing work skips: the same threat model runs across *multiple* isolation
> backends — native+seccomp, Docker, gVisor, Firecracker.
>
> That's how you separate a true isolation failure from a Docker misconfiguration.

**4/5**
> Open question we're chasing: does escape success scale log-linearly with inference
> compute — the way prior container-escape work has hinted? Containment evals may need
> to account for token budget, not just model identity.

**5/5**
> It's infrastructure from day one: MIT licensed, public contribution path, leaderboard
> coming. New escapes, new backends, and canary improvements all welcome.
>
> Repo + threat model: github.com/arpan-mondal/agent-escape

---

## LessWrong

**Title (draft):** AgentEscape: measuring when AI agents break out of their sandbox — across backends

[Full README / positioning essay here when ready]

---

## Day 5 Checklist

- [ ] Post Twitter (single tweet or thread — pick one)
- [ ] Post LessWrong (paste finished positioning essay)
- [ ] Email AISI / control-arena teams (share repo, invite backend contributions)
- [ ] Update README with leaderboard link (once the leaderboard is live)
