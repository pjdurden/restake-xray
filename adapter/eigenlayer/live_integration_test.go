package eigenlayer

import (
	"context"
	"os"
	"testing"
)

// TestLiveLRT exercises the live reader end-to-end against a real RPC.
// Skipped unless RPC_URL is set:
//
//	RPC_URL=https://ethereum-rpc.publicnode.com go test ./adapter/eigenlayer/ -run TestLiveLRT -v
//
// By default it reads the committed configs/lrts.json (ezETH + Renzo's
// OperatorDelegator stakers) and asserts a non-zero restaked total.
func TestLiveLRT(t *testing.T) {
	rpc := os.Getenv("RPC_URL")
	if rpc == "" {
		t.Skip("set RPC_URL to run live integration test")
	}
	ctx := context.Background()
	l, err := NewLive(ctx, rpc)
	if err != nil {
		t.Fatal(err)
	}
	bn, err := l.BlockNumber(ctx)
	if err != nil || bn == 0 {
		t.Fatalf("block number: %d err=%v", bn, err)
	}

	cfgs, err := LoadConfigs("../../configs/lrts.json")
	if err != nil {
		t.Fatal(err)
	}
	g, err := New(l, cfgs).Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.LRTs) == 0 {
		t.Fatal("no LRTs in snapshot")
	}
	if r := g.LRTs[0].Restaked; r == "" || r == "0" {
		t.Fatalf("live %s backing empty (restaked=%q)", g.LRTs[0].Symbol, r)
	}
	if len(g.LRTs[0].Delegations) == 0 {
		t.Fatal("no operator delegations resolved")
	}
}
