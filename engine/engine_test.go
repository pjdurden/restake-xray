package engine

import (
	"context"
	"testing"

	"github.com/prajjwalchittori/restake-xray/adapter"
	"github.com/prajjwalchittori/restake-xray/adapter/sample"
	"github.com/prajjwalchittori/restake-xray/labels"
)

func TestEngineSnapshotAppliesLabels(t *testing.T) {
	a, _ := sample.NewFromFile("../testdata/sample-graph.json")
	lp, _ := labels.LoadStatic("../testdata/labels.json")
	eng := New([]adapter.Protocol{a}, lp)
	s, err := eng.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	named := false
	for _, op := range s.Graph.Operators {
		if op.Address == "0xop1" && op.Name == "P2P" {
			named = true
		}
	}
	if !named {
		t.Fatal("expected operator 0xop1 labeled P2P")
	}
}
