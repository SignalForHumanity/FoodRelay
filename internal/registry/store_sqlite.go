package registry

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens the registry SQLite database.
func NewSQLiteStore(dsn string) (Store, *sql.DB, error) {
	db, err := sql.Open("sqlite", dsn+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, nil, fmt.Errorf("open registry db: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &sqliteStore{db: db}, db, nil
}

// UpsertNode inserts or replaces a node record (used for direct announce).
// n.Capabilities must already be a JSON string (e.g. `["sms"]`).
func (s *sqliteStore) UpsertNode(n *Node) error {
	source := n.Source
	if source == "" {
		source = "local"
	}
	_, err := s.db.Exec(`
		INSERT INTO nodes
			(node_id, public_phone, lat, lon, coverage_radius_km, capabilities, version,
			 public_key, last_seen, announce_sig, announce_ts, source, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%s','now'))
		ON CONFLICT(node_id) DO UPDATE SET
			public_phone       = excluded.public_phone,
			lat                = excluded.lat,
			lon                = excluded.lon,
			coverage_radius_km = excluded.coverage_radius_km,
			capabilities       = excluded.capabilities,
			version            = excluded.version,
			public_key         = excluded.public_key,
			last_seen          = excluded.last_seen,
			announce_sig       = excluded.announce_sig,
			announce_ts        = excluded.announce_ts,
			source             = excluded.source,
			updated_at         = strftime('%s','now')`,
		n.NodeID, n.PublicPhone,
		n.Lat, n.Lon, n.CoverageRadiusKM,
		n.Capabilities, n.Version, n.PublicKey,
		toUnix(n.LastSeen), n.AnnounceSig, n.AnnounceTS, source,
	)
	return err
}

// GetNodeByID returns a node by its node_id, or nil if not found.
func (s *sqliteStore) GetNodeByID(nodeID string) (*Node, error) {
	row := s.db.QueryRow(`
		SELECT id, node_id, public_phone, lat, lon, coverage_radius_km, capabilities, version,
		       public_key, last_seen, created_at, updated_at, source, announce_sig, announce_ts
		FROM nodes WHERE node_id = ?`, nodeID)
	return scanNode(row)
}

// UpdateLastSeen sets last_seen for a node.
func (s *sqliteStore) UpdateLastSeen(nodeID string, t time.Time) error {
	_, err := s.db.Exec(
		`UPDATE nodes SET last_seen = ?, updated_at = strftime('%s','now') WHERE node_id = ?`,
		t.Unix(), nodeID,
	)
	return err
}

// ListActiveNodes returns nodes with last_seen after since.
func (s *sqliteStore) ListActiveNodes(since time.Time) ([]*Node, error) {
	rows, err := s.db.Query(`
		SELECT id, node_id, public_phone, lat, lon, coverage_radius_km, capabilities, version,
		       public_key, last_seen, created_at, updated_at, source, announce_sig, announce_ts
		FROM nodes
		WHERE last_seen IS NOT NULL AND last_seen > ?
		ORDER BY node_id ASC`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// MergeNode inserts or updates a node record from a peer registry.
// On INSERT (new record): all fields from the verified peer payload are stored.
// On UPDATE (existing record): only mutable fields (capabilities, version, last_seen,
// announce_sig, announce_ts, source) are updated using "newest last_seen wins" logic.
// Identity fields (public_phone, lat, lon, coverage_radius_km, public_key) are never
// overwritten — only a direct announce can change those.
//
// The caller (FederationSyncer) must verify the announce signature before calling this.
func (s *sqliteStore) MergeNode(n *Node) error {
	source := n.Source
	if source == "" {
		source = "federation"
	}
	_, err := s.db.Exec(`
		INSERT INTO nodes
			(node_id, public_phone, lat, lon, coverage_radius_km, capabilities, version,
			 public_key, last_seen, announce_sig, announce_ts, source, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%s','now'))
		ON CONFLICT(node_id) DO UPDATE SET
			capabilities = CASE WHEN excluded.last_seen >= nodes.last_seen THEN excluded.capabilities ELSE nodes.capabilities END,
			version      = CASE WHEN excluded.last_seen >= nodes.last_seen THEN excluded.version      ELSE nodes.version      END,
			last_seen    = MAX(nodes.last_seen, excluded.last_seen),
			announce_sig = CASE WHEN excluded.last_seen >= nodes.last_seen THEN excluded.announce_sig ELSE nodes.announce_sig END,
			announce_ts  = CASE WHEN excluded.last_seen >= nodes.last_seen THEN excluded.announce_ts  ELSE nodes.announce_ts  END,
			source       = CASE WHEN excluded.last_seen >= nodes.last_seen THEN excluded.source       ELSE nodes.source       END,
			updated_at   = strftime('%s','now')`,
		// public_phone, lat, lon, coverage_radius_km, public_key: only set on first INSERT
		n.NodeID, n.PublicPhone,
		n.Lat, n.Lon, n.CoverageRadiusKM,
		n.Capabilities, n.Version, n.PublicKey,
		toUnix(n.LastSeen), n.AnnounceSig, n.AnnounceTS, source,
	)
	return err
}

// ListAllNodes returns all nodes with last_seen after since (broad window for federation sync).
func (s *sqliteStore) ListAllNodes(since time.Time) ([]*Node, error) {
	rows, err := s.db.Query(`
		SELECT id, node_id, public_phone, lat, lon, coverage_radius_km, capabilities, version,
		       public_key, last_seen, created_at, updated_at, source, announce_sig, announce_ts
		FROM nodes
		WHERE last_seen IS NOT NULL AND last_seen > ?
		ORDER BY node_id ASC`, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// ── scan helpers ──────────────────────────────────────────────────────────────

func scanNode(row *sql.Row) (*Node, error) {
	var n Node
	var lastSeen sql.NullInt64
	var createdAt, updatedAt int64
	err := row.Scan(
		&n.ID, &n.NodeID, &n.PublicPhone,
		&n.Lat, &n.Lon, &n.CoverageRadiusKM,
		&n.Capabilities, &n.Version, &n.PublicKey,
		&lastSeen, &createdAt, &updatedAt,
		&n.Source, &n.AnnounceSig, &n.AnnounceTS,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastSeen.Valid {
		n.LastSeen = time.Unix(lastSeen.Int64, 0).UTC()
	}
	n.CreatedAt = time.Unix(createdAt, 0).UTC()
	n.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return &n, nil
}

func scanNodes(rows *sql.Rows) ([]*Node, error) {
	var out []*Node
	for rows.Next() {
		var n Node
		var lastSeen sql.NullInt64
		var createdAt, updatedAt int64
		if err := rows.Scan(
			&n.ID, &n.NodeID, &n.PublicPhone,
			&n.Lat, &n.Lon, &n.CoverageRadiusKM,
			&n.Capabilities, &n.Version, &n.PublicKey,
			&lastSeen, &createdAt, &updatedAt,
			&n.Source, &n.AnnounceSig, &n.AnnounceTS,
		); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			n.LastSeen = time.Unix(lastSeen.Int64, 0).UTC()
		}
		n.CreatedAt = time.Unix(createdAt, 0).UTC()
		n.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		out = append(out, &n)
	}
	return out, rows.Err()
}

func toUnix(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.Unix()
}
