# restake-xray — Design Spec

**Date:** 2026-06-03
**Status:** Approved (brainstorming complete; next: implementation plan)
**Working name:** `restake-xray` (CLI binary: `xray`). Name candidates: LRTscope, eigenxray, rehypo, delegraph.

## One-liner

The open, verifiable map of what actually backs every liquid restaking token — which
operators, which AVSs, which collateral, and where they dangerously overlap.

## Goal & success criteria

- **Primary goal:** a *famous* open-source project (GitHub stars + ecosystem reach). Money is optional.
- **Builder context:** solo, nights/weekends; documented failure mode is abandoning side projects at month 3–4 when novelty fades. Every design choice must fight that.
- **Success signals (post-launch):** ~100 stars in week 1, or ≥3 unsolicited "can it do <protocol X>?" / "can it do <metric Y>?" issues.
- **Kill signal:** sub-30 stars and no engagement after a genuine launch push → the wedge isn't landing; revisit framing before sinking more nights.

## Strategic decisions (locked during brainstorming)

| Decision | Choice | Why |
|---|---|---|
| Primary user | Developers/protocols (infra) + an open public data layer | Crypto's most-starred OSS is libraries/tools, not dashboards; lowest ops burden for a solo build |
| Core wedge | **Exposure / composition X-ray** | Most data-driven (least subjective modeling); the foundation risk-scoring & contagion-sim need anyway; hardest for outsiders to replicate given deep restaking domain knowledge |
| Language | **Go** | Builder's primary backend language → fastest to ship, lowest abandonment friction; strong fintech-infra precedent. Public JSON API keeps adoption stack-agnostic |
| Data source | **Hybrid:** on-chain truth + off-chain labels | Numbers are self-verifiable (credibility/durability); off-chain metadata only for readability; free-tier RPC covers v1 |
| Data depth | Live current-state snapshot (refreshed on schedule) | Historical indexer is a multi-month lift = the month-4 trap; the X-ray's "wow" works on current state |
| v1 breadth | EigenLayer-deep first; protocol-agnostic adapter interface from day one | Dominant ecosystem = best demo + fastest ship; adapters make Symbiotic/Karak/Babylon fast-follows (each a fresh launch moment) instead of rewrites |
| Distribution | CLI + library + **git-committed JSON dataset** + **real hosted API** + thin static demo dashboard | Captures both prizes: stars (library/CLI) and reach (open dataset + API, DefiLlama playbook) |

## The core abstraction — the exposure graph

A protocol-agnostic directed graph:

```
collateral → LRT → operator delegations → AVSs / operator-sets → slashing obligations
```

Everything else is a derived view over this one graph. Each restaking protocol is an
adapter that knows how to populate this graph from its own contracts.

### Derived X-ray outputs (v1)

1. **Composition** — for a given LRT: which collateral and which operators back it, with weights.
2. **Concentration** — operator/AVS concentration per LRT (HHI-style score) → "how diversified is this LRT really."
3. **Contagion matrix** — shared-operator / shared-AVS overlap *between* LRTs: "if operator X or AVS Y fails, which LRTs bleed together." This is the headline screenshot for launch.

## Architecture (Go packages)

- **`adapter`** — `Protocol` interface each restaking protocol implements (enumerate operators, operator→AVS/operator-sets, LRT→operator delegations, strategies/collateral). v1 ships **only the EigenLayer adapter**. This interface is the anti-cyclical hedge.
- **`chain`** — abigen-generated bindings for EigenLayer core contracts (DelegationManager, AllocationManager/StrategyManager, AVSDirectory / operator-sets) + ERC20/LRT contracts; batched **multicall** reads against an RPC node; response caching.
- **`graph`** — assembles the typed exposure graph and computes derived metrics (composition, concentration, contagion).
- **`labels`** — pluggable off-chain enrichment (operator/AVS names, token symbols, optional prices) from a curated JSON + optional price source; **graceful fallback to addresses** when a label is missing.
- **`snapshot`** — serializes the computed graph to a canonical, stable-ordered **JSON dataset**; scheduled regeneration; schema-versioned.
- **`api`** — thin, read-only HTTP/JSON server over the latest snapshot. Stateless; loads the cached snapshot into memory and reloads on refresh. CORS-open, public base URL. Endpoints: `/lrts`, `/lrt/{id}/exposure`, `/operators`, `/avs`, `/contagion`, `/health` (+ snapshot timestamp/block in every response).
- **`cmd/xray`** — CLI: `xray scan`, `xray lrt <symbol>`, `xray contagion`, `xray serve`. The dev-facing star magnet.
- **Library surface** — all packages importable; primary entrypoint `engine.New(rpc).Snapshot(ctx)` returns the typed exposure graph.

### Out-of-process pieces

- **Static demo dashboard** (separate, thin) — fetches the published JSON snapshot and renders the exposure graph + contagion matrix. No backend → no ops. Sole purpose: README hero image + launch screenshot.
- **Scheduled snapshot job** — regenerates the snapshot on a cron, commits the JSON to the public repo (or GH releases), and signals the hosted API to reload.

## Data flow

```
RPC node
  → chain reader (batched multicall, cached)
  → EigenLayer adapter (normalizes into graph primitives)
  → graph builder (assemble graph + compute composition/concentration/contagion)
  → label enrichment (names, symbols, optional prices; fallback to addresses)
  → snapshot (canonical JSON, schema-versioned)
      ├── CLI prints
      ├── hosted API serves
      ├── git-committed dataset (versioned, transparent, zero-ops source of truth)
      └── static dashboard renders
```

## Ops posture (designed against the month-4 abandonment trap)

- The **git-committed dataset is the source of truth**. The hosted API and dashboard are convenience layers over it.
- The **hosted API is read-only and stateless** — single small instance or serverless, cheap, CORS-open. If it dies, CLI + dataset + dashboard all keep working (graceful degradation, never a single point of failure).
- No historical database, no write path, no user accounts in v1 → near-zero operational surface.

## Error handling & correctness (the credibility crux)

- **RPC failures:** retry with backoff; **partial-snapshot tolerance** — mark stale/failed nodes, never fail the whole scan because one contract read failed.
- **Contract upgrades:** adapter versioning; integration tests pinned to a known block.
- **Invariant checks baked into the engine:** e.g. the sum of an LRT's operator delegations must reconcile to its restaked balance; drift is flagged in output. ("We assert our own numbers reconcile" is both a correctness guard and a README credibility flex — a reconciliation instinct carried over from payments infra.)
- Every snapshot/API response carries the **block number + timestamp** it was computed at, so consumers can judge freshness and reproduce.

## Testing strategy

- **Unit tests** on graph metrics (composition, concentration HHI, contagion overlap) using synthetic graphs.
- **Golden tests** against a **pinned mainnet block** using recorded RPC fixtures → deterministic, offline, fast. Hand-verify expected exposure for 1–2 LRTs once, then lock it in.
- **Invariant assertions** run as part of `xray scan` (balances reconcile).

## v1 scope

**In:**
- EigenLayer mainnet adapter.
- Top ~6–8 LRTs (e.g. ezETH, rsETH, pufETH, cmETH, …).
- Live snapshot only.
- CLI + importable library + git-committed JSON dataset + real hosted API + static demo dashboard + README.
- Invariant checks + golden tests.

**Out (deferred):**
- Historical time-series / indexer.
- Risk *scoring* (v2 — layered on top of the exposure graph).
- Slashing/contagion *simulation* (forward-looking modeling).
- Additional protocol adapters: Symbiotic / Karak / Babylon (fast-follows, each its own launch moment).
- Prices as a core dependency (optional enrichment only).

## Anti-cyclical hedge

- Core data model is protocol-agnostic ("shared-security exposure"), not EigenLayer-specific.
- Adapter interface fixed in v1 so a second protocol is additive, not a rewrite.
- Each new adapter is a deliberate momentum relaunch (v1.1 Symbiotic, etc.) to counter novelty fade.

## Launch artifact

- `go install` one-liner for the CLI.
- Show HN / X thread led by the **contagion matrix**: "here's what really backs the top LRTs and where they secretly overlap."
- Public, versioned open dataset repo + hosted API base URL in the README.
- README hero = the exposure/contagion visual from the static dashboard.

## Open follow-ups (not blocking implementation)

- Final project name.
- Choice of RPC provider for the scheduled job (free tier for v1).
- Hosting target for the API (serverless vs small VPS) — decide at deploy time.
