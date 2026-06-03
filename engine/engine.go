// Package engine orchestrates adapters and label enrichment into a snapshot.
package engine

import (
	"context"

	"github.com/pjdurden/restake-xray/adapter"
	"github.com/pjdurden/restake-xray/graph"
	"github.com/pjdurden/restake-xray/labels"
	"github.com/pjdurden/restake-xray/snapshot"
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
