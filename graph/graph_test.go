package graph

import "testing"

func TestGraphHoldsLRTOperatorAVS(t *testing.T) {
	g := Graph{
		Block: 100, Timestamp: 1, Protocol: "eigenlayer",
		LRTs: []LRT{{
			Address: "0xlrt", Symbol: "cmETH", Restaked: "30",
			Collateral:  []Stake{{Token: "0xste", Symbol: "stETH", Amount: "30", Decimals: 18}},
			Delegations: []Delegation{{Operator: "0xop1", Amount: "20"}, {Operator: "0xop2", Amount: "10"}},
		}},
		Operators: []Operator{{Address: "0xop1", Name: "Op One", AVSs: []string{"0xavs"}}},
		AVSs:      []AVS{{Address: "0xavs", Name: "EigenDA"}},
	}
	if g.LRTs[0].Symbol != "cmETH" || len(g.LRTs[0].Delegations) != 2 {
		t.Fatalf("unexpected graph shape: %+v", g)
	}
}
