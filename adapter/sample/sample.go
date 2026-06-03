// Package sample is an offline adapter that serves a graph from a JSON fixture.
package sample

import (
	"context"
	"encoding/json"
	"os"

	"github.com/pjdurden/restake-xray/graph"
)

// Sample is an adapter.Protocol backed by a static JSON graph.
type Sample struct{ g graph.Graph }

// NewFromFile loads a graph fixture.
func NewFromFile(path string) (*Sample, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var g graph.Graph
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, err
	}
	return &Sample{g: g}, nil
}

func (s *Sample) Name() string { return s.g.Protocol }

func (s *Sample) Snapshot(_ context.Context) (graph.Graph, error) { return s.g, nil }
