# restake-xray v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the v1 open-source restaking exposure X-ray engine: from a single `xray scan` command, produce a verifiable map of what backs each liquid restaking token (collateral → operators → AVSs) plus concentration and contagion metrics, exposed as a CLI, an importable Go library, a committed JSON dataset, a read-only hosted API, and a static demo dashboard.

**Architecture:** A protocol-agnostic exposure `graph` (leaf package, pure data + pure metric functions) is populated by pluggable `adapter.Protocol` implementations. An offline `sample` adapter (loads a JSON fixture) lets the entire pipeline — metrics, snapshot serialization, CLI rendering, API, dashboard — be built and demoed with zero network. The real `eigenlayer` adapter is built last behind a narrow `Reader` port so it is unit-tested against recorded fixtures (golden test at a pinned block) and only the thin live wiring needs an RPC node.

**Tech Stack:** Go 1.22+ (user-local install, no sudo), standard library `net/http` (1.22 pattern routing), `math/big`, `encoding/json`; `github.com/ethereum/go-ethereum` (ethclient + abigen bindings) only in the live reader; plain HTML/JS for the static dashboard.

---

## Execution notes (read first)

- **Module path:** `github.com/prajjwalchittori/restake-xray`. If your GitHub handle differs, change it in `go.mod` and every import before starting — pick the final value now so imports stay consistent.
- **Commit signing:** this environment requires *your* signing key for commits. The `git commit` steps below document intended commit points. When executing as an agent that cannot sign, run the `git add` parts and let the user commit via `./commit-and-push.sh`, or set `git -c commit.gpgsign=false commit ...` for throwaway WIP. Do **not** push; pushing stays manual.
- **Working dir:** `/home/pjsump/restake-xray` (already a git repo; spec lives in `docs/superpowers/specs/`).
- **TDD:** every logic task writes the failing test first, watches it fail, implements minimally, watches it pass.

## File structure (what each file owns)

```
go.mod
graph/                         # leaf: pure data model + pure functions, no I/O
  graph.go                     # Graph, LRT, Stake, Delegation, Operator, AVS types
  metrics.go                   # OperatorConcentration, ContagionMatrix, Overlap
  invariants.go                # InvariantResult, CheckInvariants
  graph_test.go metrics_test.go invariants_test.go
snapshot/
  snapshot.go                  # Snapshot type, Build (compute+canonicalize), JSON, Read/Write
  snapshot_test.go
render/
  render.go                    # Snapshot/Graph -> terminal tables (pure strings)
  render_test.go
adapter/
  adapter.go                   # Protocol interface
  sample/sample.go             # offline adapter reading a JSON graph fixture
  sample/sample_test.go
labels/
  labels.go                    # Provider interface, Static (file), Noop
  labels_test.go
engine/
  engine.go                    # Engine.New / Snapshot: run adapters, apply labels, build snapshot
  engine_test.go
api/
  api.go                       # NewServer(load) http.Handler + CORS, JSON routes
  api_test.go
adapter/eigenlayer/
  eigenlayer.go                # Reader port, LRTConfig, Backing, Adapter (pure orchestration)
  eigenlayer_test.go           # golden test vs fixtureReader at a pinned block
  live.go                      # live Reader: ethclient + multicall + abigen bindings (build-tagged off in CI)
  bindings/                    # abigen-generated bindings (generated at impl time)
cmd/xray/
  main.go                      # subcommand dispatch: scan | lrt | contagion | serve
testdata/
  sample-graph.json            # demo fixture (NOT published as truth)
  labels.json                  # curated operator/AVS/token labels
  eigenlayer-pinned-<block>.json  # recorded reads for golden test
data/                          # published dataset output (snapshot JSON committed here)
web/
  index.html app.js style.css  # static dashboard, fetches data/latest.json
Makefile                       # build, test, snapshot, serve targets
README.md
```

---

## Task 0: Toolchain + module init

**Files:**
- Create: `go.mod`, `.gitignore`, `Makefile`

- [ ] **Step 1: Install Go locally (no sudo)**

```bash
cd /tmp
curl -fsSLO https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
mkdir -p "$HOME/go-sdk"
tar -C "$HOME/go-sdk" -xzf go1.22.5.linux-amd64.tar.gz   # creates $HOME/go-sdk/go
echo 'export PATH="$HOME/go-sdk/go/bin:$HOME/go/bin:$PATH"' >> "$HOME/.bashrc"
export PATH="$HOME/go-sdk/go/bin:$HOME/go/bin:$PATH"
```

- [ ] **Step 2: Verify Go**

Run: `go version`
Expected: `go version go1.22.5 linux/amd64`

- [ ] **Step 3: Init module + .gitignore + Makefile**

```bash
cd /home/pjsump/restake-xray
go mod init github.com/prajjwalchittori/restake-xray
printf '/xray\n/dist/\n*.out\n.env\n' > .gitignore
```

`Makefile`:
```make
.PHONY: build test scan serve snapshot
build:    ; go build -o xray ./cmd/xray
test:     ; go test ./...
scan:     ; go run ./cmd/xray scan --sample testdata/sample-graph.json
serve:    ; go run ./cmd/xray serve --data data/latest.json
```

- [ ] **Step 4: Verify build tooling**

Run: `go build ./... && echo OK`
Expected: `OK` (no packages yet is fine; exits 0)

- [ ] **Step 5: Commit** — `git add go.mod .gitignore Makefile && git commit -m "chore: go module + toolchain"`

---

## Task 1: Core graph types

**Files:**
- Create: `graph/graph.go`, `graph/graph_test.go`

- [ ] **Step 1: Write the failing test**

`graph/graph_test.go`:
```go
package graph

import "testing"

func TestGraphHoldsLRTOperatorAVS(t *testing.T) {
	g := Graph{
		Block: 100, Timestamp: 1, Protocol: "eigenlayer",
		LRTs: []LRT{{
			Address: "0xlrt", Symbol: "cmETH", Restaked: "30",
			Collateral:  []Stake{{Token: "0xste", Symbol: "stETH", Amount: "30", Decimals: 18}},
			Delegations: []Delegation{{Operator: "0xop1", Amount: "20"}, {Operator: "0xop2", Amount: "10"}},
		}},
		Operators: []Operator{{Address: "0xop1", Name: "Op One", AVSs: []string{"0xavs"}}},
		AVSs:      []AVS{{Address: "0xavs", Name: "EigenDA"}},
	}
	if g.LRTs[0].Symbol != "cmETH" || len(g.LRTs[0].Delegations) != 2 {
		t.Fatalf("unexpected graph shape: %+v", g)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./graph/...`
Expected: FAIL — undefined: Graph/LRT/Stake/Delegation/Operator/AVS

- [ ] **Step 3: Implement**

`graph/graph.go`:
```go
// Package graph defines the protocol-agnostic restaking exposure graph
// (collateral -> LRT -> operator -> AVS) and pure functions over it.
package graph

// Graph is a single point-in-time exposure snapshot for one or more protocols.
type Graph struct {
	Block     uint64     `json:"block"`
	Timestamp int64      `json:"timestamp"`
	Protocol  string     `json:"protocol"`
	LRTs      []LRT      `json:"lrts"`
	Operators []Operator `json:"operators"`
	AVSs      []AVS      `json:"avss"`
}

// LRT is a liquid restaking token and how it is backed.
type LRT struct {
	Address     string       `json:"address"`
	Symbol      string       `json:"symbol"`
	Restaked    string       `json:"restaked"` // total restaked backing, base-unit decimal string
	Collateral  []Stake      `json:"collateral"`
	Delegations []Delegation `json:"delegations"`
}

// Stake is a collateral position backing an LRT.
type Stake struct {
	Token    string `json:"token"`
	Symbol   string `json:"symbol"`
	Amount   string `json:"amount"` // base-unit decimal string
	Decimals uint8  `json:"decimals"`
}

// Delegation is restaked backing delegated from an LRT to an operator.
type Delegation struct {
	Operator string `json:"operator"`
	Amount   string `json:"amount"` // base-unit decimal string
}

// Operator restakes delegated assets and secures AVSs.
type Operator struct {
	Address string   `json:"address"`
	Name    string   `json:"name"`
	AVSs    []string `json:"avss"`
}

// AVS is a service secured by operators.
type AVS struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./graph/...` → PASS
- [ ] **Step 5: Commit** — `git add graph/ && git commit -m "feat(graph): core exposure types"`

---

## Task 2: Operator concentration metric (HHI)

**Files:**
- Create: `graph/metrics.go`, `graph/metrics_test.go`

- [ ] **Step 1: Write the failing test**

`graph/metrics_test.go`:
```go
package graph

import (
	"math"
	"testing"
)

func TestOperatorConcentration(t *testing.T) {
	cases := []struct {
		name string
		l    LRT
		want float64
	}{
		{"single operator", LRT{Delegations: []Delegation{{Operator: "a", Amount: "100"}}}, 1.0},
		{"even split", LRT{Delegations: []Delegation{{Operator: "a", Amount: "50"}, {Operator: "b", Amount: "50"}}}, 0.5},
		{"empty", LRT{}, 0.0},
		{"zero amounts", LRT{Delegations: []Delegation{{Operator: "a", Amount: "0"}}}, 0.0},
	}
	for _, c := range cases {
		got := OperatorConcentration(c.l)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./graph/ -run TestOperatorConcentration` → FAIL undefined OperatorConcentration

- [ ] **Step 3: Implement**

`graph/metrics.go`:
```go
package graph

import (
	"math/big"
	"sort"
)

// parseAmount parses a base-unit decimal string into a big.Float; invalid -> 0.
func parseAmount(s string) *big.Float {
	f, ok := new(big.Float).SetString(s)
	if !ok {
		return big.NewFloat(0)
	}
	return f
}

// OperatorConcentration is the Herfindahl-Hirschman Index (0..1) of an LRT's
// restaked amount across operators. 1 = fully concentrated, ~0 = well spread.
func OperatorConcentration(l LRT) float64 {
	total := big.NewFloat(0)
	amts := make([]*big.Float, 0, len(l.Delegations))
	for _, d := range l.Delegations {
		a := parseAmount(d.Amount)
		amts = append(amts, a)
		total.Add(total, a)
	}
	if total.Sign() == 0 {
		return 0
	}
	hhi := 0.0
	for _, a := range amts {
		w, _ := new(big.Float).Quo(a, total).Float64()
		hhi += w * w
	}
	return hhi
}

// sortedStrings returns a sorted copy (helper for deterministic outputs).
func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./graph/ -run TestOperatorConcentration` → PASS
- [ ] **Step 5: Commit** — `git add graph/metrics.go graph/metrics_test.go && git commit -m "feat(graph): operator concentration (HHI)"`

---

## Task 3: Contagion matrix

**Files:**
- Modify: `graph/metrics.go`
- Modify: `graph/metrics_test.go`

- [ ] **Step 1: Write the failing test** (append)

```go
func TestContagionMatrix(t *testing.T) {
	g := Graph{
		LRTs: []LRT{
			{Symbol: "cmETH", Delegations: []Delegation{{Operator: "op1", Amount: "1"}, {Operator: "op2", Amount: "1"}}},
			{Symbol: "ezETH", Delegations: []Delegation{{Operator: "op2", Amount: "1"}, {Operator: "op3", Amount: "1"}}},
		},
		Operators: []Operator{
			{Address: "op1", AVSs: []string{"avsA"}},
			{Address: "op2", AVSs: []string{"avsA", "avsB"}},
			{Address: "op3", AVSs: []string{"avsB"}},
		},
	}
	got := ContagionMatrix(g)
	if len(got) != 1 {
		t.Fatalf("want 1 pair, got %d: %+v", len(got), got)
	}
	o := got[0]
	if o.A != "cmETH" || o.B != "ezETH" {
		t.Fatalf("pair order wrong: %+v", o)
	}
	if len(o.SharedOperators) != 1 || o.SharedOperators[0] != "op2" {
		t.Fatalf("shared operators wrong: %+v", o.SharedOperators)
	}
	// avssOf(cmETH)={avsA,avsB}, avssOf(ezETH)={avsA,avsB} -> shared {avsA,avsB}
	if len(o.SharedAVSs) != 2 {
		t.Fatalf("shared avss wrong: %+v", o.SharedAVSs)
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./graph/ -run TestContagionMatrix` → FAIL undefined ContagionMatrix/Overlap

- [ ] **Step 3: Implement** (append to `graph/metrics.go`)

```go
// Overlap is the shared exposure between two LRTs.
type Overlap struct {
	A               string   `json:"a"`
	B               string   `json:"b"`
	SharedOperators []string `json:"shared_operators"`
	SharedAVSs      []string `json:"shared_avss"`
	Score           float64  `json:"score"` // Jaccard over AVS sets, 0..1
}

func setOf(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func intersect(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if b[k] {
			out = append(out, k)
		}
	}
	return sortedStrings(out)
}

// ContagionMatrix returns one Overlap per unordered LRT pair (A<B by symbol)
// that shares at least one operator. AVS sets are computed transitively through
// each LRT's operators.
func ContagionMatrix(g Graph) []Overlap {
	opAVS := make(map[string][]string, len(g.Operators))
	for _, op := range g.Operators {
		opAVS[op.Address] = op.AVSs
	}
	type lrtSets struct {
		sym  string
		ops  map[string]bool
		avss map[string]bool
	}
	var ls []lrtSets
	for _, l := range g.LRTs {
		ops := map[string]bool{}
		avss := map[string]bool{}
		for _, d := range l.Delegations {
			ops[d.Operator] = true
			for _, a := range opAVS[d.Operator] {
				avss[a] = true
			}
		}
		ls = append(ls, lrtSets{l.Symbol, ops, avss})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].sym < ls[j].sym })

	var out []Overlap
	for i := 0; i < len(ls); i++ {
		for j := i + 1; j < len(ls); j++ {
			sharedOps := intersect(ls[i].ops, ls[j].ops)
			if len(sharedOps) == 0 {
				continue
			}
			sharedAVS := intersect(ls[i].avss, ls[j].avss)
			union := setOf(append(sortedStrings(keys(ls[i].avss)), keys(ls[j].avss)...))
			score := 0.0
			if len(union) > 0 {
				score = float64(len(sharedAVS)) / float64(len(union))
			}
			out = append(out, Overlap{ls[i].sym, ls[j].sym, sharedOps, sharedAVS, score})
		}
	}
	return out
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./graph/...` → PASS
- [ ] **Step 5: Commit** — `git add graph/ && git commit -m "feat(graph): contagion matrix"`

---

## Task 4: Invariant checks

**Files:**
- Create: `graph/invariants.go`, `graph/invariants_test.go`

- [ ] **Step 1: Write the failing test**

`graph/invariants_test.go`:
```go
package graph

import "testing"

func TestCheckInvariants(t *testing.T) {
	g := Graph{
		LRTs: []LRT{
			{Symbol: "good", Restaked: "30", Delegations: []Delegation{{Operator: "op1", Amount: "20"}, {Operator: "op1", Amount: "10"}}},
			{Symbol: "bad-recon", Restaked: "100", Delegations: []Delegation{{Operator: "op1", Amount: "10"}}},
			{Symbol: "unknown-op", Restaked: "5", Delegations: []Delegation{{Operator: "ghost", Amount: "5"}}},
		},
		Operators: []Operator{{Address: "op1"}},
	}
	res := CheckInvariants(g)
	get := func(name, lrt string) InvariantResult {
		for _, r := range res {
			if r.Name == name && r.LRT == lrt {
				return r
			}
		}
		t.Fatalf("missing invariant %s/%s", name, lrt)
		return InvariantResult{}
	}
	if !get("delegations_reconcile_restaked", "good").OK {
		t.Error("good LRT should reconcile")
	}
	if get("delegations_reconcile_restaked", "bad-recon").OK {
		t.Error("bad-recon should fail reconciliation")
	}
	if get("delegations_reference_known_operators", "unknown-op").OK {
		t.Error("unknown-op should fail known-operator check")
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./graph/ -run TestCheckInvariants` → FAIL undefined CheckInvariants

- [ ] **Step 3: Implement**

`graph/invariants.go`:
```go
package graph

import (
	"fmt"
	"math/big"
)

// InvariantResult is one consistency check outcome for an LRT.
type InvariantResult struct {
	Name   string `json:"name"`
	LRT    string `json:"lrt"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// reconcileTolerance: relative tolerance for delegations-vs-restaked (0.5%).
const reconcileTolerance = 0.005

// CheckInvariants validates internal consistency of the graph.
func CheckInvariants(g Graph) []InvariantResult {
	known := map[string]bool{}
	for _, op := range g.Operators {
		known[op.Address] = true
	}
	var out []InvariantResult
	for _, l := range g.LRTs {
		// 1) delegations reconcile to declared restaked total
		sum := big.NewFloat(0)
		for _, d := range l.Delegations {
			sum.Add(sum, parseAmount(d.Amount))
		}
		restaked := parseAmount(l.Restaked)
		ok, detail := withinTolerance(sum, restaked)
		out = append(out, InvariantResult{"delegations_reconcile_restaked", l.Symbol, ok, detail})

		// 2) every delegation operator is known
		var missing []string
		for _, d := range l.Delegations {
			if !known[d.Operator] {
				missing = append(missing, d.Operator)
			}
		}
		out = append(out, InvariantResult{
			"delegations_reference_known_operators", l.Symbol, len(missing) == 0,
			fmt.Sprintf("missing operators: %v", sortedStrings(missing)),
		})
	}
	return out
}

func withinTolerance(sum, target *big.Float) (bool, string) {
	if target.Sign() == 0 {
		if sum.Sign() == 0 {
			return true, "both zero"
		}
		return false, "restaked is zero but delegations are not"
	}
	diff := new(big.Float).Sub(sum, target)
	diff.Abs(diff)
	rel, _ := new(big.Float).Quo(diff, target).Float64()
	return rel <= reconcileTolerance, fmt.Sprintf("rel diff %.4f%%", rel*100)
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./graph/...` → PASS
- [ ] **Step 5: Commit** — `git add graph/invariants.go graph/invariants_test.go && git commit -m "feat(graph): invariant checks"`

---

## Task 5: Snapshot build + canonical JSON

**Files:**
- Create: `snapshot/snapshot.go`, `snapshot/snapshot_test.go`

- [ ] **Step 1: Write the failing test**

`snapshot/snapshot_test.go`:
```go
package snapshot

import (
	"path/filepath"
	"testing"

	"github.com/prajjwalchittori/restake-xray/graph"
)

func sampleGraph() graph.Graph {
	return graph.Graph{
		Block: 5, Timestamp: 7, Protocol: "eigenlayer",
		LRTs: []graph.LRT{
			{Symbol: "ezETH", Restaked: "2", Delegations: []graph.Delegation{{Operator: "op1", Amount: "2"}}},
			{Symbol: "cmETH", Restaked: "2", Delegations: []graph.Delegation{{Operator: "op1", Amount: "1"}, {Operator: "op2", Amount: "1"}}},
		},
		Operators: []graph.Operator{{Address: "op2", AVSs: []string{"avs1"}}, {Address: "op1", AVSs: []string{"avs1"}}},
		AVSs:      []graph.AVS{{Address: "avs1"}},
	}
}

func TestBuildCanonicalizesAndComputes(t *testing.T) {
	s := Build(sampleGraph())
	if s.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version %d", s.SchemaVersion)
	}
	if s.Graph.LRTs[0].Symbol != "cmETH" { // sorted by symbol
		t.Fatalf("LRTs not sorted: %v", s.Graph.LRTs[0].Symbol)
	}
	if s.Graph.Operators[0].Address != "op1" { // sorted by address
		t.Fatalf("operators not sorted: %v", s.Graph.Operators[0].Address)
	}
	if _, ok := s.Concentration["cmETH"]; !ok {
		t.Fatal("missing concentration for cmETH")
	}
	if len(s.Invariants) == 0 {
		t.Fatal("expected invariants")
	}
}

func TestJSONStableAndRoundTrips(t *testing.T) {
	s := Build(sampleGraph())
	b1, err := s.JSON()
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := Build(sampleGraph()).JSON()
	if string(b1) != string(b2) {
		t.Fatal("JSON output not deterministic")
	}
	p := filepath.Join(t.TempDir(), "snap.json")
	if err := Write(p, s); err != nil {
		t.Fatal(err)
	}
	got, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Block != s.Block || len(got.Graph.LRTs) != len(s.Graph.LRTs) {
		t.Fatal("round-trip mismatch")
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./snapshot/...` → FAIL undefined Build/Snapshot/...

- [ ] **Step 3: Implement**

`snapshot/snapshot.go`:
```go
// Package snapshot serializes a computed exposure graph plus metrics and
// invariants into a canonical, deterministic JSON dataset.
package snapshot

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/prajjwalchittori/restake-xray/graph"
)

// SchemaVersion is bumped on any breaking change to the JSON shape.
const SchemaVersion = 1

// Snapshot is the published dataset unit.
type Snapshot struct {
	SchemaVersion int                     `json:"schema_version"`
	Protocol      string                  `json:"protocol"`
	Block         uint64                  `json:"block"`
	Timestamp     int64                   `json:"timestamp"`
	Graph         graph.Graph             `json:"graph"`
	Concentration map[string]float64      `json:"concentration"`
	Contagion     []graph.Overlap         `json:"contagion"`
	Invariants    []graph.InvariantResult `json:"invariants"`
}

// Build computes metrics + invariants and canonicalizes ordering so JSON output
// is deterministic (stable git diffs).
func Build(g graph.Graph) Snapshot {
	sort.Slice(g.LRTs, func(i, j int) bool { return g.LRTs[i].Symbol < g.LRTs[j].Symbol })
	sort.Slice(g.Operators, func(i, j int) bool { return g.Operators[i].Address < g.Operators[j].Address })
	sort.Slice(g.AVSs, func(i, j int) bool { return g.AVSs[i].Address < g.AVSs[j].Address })

	conc := map[string]float64{}
	for _, l := range g.LRTs {
		conc[l.Symbol] = graph.OperatorConcentration(l)
	}
	return Snapshot{
		SchemaVersion: SchemaVersion,
		Protocol:      g.Protocol,
		Block:         g.Block,
		Timestamp:     g.Timestamp,
		Graph:         g,
		Concentration: conc,
		Contagion:     graph.ContagionMatrix(g),
		Invariants:    graph.CheckInvariants(g),
	}
}

// JSON returns indented, deterministic JSON.
func (s Snapshot) JSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// Write writes the snapshot JSON to path.
func Write(path string, s Snapshot) error {
	b, err := s.JSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// Read loads a snapshot JSON from path.
func Read(path string) (Snapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var s Snapshot
	return s, json.Unmarshal(b, &s)
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./snapshot/...` → PASS
- [ ] **Step 5: Commit** — `git add snapshot/ && git commit -m "feat(snapshot): canonical dataset build + io"`

---

## Task 6: Terminal rendering

**Files:**
- Create: `render/render.go`, `render/render_test.go`

- [ ] **Step 1: Write the failing test**

`render/render_test.go`:
```go
package render

import (
	"strings"
	"testing"

	"github.com/prajjwalchittori/restake-xray/snapshot"
	"github.com/prajjwalchittori/restake-xray/graph"
)

func TestSummaryMentionsLRTAndConcentration(t *testing.T) {
	s := snapshot.Build(graph.Graph{
		Protocol: "eigenlayer", Block: 9,
		LRTs:      []graph.LRT{{Symbol: "cmETH", Restaked: "1", Delegations: []graph.Delegation{{Operator: "op1", Amount: "1"}}}},
		Operators: []graph.Operator{{Address: "op1", Name: "Op One"}},
	})
	out := Summary(s)
	if !strings.Contains(out, "cmETH") || !strings.Contains(out, "block 9") {
		t.Fatalf("summary missing fields:\n%s", out)
	}
}

func TestContagionRenderShowsPair(t *testing.T) {
	s := snapshot.Build(graph.Graph{
		LRTs: []graph.LRT{
			{Symbol: "cmETH", Delegations: []graph.Delegation{{Operator: "op1", Amount: "1"}}},
			{Symbol: "ezETH", Delegations: []graph.Delegation{{Operator: "op1", Amount: "1"}}},
		},
		Operators: []graph.Operator{{Address: "op1", Name: "Op One", AVSs: []string{"avs1"}}},
	})
	out := Contagion(s)
	if !strings.Contains(out, "cmETH") || !strings.Contains(out, "ezETH") {
		t.Fatalf("contagion missing pair:\n%s", out)
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./render/...` → FAIL undefined Summary/Contagion

- [ ] **Step 3: Implement**

`render/render.go`:
```go
// Package render turns snapshots into human-readable terminal output.
package render

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/prajjwalchittori/restake-xray/snapshot"
)

// Summary renders a high-level overview of the snapshot.
func Summary(s snapshot.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "restake-xray — %s @ block %d\n", s.Protocol, s.Block)
	fmt.Fprintf(&b, "LRTs: %d  Operators: %d  AVSs: %d\n\n", len(s.Graph.LRTs), len(s.Graph.Operators), len(s.Graph.AVSs))
	w := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "LRT\tRESTAKED\tOPERATORS\tCONCENTRATION")
	for _, l := range s.Graph.LRTs {
		fmt.Fprintf(w, "%s\t%s\t%d\t%.3f\n", l.Symbol, l.Restaked, len(l.Delegations), s.Concentration[l.Symbol])
	}
	w.Flush()
	failed := 0
	for _, inv := range s.Invariants {
		if !inv.OK {
			failed++
		}
	}
	fmt.Fprintf(&b, "\nInvariants: %d/%d passing\n", len(s.Invariants)-failed, len(s.Invariants))
	return b.String()
}

// Contagion renders the shared-exposure pairs.
func Contagion(s snapshot.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contagion (shared exposure between LRTs):\n\n")
	if len(s.Contagion) == 0 {
		fmt.Fprintln(&b, "  (no shared operators found)")
		return b.String()
	}
	w := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "A\tB\tSHARED OPS\tSHARED AVSS\tSCORE")
	for _, o := range s.Contagion {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%.3f\n", o.A, o.B, len(o.SharedOperators), len(o.SharedAVSs), o.Score)
	}
	w.Flush()
	return b.String()
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./render/...` → PASS
- [ ] **Step 5: Commit** — `git add render/ && git commit -m "feat(render): terminal summary + contagion tables"`

---

## Task 7: Adapter interface + sample (offline) adapter

**Files:**
- Create: `adapter/adapter.go`, `adapter/sample/sample.go`, `adapter/sample/sample_test.go`, `testdata/sample-graph.json`

- [ ] **Step 1: Write the failing test**

`adapter/sample/sample_test.go`:
```go
package sample

import (
	"context"
	"testing"
)

func TestSampleLoadsFixture(t *testing.T) {
	a, err := NewFromFile("../../testdata/sample-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	g, err := a.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if a.Name() == "" || len(g.LRTs) == 0 {
		t.Fatalf("empty sample graph: name=%q lrts=%d", a.Name(), len(g.LRTs))
	}
}
```

- [ ] **Step 2: Create the fixture** `testdata/sample-graph.json` (realistic shape; demo/dev only — never published as truth):
```json
{
  "block": 20000000,
  "timestamp": 1717400000,
  "protocol": "eigenlayer",
  "lrts": [
    {"address":"0xcmETH","symbol":"cmETH","restaked":"1000",
     "collateral":[{"token":"0xbeacon","symbol":"ETH","amount":"1000","decimals":18}],
     "delegations":[{"operator":"0xop1","amount":"600"},{"operator":"0xop2","amount":"400"}]},
    {"address":"0xezETH","symbol":"ezETH","restaked":"800",
     "collateral":[{"token":"0xbeacon","symbol":"ETH","amount":"800","decimals":18}],
     "delegations":[{"operator":"0xop2","amount":"500"},{"operator":"0xop3","amount":"300"}]}
  ],
  "operators": [
    {"address":"0xop1","name":"","avss":["0xavsDA"]},
    {"address":"0xop2","name":"","avss":["0xavsDA","0xavsORACLE"]},
    {"address":"0xop3","name":"","avss":["0xavsORACLE"]}
  ],
  "avss": [
    {"address":"0xavsDA","name":""},
    {"address":"0xavsORACLE","name":""}
  ]
}
```

- [ ] **Step 3: Run to verify it fails** — `go test ./adapter/sample/...` → FAIL undefined NewFromFile

- [ ] **Step 4: Implement**

`adapter/adapter.go`:
```go
// Package adapter defines the port each restaking protocol implements to
// populate the exposure graph.
package adapter

import (
	"context"

	"github.com/prajjwalchittori/restake-xray/graph"
)

// Protocol populates the exposure graph for one restaking protocol.
type Protocol interface {
	Name() string
	Snapshot(ctx context.Context) (graph.Graph, error)
}
```

`adapter/sample/sample.go`:
```go
// Package sample is an offline adapter that serves a graph from a JSON fixture.
package sample

import (
	"context"
	"encoding/json"
	"os"

	"github.com/prajjwalchittori/restake-xray/graph"
)

// Sample is an adapter.Protocol backed by a static JSON graph.
type Sample struct{ g graph.Graph }

// NewFromFile loads a graph fixture.
func NewFromFile(path string) (*Sample, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var g graph.Graph
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, err
	}
	return &Sample{g: g}, nil
}

func (s *Sample) Name() string { return s.g.Protocol }

func (s *Sample) Snapshot(_ context.Context) (graph.Graph, error) { return s.g, nil }
```

- [ ] **Step 5: Run to verify it passes** — `go test ./adapter/...` → PASS
- [ ] **Step 6: Commit** — `git add adapter/ testdata/sample-graph.json && git commit -m "feat(adapter): protocol port + offline sample adapter"`

---

## Task 8: Labels (off-chain enrichment)

**Files:**
- Create: `labels/labels.go`, `labels/labels_test.go`, `testdata/labels.json`

- [ ] **Step 1: Write the failing test**

`labels/labels_test.go`:
```go
package labels

import "testing"

func TestStaticLabels(t *testing.T) {
	p, err := LoadStatic("../testdata/labels.json")
	if err != nil {
		t.Fatal(err)
	}
	if p.OperatorName("0xop1") != "P2P" {
		t.Fatalf("operator name: %q", p.OperatorName("0xop1"))
	}
	if sym, dec, ok := p.TokenSymbol("0xbeacon"); !ok || sym != "ETH" || dec != 18 {
		t.Fatalf("token: %q %d %v", sym, dec, ok)
	}
}

func TestNoopLabels(t *testing.T) {
	var p Noop
	if p.OperatorName("x") != "" {
		t.Fatal("noop should return empty")
	}
}
```

- [ ] **Step 2: Create** `testdata/labels.json`:
```json
{
  "operators": {"0xop1": "P2P", "0xop2": "Luganodes"},
  "avss": {"0xavsDA": "EigenDA"},
  "tokens": {"0xbeacon": {"symbol": "ETH", "decimals": 18}}
}
```

- [ ] **Step 3: Run to verify it fails** — `go test ./labels/...` → FAIL undefined LoadStatic/Noop

- [ ] **Step 4: Implement**

`labels/labels.go`:
```go
// Package labels provides off-chain human-readable metadata for addresses.
package labels

import (
	"encoding/json"
	"os"
)

// Provider resolves addresses to names/symbols. Missing -> empty/!ok.
type Provider interface {
	OperatorName(addr string) string
	AVSName(addr string) string
	TokenSymbol(addr string) (symbol string, decimals uint8, ok bool)
}

type token struct {
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

// Static is a file-backed Provider.
type Static struct {
	Operators map[string]string `json:"operators"`
	AVSs      map[string]string `json:"avss"`
	Tokens    map[string]token  `json:"tokens"`
}

// LoadStatic reads a labels JSON file.
func LoadStatic(path string) (*Static, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Static
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Static) OperatorName(a string) string { return s.Operators[a] }
func (s *Static) AVSName(a string) string      { return s.AVSs[a] }
func (s *Static) TokenSymbol(a string) (string, uint8, bool) {
	t, ok := s.Tokens[a]
	return t.Symbol, t.Decimals, ok
}

// Noop returns no labels.
type Noop struct{}

func (Noop) OperatorName(string) string              { return "" }
func (Noop) AVSName(string) string                   { return "" }
func (Noop) TokenSymbol(string) (string, uint8, bool) { return "", 0, false }
```

- [ ] **Step 5: Run to verify it passes** — `go test ./labels/...` → PASS
- [ ] **Step 6: Commit** — `git add labels/ testdata/labels.json && git commit -m "feat(labels): static + noop providers"`

---

## Task 9: Engine (orchestrate adapters + labels → snapshot)

**Files:**
- Create: `engine/engine.go`, `engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

`engine/engine_test.go`:
```go
package engine

import (
	"context"
	"testing"

	"github.com/prajjwalchittori/restake-xray/adapter/sample"
	"github.com/prajjwalchittori/restake-xray/labels"
)

func TestEngineSnapshotAppliesLabels(t *testing.T) {
	a, err := sample.NewFromFile("../testdata/sample-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	lp, err := labels.LoadStatic("../testdata/labels.json")
	if err != nil {
		t.Fatal(err)
	}
	e := New([]interface {
		Name() string
		Snapshot(context.Context) (graphSnap, error)
	}{}, lp) // placeholder replaced below

	_ = e
	// Use the real constructor signature:
	eng := NewEngine([]adapterProto{a}, lp)
	s, err := eng.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var op1Named bool
	for _, op := range s.Graph.Operators {
		if op.Address == "0xop1" && op.Name == "P2P" {
			op1Named = true
		}
	}
	if !op1Named {
		t.Fatal("expected operator 0xop1 labeled P2P")
	}
}
```

> Note: simplify the test to match the final API before running — the canonical form is:
```go
package engine

import (
	"context"
	"testing"

	"github.com/prajjwalchittori/restake-xray/adapter"
	"github.com/prajjwalchittori/restake-xray/adapter/sample"
	"github.com/prajjwalchittori/restake-xray/labels"
)

func TestEngineSnapshotAppliesLabels(t *testing.T) {
	a, _ := sample.NewFromFile("../testdata/sample-graph.json")
	lp, _ := labels.LoadStatic("../testdata/labels.json")
	eng := New([]adapter.Protocol{a}, lp)
	s, err := eng.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	named := false
	for _, op := range s.Graph.Operators {
		if op.Address == "0xop1" && op.Name == "P2P" {
			named = true
		}
	}
	if !named {
		t.Fatal("expected operator 0xop1 labeled P2P")
	}
}
```
Use the canonical form; delete the placeholder block above.

- [ ] **Step 2: Run to verify it fails** — `go test ./engine/...` → FAIL undefined New

- [ ] **Step 3: Implement**

`engine/engine.go`:
```go
// Package engine orchestrates adapters and label enrichment into a snapshot.
package engine

import (
	"context"

	"github.com/prajjwalchittori/restake-xray/adapter"
	"github.com/prajjwalchittori/restake-xray/graph"
	"github.com/prajjwalchittori/restake-xray/labels"
	"github.com/prajjwalchittori/restake-xray/snapshot"
)

// Engine runs one or more protocol adapters and enriches the merged graph.
type Engine struct {
	adapters []adapter.Protocol
	labels   labels.Provider
}

// New builds an Engine.
func New(adapters []adapter.Protocol, lp labels.Provider) *Engine {
	if lp == nil {
		lp = labels.Noop{}
	}
	return &Engine{adapters: adapters, labels: lp}
}

// Snapshot runs all adapters, merges and enriches, and builds the dataset.
func (e *Engine) Snapshot(ctx context.Context) (snapshot.Snapshot, error) {
	var merged graph.Graph
	opSeen := map[string]bool{}
	avsSeen := map[string]bool{}
	for _, a := range e.adapters {
		g, err := a.Snapshot(ctx)
		if err != nil {
			return snapshot.Snapshot{}, err
		}
		if merged.Protocol == "" {
			merged.Protocol = g.Protocol
		}
		if g.Block > merged.Block {
			merged.Block = g.Block
		}
		if g.Timestamp > merged.Timestamp {
			merged.Timestamp = g.Timestamp
		}
		merged.LRTs = append(merged.LRTs, g.LRTs...)
		for _, op := range g.Operators {
			if !opSeen[op.Address] {
				opSeen[op.Address] = true
				merged.Operators = append(merged.Operators, op)
			}
		}
		for _, av := range g.AVSs {
			if !avsSeen[av.Address] {
				avsSeen[av.Address] = true
				merged.AVSs = append(merged.AVSs, av)
			}
		}
	}
	e.enrich(&merged)
	return snapshot.Build(merged), nil
}

func (e *Engine) enrich(g *graph.Graph) {
	for i := range g.Operators {
		if g.Operators[i].Name == "" {
			g.Operators[i].Name = e.labels.OperatorName(g.Operators[i].Address)
		}
	}
	for i := range g.AVSs {
		if g.AVSs[i].Name == "" {
			g.AVSs[i].Name = e.labels.AVSName(g.AVSs[i].Address)
		}
	}
	for i := range g.LRTs {
		for j := range g.LRTs[i].Collateral {
			c := &g.LRTs[i].Collateral[j]
			if c.Symbol == "" {
				if sym, dec, ok := e.labels.TokenSymbol(c.Token); ok {
					c.Symbol, c.Decimals = sym, dec
				}
			}
		}
	}
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./engine/...` → PASS
- [ ] **Step 5: Commit** — `git add engine/ && git commit -m "feat(engine): adapter orchestration + enrichment"`

---

## Task 10: CLI — `scan`, `lrt`, `contagion` (offline demoable slice 🎉)

**Files:**
- Create: `cmd/xray/main.go`

This is the first end-to-end demo: `make scan` prints the X-ray from the sample fixture and can write the dataset JSON.

- [ ] **Step 1: Implement the CLI** (dispatch + the three read commands)

`cmd/xray/main.go`:
```go
// Command xray is the restake-xray CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/prajjwalchittori/restake-xray/adapter"
	"github.com/prajjwalchittori/restake-xray/adapter/sample"
	"github.com/prajjwalchittori/restake-xray/engine"
	"github.com/prajjwalchittori/restake-xray/labels"
	"github.com/prajjwalchittori/restake-xray/render"
	"github.com/prajjwalchittori/restake-xray/snapshot"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "scan":
		cmdScan(os.Args[2:])
	case "lrt":
		cmdLRT(os.Args[2:])
	case "contagion":
		cmdContagion(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:]) // implemented in Task 12
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: xray <scan|lrt|contagion|serve> [flags]")
	os.Exit(2)
}

// buildSnapshot wires the offline sample adapter (live wiring added in Task 17).
func buildSnapshot(fixture, labelsPath string) (snapshot.Snapshot, error) {
	a, err := sample.NewFromFile(fixture)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	var lp labels.Provider = labels.Noop{}
	if labelsPath != "" {
		if s, err := labels.LoadStatic(labelsPath); err == nil {
			lp = s
		}
	}
	e := engine.New([]adapter.Protocol{a}, lp)
	return e.Snapshot(context.Background())
}

func cmdScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	fixture := fs.String("sample", "testdata/sample-graph.json", "graph fixture (offline)")
	labelsPath := fs.String("labels", "testdata/labels.json", "labels file")
	out := fs.String("out", "", "write dataset JSON to this path")
	fs.Parse(args)

	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	fmt.Print(render.Summary(s))
	if *out != "" {
		must(snapshot.Write(*out, s))
		fmt.Printf("\nwrote dataset -> %s\n", *out)
	}
}

func cmdContagion(args []string) {
	fs := flag.NewFlagSet("contagion", flag.ExitOnError)
	fixture := fs.String("sample", "testdata/sample-graph.json", "graph fixture")
	labelsPath := fs.String("labels", "testdata/labels.json", "labels file")
	fs.Parse(args)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	fmt.Print(render.Contagion(s))
}

func cmdLRT(args []string) {
	fs := flag.NewFlagSet("lrt", flag.ExitOnError)
	fixture := fs.String("sample", "testdata/sample-graph.json", "graph fixture")
	labelsPath := fs.String("labels", "testdata/labels.json", "labels file")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: xray lrt <SYMBOL>")
		os.Exit(2)
	}
	sym := fs.Arg(0)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	for _, l := range s.Graph.LRTs {
		if l.Symbol == sym {
			fmt.Printf("%s @ block %d\n  restaked: %s\n  concentration: %.3f\n  operators:\n", l.Symbol, s.Block, l.Restaked, s.Concentration[l.Symbol])
			for _, d := range l.Delegations {
				fmt.Printf("    %s  %s\n", d.Operator, d.Amount)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "LRT %q not found\n", sym)
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

> `cmdServe` is referenced here but implemented in Task 12. To keep this task compiling on its own, add a temporary stub at the bottom of `main.go`:
```go
func cmdServe(args []string) { fmt.Fprintln(os.Stderr, "serve: implemented in Task 12"); os.Exit(2) }
```
Remove the stub in Task 12 when the real `cmdServe` lands.

- [ ] **Step 2: Build & run the demo**

Run:
```bash
go build -o xray ./cmd/xray
./xray scan
./xray contagion
./xray lrt cmETH
```
Expected: a summary table listing cmETH/ezETH with concentration values, a contagion table showing the cmETH↔ezETH pair, and a per-LRT breakdown. **This is the demoable milestone.**

- [ ] **Step 3: Commit** — `git add cmd/ && git commit -m "feat(cli): scan/lrt/contagion over offline sample"`

---

## Task 11: Hosted API server

**Files:**
- Create: `api/api.go`, `api/api_test.go`

- [ ] **Step 1: Write the failing test**

`api/api_test.go`:
```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prajjwalchittori/restake-xray/graph"
	"github.com/prajjwalchittori/restake-xray/snapshot"
)

func testSnap() snapshot.Snapshot {
	return snapshot.Build(graph.Graph{
		Protocol: "eigenlayer", Block: 42,
		LRTs:      []graph.LRT{{Symbol: "cmETH", Restaked: "1", Delegations: []graph.Delegation{{Operator: "op1", Amount: "1"}}}},
		Operators: []graph.Operator{{Address: "op1", Name: "P2P", AVSs: []string{"avs1"}}},
		AVSs:      []graph.AVS{{Address: "avs1", Name: "EigenDA"}},
	})
}

func TestRoutes(t *testing.T) {
	srv := NewServer(func() (snapshot.Snapshot, error) { return testSnap(), nil })
	for _, path := range []string{"/health", "/lrts", "/operators", "/avs", "/contagion", "/lrt/cmETH/exposure"} {
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s -> %d", path, rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Fatalf("%s missing CORS header", path)
		}
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	var h map[string]any
	json.Unmarshal(rec.Body.Bytes(), &h)
	if h["block"].(float64) != 42 {
		t.Fatalf("health block: %v", h["block"])
	}
}

func TestUnknownLRT404(t *testing.T) {
	srv := NewServer(func() (snapshot.Snapshot, error) { return testSnap(), nil })
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/lrt/NOPE/exposure", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./api/...` → FAIL undefined NewServer

- [ ] **Step 3: Implement**

`api/api.go`:
```go
// Package api serves a read-only HTTP/JSON view of the latest snapshot.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/prajjwalchittori/restake-xray/snapshot"
)

// Loader returns the current snapshot to serve.
type Loader func() (snapshot.Snapshot, error)

// NewServer builds the HTTP handler. load is called per request so the served
// data refreshes when the underlying snapshot changes.
func NewServer(load Loader) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		s, err := load()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "protocol": s.Protocol, "block": s.Block, "timestamp": s.Timestamp})
	})
	mux.HandleFunc("GET /lrts", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Graph.LRTs })
	})
	mux.HandleFunc("GET /operators", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Graph.Operators })
	})
	mux.HandleFunc("GET /avs", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Graph.AVSs })
	})
	mux.HandleFunc("GET /contagion", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Contagion })
	})
	mux.HandleFunc("GET /lrt/{sym}/exposure", func(w http.ResponseWriter, r *http.Request) {
		s, err := load()
		if err != nil {
			writeErr(w, err)
			return
		}
		sym := r.PathValue("sym")
		for _, l := range s.Graph.LRTs {
			if l.Symbol == sym {
				writeJSON(w, map[string]any{"lrt": l, "concentration": s.Concentration[sym]})
				return
			}
		}
		http.Error(w, "lrt not found", http.StatusNotFound)
	})

	return cors(mux)
}

func serve(w http.ResponseWriter, load Loader, pick func(snapshot.Snapshot) any) {
	s, err := load()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, pick(s))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./api/...` → PASS
- [ ] **Step 5: Commit** — `git add api/ && git commit -m "feat(api): read-only JSON server"`

---

## Task 12: CLI `serve` command

**Files:**
- Modify: `cmd/xray/main.go` (replace the `cmdServe` stub)

- [ ] **Step 1: Replace the stub** with:
```go
func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dataPath := fs.String("data", "data/latest.json", "snapshot JSON to serve")
	addr := fs.String("addr", ":8080", "listen address")
	fs.Parse(args)

	load := func() (snapshot.Snapshot, error) { return snapshot.Read(*dataPath) }
	if _, err := load(); err != nil {
		must(fmt.Errorf("cannot read %s: %w", *dataPath, err))
	}
	fmt.Printf("serving %s on %s\n", *dataPath, *addr)
	must(http.ListenAndServe(*addr, api.NewServer(load)))
}
```
Add imports `net/http` and `github.com/prajjwalchittori/restake-xray/api` to `main.go`.

- [ ] **Step 2: Verify end-to-end**

Run:
```bash
go build -o xray ./cmd/xray
mkdir -p data && ./xray scan --out data/latest.json
( ./xray serve --addr :8080 & sleep 1; curl -s localhost:8080/contagion | head -c 200; curl -s localhost:8080/health; kill %1 )
```
Expected: dataset written, `/contagion` returns JSON, `/health` shows the block.

- [ ] **Step 3: Commit** — `git add cmd/ && git commit -m "feat(cli): serve hosted API from dataset"`

---

## Task 13: Static demo dashboard

**Files:**
- Create: `web/index.html`, `web/app.js`, `web/style.css`

- [ ] **Step 1: Implement a static page** that fetches `data/latest.json` (same-origin when served from repo root or GitHub Pages) and renders (a) an LRT table with concentration and (b) the contagion pairs.

`web/index.html`:
```html
<!doctype html><html><head><meta charset="utf-8">
<title>restake-xray</title><link rel="stylesheet" href="style.css"></head>
<body>
<h1>restake-xray</h1>
<p id="meta">loading…</p>
<h2>LRTs</h2><table id="lrts"><thead><tr><th>LRT</th><th>Restaked</th><th>Operators</th><th>Concentration</th></tr></thead><tbody></tbody></table>
<h2>Contagion</h2><table id="contagion"><thead><tr><th>A</th><th>B</th><th>Shared ops</th><th>Shared AVSs</th><th>Score</th></tr></thead><tbody></tbody></table>
<script src="app.js"></script></body></html>
```

`web/app.js`:
```js
const DATA_URL = (location.search.match(/[?&]data=([^&]+)/) || [,'../data/latest.json'])[1];
async function main() {
  const s = await (await fetch(DATA_URL)).json();
  document.getElementById('meta').textContent =
    `${s.protocol} @ block ${s.block} — ${s.graph.lrts.length} LRTs, ${s.graph.operators.length} operators`;
  const lrtBody = document.querySelector('#lrts tbody');
  for (const l of s.graph.lrts) {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${l.symbol}</td><td>${l.restaked}</td><td>${l.delegations.length}</td><td>${(s.concentration[l.symbol]||0).toFixed(3)}</td>`;
    lrtBody.appendChild(tr);
  }
  const cBody = document.querySelector('#contagion tbody');
  for (const o of (s.contagion||[])) {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${o.a}</td><td>${o.b}</td><td>${o.shared_operators.length}</td><td>${o.shared_avss.length}</td><td>${o.score.toFixed(3)}</td>`;
    cBody.appendChild(tr);
  }
}
main().catch(e => document.getElementById('meta').textContent = 'error: ' + e);
```

`web/style.css`:
```css
body{font:14px/1.5 system-ui,sans-serif;max-width:900px;margin:2rem auto;padding:0 1rem}
table{border-collapse:collapse;width:100%;margin:.5rem 0 1.5rem}
th,td{border:1px solid #ddd;padding:.4rem .6rem;text-align:left}
th{background:#f5f5f5}
```

- [ ] **Step 2: Verify visually**

Run:
```bash
./xray scan --out data/latest.json
( cd . && python3 -m http.server 8090 & sleep 1; curl -s "localhost:8090/web/index.html" | head -c 120; kill %1 )
```
Expected: HTML served; opening `http://localhost:8090/web/index.html` in a browser shows the tables populated from `data/latest.json`.

- [ ] **Step 3: Commit** — `git add web/ data/ && git commit -m "feat(web): static demo dashboard"`

---

## Task 14: EigenLayer adapter — Reader port + golden test (fixture)

**Files:**
- Create: `adapter/eigenlayer/eigenlayer.go`, `adapter/eigenlayer/eigenlayer_test.go`, `testdata/eigenlayer-pinned.json`

This builds the real adapter's pure orchestration against a recorded fixture so it's deterministic and offline. The live RPC wiring is Task 15.

- [ ] **Step 1: Write the failing test** (fixture reader → graph)

`adapter/eigenlayer/eigenlayer_test.go`:
```go
package eigenlayer

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// fixtureReader replays recorded reads for a pinned block.
type fixtureReader struct {
	Block    uint64                `json:"block"`
	Backings map[string]Backing    `json:"backings"`     // by LRT address
	OpAVSs   map[string][]string   `json:"operator_avss"`// by operator address
}

func loadFixture(t *testing.T, path string) *fixtureReader {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f fixtureReader
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return &f
}

func (f *fixtureReader) BlockNumber(context.Context) (uint64, error) { return f.Block, nil }
func (f *fixtureReader) LRTBacking(_ context.Context, c LRTConfig) (Backing, error) {
	return f.Backings[c.Address], nil
}
func (f *fixtureReader) OperatorAVSs(_ context.Context, op string) ([]string, error) {
	return f.OpAVSs[op], nil
}

func TestAdapterBuildsGraphFromReader(t *testing.T) {
	r := loadFixture(t, "../../testdata/eigenlayer-pinned.json")
	a := New(r, []LRTConfig{{Symbol: "cmETH", Address: "0xcmETH"}})
	g, err := a.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if g.Protocol != "eigenlayer" || g.Block == 0 {
		t.Fatalf("bad header: %+v", g)
	}
	var found bool
	for _, l := range g.LRTs {
		if l.Symbol == "cmETH" && len(l.Delegations) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("cmETH not built with delegations")
	}
	// operators referenced by delegations must appear in g.Operators with AVSs resolved
	if len(g.Operators) == 0 || len(g.AVSs) == 0 {
		t.Fatalf("operators/avss not populated: ops=%d avss=%d", len(g.Operators), len(g.AVSs))
	}
}
```

- [ ] **Step 2: Create the fixture** `testdata/eigenlayer-pinned.json` (hand-built, small, one LRT; replace with recorded mainnet reads in Task 15):
```json
{
  "block": 20000000,
  "backings": {
    "0xcmETH": {
      "restaked": "1000",
      "collateral": [{"token":"0xbeacon","symbol":"ETH","amount":"1000","decimals":18}],
      "delegations": [{"operator":"0xop1","amount":"600"},{"operator":"0xop2","amount":"400"}]
    }
  },
  "operator_avss": {
    "0xop1": ["0xavsDA"],
    "0xop2": ["0xavsDA","0xavsORACLE"]
  }
}
```

- [ ] **Step 3: Run to verify it fails** — `go test ./adapter/eigenlayer/...` → FAIL undefined New/Reader/LRTConfig/Backing

- [ ] **Step 4: Implement the adapter (pure orchestration)**

`adapter/eigenlayer/eigenlayer.go`:
```go
// Package eigenlayer builds the exposure graph for EigenLayer restaking.
package eigenlayer

import (
	"context"

	"github.com/prajjwalchittori/restake-xray/graph"
)

// Reader is the narrow on-chain data port the adapter needs. The live
// implementation (live.go) uses go-ethereum bindings + multicall; tests use a
// recorded fixture.
type Reader interface {
	BlockNumber(ctx context.Context) (uint64, error)
	LRTBacking(ctx context.Context, lrt LRTConfig) (Backing, error)
	OperatorAVSs(ctx context.Context, operator string) ([]string, error)
}

// LRTConfig identifies an LRT and the addresses needed to read its backing.
type LRTConfig struct {
	Symbol  string `json:"symbol"`
	Address string `json:"address"`
	// Additional protocol-specific addresses (strategies, restaking pools) live
	// here; the live Reader uses them. Kept opaque to the orchestration layer.
	Extra map[string]string `json:"extra,omitempty"`
}

// Backing is what a single LRT is backed by, as read on-chain.
type Backing struct {
	Restaked    string             `json:"restaked"`
	Collateral  []graph.Stake      `json:"collateral"`
	Delegations []graph.Delegation `json:"delegations"`
}

// Adapter implements adapter.Protocol for EigenLayer.
type Adapter struct {
	r    Reader
	lrts []LRTConfig
}

// New builds the adapter from a Reader and the LRT set to scan.
func New(r Reader, lrts []LRTConfig) *Adapter { return &Adapter{r: r, lrts: lrts} }

func (a *Adapter) Name() string { return "eigenlayer" }

// Snapshot reads each configured LRT's backing, then resolves the AVS set for
// every referenced operator.
func (a *Adapter) Snapshot(ctx context.Context) (graph.Graph, error) {
	block, err := a.r.BlockNumber(ctx)
	if err != nil {
		return graph.Graph{}, err
	}
	g := graph.Graph{Protocol: "eigenlayer", Block: block}

	opSet := map[string]bool{}
	for _, cfg := range a.lrts {
		b, err := a.r.LRTBacking(ctx, cfg)
		if err != nil {
			return graph.Graph{}, err
		}
		g.LRTs = append(g.LRTs, graph.LRT{
			Address: cfg.Address, Symbol: cfg.Symbol,
			Restaked: b.Restaked, Collateral: b.Collateral, Delegations: b.Delegations,
		})
		for _, d := range b.Delegations {
			opSet[d.Operator] = true
		}
	}

	avsSet := map[string]bool{}
	for op := range opSet {
		avss, err := a.r.OperatorAVSs(ctx, op)
		if err != nil {
			return graph.Graph{}, err
		}
		g.Operators = append(g.Operators, graph.Operator{Address: op, AVSs: avss})
		for _, av := range avss {
			avsSet[av] = true
		}
	}
	for av := range avsSet {
		g.AVSs = append(g.AVSs, graph.AVS{Address: av})
	}
	return g, nil
}
```

- [ ] **Step 5: Run to verify it passes** — `go test ./adapter/eigenlayer/...` → PASS
- [ ] **Step 6: Commit** — `git add adapter/eigenlayer/ testdata/eigenlayer-pinned.json && git commit -m "feat(eigenlayer): adapter orchestration + golden fixture test"`

---

## Task 15: Live EigenLayer Reader (real RPC) + the target LRT first

**Files:**
- Create: `adapter/eigenlayer/live.go`, `adapter/eigenlayer/bindings/` (generated), `adapter/eigenlayer/live_integration_test.go`

> This is the only task that needs network + real ABIs. Build it behind the `Reader` interface so nothing else changes. Start with **one LRT (cmETH)** as the first live LRT — the builder knows that architecture cold, which de-risks the hardest part (mapping an LRT to its operator delegations). Other LRTs are fast-follows that only add `LRTConfig`s + per-LRT backing logic.

- [ ] **Step 1: Add dependency**

Run: `go get github.com/ethereum/go-ethereum@latest`

- [ ] **Step 2: Generate bindings from official ABIs**

Install abigen and generate typed bindings for the contracts the live reader calls. Fetch ABIs from the official EigenLayer release (etherscan-verified ABIs / eigenlayer-contracts repo) for:
- `DelegationManager` — operator shares & delegation (`getOperatorShares(operator, strategies[])`, `delegatedTo(staker)`).
- `StrategyManager` / `AllocationManager` — strategy share accounting and operator-set / AVS allocation.
- `AVSDirectory` (and/or operator-sets registry) — operator → AVS registration.
- The relevant ERC20 / beacon-strategy contracts for collateral + decimals.

```bash
go install github.com/ethereum/go-ethereum/cmd/abigen@latest
# for each ABI file saved under adapter/eigenlayer/abi/<Name>.json:
abigen --abi adapter/eigenlayer/abi/DelegationManager.json \
  --pkg bindings --type DelegationManager \
  --out adapter/eigenlayer/bindings/delegationmanager.go
# repeat for StrategyManager/AllocationManager, AVSDirectory, ERC20
```

- [ ] **Step 3: Implement the live Reader**

`adapter/eigenlayer/live.go`:
```go
package eigenlayer

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/prajjwalchittori/restake-xray/graph"
	"github.com/prajjwalchittori/restake-xray/adapter/eigenlayer/bindings"
)

// Live reads EigenLayer state directly from an Ethereum RPC endpoint.
type Live struct {
	ec  *ethclient.Client
	dm  *bindings.DelegationManager
	avs *bindings.AVSDirectory
	// add StrategyManager/AllocationManager + ERC20 handles as needed
}

// NewLive dials rpcURL and binds the EigenLayer core contracts (mainnet addresses).
func NewLive(ctx context.Context, rpcURL string) (*Live, error) {
	ec, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	dm, err := bindings.NewDelegationManager(common.HexToAddress(addrDelegationManager), ec)
	if err != nil {
		return nil, err
	}
	avs, err := bindings.NewAVSDirectory(common.HexToAddress(addrAVSDirectory), ec)
	if err != nil {
		return nil, err
	}
	return &Live{ec: ec, dm: dm, avs: avs}, nil
}

// Mainnet core addresses (verify against the current EigenLayer deployment before shipping).
const (
	addrDelegationManager = "0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A"
	addrAVSDirectory      = "0x135DDa560e946695d6f155dACaFC6f1F25C1F5AF"
)

func (l *Live) BlockNumber(ctx context.Context) (uint64, error) {
	return l.ec.BlockNumber(ctx)
}

// LRTBacking reads one LRT's backing. The mapping from an LRT to its operator
// delegations is protocol-specific; implement per LRT (the target LRT first) using the
// addresses in cfg.Extra. Use call batching (multicall) for many operators.
func (l *Live) LRTBacking(ctx context.Context, cfg LRTConfig) (Backing, error) {
	switch cfg.Symbol {
	case "cmETH", "cmETH":
		return l.lrtBacking(ctx, cfg)
	default:
		return Backing{}, errUnsupportedLRT(cfg.Symbol)
	}
}

// OperatorAVSs resolves which AVSs an operator secures via the AVS/operator-set registry.
func (l *Live) OperatorAVSs(ctx context.Context, operator string) ([]string, error) {
	// Query the operator-set / AVSDirectory registry for `operator`'s registrations.
	// Return AVS addresses (lowercased hex).
	return queryOperatorAVSs(ctx, l.avs, common.HexToAddress(operator))
}

// --- helpers (implement with real contract calls) ---

func formatWei(x *big.Int) string { return x.String() }

// lrtBacking, queryOperatorAVSs, errUnsupportedLRT: implement using the target LRT's
// restaking architecture (the builder's domain). lrtBacking returns
// Restaked (total restaked ETH-equiv), Collateral, and per-operator Delegations.
// Convert big.Int amounts via formatWei. Keep collateral symbol/decimals empty;
// the labels layer fills them.
```

> The bodies of `lrtBacking`, `queryOperatorAVSs`, and `errUnsupportedLRT` are implemented against the actual contracts during this task — they are the the target LRT-specific reads. This is the deliberate seam where insider knowledge does the work; everything upstream is already tested.

- [ ] **Step 4: Wire live into the CLI** (add a `--rpc` flag to `scan`/dataset path)

In `cmd/xray/main.go`, add an alternate builder used when `--rpc` is set:
```go
// in cmdScan: add fs.String("rpc", "", "Ethereum RPC URL (live mode)")
// and a fs.String("lrts", "configs/lrts.json", "live LRT configs")
// when *rpc != "": build via eigenlayer.NewLive + eigenlayer.New(reader, cfgs)
//                  through engine.New(...) instead of the sample adapter.
```
Add `configs/lrts.json` with the the target LRT `LRTConfig` (symbol, address, Extra addresses).

- [ ] **Step 5: Integration test (guarded)**

`adapter/eigenlayer/live_integration_test.go`:
```go
package eigenlayer

import (
	"context"
	"os"
	"testing"
)

func TestLiveLRT(t *testing.T) {
	rpc := os.Getenv("RPC_URL")
	if rpc == "" {
		t.Skip("set RPC_URL to run live integration test")
	}
	l, err := NewLive(context.Background(), rpc)
	if err != nil {
		t.Fatal(err)
	}
	a := New(l, []LRTConfig{{Symbol: "cmETH", Address: "0x35fA164735182de50811E8e2E824cFb9B6118ac2"}})
	g, err := a.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(g.LRTs) == 0 || g.LRTs[0].Restaked == "" {
		t.Fatal("live cmETH backing empty")
	}
}
```

- [ ] **Step 6: Record fixture from live** — once the live read works, dump the reads at a pinned block into `testdata/eigenlayer-pinned.json` so the Task 14 golden test runs against real recorded data. Hand-verify cmETH's numbers once against Etherscan.

- [ ] **Step 7: Run tests**

Run: `go test ./...` (live test skips without RPC_URL) → PASS
Run (manual, with a node): `RPC_URL=<https-rpc> go test ./adapter/eigenlayer/ -run TestLiveLRT -v`

- [ ] **Step 8: Commit** — `git add adapter/eigenlayer/ cmd/ configs/ go.mod go.sum && git commit -m "feat(eigenlayer): live RPC reader (the target LRT first) + integration test"`

---

## Task 16: Snapshot job + README + launch polish

**Files:**
- Create: `scripts/snapshot.sh`, `README.md`
- Modify: `Makefile`

- [ ] **Step 1: Snapshot/dataset script**

`scripts/snapshot.sh`:
```bash
#!/usr/bin/env bash
# Regenerate the published dataset from live data and stage it.
set -euo pipefail
cd "$(dirname "$0")/.."
: "${RPC_URL:?set RPC_URL}"
go build -o xray ./cmd/xray
./xray scan --rpc "$RPC_URL" --lrts configs/lrts.json --labels testdata/labels.json --out data/latest.json
cp data/latest.json "data/snapshot-$(date -u +%Y%m%dT%H%M%SZ).json"
echo "dataset updated; commit with ./commit-and-push.sh"
```
Make executable: `chmod +x scripts/snapshot.sh`. (Schedule later via cron/systemd; out of v1 scope to automate.)

- [ ] **Step 2: README** — lead with the one-liner, a `go install` line, the contagion screenshot, the public API base URL, the "we assert our own numbers reconcile" invariant note, and a "supported protocols: EigenLayer (more via the `adapter.Protocol` interface — PRs welcome)" section. Include an `Add an adapter` subsection pointing at `adapter/adapter.go` to invite contributors.

- [ ] **Step 3: Makefile targets** — add:
```make
snapshot-live: ; ./scripts/snapshot.sh
```

- [ ] **Step 4: Full test + build sanity**

Run: `go test ./... && go build -o xray ./cmd/xray && ./xray scan && echo OK`
Expected: all tests pass, binary builds, sample scan prints, ends `OK`.

- [ ] **Step 5: Commit** — `git add README.md scripts/ Makefile && git commit -m "docs+ops: snapshot script, README, launch polish"`

---

## Self-review (completed by plan author)

**Spec coverage:**
- Exposure graph model → Tasks 1, 14. Composition/concentration/contagion → Tasks 2–3. Invariants → Task 4. ✓
- Hybrid data (on-chain + labels) → Tasks 8 (labels), 14–15 (on-chain). ✓
- Live snapshot depth (no historical) → snapshot is point-in-time; historical explicitly omitted. ✓
- CLI → Tasks 10, 12; library surface → all packages exported + `engine.New`; git-committed dataset → Tasks 12/16; hosted API → Tasks 11–12; static dashboard → Task 13. ✓
- EigenLayer-deep + adapter interface for fast-follows → Tasks 7, 14; anti-cyclical interface → `adapter.Protocol`. ✓
- Correctness/golden tests at pinned block → Tasks 14–15; invariant assertions in output → Tasks 4 + render/api surfacing. ✓
- v1 cut list (no risk scoring / no slashing sim / no other protocols / prices optional) → respected; prices only via optional labels. ✓
- Demoable-early sequencing → Task 10 is a full offline end-to-end demo before any network work. ✓

**Placeholder scan:** Task 15 intentionally defers the target LRT-specific contract-read bodies to implementation time (real ABIs/addresses must be fetched, not invented) — the `Reader` seam, signatures, and tests around it are fully specified, so this is a bounded, well-defined interface, not a vague placeholder. The mainnet addresses are flagged "verify against current deployment." All other tasks contain complete code.

**Type consistency:** `graph.*` types, `Backing`/`LRTConfig`/`Reader`, `snapshot.Snapshot` fields, and `api`/`render` consumers use consistent names and signatures across tasks. The Task 9 test shows a placeholder block then the canonical block — instruction says use the canonical one and delete the placeholder.
