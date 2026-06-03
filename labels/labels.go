// Package labels provides off-chain human-readable metadata for addresses.
package labels

import (
	"encoding/json"
	"os"
)

// Provider resolves addresses to names/symbols. Missing -> empty/!ok.
type Provider interface {
	OperatorName(addr string) string
	AVSName(addr string) string
	TokenSymbol(addr string) (symbol string, decimals uint8, ok bool)
}

type token struct {
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

// Static is a file-backed Provider.
type Static struct {
	Operators map[string]string `json:"operators"`
	AVSs      map[string]string `json:"avss"`
	Tokens    map[string]token  `json:"tokens"`
}

// LoadStatic reads a labels JSON file.
func LoadStatic(path string) (*Static, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Static
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Static) OperatorName(a string) string { return s.Operators[a] }
func (s *Static) AVSName(a string) string      { return s.AVSs[a] }
func (s *Static) TokenSymbol(a string) (string, uint8, bool) {
	t, ok := s.Tokens[a]
	return t.Symbol, t.Decimals, ok
}

// Noop returns no labels.
type Noop struct{}

func (Noop) OperatorName(string) string               { return "" }
func (Noop) AVSName(string) string                    { return "" }
func (Noop) TokenSymbol(string) (string, uint8, bool) { return "", 0, false }
