package graph

import (
	"math/big"
	"sort"
)

// parseAmount parses a base-unit decimal string into a big.Float; invalid -> 0.
func parseAmount(s string) *big.Float {
	f, ok := new(big.Float).SetString(s)
	if !ok {
		return big.NewFloat(0)
	}
	return f
}

// OperatorConcentration is the Herfindahl-Hirschman Index (0..1) of an LRT's
// restaked amount across operators. 1 = fully concentrated, ~0 = well spread.
func OperatorConcentration(l LRT) float64 {
	total := big.NewFloat(0)
	amts := make([]*big.Float, 0, len(l.Delegations))
	for _, d := range l.Delegations {
		a := parseAmount(d.Amount)
		amts = append(amts, a)
		total.Add(total, a)
	}
	if total.Sign() == 0 {
		return 0
	}
	hhi := 0.0
	for _, a := range amts {
		w, _ := new(big.Float).Quo(a, total).Float64()
		hhi += w * w
	}
	return hhi
}

// sortedStrings returns a sorted copy (helper for deterministic outputs).
func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// formatFloat renders a big.Float as a plain decimal string (no exponent),
// using the minimal number of digits.
func formatFloat(f *big.Float) string { return f.Text('f', -1) }

// Overlap is the shared exposure between two LRTs.
type Overlap struct {
	A               string   `json:"a"`
	B               string   `json:"b"`
	SharedOperators []string `json:"shared_operators"`
	SharedAVSs      []string `json:"shared_avss"`
	Score           float64  `json:"score"` // Jaccard over AVS sets, 0..1
}

func setOf(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func intersect(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if b[k] {
			out = append(out, k)
		}
	}
	return sortedStrings(out)
}

// ContagionMatrix returns one Overlap per unordered LRT pair (A<B by symbol)
// that shares at least one operator. AVS sets are computed transitively through
// each LRT's operators.
func ContagionMatrix(g Graph) []Overlap {
	opAVS := make(map[string][]string, len(g.Operators))
	for _, op := range g.Operators {
		opAVS[op.Address] = op.AVSs
	}
	type lrtSets struct {
		sym  string
		ops  map[string]bool
		avss map[string]bool
	}
	var ls []lrtSets
	for _, l := range g.LRTs {
		ops := map[string]bool{}
		avss := map[string]bool{}
		for _, d := range l.Delegations {
			ops[d.Operator] = true
			for _, a := range opAVS[d.Operator] {
				avss[a] = true
			}
		}
		ls = append(ls, lrtSets{l.Symbol, ops, avss})
	}
	sort.Slice(ls, func(i, j int) bool { return ls[i].sym < ls[j].sym })

	var out []Overlap
	for i := 0; i < len(ls); i++ {
		for j := i + 1; j < len(ls); j++ {
			sharedOps := intersect(ls[i].ops, ls[j].ops)
			if len(sharedOps) == 0 {
				continue
			}
			sharedAVS := intersect(ls[i].avss, ls[j].avss)
			union := setOf(append(sortedStrings(keys(ls[i].avss)), keys(ls[j].avss)...))
			score := 0.0
			if len(union) > 0 {
				score = float64(len(sharedAVS)) / float64(len(union))
			}
			out = append(out, Overlap{ls[i].sym, ls[j].sym, sharedOps, sharedAVS, score})
		}
	}
	return out
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
