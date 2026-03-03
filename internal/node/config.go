package node

import (
	"strconv"
	"strings"

	"github.com/foodrelay/foodrelay/internal/common"
)

// Config holds all runtime configuration for the node process.
type Config struct {
	NodeID        string
	NodePhone     string
	Addr          string
	DBPath        string
	AdminPhones   []string
	TwilioSID     string
	TwilioToken   string
	TwilioFrom    string
	RegistryURLs  []string // all registry URLs to announce/heartbeat to
	NodeLat       float64
	NodeLon       float64
	CoverageKM    float64
	PrivKeySeed   string // hex seed for ed25519; generated on first run if empty
	EscalateAfter int    // minutes before alerting admins about unmatched READY
	RunRegistry   bool   // if true, start an embedded registry server in this process
}

// LoadConfig reads config from environment variables.
func LoadConfig() Config {
	lat, _ := strconv.ParseFloat(common.Env("NODE_LAT", "0"), 64)
	lon, _ := strconv.ParseFloat(common.Env("NODE_LON", "0"), 64)
	cov, _ := strconv.ParseFloat(common.Env("NODE_COVERAGE_KM", "10"), 64)
	esc, _ := strconv.Atoi(common.Env("READY_ESCALATE_MINUTES", "30"))

	var admins []string
	if v := common.Env("ADMIN_PHONES", ""); v != "" {
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				admins = append(admins, common.NormalizePhone(p))
			}
		}
	}

	var registryURLs []string
	// Support both REGISTRY_URLS (new, comma-separated) and REGISTRY_URL (legacy single).
	if v := common.Env("REGISTRY_URLS", ""); v != "" {
		for _, u := range strings.Split(v, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				registryURLs = append(registryURLs, u)
			}
		}
	} else if v := common.Env("REGISTRY_URL", ""); v != "" {
		registryURLs = []string{v}
	}

	runRegistry := strings.EqualFold(common.Env("NODE_RUN_REGISTRY", "false"), "true")

	return Config{
		NodeID:        common.Env("NODE_ID", "node-local-01"),
		NodePhone:     common.Env("NODE_PHONE", ""),
		Addr:          common.Env("NODE_ADDR", ":8080"),
		DBPath:        common.Env("NODE_DB_PATH", "./node.db"),
		AdminPhones:   admins,
		TwilioSID:     common.Env("TWILIO_ACCOUNT_SID", ""),
		TwilioToken:   common.Env("TWILIO_AUTH_TOKEN", ""),
		TwilioFrom:    common.Env("TWILIO_FROM_PHONE", ""),
		RegistryURLs:  registryURLs,
		NodeLat:       lat,
		NodeLon:       lon,
		CoverageKM:    cov,
		PrivKeySeed:   common.Env("NODE_PRIVATE_KEY_HEX", ""),
		EscalateAfter: esc,
		RunRegistry:   runRegistry,
	}
}
