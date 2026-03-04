package node

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) the SQLite database.
// Returns a Store and the underlying *sql.DB (for migrations).
func NewSQLiteStore(dsn string) (Store, *sql.DB, error) {
	db, err := sql.Open("sqlite", dsn+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, nil, fmt.Errorf("open node db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	return &sqliteStore{db: db}, db, nil
}

// ── Offers ────────────────────────────────────────────────────────────────────

func (s *sqliteStore) CreateOffer(o *Offer) error {
	_, err := s.db.Exec(`
		INSERT INTO offers (phone, description, window_start, window_end, location, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		o.Phone, o.Description,
		toUnix(o.WindowStart), toUnix(o.WindowEnd),
		o.Location, o.Status,
	)
	return err
}

func (s *sqliteStore) GetOfferByID(id int64) (*Offer, error) {
	row := s.db.QueryRow(`
		SELECT id, phone, description, window_start, window_end, location, status, created_at, updated_at
		FROM offers WHERE id = ?`, id)
	return scanOffer(row)
}

func (s *sqliteStore) GetLatestOfferByPhone(phone string) (*Offer, error) {
	row := s.db.QueryRow(`
		SELECT id, phone, description, window_start, window_end, location, status, created_at, updated_at
		FROM offers WHERE phone = ? ORDER BY id DESC LIMIT 1`, phone)
	return scanOffer(row)
}

func (s *sqliteStore) GetOffersByStatus(status string) ([]*Offer, error) {
	rows, err := s.db.Query(`
		SELECT id, phone, description, window_start, window_end, location, status, created_at, updated_at
		FROM offers WHERE status = ? ORDER BY id ASC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOffers(rows)
}

func (s *sqliteStore) UpdateOfferStatus(id int64, status string) error {
	_, err := s.db.Exec(
		`UPDATE offers SET status = ?, updated_at = strftime('%s','now') WHERE id = ?`,
		status, id,
	)
	return err
}

func (s *sqliteStore) ExpireOffers(before time.Time) (int64, error) {
	res, err := s.db.Exec(`
		UPDATE offers
		SET status = 'expired', updated_at = strftime('%s','now')
		WHERE status IN ('awaiting_ready','ready')
		  AND window_end IS NOT NULL
		  AND window_end < ?`, before.Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ── Needs ─────────────────────────────────────────────────────────────────────

func (s *sqliteStore) CreateNeed(n *Need) error {
	_, err := s.db.Exec(`
		INSERT INTO needs (phone, location, status)
		VALUES (?, ?, ?)`,
		n.Phone, n.Location, n.Status,
	)
	return err
}

func (s *sqliteStore) GetNeedByID(id int64) (*Need, error) {
	row := s.db.QueryRow(`
		SELECT id, phone, location, status, created_at, updated_at
		FROM needs WHERE id = ?`, id)
	return scanNeed(row)
}

func (s *sqliteStore) GetLatestNeedByPhone(phone string) (*Need, error) {
	row := s.db.QueryRow(`
		SELECT id, phone, location, status, created_at, updated_at
		FROM needs WHERE phone = ? ORDER BY id DESC LIMIT 1`, phone)
	return scanNeed(row)
}

func (s *sqliteStore) GetNeedsByStatus(status string) ([]*Need, error) {
	rows, err := s.db.Query(`
		SELECT id, phone, location, status, created_at, updated_at
		FROM needs WHERE status = ? ORDER BY id ASC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNeeds(rows)
}

func (s *sqliteStore) UpdateNeedStatus(id int64, status string) error {
	_, err := s.db.Exec(
		`UPDATE needs SET status = ?, updated_at = strftime('%s','now') WHERE id = ?`,
		status, id,
	)
	return err
}

// ExpireNeeds expires open needs older than the cutoff time.
// The scheduler passes now.Add(-24*time.Hour) so needs live for 24 hours.
func (s *sqliteStore) ExpireNeeds(before time.Time) (int64, error) {
	res, err := s.db.Exec(`
		UPDATE needs
		SET status = 'expired', updated_at = strftime('%s','now')
		WHERE status = 'open'
		  AND created_at < ?`, before.Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ── Jobs ──────────────────────────────────────────────────────────────────────

func (s *sqliteStore) CreateJob(j *Job) error {
	_, err := s.db.Exec(`
		INSERT INTO jobs (offer_id, need_id, carrier, status, match_score, match_reason)
		VALUES (?, ?, ?, ?, ?, ?)`,
		j.OfferID, j.NeedID, j.Carrier, j.Status, j.MatchScore, j.MatchReason,
	)
	return err
}

func (s *sqliteStore) GetJobByOfferID(offerID int64) (*Job, error) {
	row := s.db.QueryRow(`
		SELECT id, offer_id, need_id, carrier, status, match_score, match_reason, created_at, updated_at
		FROM jobs WHERE offer_id = ? ORDER BY id DESC LIMIT 1`, offerID)
	return scanJob(row)
}

func (s *sqliteStore) UpdateJobStatus(id int64, status string) error {
	_, err := s.db.Exec(
		`UPDATE jobs SET status = ?, updated_at = strftime('%s','now') WHERE id = ?`,
		status, id,
	)
	return err
}

// ── scan helpers ──────────────────────────────────────────────────────────────

func scanOffer(row *sql.Row) (*Offer, error) {
	var o Offer
	var ws, we sql.NullInt64
	var createdAt, updatedAt int64
	err := row.Scan(
		&o.ID, &o.Phone, &o.Description,
		&ws, &we,
		&o.Location, &o.Status,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	o.WindowStart = fromUnix(ws)
	o.WindowEnd = fromUnix(we)
	o.CreatedAt = time.Unix(createdAt, 0).UTC()
	o.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return &o, nil
}

func scanOffers(rows *sql.Rows) ([]*Offer, error) {
	var out []*Offer
	for rows.Next() {
		var o Offer
		var ws, we sql.NullInt64
		var createdAt, updatedAt int64
		if err := rows.Scan(
			&o.ID, &o.Phone, &o.Description,
			&ws, &we,
			&o.Location, &o.Status,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		o.WindowStart = fromUnix(ws)
		o.WindowEnd = fromUnix(we)
		o.CreatedAt = time.Unix(createdAt, 0).UTC()
		o.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		out = append(out, &o)
	}
	return out, rows.Err()
}

func scanNeed(row *sql.Row) (*Need, error) {
	var n Need
	var createdAt, updatedAt int64
	err := row.Scan(
		&n.ID, &n.Phone, &n.Location, &n.Status,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.CreatedAt = time.Unix(createdAt, 0).UTC()
	n.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return &n, nil
}

func scanNeeds(rows *sql.Rows) ([]*Need, error) {
	var out []*Need
	for rows.Next() {
		var n Need
		var createdAt, updatedAt int64
		if err := rows.Scan(
			&n.ID, &n.Phone, &n.Location, &n.Status,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		n.CreatedAt = time.Unix(createdAt, 0).UTC()
		n.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		out = append(out, &n)
	}
	return out, rows.Err()
}

func scanJob(row *sql.Row) (*Job, error) {
	var j Job
	var createdAt, updatedAt int64
	err := row.Scan(
		&j.ID, &j.OfferID, &j.NeedID, &j.Carrier, &j.Status,
		&j.MatchScore, &j.MatchReason,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	j.CreatedAt = time.Unix(createdAt, 0).UTC()
	j.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return &j, nil
}

// ── time helpers ──────────────────────────────────────────────────────────────

// toUnix converts a time.Time to a Unix int64, or nil if zero.
func toUnix(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.Unix()
}

// fromUnix converts a nullable int64 to time.Time.
func fromUnix(n sql.NullInt64) time.Time {
	if !n.Valid {
		return time.Time{}
	}
	return time.Unix(n.Int64, 0).UTC()
}
