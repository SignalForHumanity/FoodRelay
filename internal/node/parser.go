package node

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/foodrelay/foodrelay/internal/common"
)

var (
	qtyRe  = regexp.MustCompile(`^(\d+)\s*`)
	unitRe = regexp.MustCompile(`(?i)^(meals?|items?|boxes?|bags?|portions?|servings?|units?|packages?|lbs?|kg)\b\s*`)
	prioRe = regexp.MustCompile(`(?i)\bpriority:\s*(\d+)`)
)

// Parse parses an inbound SMS body into a ParsedCommand.
func Parse(text string, now time.Time) ParsedCommand {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ParsedCommand{Type: "UNKNOWN", RawText: text}
	}

	words := strings.Fields(trimmed)
	cmd := strings.ToUpper(words[0])
	rest := strings.TrimSpace(trimmed[len(words[0]):])

	switch cmd {
	case "OFFER":
		o, err := parseOffer(rest, now)
		if err != nil {
			return ParsedCommand{Type: "UNKNOWN", RawText: text}
		}
		return ParsedCommand{Type: "OFFER", Offer: o, RawText: text}

	case "NEED":
		n, err := parseNeed(rest, now)
		if err != nil {
			return ParsedCommand{Type: "UNKNOWN", RawText: text}
		}
		return ParsedCommand{Type: "NEED", Need: n, RawText: text}

	case "READY":
		return ParsedCommand{Type: "READY", RawText: text}

	case "CANCEL":
		return ParsedCommand{Type: "CANCEL", RawText: text}

	case "HELP", "?":
		return ParsedCommand{Type: "HELP", RawText: text}

	default:
		return ParsedCommand{Type: "UNKNOWN", RawText: text}
	}
}

// parseOffer parses the body after "OFFER ".
// Grammar: <qty> [unit] <timewindow> <location> [notes: ...] [allergens: ...]
func parseOffer(s string, now time.Time) (*Offer, error) {
	notes := extractKeyed(&s, "notes:")
	allergens := extractKeyed(&s, "allergens:")

	qty, unit, err := parseQtyUnit(&s)
	if err != nil {
		return nil, err
	}

	wStart, wEnd, rem, ok := common.FindTimeWindow(s, now)
	if !ok {
		return nil, fmt.Errorf("could not parse time window from %q", s)
	}

	location := strings.TrimSpace(rem)
	if location == "" {
		return nil, fmt.Errorf("missing location")
	}

	return &Offer{
		Qty:         qty,
		Unit:        unit,
		WindowStart: wStart,
		WindowEnd:   wEnd,
		Location:    location,
		Notes:       notes,
		Allergens:   allergens,
		Status:      "awaiting_ready",
	}, nil
}

// parseNeed parses the body after "NEED ".
// Grammar: <qty> [unit] <timewindow> <location> [priority: N]
func parseNeed(s string, now time.Time) (*Need, error) {
	priority := 1
	if m := prioRe.FindStringSubmatch(s); m != nil {
		p, _ := strconv.Atoi(m[1])
		priority = p
		s = prioRe.ReplaceAllString(s, "")
	}

	qty, unit, err := parseQtyUnit(&s)
	if err != nil {
		return nil, err
	}

	wStart, wEnd, rem, ok := common.FindTimeWindow(s, now)
	if !ok {
		return nil, fmt.Errorf("could not parse time window from %q", s)
	}

	location := strings.TrimSpace(rem)
	if location == "" {
		return nil, fmt.Errorf("missing location")
	}

	return &Need{
		Qty:         qty,
		Unit:        unit,
		WindowStart: wStart,
		WindowEnd:   wEnd,
		Location:    location,
		Priority:    priority,
		Status:      "open",
	}, nil
}

// parseQtyUnit consumes "<qty> [unit] " from the front of *s.
func parseQtyUnit(s *string) (qty int, unit string, err error) {
	m := qtyRe.FindStringSubmatch(*s)
	if m == nil {
		return 0, "", fmt.Errorf("expected quantity number, got %q", *s)
	}
	qty, _ = strconv.Atoi(m[1])
	*s = (*s)[len(m[0]):]

	um := unitRe.FindStringSubmatch(*s)
	if um != nil {
		unit = strings.ToLower(um[1])
		*s = (*s)[len(um[0]):]
	} else {
		unit = "meals"
	}
	return qty, unit, nil
}

// extractKeyed removes "key <value>" from *s and returns value.
// Handles multiple keys by stopping at the next known keyword.
func extractKeyed(s *string, key string) string {
	lower := strings.ToLower(*s)
	idx := strings.Index(lower, strings.ToLower(key))
	if idx < 0 {
		return ""
	}

	afterKey := strings.TrimSpace((*s)[idx+len(key):])
	lowerAfter := strings.ToLower(afterKey)

	otherKeys := []string{"notes:", "allergens:", "priority:"}
	nextIdx := len(afterKey)
	for _, k := range otherKeys {
		if k == key {
			continue
		}
		if i := strings.Index(lowerAfter, k); i >= 0 && i < nextIdx {
			nextIdx = i
		}
	}

	value := strings.TrimSpace(afterKey[:nextIdx])
	before := strings.TrimSpace((*s)[:idx])
	after := ""
	if nextIdx < len(afterKey) {
		after = " " + afterKey[nextIdx:]
	}
	*s = strings.TrimSpace(before + after)
	return value
}
