package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pjdurden/restake-xray/graph"
	"github.com/pjdurden/restake-xray/snapshot"
)

func testSnap() snapshot.Snapshot {
	return snapshot.Build(graph.Graph{
		Protocol: "eigenlayer", Block: 42,
		LRTs:      []graph.LRT{{Symbol: "cmETH", Restaked: "1", Delegations: []graph.Delegation{{Operator: "op1", Amount: "1"}}}},
		Operators: []graph.Operator{{Address: "op1", Name: "P2P", AVSs: []string{"avs1"}}},
		AVSs:      []graph.AVS{{Address: "avs1", Name: "EigenDA"}},
	})
}

func TestRoutes(t *testing.T) {
	srv := NewServer(func() (snapshot.Snapshot, error) { return testSnap(), nil })
	for _, path := range []string{"/health", "/lrts", "/operators", "/avs", "/contagion", "/systemic", "/warnings", "/lrt/cmETH/exposure", "/operator/op1", "/avs/avs1"} {
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s -> %d", path, rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Fatalf("%s missing CORS header", path)
		}
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	var h map[string]any
	json.Unmarshal(rec.Body.Bytes(), &h)
	if h["block"].(float64) != 42 {
		t.Fatalf("health block: %v", h["block"])
	}
}

func TestUnknownLRT404(t *testing.T) {
	srv := NewServer(func() (snapshot.Snapshot, error) { return testSnap(), nil })
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/lrt/NOPE/exposure", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", rec.Code)
	}
}
