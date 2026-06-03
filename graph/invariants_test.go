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
