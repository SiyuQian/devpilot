package store

const schemaSQL = `
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  path TEXT NOT NULL,
  name TEXT NOT NULL,
  container TEXT,
  language TEXT NOT NULL,
  start_line INTEGER,
  end_line INTEGER,
  is_exported INTEGER NOT NULL DEFAULT 0,
  signature_hash TEXT
);

CREATE TABLE IF NOT EXISTS edges (
  src TEXT NOT NULL,
  dst TEXT NOT NULL,
  kind TEXT NOT NULL,
  PRIMARY KEY (src, dst, kind)
);

CREATE INDEX IF NOT EXISTS idx_edges_dst_kind ON edges(dst, kind);
CREATE INDEX IF NOT EXISTS idx_edges_src_kind ON edges(src, kind);
CREATE INDEX IF NOT EXISTS idx_nodes_path ON nodes(path);
`

const currentSchemaVersion = 1
