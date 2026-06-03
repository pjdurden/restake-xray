package graph

import (
	"math"
	"sort"
)

// concentrationDeltaThreshold: minimum absolute change reported in a diff.
const concentrationDeltaThreshold = 0.001

// ConcChange is an LRT's operator-concentration change between two graphs.
type ConcChange struct {
	LRT   string  `json:"lrt"`
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Delta float64 `json:"delta"`
}

// Diff summarizes what changed between two exposure graphs.
type Diff struct {
	FromBlock        uint64       `json:"from_block"`
	ToBlock          uint64       `json:"to_block"`
	AddedLRTs        []string     `json:"added_lrts"`
	RemovedLRTs      []string     `json:"removed_lrts"`
	AddedOperators   []string     `json:"added_operators"`
	RemovedOperators []string     `json:"removed_operators"`
	AddedAVSs        []string     `json:"added_avss"`
	RemovedAVSs      []string     `json:"removed_avss"`
	Concentration    []ConcChange `json:"concentration_changes"`
}

// DiffGraphs computes a -> b changes.
func DiffGraphs(a, b Graph) Diff {
	d := Diff{FromBlock: a.Block, ToBlock: b.Block}

	aL, bL := lrtSymbols(a), lrtSymbols(b)
	d.AddedLRTs = diffKeys(bL, aL)
	d.RemovedLRTs = diffKeys(aL, bL)

	aO, bO := operatorAddrs(a), operatorAddrs(b)
	d.AddedOperators = diffKeys(bO, aO)
	d.RemovedOperators = diffKeys(aO, bO)

	aV, bV := avsAddrs(a), avsAddrs(b)
	d.AddedAVSs = diffKeys(bV, aV)
	d.RemovedAVSs = diffKeys(aV, bV)

	aConc := concBySymbol(a)
	bConc := concBySymbol(b)
	for sym, from := range aConc {
		if to, ok := bConc[sym]; ok {
			if math.Abs(to-from) >= concentrationDeltaThreshold {
				d.Concentration = append(d.Concentration, ConcChange{sym, from, to, to - from})
			}
		}
	}
	sort.Slice(d.Concentration, func(i, j int) bool { return d.Concentration[i].LRT < d.Concentration[j].LRT })
	return d
}

func lrtSymbols(g Graph) map[string]bool {
	m := map[string]bool{}
	for _, l := range g.LRTs {
		m[l.Symbol] = true
	}
	return m
}

func operatorAddrs(g Graph) map[string]bool {
	m := map[string]bool{}
	for _, o := range g.Operators {
		m[o.Address] = true
	}
	return m
}

func avsAddrs(g Graph) map[string]bool {
	m := map[string]bool{}
	for _, a := range g.AVSs {
		m[a.Address] = true
	}
	return m
}

func concBySymbol(g Graph) map[string]float64 {
	m := map[string]float64{}
	for _, l := range g.LRTs {
		m[l.Symbol] = OperatorConcentration(l)
	}
	return m
}

// diffKeys returns keys present in a but not b, sorted.
func diffKeys(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	return sortedStrings(out)
}
