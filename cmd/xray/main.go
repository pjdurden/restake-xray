// Command xray is the restake-xray CLI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/pjdurden/restake-xray/adapter"
	"github.com/pjdurden/restake-xray/adapter/eigenlayer"
	"github.com/pjdurden/restake-xray/adapter/sample"
	"github.com/pjdurden/restake-xray/api"
	"github.com/pjdurden/restake-xray/engine"
	"github.com/pjdurden/restake-xray/graph"
	"github.com/pjdurden/restake-xray/labels"
	"github.com/pjdurden/restake-xray/render"
	"github.com/pjdurden/restake-xray/snapshot"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "scan":
		cmdScan(os.Args[2:])
	case "lrt":
		cmdLRT(os.Args[2:])
	case "contagion":
		cmdContagion(os.Args[2:])
	case "systemic":
		cmdSystemic(os.Args[2:])
	case "warnings":
		cmdWarnings(os.Args[2:])
	case "operator":
		cmdOperator(os.Args[2:])
	case "avs":
		cmdAVS(os.Args[2:])
	case "report":
		cmdReport(os.Args[2:])
	case "graph":
		cmdGraph(os.Args[2:])
	case "diff":
		cmdDiff(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: xray <command> [flags]")
	fmt.Fprintln(os.Stderr, "commands: scan lrt contagion systemic warnings operator avs report graph diff serve")
	os.Exit(2)
}

// commonFlags registers the offline-source flags shared by read commands.
func commonFlags(fs *flag.FlagSet) (fixture, labelsPath *string, asJSON *bool) {
	fixture = fs.String("sample", "testdata/sample-graph.json", "graph fixture (offline)")
	labelsPath = fs.String("labels", "testdata/labels.json", "labels file")
	asJSON = fs.Bool("json", false, "output JSON instead of tables")
	return
}

// buildSnapshot wires the offline sample adapter (live wiring lives in adapter/eigenlayer).
func buildSnapshot(fixture, labelsPath string) (snapshot.Snapshot, error) {
	a, err := sample.NewFromFile(fixture)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	var lp labels.Provider = labels.Noop{}
	if labelsPath != "" {
		if s, err := labels.LoadStatic(labelsPath); err == nil {
			lp = s
		}
	}
	e := engine.New([]adapter.Protocol{a}, lp)
	return e.Snapshot(context.Background())
}

// buildLiveSnapshot reads the exposure graph directly from chain via the
// EigenLayer live reader (used when --rpc is set).
func buildLiveSnapshot(rpc, lrtsPath, labelsPath string) (snapshot.Snapshot, error) {
	ctx := context.Background()
	reader, err := eigenlayer.NewLive(ctx, rpc)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	cfgs, err := eigenlayer.LoadConfigs(lrtsPath)
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	a := eigenlayer.New(reader, cfgs)
	var lp labels.Provider = labels.Noop{}
	if labelsPath != "" {
		if s, err := labels.LoadStatic(labelsPath); err == nil {
			lp = s
		}
	}
	e := engine.New([]adapter.Protocol{a}, lp)
	return e.Snapshot(ctx)
}

func cmdScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	fixture, labelsPath, asJSON := commonFlags(fs)
	out := fs.String("out", "", "write dataset JSON to this path")
	rpc := fs.String("rpc", "", "Ethereum RPC URL (live mode; reads on-chain instead of --sample)")
	lrtsPath := fs.String("lrts", "configs/lrts.json", "live LRT configs (used with --rpc)")
	fs.Parse(args)

	var s snapshot.Snapshot
	var err error
	if *rpc != "" {
		s, err = buildLiveSnapshot(*rpc, *lrtsPath, *labelsPath)
	} else {
		s, err = buildSnapshot(*fixture, *labelsPath)
	}
	must(err)
	if *asJSON {
		printJSON(s)
	} else {
		fmt.Print(render.Summary(s))
	}
	if *out != "" {
		must(snapshot.Write(*out, s))
		fmt.Fprintf(os.Stderr, "wrote dataset -> %s\n", *out)
	}
}

func cmdContagion(args []string) {
	fs := flag.NewFlagSet("contagion", flag.ExitOnError)
	fixture, labelsPath, asJSON := commonFlags(fs)
	fs.Parse(args)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	if *asJSON {
		printJSON(s.Contagion)
		return
	}
	fmt.Print(render.Contagion(s))
}

func cmdSystemic(args []string) {
	fs := flag.NewFlagSet("systemic", flag.ExitOnError)
	fixture, labelsPath, asJSON := commonFlags(fs)
	fs.Parse(args)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	if *asJSON {
		printJSON(s.Systemic)
		return
	}
	fmt.Print(render.Systemic(s))
}

func cmdWarnings(args []string) {
	fs := flag.NewFlagSet("warnings", flag.ExitOnError)
	fixture, labelsPath, asJSON := commonFlags(fs)
	fs.Parse(args)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	if *asJSON {
		printJSON(s.Warnings)
		return
	}
	fmt.Print(render.Warnings(s))
}

func cmdReport(args []string) {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	fixture, labelsPath, _ := commonFlags(fs)
	fs.Parse(args)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	fmt.Print(render.Report(s))
}

func cmdGraph(args []string) {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	fixture, labelsPath, _ := commonFlags(fs)
	dot := fs.Bool("dot", false, "emit Graphviz DOT")
	from := fs.String("from", "", "read graph from an existing dataset JSON instead of --sample")
	fs.Parse(args)

	var g graph.Graph
	if *from != "" {
		s, err := snapshot.Read(*from)
		must(err)
		g = s.Graph
	} else {
		s, err := buildSnapshot(*fixture, *labelsPath)
		must(err)
		g = s.Graph
	}
	if *dot {
		fmt.Print(graph.DOT(g))
		return
	}
	fmt.Fprintln(os.Stderr, "specify --dot to emit Graphviz DOT")
	os.Exit(2)
}

func cmdLRT(args []string) {
	fs := flag.NewFlagSet("lrt", flag.ExitOnError)
	fixture, labelsPath, asJSON := commonFlags(fs)
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: xray lrt <SYMBOL>")
		os.Exit(2)
	}
	sym := fs.Arg(0)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	for _, l := range s.Graph.LRTs {
		if l.Symbol == sym {
			if *asJSON {
				printJSON(map[string]any{"lrt": l, "concentration": s.Concentration[sym]})
				return
			}
			fmt.Printf("%s @ block %d\n  restaked: %s\n  concentration: %.3f\n  operators:\n", l.Symbol, s.Block, l.Restaked, s.Concentration[sym])
			for _, d := range l.Delegations {
				fmt.Printf("    %s  %s\n", d.Operator, d.Amount)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "LRT %q not found\n", sym)
	os.Exit(1)
}

func cmdOperator(args []string) {
	fs := flag.NewFlagSet("operator", flag.ExitOnError)
	fixture, labelsPath, asJSON := commonFlags(fs)
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: xray operator <ADDRESS>")
		os.Exit(2)
	}
	addr := fs.Arg(0)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	for _, o := range s.Systemic.Operators {
		if o.Operator == addr {
			if *asJSON {
				printJSON(o)
				return
			}
			fmt.Printf("operator %s (%s)\n  total restaked: %s\n  LRTs: %v\n  AVSs: %v\n", o.Operator, o.Name, o.TotalAmount, o.LRTs, o.AVSs)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "operator %q not found\n", addr)
	os.Exit(1)
}

func cmdAVS(args []string) {
	fs := flag.NewFlagSet("avs", flag.ExitOnError)
	fixture, labelsPath, asJSON := commonFlags(fs)
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: xray avs <ADDRESS>")
		os.Exit(2)
	}
	addr := fs.Arg(0)
	s, err := buildSnapshot(*fixture, *labelsPath)
	must(err)
	for _, a := range s.Systemic.AVSs {
		if a.AVS == addr {
			if *asJSON {
				printJSON(a)
				return
			}
			fmt.Printf("avs %s (%s)\n  exposed LRTs: %v\n  operators: %v\n", a.AVS, a.Name, a.LRTs, a.Operators)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "avs %q not found\n", addr)
	os.Exit(1)
}

func cmdDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output JSON instead of text")
	fs.Parse(args)
	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: xray diff <old.json> <new.json>")
		os.Exit(2)
	}
	a, err := snapshot.Read(fs.Arg(0))
	must(err)
	b, err := snapshot.Read(fs.Arg(1))
	must(err)
	d := graph.DiffGraphs(a.Graph, b.Graph)
	if *asJSON {
		printJSON(d)
		return
	}
	fmt.Printf("diff block %d -> %d\n", d.FromBlock, d.ToBlock)
	fmt.Printf("  LRTs:      +%v -%v\n", d.AddedLRTs, d.RemovedLRTs)
	fmt.Printf("  operators: +%v -%v\n", d.AddedOperators, d.RemovedOperators)
	fmt.Printf("  AVSs:      +%v -%v\n", d.AddedAVSs, d.RemovedAVSs)
	if len(d.Concentration) > 0 {
		fmt.Println("  concentration changes:")
		for _, c := range d.Concentration {
			fmt.Printf("    %s  %.3f -> %.3f  (%+.3f)\n", c.LRT, c.From, c.To, c.Delta)
		}
	}
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dataPath := fs.String("data", "data/latest.json", "snapshot JSON to serve")
	addr := fs.String("addr", ":8080", "listen address")
	fs.Parse(args)

	load := func() (snapshot.Snapshot, error) { return snapshot.Read(*dataPath) }
	if _, err := load(); err != nil {
		must(fmt.Errorf("cannot read %s: %w", *dataPath, err))
	}
	fmt.Printf("serving %s on %s\n", *dataPath, *addr)
	must(http.ListenAndServe(*addr, api.NewServer(load)))
}

func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	must(err)
	fmt.Println(string(b))
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
