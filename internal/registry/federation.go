package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/foodrelay/foodrelay/internal/common"
)

// federationClient is used for all peer-sync HTTP requests.
// A 20-second timeout prevents a slow peer from blocking the sync goroutine indefinitely.
var federationClient = &http.Client{Timeout: 20 * time.Second}

// maxFederationResponseBytes caps how much data we read from a peer (1 MB).
const maxFederationResponseBytes = 1 << 20

// FederationSyncer pulls node records from peer registries every 5 minutes
// and merges them into the local store using "newest last_seen wins" logic.
type FederationSyncer struct {
	Store Store
	Peers []string
	Self  string // this registry's own URL — skipped to avoid self-sync
}

// Run starts the periodic gossip loop. Blocks until stop is closed.
func (f *FederationSyncer) Run(stop <-chan struct{}) {
	if len(f.Peers) == 0 {
		log.Println("[federation] no peers configured; skipping federation sync")
		return
	}

	// Sync immediately on startup, then every 5 minutes.
	f.syncAll()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.syncAll()
		case <-stop:
			return
		}
	}
}

func (f *FederationSyncer) syncAll() {
	for _, peer := range f.Peers {
		if f.Self != "" && sameOrigin(peer, f.Self) {
			continue // skip self
		}
		f.syncPeer(peer)
	}
}

func (f *FederationSyncer) syncPeer(peerURL string) {
	endpoint := strings.TrimRight(peerURL, "/") + "/v1/federation/nodes"
	resp, err := federationClient.Get(endpoint)
	if err != nil {
		log.Printf("[federation] fetch %s: %v", endpoint, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[federation] peer %s returned %d", endpoint, resp.StatusCode)
		return
	}

	var nodes []FederationNode
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxFederationResponseBytes)).Decode(&nodes); err != nil {
		log.Printf("[federation] decode %s: %v", endpoint, err)
		return
	}

	merged, skipped := 0, 0
	for i := range nodes {
		fn := &nodes[i]

		// Verify the original announce signature before accepting the record.
		// This proves the phone number in the payload was signed by the node's
		// private key — a compromised peer cannot substitute an arbitrary phone.
		if fn.AnnounceSig == "" {
			log.Printf("[federation] node %s from %s has no announce_sig, skipping", fn.NodeID, peerURL)
			skipped++
			continue
		}
		msg := AnnounceSignatureMessage(fn.NodeID, fn.PublicPhone, fn.AnnounceTS)
		if !common.Verify(fn.PublicKey, fn.AnnounceSig, []byte(msg)) {
			log.Printf("[federation] invalid announce_sig for node %s from %s, skipping", fn.NodeID, peerURL)
			skipped++
			continue
		}

		caps, _ := json.Marshal(fn.Capabilities)
		n := &Node{
			NodeID:           fn.NodeID,
			PublicPhone:      fn.PublicPhone,
			Lat:              fn.Lat,
			Lon:              fn.Lon,
			CoverageRadiusKM: fn.CoverageRadiusKM,
			Capabilities:     string(caps),
			Version:          fn.Version,
			PublicKey:        fn.PublicKey,
			LastSeen:         time.Unix(fn.LastSeen, 0).UTC(),
			AnnounceSig:      fn.AnnounceSig,
			AnnounceTS:       fn.AnnounceTS,
			Source:           fmt.Sprintf("federation:%s", peerURL),
		}
		if err := f.Store.MergeNode(n); err != nil {
			log.Printf("[federation] merge node %s from %s: %v", fn.NodeID, peerURL, err)
			skipped++
			continue
		}
		merged++
	}
	log.Printf("[federation] synced %s: %d merged, %d skipped", peerURL, merged, skipped)
}

// sameOrigin returns true if two URLs refer to the same host and port,
// avoiding false negatives from trailing slashes or differing schemes.
func sameOrigin(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return strings.EqualFold(ua.Hostname(), ub.Hostname()) && ua.Port() == ub.Port()
}
