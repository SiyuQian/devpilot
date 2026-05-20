// Package query implements the read-side graph operations consumed by the
// devpilot CLI and the devpilot-pr-review skill.
//
// All queries are pure functions of a Reader plus primitive arguments; they
// never mutate the underlying store.
package query

import "github.com/siyuqian/devpilot/internal/graph/store"

// Reader is the narrow read-only surface that every query depends on. It is
// satisfied by *store.Store; tests construct in-memory stores to feed it.
type Reader interface {
	GetNode(id string) (store.Node, error)
	NodesByPath(path string) ([]store.Node, error)
	AllNodes() ([]store.Node, error)
	EdgesByDst(dst, kind string) ([]store.Edge, error)
	EdgesBySrc(src, kind string) ([]store.Edge, error)
	CountEdgesByKind(dst, kind string) (int, error)
}
