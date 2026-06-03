package graph

import (
	"fmt"
	"sort"
)

// Severity classifies a warning.
type Severity string

const (
	SevInfo Severity = "info"
	SevWarn Severity = "warn"
	SevHigh Severity = "high"
)

// highConcentrationThreshold: HHI at/above which an LRT is flagged (warn).
const highConcentrationThreshold = 0.5

// Warning is a derived, rule-based health flag (not a subjective risk score).
type Warning struct {
	Severity Severity `json:"severity"`
	LRT      string   `json:"lrt,omitempty"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
}

// Warnings derives health flags from the graph: single-operator dependency,
// high operator concentration, and failed invariants.
func Warnings(g Graph) []Warning {
	var out []Warning
	for _, l := range g.LRTs {
		ops := map[string]bool{}
		for _, d := range l.Delegations {
			ops[d.Operator] = true
		}
		switch {
		case len(ops) == 1:
			out = append(out, Warning{SevHigh, l.Symbol, "single_operator_dependency",
				"all restaked backing is delegated to a single operator"})
		default:
			if hhi := OperatorConcentration(l); hhi >= highConcentrationThreshold {
				out = append(out, Warning{SevWarn, l.Symbol, "high_concentration",
					fmt.Sprintf("operator concentration HHI %.3f >= %.2f", hhi, highConcentrationThreshold)})
			}
		}
	}
	for _, inv := range CheckInvariants(g) {
		if !inv.OK {
			out = append(out, Warning{SevHigh, inv.LRT, "invariant_failed",
				fmt.Sprintf("%s: %s", inv.Name, inv.Detail)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LRT != out[j].LRT {
			return out[i].LRT < out[j].LRT
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Severity < out[j].Severity
	})
	return out
}
