#!/usr/bin/env bash
# Regenerate the published dataset from live data, render the exposure-graph SVG,
# and stage timestamped copies.
# Requires the live EigenLayer reader (see adapter/eigenlayer/LIVE.md) and RPC_URL.
set -euo pipefail
cd "$(dirname "$0")/.."
: "${RPC_URL:?set RPC_URL}"

go build -o xray ./cmd/xray
mkdir -p data

# 1) dataset
./xray scan --rpc "$RPC_URL" --lrts configs/lrts.json --labels testdata/labels.json --out data/latest.json
ts="$(date -u +%Y%m%dT%H%M%SZ)"
cp data/latest.json "data/snapshot-${ts}.json"

# 2) exposure-graph diagram (DOT always; SVG if Graphviz is installed)
./xray graph --from data/latest.json --dot > data/latest.dot
if command -v dot >/dev/null 2>&1; then
  dot -Tsvg data/latest.dot -o data/latest.svg
  cp data/latest.svg "data/snapshot-${ts}.svg"
  echo "rendered data/latest.svg"
else
  echo "graphviz 'dot' not found — wrote data/latest.dot only (install graphviz to get SVG)"
fi

echo "dataset updated; commit with ./commit-and-push.sh"
