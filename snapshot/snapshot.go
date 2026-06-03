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
const SchemaVersion = 2

// Systemic holds ecosystem-wide single-point-of-failure rankings.
type Systemic struct {
	Operators []graph.SystemicOperator `json:"operators"`
	AVSs      []graph.SystemicAVS      `json:"avss"`
}

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
	Systemic      Systemic                `json:"systemic"`
	Warnings      []graph.Warning         `json:"warnings"`
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
		Systemic: Systemic{
			Operators: graph.SystemicOperators(g),
			AVSs:      graph.SystemicAVSs(g),
		},
		Warnings: graph.Warnings(g),
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
