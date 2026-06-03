package sample

import (
	"context"
	"testing"
)

func TestSampleLoadsFixture(t *testing.T) {
	a, err := NewFromFile("../../testdata/sample-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	g, err := a.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if a.Name() == "" || len(g.LRTs) == 0 {
		t.Fatalf("empty sample graph: name=%q lrts=%d", a.Name(), len(g.LRTs))
	}
}
