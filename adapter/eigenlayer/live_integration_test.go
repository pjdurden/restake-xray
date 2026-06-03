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

	stakers := os.Getenv("CMETH_STAKERS")
	if stakers == "" {
		t.Skip("set CMETH_STAKERS (comma-separated EigenLayer staker addresses) to assert backing")
	}
	a := New(l, []LRTConfig{{
		Symbol:  "cmETH",
		Address: "0xE6829d9a7eE3040e1276Fa75293Bde931859e8fA",
		Extra:   map[string]string{"stakers": stakers},
	}})
	g, err := a.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.LRTs) == 0 || g.LRTs[0].Restaked == "" {
		t.Fatal("live cmETH backing empty")
	}
}
