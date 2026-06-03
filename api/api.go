// Package api serves a read-only HTTP/JSON view of the latest snapshot.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/prajjwalchittori/restake-xray/snapshot"
)

// Loader returns the current snapshot to serve.
type Loader func() (snapshot.Snapshot, error)

// NewServer builds the HTTP handler. load is called per request so the served
// data refreshes when the underlying snapshot changes.
func NewServer(load Loader) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		s, err := load()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "protocol": s.Protocol, "block": s.Block, "timestamp": s.Timestamp})
	})
	mux.HandleFunc("GET /lrts", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Graph.LRTs })
	})
	mux.HandleFunc("GET /operators", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Graph.Operators })
	})
	mux.HandleFunc("GET /avs", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Graph.AVSs })
	})
	mux.HandleFunc("GET /contagion", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Contagion })
	})
	mux.HandleFunc("GET /systemic", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Systemic })
	})
	mux.HandleFunc("GET /warnings", func(w http.ResponseWriter, r *http.Request) {
		serve(w, load, func(s snapshot.Snapshot) any { return s.Warnings })
	})
	mux.HandleFunc("GET /operator/{addr}", func(w http.ResponseWriter, r *http.Request) {
		s, err := load()
		if err != nil {
			writeErr(w, err)
			return
		}
		addr := r.PathValue("addr")
		for _, o := range s.Systemic.Operators {
			if o.Operator == addr {
				writeJSON(w, o)
				return
			}
		}
		http.Error(w, "operator not found", http.StatusNotFound)
	})
	mux.HandleFunc("GET /avs/{addr}", func(w http.ResponseWriter, r *http.Request) {
		s, err := load()
		if err != nil {
			writeErr(w, err)
			return
		}
		addr := r.PathValue("addr")
		for _, a := range s.Systemic.AVSs {
			if a.AVS == addr {
				writeJSON(w, a)
				return
			}
		}
		http.Error(w, "avs not found", http.StatusNotFound)
	})
	mux.HandleFunc("GET /lrt/{sym}/exposure", func(w http.ResponseWriter, r *http.Request) {
		s, err := load()
		if err != nil {
			writeErr(w, err)
			return
		}
		sym := r.PathValue("sym")
		for _, l := range s.Graph.LRTs {
			if l.Symbol == sym {
				writeJSON(w, map[string]any{"lrt": l, "concentration": s.Concentration[sym]})
				return
			}
		}
		http.Error(w, "lrt not found", http.StatusNotFound)
	})

	return cors(mux)
}

func serve(w http.ResponseWriter, load Loader, pick func(snapshot.Snapshot) any) {
	s, err := load()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, pick(s))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	})
}
