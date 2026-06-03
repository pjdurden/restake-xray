// Live EigenLayer reader: reads the restaking exposure graph directly from an
// Ethereum RPC endpoint via go-ethereum, without code-generated bindings.
//
// Methodology (all reads pinned to one block for a consistent snapshot):
//   - An LRT's EigenLayer position is held by one or more "staker" contracts
//     (configured per LRT in LRTConfig.Extra["stakers"]).
//   - For each staker: StrategyManager.getDeposits(staker) -> (strategies, shares)
//     gives the collateral, and DelegationManager.delegatedTo(staker) -> operator
//     gives who it is delegated to. Summed shares are the restaked backing.
//   - OperatorAVSs resolves an operator's AVS registrations from AVSDirectory
//     logs (OperatorAVSRegistrationStatusUpdated), latest-status-wins.
//
// Mainnet core addresses are verified against the live deployment before shipping
// (see LIVE.md). The seam that needs protocol-specific knowledge is the staker
// set per LRT, which is data (configs/lrts.json), not code.
package eigenlayer

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/pjdurden/restake-xray/graph"
)

// Mainnet core addresses (verified against the current EigenLayer deployment).
const (
	addrDelegationManager = "0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A"
	addrStrategyManager   = "0x858646372CC42E1A627fcE94aa7A7033e7CF075A"
	addrAVSDirectory      = "0x135DDa560e946695d6f155dACaFC6f1F25C1F5AF"
)

// avsRegEventSig is keccak256("OperatorAVSRegistrationStatusUpdated(address,address,uint8)").
var avsRegEventSig = crypto.Keccak256Hash([]byte("OperatorAVSRegistrationStatusUpdated(address,address,uint8)"))

const (
	abiDelegationManager = `[
	  {"name":"delegatedTo","type":"function","stateMutability":"view","inputs":[{"name":"staker","type":"address"}],"outputs":[{"name":"","type":"address"}]},
	  {"name":"getOperatorShares","type":"function","stateMutability":"view","inputs":[{"name":"operator","type":"address"},{"name":"strategies","type":"address[]"}],"outputs":[{"name":"","type":"uint256[]"}]}
	]`
	abiStrategyManager = `[
	  {"name":"getDeposits","type":"function","stateMutability":"view","inputs":[{"name":"staker","type":"address"}],"outputs":[{"name":"","type":"address[]"},{"name":"","type":"uint256[]"}]}
	]`
)

// Live reads EigenLayer state directly from an Ethereum RPC endpoint.
type Live struct {
	ec    *ethclient.Client
	block *big.Int // pinned snapshot block; all reads use it

	dm common.Address
	sm common.Address

	dmABI abi.ABI
	smABI abi.ABI

	// avsFromBlock bounds the AVSDirectory log scan (public RPCs cap getLogs range).
	avsFromBlock uint64
}

// NewLive dials rpcURL, pins the current head block, and prepares the core
// contract call interfaces.
func NewLive(ctx context.Context, rpcURL string) (*Live, error) {
	ec, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}
	head, err := ec.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("get head block: %w", err)
	}
	dmABI, err := abi.JSON(strings.NewReader(abiDelegationManager))
	if err != nil {
		return nil, err
	}
	smABI, err := abi.JSON(strings.NewReader(abiStrategyManager))
	if err != nil {
		return nil, err
	}
	from := uint64(0)
	if head > 250_000 { // ~5 weeks of recent registrations by default
		from = head - 250_000
	}
	return &Live{
		ec:           ec,
		block:        new(big.Int).SetUint64(head),
		dm:           common.HexToAddress(addrDelegationManager),
		sm:           common.HexToAddress(addrStrategyManager),
		dmABI:        dmABI,
		smABI:        smABI,
		avsFromBlock: from,
	}, nil
}

// SetAVSFromBlock overrides the lower bound of the AVSDirectory log scan.
func (l *Live) SetAVSFromBlock(b uint64) { l.avsFromBlock = b }

func (l *Live) callOpts(ctx context.Context) *bind.CallOpts {
	return &bind.CallOpts{Context: ctx, BlockNumber: l.block}
}

func (l *Live) BlockNumber(ctx context.Context) (uint64, error) { return l.block.Uint64(), nil }

// LRTBacking reads one LRT's backing from its configured staker contract(s).
func (l *Live) LRTBacking(ctx context.Context, cfg LRTConfig) (Backing, error) {
	stakers := parseAddrList(cfg.Extra["stakers"])
	if len(stakers) == 0 {
		return Backing{}, fmt.Errorf("eigenlayer: no stakers configured for %s; set Extra.stakers in configs/lrts.json", cfg.Symbol)
	}
	dm := bind.NewBoundContract(l.dm, l.dmABI, l.ec, l.ec, l.ec)
	sm := bind.NewBoundContract(l.sm, l.smABI, l.ec, l.ec, l.ec)

	total := new(big.Int)
	collByStrat := map[common.Address]*big.Int{}
	delByOp := map[common.Address]*big.Int{}

	for _, staker := range stakers {
		// strategies + shares for this staker
		var depOut []interface{}
		if err := sm.Call(l.callOpts(ctx), &depOut, "getDeposits", staker); err != nil {
			return Backing{}, fmt.Errorf("getDeposits(%s): %w", staker, err)
		}
		strategies, _ := depOut[0].([]common.Address)
		shares, _ := depOut[1].([]*big.Int)

		stakerTotal := new(big.Int)
		for i := range strategies {
			amt := shares[i]
			if amt == nil {
				continue
			}
			stakerTotal.Add(stakerTotal, amt)
			if collByStrat[strategies[i]] == nil {
				collByStrat[strategies[i]] = new(big.Int)
			}
			collByStrat[strategies[i]].Add(collByStrat[strategies[i]], amt)
		}
		total.Add(total, stakerTotal)

		// who this staker delegates to
		var delOut []interface{}
		if err := dm.Call(l.callOpts(ctx), &delOut, "delegatedTo", staker); err != nil {
			return Backing{}, fmt.Errorf("delegatedTo(%s): %w", staker, err)
		}
		op, _ := delOut[0].(common.Address)
		if op != (common.Address{}) {
			if delByOp[op] == nil {
				delByOp[op] = new(big.Int)
			}
			delByOp[op].Add(delByOp[op], stakerTotal)
		}
	}

	b := Backing{Restaked: total.String()}
	for strat, amt := range collByStrat {
		b.Collateral = append(b.Collateral, stakeCollateral(strat, amt))
	}
	for op, amt := range delByOp {
		b.Delegations = append(b.Delegations, delegation(op, amt))
	}
	return b, nil
}

// OperatorAVSs resolves which AVSs an operator secures by scanning AVSDirectory
// registration logs in the [avsFromBlock, pinned block] window, latest-status-wins.
func (l *Live) OperatorAVSs(ctx context.Context, operator string) ([]string, error) {
	opTopic := common.BytesToHash(common.LeftPadBytes(common.HexToAddress(operator).Bytes(), 32))
	avsDir := common.HexToAddress(addrAVSDirectory)

	latest := map[common.Address]struct {
		block  uint64
		status uint8
	}{}

	const chunk = 9000
	for from := l.avsFromBlock; from <= l.block.Uint64(); from += chunk + 1 {
		to := from + chunk
		if to > l.block.Uint64() {
			to = l.block.Uint64()
		}
		q := ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(from),
			ToBlock:   new(big.Int).SetUint64(to),
			Addresses: []common.Address{avsDir},
			Topics:    [][]common.Hash{{avsRegEventSig}, {opTopic}},
		}
		logs, err := l.ec.FilterLogs(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("avs log scan [%d,%d]: %w", from, to, err)
		}
		for _, lg := range logs {
			if len(lg.Topics) < 3 || len(lg.Data) == 0 {
				continue
			}
			avs := common.BytesToAddress(lg.Topics[2].Bytes())
			status := lg.Data[len(lg.Data)-1] // uint8 in the last byte of the 32-byte word
			if cur, ok := latest[avs]; !ok || lg.BlockNumber >= cur.block {
				latest[avs] = struct {
					block  uint64
					status uint8
				}{lg.BlockNumber, status}
			}
		}
	}

	var out []string
	for avs, st := range latest {
		if st.status == 1 { // REGISTERED
			out = append(out, strings.ToLower(avs.Hex()))
		}
	}
	return out, nil
}

func parseAddrList(s string) []common.Address {
	var out []common.Address
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, common.HexToAddress(p))
	}
	return out
}

func stakeCollateral(strategy common.Address, amount *big.Int) graph.Stake {
	// Symbol/decimals are filled by the labels layer; we record the raw on-chain
	// strategy address and share amount.
	return graph.Stake{Token: strings.ToLower(strategy.Hex()), Amount: amount.String()}
}

func delegation(operator common.Address, amount *big.Int) graph.Delegation {
	return graph.Delegation{Operator: strings.ToLower(operator.Hex()), Amount: amount.String()}
}
