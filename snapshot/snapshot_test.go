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
