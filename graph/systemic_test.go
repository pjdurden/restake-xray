package graph

import "testing"

func systemicFixture() Graph {
	return Graph{
		LRTs: []LRT{
			{Symbol: "cmETH", Delegations: []Delegation{{Operator: "op1", Amount: "600"}, {Operator: "op2", Amount: "400"}}},
			{Symbol: "ezETH", Delegations: []Delegation{{Operator: "op2", Amount: "500"}, {Operator: "op3", Amount: "300"}}},
		},
		Operators: []Operator{
			{Address: "op1", Name: "P2P", AVSs: []string{"avsDA"}},
			{Address: "op2", Name: "Luga", AVSs: []string{"avsDA", "avsORACLE"}},
			{Address: "op3", Name: "Figment", AVSs: []string{"avsORACLE"}},
		},
		AVSs: []AVS{{Address: "avsDA", Name: "EigenDA"}, {Address: "avsORACLE", Name: "eOracle"}},
	}
}

func TestSystemicOperators(t *testing.T) {
	got := SystemicOperators(systemicFixture())
	if len(got) != 3 {
		t.Fatalf("want 3 operators, got %d", len(got))
	}
	// op2 has 400+500=900 total -> ranked first
	if got[0].Operator != "op2" || got[0].TotalAmount != "900" {
		t.Fatalf("expected op2/900 first, got %+v", got[0])
	}
	if len(got[0].LRTs) != 2 {
		t.Fatalf("op2 should serve 2 LRTs, got %v", got[0].LRTs)
	}
}

func TestSystemicAVSs(t *testing.T) {
	got := SystemicAVSs(systemicFixture())
	if len(got) != 2 {
		t.Fatalf("want 2 AVSs, got %d", len(got))
	}
	// avsDA exposed via op1(cmETH)+op2(cmETH,ezETH) -> {cmETH,ezETH} (2)
	// avsORACLE via op2(cmETH,ezETH)+op3(ezETH) -> {cmETH,ezETH} (2); tie -> sort by addr
	if got[0].AVS != "avsDA" {
		t.Fatalf("expected avsDA first (tie broken by addr), got %+v", got[0])
	}
	if len(got[0].LRTs) != 2 {
		t.Fatalf("avsDA should expose 2 LRTs, got %v", got[0].LRTs)
	}
}
