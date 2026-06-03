package eigenlayer

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// fixtureReader replays recorded reads for a pinned block.
type fixtureReader struct {
	Block    uint64              `json:"block"`
	Backings map[string]Backing  `json:"backings"`      // by LRT address
	OpAVSs   map[string][]string `json:"operator_avss"` // by operator address
}

func loadFixture(t *testing.T, path string) *fixtureReader {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f fixtureReader
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return &f
}

func (f *fixtureReader) BlockNumber(context.Context) (uint64, error) { return f.Block, nil }
func (f *fixtureReader) LRTBacking(_ context.Context, c LRTConfig) (Backing, error) {
	return f.Backings[c.Address], nil
}
func (f *fixtureReader) OperatorAVSs(_ context.Context, op string) ([]string, error) {
	return f.OpAVSs[op], nil
}

func TestAdapterBuildsGraphFromReader(t *testing.T) {
	r := loadFixture(t, "../../testdata/eigenlayer-pinned.json")
	a := New(r, []LRTConfig{{Symbol: "cmETH", Address: "0xcmETH"}})
	g, err := a.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if g.Protocol != "eigenlayer" || g.Block == 0 {
		t.Fatalf("bad header: %+v", g)
	}
	var found bool
	for _, l := range g.LRTs {
		if l.Symbol == "cmETH" && len(l.Delegations) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("cmETH not built with delegations")
	}
	if len(g.Operators) == 0 || len(g.AVSs) == 0 {
		t.Fatalf("operators/avss not populated: ops=%d avss=%d", len(g.Operators), len(g.AVSs))
	}
}
