package query

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// PreflightInput configures Preflight.
type PreflightInput struct {
	RepoRoot     string
	Base, Head   string
	HubThreshold int // default 10
	CallerSample int // default 10
	SymbolBudget int // default 50 (top-N by risk)
}

// PreflightResult mirrors §6 of the design doc; field names map to JSON
// keys when emitted via the envelope.
type PreflightResult struct {
	Mode             string                `json:"mode"`
	Graph            GraphMeta             `json:"graph"`
	ChangedSymbols   []ChangedSymbolDetail `json:"changed_symbols"`
	CrossCommunity   []CrossCommunityEdge  `json:"cross_community_edges"`
	RiskSummary      RiskSummary           `json:"risk_summary"`
	TruncatedSymbols []string              `json:"truncated_symbols"`
}

// GraphMeta holds metadata about the graph state.
type GraphMeta struct {
	Freshness    Freshness `json:"freshness"`
	Languages    []string  `json:"languages"`
	SkippedFiles []string  `json:"skipped_files"`
}

// Freshness describes whether the graph covers the requested revision.
type Freshness struct {
	CoversBaseSHA bool `json:"covers_base_sha"`
	StaleFiles    int  `json:"stale_files"`
}

// ChangedSymbolDetail is the per-symbol payload in PreflightResult.
type ChangedSymbolDetail struct {
	ID             string        `json:"id"`
	Kind           string        `json:"kind"`
	IsExported     bool          `json:"is_exported"`
	IsNew          bool          `json:"is_new"`
	ChangeType     string        `json:"change_type"`
	Callers        CallerSummary `json:"callers"`
	CalleesChanged []string      `json:"callees_changed"`
	Tests          TestSummary   `json:"tests"`
	ImplementorsOf []string      `json:"implementors_of"`
	Implements     []string      `json:"implements"`
	Community      string        `json:"community"`
	RiskFactors    []string      `json:"risk_factors"`
	Risk           int           `json:"-"` // used for sorting/truncation; not in §6 schema
}

// CallerSummary summarises inbound callers for a changed symbol.
type CallerSummary struct {
	Count  int      `json:"count"`
	InHub  bool     `json:"in_hub"`
	Sample []string `json:"sample"`
}

// TestSummary captures test coverage for a changed symbol.
type TestSummary struct {
	HasTests    bool     `json:"has_tests"`
	TestSymbols []string `json:"test_symbols"`
}

// CrossCommunityEdge represents a call that crosses community boundaries.
type CrossCommunityEdge struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	CountAdded int      `json:"count_added"`
	Samples    []string `json:"samples"`
}

// RiskSummary is the aggregate risk tally in PreflightResult.
type RiskSummary struct {
	HubNodesModified       int `json:"hub_nodes_modified"`
	UntestedPublicChanges  int `json:"untested_public_changes"`
	InterfaceChanges       int `json:"interface_changes"`
	NewCrossCommunityEdges int `json:"new_cross_community_edges"`
}

// hubSet is a quick membership lookup of hub-node IDs.
type hubSet map[string]bool

func (h hubSet) contains(id string) bool { return h[id] }

func enrichChangedSymbol(r Reader, ch ChangedSymbol, hubs hubSet, callerSample int) (ChangedSymbolDetail, error) {
	out := ChangedSymbolDetail{
		ID:         ch.ID,
		Kind:       ch.Kind,
		IsExported: ch.IsExported,
		IsNew:      ch.IsNew,
		ChangeType: ch.ChangeType,
	}
	node, err := r.GetNode(ch.ID)
	if err == nil {
		out.Community = communityFromPath(node.Path)
	} else {
		out.Community = communityFromPath(ch.ID)
	}

	count, err := r.CountEdgesByKind(ch.ID, "calls")
	if err != nil {
		return out, err
	}
	out.Callers.Count = count
	out.Callers.InHub = hubs.contains(ch.ID)

	callerEdges, err := r.EdgesByDst(ch.ID, "calls")
	if err != nil {
		return out, err
	}
	sample := pickCallerSample(r, callerEdges, callerSample)
	out.Callers.Sample = sample

	tests, err := TestsFor(r, ch.ID)
	if err != nil {
		return out, err
	}
	out.Tests = TestSummary{HasTests: len(tests) > 0, TestSymbols: tests}

	impls, err := ImplementorsOf(r, ch.ID)
	if err != nil {
		return out, err
	}
	if len(impls) > 0 {
		out.ImplementorsOf = impls
	}

	// `implements` edges originating from this symbol (struct/class) toward interfaces.
	implEdges, err := r.EdgesBySrc(ch.ID, "implements")
	if err != nil {
		return out, err
	}
	for _, e := range implEdges {
		out.Implements = append(out.Implements, e.Dst)
	}

	// Risk factors + score.
	factors := riskFactors(out, hubs)
	out.RiskFactors = factors
	out.Risk = RiskScore(RiskInputs{
		IsExported:      out.IsExported,
		InHub:           out.Callers.InHub,
		InterfaceChange: containsString(factors, "interface_change"),
		Untested:        !out.Tests.HasTests && out.IsExported,
	})
	return out, nil
}

func pickCallerSample(r Reader, edges []store.Edge, limit int) []string {
	type entry struct {
		id       string
		exported bool
	}
	pool := make([]entry, 0, len(edges))
	for _, e := range edges {
		n, err := r.GetNode(e.Src)
		if err != nil {
			pool = append(pool, entry{id: e.Src})
			continue
		}
		pool = append(pool, entry{id: e.Src, exported: n.IsExported})
	}
	sort.Slice(pool, func(i, j int) bool {
		if pool[i].exported != pool[j].exported {
			return pool[i].exported // true first
		}
		return pool[i].id < pool[j].id
	})
	if limit > 0 && len(pool) > limit {
		pool = pool[:limit]
	}
	out := make([]string, len(pool))
	for i, e := range pool {
		out[i] = e.id
	}
	return out
}

func riskFactors(d ChangedSymbolDetail, hubs hubSet) []string {
	var f []string
	if d.IsExported && !d.Tests.HasTests {
		f = append(f, "untested_public")
	}
	if hubs.contains(d.ID) {
		f = append(f, "hub")
	}
	if d.Kind == "interface" || len(d.ImplementorsOf) > 0 {
		f = append(f, "interface_change")
	}
	return f
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// communityFromPath returns the shallowest directory containing the file,
// capped at depth 3, per design §6 "Community definition".
func communityFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) <= 1 {
		return ""
	}
	dirs := parts[:len(parts)-1]
	if len(dirs) > 3 {
		dirs = dirs[:3]
	}
	return strings.Join(dirs, "/")
}
