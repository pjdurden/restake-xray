package graph

import "testing"

func TestWarnings(t *testing.T) {
	g := Graph{
		LRTs: []LRT{
			{Symbol: "solo", Restaked: "10", Delegations: []Delegation{{Operator: "op1", Amount: "10"}}},
			{Symbol: "concentrated", Restaked: "10", Delegations: []Delegation{{Operator: "op1", Amount: "9"}, {Operator: "op2", Amount: "1"}}},
			{Symbol: "spread", Restaked: "10", Delegations: []Delegation{{Operator: "op1", Amount: "3"}, {Operator: "op2", Amount: "3"}, {Operator: "op3", Amount: "4"}}},
			{Symbol: "broken", Restaked: "100", Delegations: []Delegation{{Operator: "op1", Amount: "1"}, {Operator: "op2", Amount: "1"}}},
		},
		Operators: []Operator{{Address: "op1"}, {Address: "op2"}, {Address: "op3"}},
	}
	ws := Warnings(g)
	has := func(lrt, code string, sev Severity) bool {
		for _, w := range ws {
			if w.LRT == lrt && w.Code == code && w.Severity == sev {
				return true
			}
		}
		return false
	}
	if !has("solo", "single_operator_dependency", SevHigh) {
		t.Error("expected single_operator_dependency for solo")
	}
	if !has("concentrated", "high_concentration", SevWarn) {
		t.Error("expected high_concentration for concentrated")
	}
	if !has("broken", "invariant_failed", SevHigh) {
		t.Error("expected invariant_failed for broken (1+1 != 100)")
	}
	for _, w := range ws {
		if w.LRT == "spread" && w.Code == "high_concentration" {
			t.Error("spread should not be flagged high_concentration")
		}
	}
}
