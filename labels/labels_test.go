package labels

import "testing"

func TestStaticLabels(t *testing.T) {
	p, err := LoadStatic("../testdata/labels.json")
	if err != nil {
		t.Fatal(err)
	}
	if p.OperatorName("0xop1") != "P2P" {
		t.Fatalf("operator name: %q", p.OperatorName("0xop1"))
	}
	if sym, dec, ok := p.TokenSymbol("0xbeacon"); !ok || sym != "ETH" || dec != 18 {
		t.Fatalf("token: %q %d %v", sym, dec, ok)
	}
}

func TestNoopLabels(t *testing.T) {
	var p Noop
	if p.OperatorName("x") != "" {
		t.Fatal("noop should return empty")
	}
}
