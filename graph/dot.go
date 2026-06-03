package graph

import (
	"fmt"
	"sort"
	"strings"
)

// DOT renders the exposure graph as Graphviz DOT:
// LRTs (boxes) -> operators (ellipses) -> AVSs (diamonds).
// Output is deterministic (stable node/edge ordering).
func DOT(g Graph) string {
	var b strings.Builder
	b.WriteString("digraph restake_xray {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [fontname=\"sans-serif\"];\n")

	// nodes
	lrts := append([]LRT(nil), g.LRTs...)
	sort.Slice(lrts, func(i, j int) bool { return lrts[i].Symbol < lrts[j].Symbol })
	for _, l := range lrts {
		fmt.Fprintf(&b, "  %q [shape=box, style=filled, fillcolor=\"#dbeafe\"];\n", lrtNode(l.Symbol))
	}
	ops := append([]Operator(nil), g.Operators...)
	sort.Slice(ops, func(i, j int) bool { return ops[i].Address < ops[j].Address })
	for _, op := range ops {
		fmt.Fprintf(&b, "  %q [shape=ellipse];\n", opNode(op))
	}
	avss := append([]AVS(nil), g.AVSs...)
	sort.Slice(avss, func(i, j int) bool { return avss[i].Address < avss[j].Address })
	for _, av := range avss {
		fmt.Fprintf(&b, "  %q [shape=diamond, style=filled, fillcolor=\"#fee2e2\"];\n", avsNode(av))
	}

	// edges LRT -> operator
	opByAddr := map[string]Operator{}
	for _, op := range g.Operators {
		opByAddr[op.Address] = op
	}
	for _, l := range lrts {
		ds := append([]Delegation(nil), l.Delegations...)
		sort.Slice(ds, func(i, j int) bool { return ds[i].Operator < ds[j].Operator })
		for _, d := range ds {
			fmt.Fprintf(&b, "  %q -> %q [label=%q];\n", lrtNode(l.Symbol), opNode(opByAddr[d.Operator]), d.Amount)
		}
	}
	// edges operator -> AVS
	avsByAddr := map[string]AVS{}
	for _, av := range g.AVSs {
		avsByAddr[av.Address] = av
	}
	for _, op := range ops {
		for _, av := range sortedStrings(op.AVSs) {
			fmt.Fprintf(&b, "  %q -> %q;\n", opNode(op), avsNode(avsByAddr[av]))
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func lrtNode(sym string) string { return "LRT: " + sym }

func opNode(op Operator) string {
	if op.Name != "" {
		return "Op: " + op.Name
	}
	return "Op: " + short(op.Address)
}

func avsNode(av AVS) string {
	if av.Name != "" {
		return "AVS: " + av.Name
	}
	return "AVS: " + short(av.Address)
}

func short(addr string) string {
	if len(addr) > 10 {
		return addr[:6] + "…" + addr[len(addr)-4:]
	}
	return addr
}
