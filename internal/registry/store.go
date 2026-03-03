package registry

import "time"

// Store is the registry's data access interface.
type Store interface {
	UpsertNode(n *Node) error
	GetNodeByID(nodeID string) (*Node, error)
	UpdateLastSeen(nodeID string, t time.Time) error
	ListActiveNodes(since time.Time) ([]*Node, error)
	// MergeNode inserts or updates a node record from a peer registry.
	// The caller must verify the announce signature before calling this.
	// On INSERT, all fields from the peer are stored. On UPDATE (conflict on node_id),
	// only mutable fields are updated using "newest last_seen wins" logic; identity
	// fields (public_phone, lat, lon, coverage_radius_km, public_key) are never overwritten.
	MergeNode(n *Node) error
	// ListAllNodes returns all nodes with last_seen after since (generous window, for federation sync).
	ListAllNodes(since time.Time) ([]*Node, error)
}
