package query

import (
	"path/filepath"
	"strings"
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
