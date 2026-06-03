package graph

import (
	"strings"
	"testing"
)

func TestDOT(t *testing.T) {
	g := Graph{
		LRTs:      []LRT{{Symbol: "cmETH", Delegations: []Delegation{{Operator: "op1", Amount: "5"}}}},
		Operators: []Operator{{Address: "op1", Name: "P2P", AVSs: []string{"avs1"}}},
		AVSs:      []AVS{{Address: "avs1", Name: "EigenDA"}},
	}
	out := DOT(g)
	for _, want := range []string{"digraph restake_xray", "LRT: cmETH", "Op: P2P", "AVS: EigenDA", "->"} {
		if !strings.Contains(out, want) {
			t.Errorf("DOT missing %q:\n%s", want, out)
		}
	}
	// deterministic
	if DOT(g) != out {
		t.Error("DOT not deterministic")
	}
}
