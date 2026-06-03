package graph

import "testing"

func TestDiffGraphs(t *testing.T) {
	a := Graph{
		Block:     100,
		LRTs:      []LRT{{Symbol: "cmETH", Delegations: []Delegation{{Operator: "op1", Amount: "1"}}}},
		Operators: []Operator{{Address: "op1"}},
		AVSs:      []AVS{{Address: "avs1"}},
	}
	b := Graph{
		Block: 200,
		LRTs: []LRT{
			{Symbol: "cmETH", Delegations: []Delegation{{Operator: "op1", Amount: "1"}, {Operator: "op2", Amount: "1"}}},
			{Symbol: "ezETH", Delegations: []Delegation{{Operator: "op2", Amount: "1"}}},
		},
		Operators: []Operator{{Address: "op1"}, {Address: "op2"}},
		AVSs:      []AVS{{Address: "avs1"}, {Address: "avs2"}},
	}
	d := DiffGraphs(a, b)
	if d.FromBlock != 100 || d.ToBlock != 200 {
		t.Fatalf("blocks: %+v", d)
	}
	if len(d.AddedLRTs) != 1 || d.AddedLRTs[0] != "ezETH" {
		t.Fatalf("added lrts: %v", d.AddedLRTs)
	}
	if len(d.AddedOperators) != 1 || d.AddedOperators[0] != "op2" {
		t.Fatalf("added operators: %v", d.AddedOperators)
	}
	if len(d.AddedAVSs) != 1 || d.AddedAVSs[0] != "avs2" {
		t.Fatalf("added avss: %v", d.AddedAVSs)
	}
	// cmETH went from 1.0 (single op) to 0.5 (even split) -> delta -0.5
	var found bool
	for _, c := range d.Concentration {
		if c.LRT == "cmETH" {
			found = true
			if c.From != 1.0 || c.To != 0.5 {
				t.Fatalf("cmETH conc change wrong: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("expected cmETH concentration change")
	}
}
