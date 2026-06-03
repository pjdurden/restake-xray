// Package eigenlayer builds the exposure graph for EigenLayer restaking.
package eigenlayer

import (
	"context"

	"github.com/pjdurden/restake-xray/graph"
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
	// Extra carries protocol-specific addresses (strategies, restaking pools)
	// used by the live Reader. Opaque to the orchestration layer.
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
