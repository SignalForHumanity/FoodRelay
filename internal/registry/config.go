package registry

import (
	"strings"

	"github.com/foodrelay/foodrelay/internal/common"
)

// Config holds registry runtime configuration.
type Config struct {
	Addr   string
	DBPath string
	Peers  []string // peer registry URLs for gossip federation
}

// LoadConfig reads config from environment variables.
func LoadConfig() Config {
	var peers []string
	if v := common.Env("REGISTRY_PEERS", ""); v != "" {
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				peers = append(peers, p)
			}
		}
	}
	return Config{
		Addr:   common.Env("REGISTRY_ADDR", ":8081"),
		DBPath: common.Env("REGISTRY_DB_PATH", "./registry.db"),
		Peers:  peers,
	}
}
