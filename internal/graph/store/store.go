package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if _, err := db.Exec(
		`INSERT INTO schema_version (version) SELECT ? WHERE NOT EXISTS (SELECT 1 FROM schema_version)`,
		currentSchemaVersion,
	); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("seed schema_version: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

type Node struct {
	ID, Kind, Path, Name, Container, Language string
	StartLine, EndLine                        int
	IsExported                                bool
	SignatureHash                             string
}

type Edge struct {
	Src, Dst, Kind string
}

// InsertNodes inserts or replaces nodes in a single transaction.
func (s *Store) InsertNodes(nodes []Node) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO nodes
		(id, kind, path, name, container, language, start_line, end_line, is_exported, signature_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()
	for _, n := range nodes {
		var exported int
		if n.IsExported {
			exported = 1
		}
		if _, err := stmt.Exec(n.ID, n.Kind, n.Path, n.Name, n.Container, n.Language,
			n.StartLine, n.EndLine, exported, n.SignatureHash); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// InsertEdges inserts edges, ignoring duplicates, in a single transaction.
func (s *Store) InsertEdges(edges []Edge) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO edges (src, dst, kind) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()
	for _, e := range edges {
		if _, err := stmt.Exec(e.Src, e.Dst, e.Kind); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetNode loads a node by ID.
func (s *Store) GetNode(id string) (Node, error) {
	var n Node
	var exported int
	var container, sigHash sql.NullString
	err := s.db.QueryRow(
		`SELECT id, kind, path, name, container, language, start_line, end_line, is_exported, signature_hash FROM nodes WHERE id = ?`,
		id,
	).Scan(
		&n.ID, &n.Kind, &n.Path, &n.Name, &container, &n.Language,
		&n.StartLine, &n.EndLine, &exported, &sigHash,
	)
	if err != nil {
		return n, err
	}
	n.Container = container.String
	n.SignatureHash = sigHash.String
	n.IsExported = exported == 1
	return n, nil
}

// CallersOf returns the source node IDs of all `calls` edges targeting id.
func (s *Store) CallersOf(id string) ([]string, error) {
	rows, err := s.db.Query(`SELECT src FROM edges WHERE dst = ? AND kind = 'calls'`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var src string
		if err := rows.Scan(&src); err != nil {
			return nil, err
		}
		out = append(out, src)
	}
	return out, rows.Err()
}
