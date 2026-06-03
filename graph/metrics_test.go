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
	if len(o.SharedAVSs) != 2 {
		t.Fatalf("shared avss wrong: %+v", o.SharedAVSs)
	}
}
