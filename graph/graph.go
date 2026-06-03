// Package graph defines the protocol-agnostic restaking exposure graph
// (collateral -> LRT -> operator -> AVS) and pure functions over it.
package graph

// Graph is a single point-in-time exposure snapshot for one or more protocols.
type Graph struct {
	Block     uint64     `json:"block"`
	Timestamp int64      `json:"timestamp"`
	Protocol  string     `json:"protocol"`
	LRTs      []LRT      `json:"lrts"`
	Operators []Operator `json:"operators"`
	AVSs      []AVS      `json:"avss"`
}

// LRT is a liquid restaking token and how it is backed.
type LRT struct {
	Address     string       `json:"address"`
	Symbol      string       `json:"symbol"`
	Restaked    string       `json:"restaked"` // total restaked backing, base-unit decimal string
	Collateral  []Stake      `json:"collateral"`
	Delegations []Delegation `json:"delegations"`
}

// Stake is a collateral position backing an LRT.
type Stake struct {
	Token    string `json:"token"`
	Symbol   string `json:"symbol"`
	Amount   string `json:"amount"` // base-unit decimal string
	Decimals uint8  `json:"decimals"`
}

// Delegation is restaked backing delegated from an LRT to an operator.
type Delegation struct {
	Operator string `json:"operator"`
	Amount   string `json:"amount"` // base-unit decimal string
}

// Operator restakes delegated assets and secures AVSs.
type Operator struct {
	Address string   `json:"address"`
	Name    string   `json:"name"`
	AVSs    []string `json:"avss"`
}

// AVS is a service secured by operators.
type AVS struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}
