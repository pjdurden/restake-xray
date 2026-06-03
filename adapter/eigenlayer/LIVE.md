# Live EigenLayer reader — implementation notes

The adapter orchestration (`eigenlayer.go`) and its golden test are complete and
run offline against `testdata/eigenlayer-pinned.json`. What remains is the **live
`Reader`** that reads mainnet directly. It is deliberately the last piece because
it needs three things this environment doesn't have: a funded RPC endpoint, the
current verified ABIs, and LRT-specific read logic.

## Steps

1. **Dependency:** `go get github.com/ethereum/go-ethereum@latest`
2. **Bindings:** install `abigen`, save verified ABIs under
   `adapter/eigenlayer/abi/`, generate into `adapter/eigenlayer/bindings/` for:
   - `DelegationManager` — `getOperatorShares(operator, strategies[])`, `delegatedTo(staker)`
   - `StrategyManager` / `AllocationManager` — strategy shares, operator-set/AVS allocation
   - `AVSDirectory` / operator-sets registry — operator → AVS registration
   - relevant ERC20 / beacon-strategy contracts for collateral + decimals
3. **`live.go`:** implement `Reader` with `ethclient`. Mainnet core addresses
   (verify against the current EigenLayer deployment before shipping):
   - DelegationManager `0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A`
   - AVSDirectory `0x135DDa560e946695d6f155dACaFC6f1F25C1F5AF`
4. **`LRTBacking` per LRT** (start with one LRT, e.g. `ezETH`): this is the seam
   where deep restaking domain knowledge does the work — map the LRT to its operator
   delegations using the addresses in `LRTConfig.Extra`. Batch operator reads
   with multicall.
5. **Integration test** (guarded by `RPC_URL`):
   `RPC_URL=<rpc> go test ./adapter/eigenlayer/ -run TestLiveLRT -v`
6. **Record fixture:** dump the live reads at a pinned block into
   `testdata/eigenlayer-pinned.json` and hand-verify ezETH's numbers once against
   Etherscan, so the offline golden test runs against real recorded data.

Everything upstream of `Reader` (graph, metrics, snapshot, CLI, API, dashboard)
is already tested and unchanged by this work.
