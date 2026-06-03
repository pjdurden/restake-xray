package graph

import (
	"fmt"
	"math/big"
)

// InvariantResult is one consistency check outcome for an LRT.
type InvariantResult struct {
	Name   string `json:"name"`
	LRT    string `json:"lrt"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// reconcileTolerance: relative tolerance for delegations-vs-restaked (0.5%).
const reconcileTolerance = 0.005

// CheckInvariants validates internal consistency of the graph.
func CheckInvariants(g Graph) []InvariantResult {
	known := map[string]bool{}
	for _, op := range g.Operators {
		known[op.Address] = true
	}
	var out []InvariantResult
	for _, l := range g.LRTs {
		// 1) delegations reconcile to declared restaked total
		sum := big.NewFloat(0)
		for _, d := range l.Delegations {
			sum.Add(sum, parseAmount(d.Amount))
		}
		restaked := parseAmount(l.Restaked)
		ok, detail := withinTolerance(sum, restaked)
		out = append(out, InvariantResult{"delegations_reconcile_restaked", l.Symbol, ok, detail})

		// 2) every delegation operator is known
		var missing []string
		for _, d := range l.Delegations {
			if !known[d.Operator] {
				missing = append(missing, d.Operator)
			}
		}
		out = append(out, InvariantResult{
			"delegations_reference_known_operators", l.Symbol, len(missing) == 0,
			fmt.Sprintf("missing operators: %v", sortedStrings(missing)),
		})
	}
	return out
}

func withinTolerance(sum, target *big.Float) (bool, string) {
	if target.Sign() == 0 {
		if sum.Sign() == 0 {
			return true, "both zero"
		}
		return false, "restaked is zero but delegations are not"
	}
	diff := new(big.Float).Sub(sum, target)
	diff.Abs(diff)
	rel, _ := new(big.Float).Quo(diff, target).Float64()
	return rel <= reconcileTolerance, fmt.Sprintf("rel diff %.4f%%", rel*100)
}
