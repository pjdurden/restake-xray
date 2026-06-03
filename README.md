# restake-xray

**The open, verifiable map of what actually backs every liquid restaking token** — which operators, which AVSs, which collateral, and where they dangerously overlap.

restake-xray reads the restaking exposure graph directly from chain
(`collateral → LRT → operator → AVS`) and computes the things holders and
allocators actually need but can't easily see: operator **concentration**, AVS
exposure, and the **contagion matrix** of shared exposure between LRTs.

> Verifiable by design: every snapshot asserts its own invariants (e.g. each
> LRT's per-operator delegations reconcile to its restaked total) and is stamped
> with the block it was computed at. The numbers are read from chain, not scraped.

## Install

```bash
go install github.com/prajjwalchittori/restake-xray/cmd/xray@latest
```

## Use

```bash
xray scan                 # exposure summary: per-LRT restaked, operators, concentration
xray contagion            # shared-exposure matrix between LRTs
xray systemic             # ecosystem single-points-of-failure (operators & AVSs ranked)
xray warnings             # derived health flags (single-operator dep, high concentration, invariant fails)
xray lrt cmETH            # one LRT's full backing breakdown
xray operator <addr>      # what one operator secures + which LRTs depend on it
xray avs <addr>           # which operators/LRTs are exposed to one AVS
xray report               # shareable Markdown report
xray graph --dot          # exposure graph as Graphviz DOT (pipe to `dot -Tsvg`)
xray diff old.json new.json   # what changed between two snapshots
xray scan --out data/latest.json   # write the JSON dataset
xray serve --data data/latest.json # read-only JSON API on :8080
```

Add `--json` to `scan`, `contagion`, `systemic`, `warnings`, `lrt`, `operator`,
`avs`, and `diff` for machine-readable output.

By default the commands run against the bundled offline sample
(`testdata/sample-graph.json`) so you can try everything with zero setup. Live
mode (reading mainnet via an RPC endpoint) is wired through the same
`adapter.Protocol` interface — see [Live data](#live-data).

## As a library

```go
import (
    "github.com/prajjwalchittori/restake-xray/engine"
    "github.com/prajjwalchittori/restake-xray/adapter"
    "github.com/prajjwalchittori/restake-xray/labels"
)

e := engine.New([]adapter.Protocol{myAdapter}, labels.Noop{})
snap, err := e.Snapshot(ctx) // typed exposure graph + metrics + invariants
```

## API

Read-only JSON, CORS-open. `GET /health`, `/lrts`, `/operators`, `/avs`,
`/contagion`, `/systemic`, `/warnings`, `/lrt/{symbol}/exposure`,
`/operator/{addr}`, `/avs/{addr}`. Every response is derived from the latest
committed snapshot, so the API is a stateless convenience layer over the dataset
— if it's down, the CLI and the committed `data/` snapshots still work.

## Supported protocols

- **EigenLayer** (mainnet).

More protocols (Symbiotic, Karak, Babylon, …) plug in through the
`adapter.Protocol` interface in [`adapter/adapter.go`](adapter/adapter.go) — the
exposure graph itself is protocol-agnostic. **PRs adding adapters are welcome.**

### Add an adapter

Implement `Name()` and `Snapshot(ctx) (graph.Graph, error)`, populating the
`collateral → LRT → operator → AVS` graph from your protocol's contracts. The
EigenLayer adapter ([`adapter/eigenlayer`](adapter/eigenlayer)) is the reference:
it reads through a narrow `Reader` port so it's unit-tested against recorded
fixtures and only the thin live wiring needs an RPC node.

## Live data

Live reads go through `adapter/eigenlayer`'s `Reader` port. The live
implementation (go-ethereum bindings + multicall) is tracked in
[`adapter/eigenlayer/LIVE.md`](adapter/eigenlayer/LIVE.md). Regenerate the
published dataset with:

```bash
RPC_URL=<https-rpc> ./scripts/snapshot.sh
```

Each run writes `data/latest.json` (the dataset), `data/latest.dot`, and — if
[Graphviz](https://graphviz.org/) is installed — `data/latest.svg`, the
exposure-graph diagram, plus timestamped copies. Generate the diagram from any
dataset without a node:

```bash
xray graph --from data/latest.json --dot | dot -Tsvg -o exposure.svg
```

## Development

```bash
make test    # go test ./...
make build   # -> ./xray
make scan    # run against the offline sample
```

## License

MIT.
