// Package envelope defines the uniform JSON output shape used by every
// `devpilot graph <verb>` subcommand. Downstream skills depend on this exact
// shape; changes must bump the schema_version constant and add a new schema
// file under schemas/.
package envelope

import (
	"encoding/json"
	"time"
)

// SchemaVersion is the wire-protocol version of the envelope.
const SchemaVersion = "1"

// Envelope is the canonical CLI output shape.
type Envelope struct {
	SchemaVersionField  string   `json:"schema_version"`
	Command             string   `json:"command"`
	OKFlag              bool     `json:"ok"`
	Data                any      `json:"data"`
	Error               *ErrInfo `json:"error"`
	Warnings            []string `json:"warnings"`
	NextToolSuggestions []string `json:"next_tool_suggestions"`
	ElapsedMS           int64    `json:"elapsed_ms"`

	startedAt time.Time
}

// ErrInfo carries a structured error payload.
type ErrInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// New starts a builder for the given fully-qualified command (e.g. "graph.preflight").
func New(command string) *Envelope {
	return &Envelope{
		SchemaVersionField:  SchemaVersion,
		Command:             command,
		Warnings:            []string{},
		NextToolSuggestions: []string{},
		startedAt:           time.Now(),
	}
}

// OK marks the envelope as successful and attaches the payload.
func (e *Envelope) OK(data any) *Envelope {
	e.OKFlag = true
	e.Data = data
	e.Error = nil
	return e
}

// Err marks the envelope as failed and clears the payload.
func (e *Envelope) Err(code, msg string) *Envelope {
	e.OKFlag = false
	e.Data = nil
	e.Error = &ErrInfo{Code: code, Message: msg}
	return e
}

// Warn appends a non-fatal warning.
func (e *Envelope) Warn(msg string) *Envelope {
	e.Warnings = append(e.Warnings, msg)
	return e
}

// Suggest appends one or more next-tool hints.
func (e *Envelope) Suggest(cmds ...string) *Envelope {
	e.NextToolSuggestions = append(e.NextToolSuggestions, cmds...)
	return e
}

// Marshal finalises elapsed_ms and returns canonical JSON bytes.
func (e *Envelope) Marshal() ([]byte, error) {
	if e.ElapsedMS == 0 && !e.startedAt.IsZero() {
		e.ElapsedMS = time.Since(e.startedAt).Milliseconds()
	}
	return json.Marshal(e)
}
