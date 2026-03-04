package node

import (
	"fmt"
	"strings"
	"time"

	"github.com/foodrelay/foodrelay/internal/common"
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
		return ParsedCommand{
			Type:    "NEED",
			Need:    &Need{Location: strings.TrimSpace(rest), Status: "open"},
			RawText: text,
		}

	case "READY":
		return ParsedCommand{Type: "READY", RawText: text}

	case "CANCEL":
		return ParsedCommand{Type: "CANCEL", RawText: text}

	case "FOOD", "?":
		return ParsedCommand{Type: "FOOD", RawText: text}

	default:
		return ParsedCommand{Type: "UNKNOWN", RawText: text}
	}
}

// parseOffer parses the body after "OFFER ".
// Grammar: <description> <timewindow> <location>
// The time window (e.g. "until 8pm", "7-9pm") splits description from location.
func parseOffer(s string, now time.Time) (*Offer, error) {
	description, wStart, wEnd, location, ok := common.SplitAtTimeWindow(s, now)
	if !ok {
		return nil, fmt.Errorf("could not find time window in %q; use e.g. 'until 8pm' or '7-9pm'", s)
	}

	description = strings.TrimSpace(description)
	location = strings.TrimSpace(location)

	if description == "" {
		return nil, fmt.Errorf("missing description before time window")
	}
	if location == "" {
		return nil, fmt.Errorf("missing location after time window")
	}

	return &Offer{
		Description: description,
		WindowStart: wStart,
		WindowEnd:   wEnd,
		Location:    location,
		Status:      "awaiting_ready",
	}, nil
}
