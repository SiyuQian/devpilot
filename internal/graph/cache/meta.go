package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CurrentSchemaVersion mirrors store.currentSchemaVersion. Bump when the on-disk
// SQLite schema changes incompatibly; cache directories with a mismatched value
// are rebuilt from scratch.
const CurrentSchemaVersion = 1

// Meta is persisted to <cache>/meta.json alongside graph.db.
type Meta struct {
	SchemaVersion int      `json:"schema_version"`
	HeadSHA       string   `json:"head_sha"`
	ParserVersion string   `json:"parser_version"`
	Languages     []string `json:"languages"`
	BuiltAtUnix   int64    `json:"built_at_unix"`
}

// ReadMeta loads meta.json. Returns a zero Meta and nil error when the file is missing.
func ReadMeta(path string) (Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Meta{}, nil
		}
		return Meta{}, fmt.Errorf("read %s: %w", path, err)
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return Meta{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// WriteMeta atomically writes meta.json (write-temp-then-rename).
func WriteMeta(path string, m Meta) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}
