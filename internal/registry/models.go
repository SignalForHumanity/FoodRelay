package registry

import "time"

// Node represents a registered FoodRelay node.
// Privacy: we store ONLY public metadata — never offer/need details or private phones.
type Node struct {
	ID               int64
	NodeID           string
	PublicPhone      string
	Lat              float64
	Lon              float64
	CoverageRadiusKM float64
	Capabilities     string // JSON array string
	Version          string
	PublicKey        string // hex-encoded ed25519 public key
	LastSeen         time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Source           string // 'local' (direct announce) or 'federation:<peerURL>' (replicated from peer)
	AnnounceSig      string // original ed25519 signature from the announce request
	AnnounceTS       int64  // Unix timestamp used when the announce was signed
}

// FederationNode is the wire format used by GET /v1/federation/nodes.
// It carries the original announce signature so peer registries can verify
// a record before inserting it — preventing phantom node injection.
type FederationNode struct {
	NodeID           string   `json:"node_id"`
	PublicPhone      string   `json:"public_phone"`
	Lat              float64  `json:"lat"`
	Lon              float64  `json:"lon"`
	CoverageRadiusKM float64  `json:"coverage_radius_km"`
	Capabilities     []string `json:"capabilities"`
	Version          string   `json:"version"`
	PublicKey        string   `json:"public_key"`
	LastSeen         int64    `json:"last_seen"`    // Unix epoch for easy comparison
	Source           string   `json:"source"`
	AnnounceSig      string   `json:"announce_sig"` // original announce signature
	AnnounceTS       int64    `json:"announce_ts"`  // timestamp used when announce was signed
}

// AnnounceRequest is the body for POST /v1/nodes/announce.
type AnnounceRequest struct {
	NodeID           string   `json:"node_id"`
	PublicPhone      string   `json:"public_phone"`
	Lat              float64  `json:"lat"`
	Lon              float64  `json:"lon"`
	CoverageRadiusKM float64  `json:"coverage_radius_km"`
	Capabilities     []string `json:"capabilities"`
	Version          string   `json:"version"`
	PublicKey        string   `json:"public_key"`
	Timestamp        int64    `json:"timestamp"`
	Signature        string   `json:"signature"`
}

// HeartbeatRequest is the body for POST /v1/nodes/heartbeat.
type HeartbeatRequest struct {
	NodeID    string `json:"node_id"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

// NodeResponse is the public representation returned by GET /v1/nodes.
type NodeResponse struct {
	NodeID           string   `json:"node_id"`
	PublicPhone      string   `json:"public_phone"`
	Lat              float64  `json:"lat"`
	Lon              float64  `json:"lon"`
	CoverageRadiusKM float64  `json:"coverage_radius_km"`
	Capabilities     []string `json:"capabilities"`
	Version          string   `json:"version"`
	LastSeen         string   `json:"last_seen"` // RFC3339
}
