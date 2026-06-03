.PHONY: build test scan serve snapshot-live diagram
build:         ; go build -o xray ./cmd/xray
test:          ; go test ./...
scan:          ; go run ./cmd/xray scan --sample testdata/sample-graph.json
serve:         ; go run ./cmd/xray serve --data data/latest.json
snapshot-live: ; ./scripts/snapshot.sh
diagram: build
	mkdir -p data
	./xray scan --out data/latest.json
	./xray graph --from data/latest.json --dot > data/latest.dot
	@command -v dot >/dev/null 2>&1 && dot -Tsvg data/latest.dot -o data/latest.svg && echo "rendered data/latest.svg" || echo "graphviz not found; wrote data/latest.dot"
