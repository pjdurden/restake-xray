package graph

import (
	"math/big"
	"sort"
)

// SystemicOperator measures how much the whole ecosystem leans on one operator:
// the total restaked amount delegated to it across all LRTs, and which LRTs and
// AVSs depend on it.
type SystemicOperator struct {
	Operator    string   `json:"operator"`
	Name        string   `json:"name"`
	TotalAmount string   `json:"total_amount"` // sum of delegations across all LRTs
	LRTs        []string `json:"lrts"`         // LRT symbols depending on this operator
	AVSs        []string `json:"avss"`         // AVSs this operator secures
}

// SystemicOperators ranks operators by total restaked depending on them
// (descending), tie-broken by operator address.
func SystemicOperators(g Graph) []SystemicOperator {
	opAVS := map[string][]string{}
	opName := map[string]string{}
	for _, op := range g.Operators {
		opAVS[op.Address] = op.AVSs
		opName[op.Address] = op.Name
	}
	totals := map[string]*big.Float{}
	lrts := map[string]map[string]bool{}
	for _, l := range g.LRTs {
		for _, d := range l.Delegations {
			if totals[d.Operator] == nil {
				totals[d.Operator] = big.NewFloat(0)
				lrts[d.Operator] = map[string]bool{}
			}
			totals[d.Operator].Add(totals[d.Operator], parseAmount(d.Amount))
			lrts[d.Operator][l.Symbol] = true
		}
	}
	out := make([]SystemicOperator, 0, len(totals))
	for op, total := range totals {
		out = append(out, SystemicOperator{
			Operator:    op,
			Name:        opName[op],
			TotalAmount: formatFloat(total),
			LRTs:        sortedStrings(keys(lrts[op])),
			AVSs:        sortedStrings(opAVS[op]),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		ai, _ := new(big.Float).SetString(out[i].TotalAmount)
		aj, _ := new(big.Float).SetString(out[j].TotalAmount)
		if ai.Cmp(aj) != 0 {
			return ai.Cmp(aj) > 0 // descending
		}
		return out[i].Operator < out[j].Operator
	})
	return out
}

// SystemicAVS measures ecosystem exposure to one AVS: which operators secure it
// and which LRTs are transitively exposed through those operators.
type SystemicAVS struct {
	AVS       string   `json:"avs"`
	Name      string   `json:"name"`
	Operators []string `json:"operators"`
	LRTs      []string `json:"lrts"`
}

// SystemicAVSs ranks AVSs by the number of exposed LRTs (descending),
// tie-broken by AVS address.
func SystemicAVSs(g Graph) []SystemicAVS {
	avsName := map[string]string{}
	for _, a := range g.AVSs {
		avsName[a.Address] = a.Name
	}
	// operator -> AVSs, and AVS -> operators
	avsOps := map[string]map[string]bool{}
	for _, op := range g.Operators {
		for _, av := range op.AVSs {
			if avsOps[av] == nil {
				avsOps[av] = map[string]bool{}
			}
			avsOps[av][op.Address] = true
		}
	}
	// operator -> LRT symbols delegating to it
	opLRTs := map[string]map[string]bool{}
	for _, l := range g.LRTs {
		for _, d := range l.Delegations {
			if opLRTs[d.Operator] == nil {
				opLRTs[d.Operator] = map[string]bool{}
			}
			opLRTs[d.Operator][l.Symbol] = true
		}
	}
	out := make([]SystemicAVS, 0, len(avsOps))
	for av, ops := range avsOps {
		lrtSet := map[string]bool{}
		for op := range ops {
			for sym := range opLRTs[op] {
				lrtSet[sym] = true
			}
		}
		out = append(out, SystemicAVS{
			AVS:       av,
			Name:      avsName[av],
			Operators: sortedStrings(keys(ops)),
			LRTs:      sortedStrings(keys(lrtSet)),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].LRTs) != len(out[j].LRTs) {
			return len(out[i].LRTs) > len(out[j].LRTs)
		}
		return out[i].AVS < out[j].AVS
	})
	return out
}
