// Package render turns snapshots into human-readable terminal output.
package render

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/pjdurden/restake-xray/snapshot"
)

// Summary renders a high-level overview of the snapshot.
func Summary(s snapshot.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "restake-xray — %s @ block %d\n", s.Protocol, s.Block)
	fmt.Fprintf(&b, "LRTs: %d  Operators: %d  AVSs: %d\n\n", len(s.Graph.LRTs), len(s.Graph.Operators), len(s.Graph.AVSs))
	w := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "LRT\tRESTAKED\tOPERATORS\tCONCENTRATION")
	for _, l := range s.Graph.LRTs {
		fmt.Fprintf(w, "%s\t%s\t%d\t%.3f\n", l.Symbol, l.Restaked, len(l.Delegations), s.Concentration[l.Symbol])
	}
	w.Flush()
	failed := 0
	for _, inv := range s.Invariants {
		if !inv.OK {
			failed++
		}
	}
	fmt.Fprintf(&b, "\nInvariants: %d/%d passing", len(s.Invariants)-failed, len(s.Invariants))
	if len(s.Warnings) > 0 {
		fmt.Fprintf(&b, "  |  Warnings: %d", len(s.Warnings))
	}
	fmt.Fprintln(&b)
	return b.String()
}

// Systemic renders the ecosystem single-points-of-failure rankings.
func Systemic(s snapshot.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Systemic operators (most restaked depending on them):\n\n")
	w := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "OPERATOR\tTOTAL\tLRTS\tAVSS")
	for _, o := range s.Systemic.Operators {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\n", nameOr(o.Name, o.Operator), o.TotalAmount, len(o.LRTs), len(o.AVSs))
	}
	w.Flush()
	fmt.Fprintf(&b, "\nSystemic AVSs (most LRTs exposed):\n\n")
	w = tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "AVS\tLRTS\tOPERATORS")
	for _, a := range s.Systemic.AVSs {
		fmt.Fprintf(w, "%s\t%d\t%d\n", nameOr(a.Name, a.AVS), len(a.LRTs), len(a.Operators))
	}
	w.Flush()
	return b.String()
}

// Warnings renders the derived health flags.
func Warnings(s snapshot.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Health warnings:\n\n")
	if len(s.Warnings) == 0 {
		fmt.Fprintln(&b, "  (none)")
		return b.String()
	}
	w := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "SEVERITY\tLRT\tCODE\tMESSAGE")
	for _, wn := range s.Warnings {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", wn.Severity, wn.LRT, wn.Code, wn.Message)
	}
	w.Flush()
	return b.String()
}

// Report renders a shareable Markdown report of the snapshot.
func Report(s snapshot.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# restake-xray report — %s\n\n", s.Protocol)
	fmt.Fprintf(&b, "Block `%d` · %d LRTs · %d operators · %d AVSs\n\n", s.Block, len(s.Graph.LRTs), len(s.Graph.Operators), len(s.Graph.AVSs))

	fmt.Fprintf(&b, "## LRTs\n\n| LRT | Restaked | Operators | Concentration |\n|---|---|---|---|\n")
	for _, l := range s.Graph.LRTs {
		fmt.Fprintf(&b, "| %s | %s | %d | %.3f |\n", l.Symbol, l.Restaked, len(l.Delegations), s.Concentration[l.Symbol])
	}

	fmt.Fprintf(&b, "\n## Systemic operators\n\n| Operator | Total restaked | LRTs | AVSs |\n|---|---|---|---|\n")
	for _, o := range s.Systemic.Operators {
		fmt.Fprintf(&b, "| %s | %s | %d | %d |\n", nameOr(o.Name, o.Operator), o.TotalAmount, len(o.LRTs), len(o.AVSs))
	}

	if len(s.Warnings) > 0 {
		fmt.Fprintf(&b, "\n## Warnings\n\n| Severity | LRT | Message |\n|---|---|---|\n")
		for _, wn := range s.Warnings {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", wn.Severity, wn.LRT, wn.Message)
		}
	}
	return b.String()
}

func nameOr(name, fallback string) string {
	if name != "" {
		return name
	}
	return fallback
}

// Contagion renders the shared-exposure pairs.
func Contagion(s snapshot.Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contagion (shared exposure between LRTs):\n\n")
	if len(s.Contagion) == 0 {
		fmt.Fprintln(&b, "  (no shared operators found)")
		return b.String()
	}
	w := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "A\tB\tSHARED OPS\tSHARED AVSS\tSCORE")
	for _, o := range s.Contagion {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%.3f\n", o.A, o.B, len(o.SharedOperators), len(o.SharedAVSs), o.Score)
	}
	w.Flush()
	return b.String()
}
