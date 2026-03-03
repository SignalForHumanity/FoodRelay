package node

import "time"

// Offer represents a food donation offer from a donor.
type Offer struct {
	ID          int64
	Phone       string
	Qty         int
	Unit        string
	WindowStart time.Time
	WindowEnd   time.Time
	Location    string
	Notes       string
	Allergens   string
	Status      string // awaiting_ready | ready | matched | done | canceled | expired
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Need represents a food request from a recipient/hub.
type Need struct {
	ID          int64
	Phone       string
	Qty         int
	Unit        string
	WindowStart time.Time
	WindowEnd   time.Time
	Location    string
	Priority    int
	Status      string // open | matched | done | canceled | expired
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Job represents a matched offer+need pair pending delivery.
type Job struct {
	ID          int64
	OfferID     int64
	NeedID      int64
	Carrier     string
	Status      string // matched | dispatched | picked_up | delivered | failed | canceled
	MatchScore  float64
	MatchReason string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ParsedCommand is the result of parsing an inbound SMS.
type ParsedCommand struct {
	Type    string // OFFER | NEED | READY | CANCEL | HELP | UNKNOWN
	Offer   *Offer
	Need    *Need
	RawText string
}
