package node

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/foodrelay/foodrelay/internal/common"
)

// HeartbeatClient announces the node to all configured registries and sends periodic heartbeats.
type HeartbeatClient struct {
	Cfg       Config
	PubKeyHex string
	PrivKey   ed25519.PrivateKey
}

type announcePayload struct {
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

type heartbeatPayload struct {
	NodeID    string `json:"node_id"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

// AnnounceAll sends POST /v1/nodes/announce to every configured registry concurrently.
// Per-registry failures are logged but do not block the others.
func (h *HeartbeatClient) AnnounceAll() {
	if len(h.Cfg.RegistryURLs) == 0 {
		log.Println("[heartbeat] no registry URLs configured; skipping announce")
		return
	}

	ts := time.Now().Unix()
	// Announce signature covers nodeID|publicPhone|timestamp so that peer registries
	// can verify the phone number has not been substituted during federation propagation.
	sigData := h.Cfg.NodeID + "|" + h.Cfg.NodePhone + "|" + strconv.FormatInt(ts, 10)

	payload := announcePayload{
		NodeID:           h.Cfg.NodeID,
		PublicPhone:      h.Cfg.NodePhone,
		Lat:              h.Cfg.NodeLat,
		Lon:              h.Cfg.NodeLon,
		CoverageRadiusKM: h.Cfg.CoverageKM,
		Capabilities:     []string{"sms"},
		Version:          "0.1.0",
		PublicKey:        h.PubKeyHex,
		Timestamp:        ts,
		Signature:        common.Sign(h.PrivKey, []byte(sigData)),
	}

	h.fanOut("/v1/nodes/announce", payload)
}

// HeartbeatAll sends POST /v1/nodes/heartbeat to every configured registry concurrently.
func (h *HeartbeatClient) HeartbeatAll() {
	if len(h.Cfg.RegistryURLs) == 0 {
		return
	}

	ts := time.Now().Unix()
	sigData := h.Cfg.NodeID + "|" + strconv.FormatInt(ts, 10)

	payload := heartbeatPayload{
		NodeID:    h.Cfg.NodeID,
		Timestamp: ts,
		Signature: common.Sign(h.PrivKey, []byte(sigData)),
	}

	h.fanOut("/v1/nodes/heartbeat", payload)
}

// Run sends an initial announce then heartbeats every 5 minutes.
func (h *HeartbeatClient) Run(stop <-chan struct{}) {
	h.AnnounceAll()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.HeartbeatAll()
		case <-stop:
			return
		}
	}
}

// fanOut sends path to every registry URL concurrently, logging errors.
func (h *HeartbeatClient) fanOut(path string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[heartbeat] marshal %s: %v", path, err)
		return
	}

	var wg sync.WaitGroup
	for _, baseURL := range h.Cfg.RegistryURLs {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			if err := postJSON(u, path, data); err != nil {
				log.Printf("[heartbeat] %s -> %s: %v", path, u, err)
			}
		}(baseURL)
	}
	wg.Wait()
}

func postJSON(baseURL, path string, data []byte) error {
	url := strings.TrimRight(baseURL, "/") + path
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("registry returned %d", resp.StatusCode)
	}
	return nil
}
