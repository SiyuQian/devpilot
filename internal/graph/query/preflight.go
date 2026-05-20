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

// Preflight composes DetectChanges + enrichment + cross-community detection
// into the §6 payload. It does not write anything.
func Preflight(r Reader, in PreflightInput) (PreflightResult, error) {
	if in.HubThreshold <= 0 {
		in.HubThreshold = 10
	}
	if in.CallerSample <= 0 {
		in.CallerSample = 10
	}
	if in.SymbolBudget <= 0 {
		in.SymbolBudget = 50
	}

	changes, err := DetectChanges(r, in.RepoRoot, in.Base, in.Head)
	if err != nil {
		return PreflightResult{}, err
	}

	hubs, err := Hubs(r, in.HubThreshold)
	if err != nil {
		return PreflightResult{}, err
	}
	set := hubSet{}
	for _, h := range hubs {
		set[h.ID] = true
	}

	details := make([]ChangedSymbolDetail, 0, len(changes))
	for _, ch := range changes {
		if ch.Kind == "file" {
			details = append(details, ChangedSymbolDetail{
				ID:         ch.ID,
				Kind:       "file",
				ChangeType: ch.ChangeType,
				IsNew:      ch.IsNew,
				Community:  communityFromPath(ch.ID),
			})
			continue
		}
		d, err := enrichChangedSymbol(r, ch, set, in.CallerSample)
		if err != nil {
			return PreflightResult{}, err
		}
		details = append(details, d)
	}

	// Rank by risk descending, then by id for determinism.
	sort.SliceStable(details, func(i, j int) bool {
		if details[i].Risk != details[j].Risk {
			return details[i].Risk > details[j].Risk
		}
		return details[i].ID < details[j].ID
	})

	var truncated []string
	if len(details) > in.SymbolBudget {
		for _, d := range details[in.SymbolBudget:] {
			truncated = append(truncated, d.ID)
		}
		details = details[:in.SymbolBudget]
	}

	cross := crossCommunityEdges(r, details)
	summary := buildRiskSummary(details, cross)

	return PreflightResult{
		Mode:             "built",
		Graph:            GraphMeta{Freshness: Freshness{CoversBaseSHA: true}, Languages: detectLanguages(r), SkippedFiles: nil},
		ChangedSymbols:   details,
		CrossCommunity:   cross,
		RiskSummary:      summary,
		TruncatedSymbols: truncated,
	}, nil
}

func crossCommunityEdges(r Reader, details []ChangedSymbolDetail) []CrossCommunityEdge {
	type key struct{ from, to string }
	agg := map[key]*CrossCommunityEdge{}
	for _, d := range details {
		dstNode, err := r.GetNode(d.ID)
		if err != nil {
			continue
		}
		toCom := communityFromPath(dstNode.Path)
		for _, callerID := range d.Callers.Sample {
			n, err := r.GetNode(callerID)
			if err != nil {
				continue
			}
			fromCom := communityFromPath(n.Path)
			if fromCom == "" || fromCom == toCom {
				continue
			}
			k := key{fromCom, toCom}
			e, ok := agg[k]
			if !ok {
				e = &CrossCommunityEdge{From: fromCom, To: toCom}
				agg[k] = e
			}
			e.CountAdded++
			if len(e.Samples) < 5 {
				e.Samples = append(e.Samples, callerID+" → "+d.ID)
			}
		}
	}
	out := make([]CrossCommunityEdge, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

func buildRiskSummary(details []ChangedSymbolDetail, cross []CrossCommunityEdge) RiskSummary {
	var s RiskSummary
	for _, d := range details {
		if d.Callers.InHub {
			s.HubNodesModified++
		}
		if d.IsExported && !d.Tests.HasTests {
			s.UntestedPublicChanges++
		}
		if containsString(d.RiskFactors, "interface_change") {
			s.InterfaceChanges++
		}
	}
	for _, c := range cross {
		s.NewCrossCommunityEdges += c.CountAdded
	}
	return s
}

func detectLanguages(r Reader) []string {
	all, err := r.AllNodes()
	if err != nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, n := range all {
		if n.Language != "" {
			set[n.Language] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for l := range set {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
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
