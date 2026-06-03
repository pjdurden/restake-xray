package render

import (
	"strings"
	"testing"

	"github.com/pjdurden/restake-xray/graph"
	"github.com/pjdurden/restake-xray/snapshot"
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
