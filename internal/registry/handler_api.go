package registry

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/foodrelay/foodrelay/internal/common"
)

// Handler implements the registry HTTP API.
type Handler struct {
	Store Store
}

// Announce handles POST /v1/nodes/announce.
func (h *Handler) Announce(w http.ResponseWriter, r *http.Request) {
	var req AnnounceRequest
	if err := common.ReadJSON(w, r, &req); err != nil {
		common.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if err := common.RequireFields(map[string]string{
		"node_id":      req.NodeID,
		"public_phone": req.PublicPhone,
		"public_key":   req.PublicKey,
		"signature":    req.Signature,
	}); err != nil {
		common.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Look up any existing record to decide which public key to verify against.
	// On first announce (TOFU) we use the provided key; on re-announce we require
	// a valid signature under the already-stored key so an outsider can't rotate it.
	existing, err := h.Store.GetNodeByID(req.NodeID)
	if err != nil {
		common.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	verifyKey := req.PublicKey // TOFU: first announce
	if existing != nil {
		verifyKey = existing.PublicKey
	}

	if err := VerifyAnnounceSignature(verifyKey, req.NodeID, req.PublicPhone, req.Timestamp, req.Signature); err != nil {
		log.Printf("[registry] announce verify failed node=%s: %v", req.NodeID, err)
		common.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "signature invalid"})
		return
	}

	caps, _ := json.Marshal(req.Capabilities)
	node := &Node{
		NodeID:           req.NodeID,
		PublicPhone:      req.PublicPhone,
		Lat:              req.Lat,
		Lon:              req.Lon,
		CoverageRadiusKM: req.CoverageRadiusKM,
		Capabilities:     string(caps),
		Version:          req.Version,
		PublicKey:        req.PublicKey,
		LastSeen:         time.Now(),
		AnnounceSig:      req.Signature,
		AnnounceTS:       req.Timestamp,
	}

	if err := h.Store.UpsertNode(node); err != nil {
		log.Printf("[registry] upsert node=%s: %v", req.NodeID, err)
		common.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}

	log.Printf("[registry] node announced: %s", req.NodeID)
	common.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Heartbeat handles POST /v1/nodes/heartbeat.
func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := common.ReadJSON(w, r, &req); err != nil {
		common.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	node, err := h.Store.GetNodeByID(req.NodeID)
	if err != nil {
		common.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	if node == nil {
		common.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "node not found; call /announce first"})
		return
	}

	if err := VerifySignature(node.PublicKey, req.NodeID, req.Timestamp, req.Signature); err != nil {
		log.Printf("[registry] heartbeat verify failed node=%s: %v", req.NodeID, err)
		common.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "signature invalid"})
		return
	}

	if err := h.Store.UpdateLastSeen(req.NodeID, time.Now()); err != nil {
		log.Printf("[registry] update last_seen node=%s: %v", req.NodeID, err)
		common.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListNodes handles GET /v1/nodes.
// Returns nodes active within the last 15 minutes.
func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-48 * time.Hour)
	nodes, err := h.Store.ListActiveNodes(since)
	if err != nil {
		common.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}

	var out []NodeResponse
	for _, n := range nodes {
		var caps []string
		json.Unmarshal([]byte(n.Capabilities), &caps) //nolint:errcheck
		out = append(out, NodeResponse{
			NodeID:           n.NodeID,
			PublicPhone:      n.PublicPhone,
			Lat:              n.Lat,
			Lon:              n.Lon,
			CoverageRadiusKM: n.CoverageRadiusKM,
			Capabilities:     caps,
			Version:          n.Version,
			LastSeen:         n.LastSeen.Format(time.RFC3339),
		})
	}
	if out == nil {
		out = []NodeResponse{}
	}
	common.WriteJSON(w, http.StatusOK, out)
}

// FederationNodes handles GET /v1/federation/nodes.
// Returns all nodes seen in the last 24 hours, including public_key and last_seen as int64.
// This endpoint is used exclusively by peer registries for gossip synchronisation.
func (h *Handler) FederationNodes(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-72 * time.Hour)
	nodes, err := h.Store.ListAllNodes(since)
	if err != nil {
		common.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}

	var out []FederationNode
	for _, n := range nodes {
		var caps []string
		json.Unmarshal([]byte(n.Capabilities), &caps) //nolint:errcheck
		src := n.Source
		if src == "" {
			src = "local"
		}
		out = append(out, FederationNode{
			NodeID:           n.NodeID,
			PublicPhone:      n.PublicPhone,
			Lat:              n.Lat,
			Lon:              n.Lon,
			CoverageRadiusKM: n.CoverageRadiusKM,
			Capabilities:     caps,
			Version:          n.Version,
			PublicKey:        n.PublicKey,
			LastSeen:         n.LastSeen.Unix(),
			Source:           src,
			AnnounceSig:      n.AnnounceSig,
			AnnounceTS:       n.AnnounceTS,
		})
	}
	if out == nil {
		out = []FederationNode{}
	}
	common.WriteJSON(w, http.StatusOK, out)
}

// HealthCheck handles GET /health.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	common.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

