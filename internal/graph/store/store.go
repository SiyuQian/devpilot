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

// AllNodes returns every node currently in the database.
// Intended for cache.Builder; not for query hot-paths.
func (s *Store) AllNodes() ([]Node, error) {
	rows, err := s.db.Query(
		`SELECT id, kind, path, name, container, language, start_line, end_line, is_exported, signature_hash FROM nodes`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Node
	for rows.Next() {
		var n Node
		var container, sigHash sql.NullString
		var exported int
		if err := rows.Scan(
			&n.ID, &n.Kind, &n.Path, &n.Name, &container, &n.Language,
			&n.StartLine, &n.EndLine, &exported, &sigHash,
		); err != nil {
			return nil, err
		}
		n.Container = container.String
		n.SignatureHash = sigHash.String
		n.IsExported = exported == 1
		out = append(out, n)
	}
	return out, rows.Err()
}

// DeleteByPaths deletes every node whose path is in paths, plus edges whose
// src or dst belongs to a deleted node. Returns counts deleted.
func (s *Store) DeleteByPaths(paths []string) (nodes, edges int, err error) {
	if len(paths) == 0 {
		return 0, 0, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var ids []string
	for _, p := range paths {
		rows, err := tx.Query(`SELECT id FROM nodes WHERE path = ?`, p)
		if err != nil {
			return 0, 0, err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return 0, 0, err
			}
			ids = append(ids, id)
		}
		_ = rows.Close()
	}
	for _, id := range ids {
		r, err := tx.Exec(`DELETE FROM edges WHERE src = ? OR dst = ?`, id, id)
		if err != nil {
			return 0, 0, err
		}
		n, _ := r.RowsAffected()
		edges += int(n)
	}
	for _, p := range paths {
		r, err := tx.Exec(`DELETE FROM nodes WHERE path = ?`, p)
		if err != nil {
			return 0, 0, err
		}
		n, _ := r.RowsAffected()
		nodes += int(n)
	}
	return nodes, edges, tx.Commit()
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

// EdgesByDst returns all edges of the given kind that end at dst.
func (s *Store) EdgesByDst(dst, kind string) ([]Edge, error) {
	rows, err := s.db.Query(`SELECT src, dst, kind FROM edges WHERE dst = ? AND kind = ?`, dst, kind)
	if err != nil {
		return nil, fmt.Errorf("EdgesByDst: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Src, &e.Dst, &e.Kind); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EdgesBySrc returns all edges of the given kind that start at src.
func (s *Store) EdgesBySrc(src, kind string) ([]Edge, error) {
	rows, err := s.db.Query(`SELECT src, dst, kind FROM edges WHERE src = ? AND kind = ?`, src, kind)
	if err != nil {
		return nil, fmt.Errorf("EdgesBySrc: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Src, &e.Dst, &e.Kind); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// NodesByPath returns all nodes whose path equals the given path (excluding the file node itself).
func (s *Store) NodesByPath(path string) ([]Node, error) {
	rows, err := s.db.Query(
		`SELECT id, kind, path, name, container, language, start_line, end_line, is_exported, signature_hash
		 FROM nodes WHERE path = ? AND kind != 'file'`, path)
	if err != nil {
		return nil, fmt.Errorf("NodesByPath: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Node
	for rows.Next() {
		var n Node
		var exp int
		var container, sigHash sql.NullString
		if err := rows.Scan(&n.ID, &n.Kind, &n.Path, &n.Name, &container, &n.Language,
			&n.StartLine, &n.EndLine, &exp, &sigHash); err != nil {
			return nil, err
		}
		n.Container = container.String
		n.SignatureHash = sigHash.String
		n.IsExported = exp == 1
		out = append(out, n)
	}
	return out, rows.Err()
}

// CountEdgesByKind returns the inbound edge count toward dst of the given kind.
func (s *Store) CountEdgesByKind(dst, kind string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM edges WHERE dst = ? AND kind = ?`, dst, kind).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("CountEdgesByKind: %w", err)
	}
	return n, nil
}

// CountNodes returns the total number of node rows.
func (s *Store) CountNodes() (int, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&n); err != nil {
		return 0, fmt.Errorf("CountNodes: %w", err)
	}
	return n, nil
}

// CountEdges returns the total number of edge rows.
func (s *Store) CountEdges() (int, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&n); err != nil {
		return 0, fmt.Errorf("CountEdges: %w", err)
	}
	return n, nil
}

// HubsByCalls returns dst IDs with at least minCallers inbound `calls` edges,
// ordered by caller count descending then id ascending for determinism.
func (s *Store) HubsByCalls(minCallers int) ([]struct {
	ID    string
	Count int
}, error) {
	rows, err := s.db.Query(
		`SELECT dst, COUNT(*) AS c FROM edges WHERE kind='calls'
		   GROUP BY dst HAVING c >= ?
		   ORDER BY c DESC, dst ASC`, minCallers)
	if err != nil {
		return nil, fmt.Errorf("HubsByCalls: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []struct {
		ID    string
		Count int
	}
	for rows.Next() {
		var e struct {
			ID    string
			Count int
		}
		if err := rows.Scan(&e.ID, &e.Count); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
