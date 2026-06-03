// Package adapter defines the port each restaking protocol implements to
// populate the exposure graph.
package adapter

import (
	"context"

	"github.com/prajjwalchittori/restake-xray/graph"
)

// Protocol populates the exposure graph for one restaking protocol.
type Protocol interface {
	Name() string
	Snapshot(ctx context.Context) (graph.Graph, error)
}
