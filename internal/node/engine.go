package node

import (
	"fmt"
	"log"
)

// Engine performs matching between ready offers and open needs.
type Engine struct {
	Store  Store
	Sender *Sender
}

// MatchOffer attempts to find the best open need for a READY offer.
// Picks the oldest open need (FIFO — GetNeedsByStatus orders by id ASC).
// Returns the created Job on success, or nil if no open needs exist.
func (e *Engine) MatchOffer(offer *Offer) (*Job, error) {
	needs, err := e.Store.GetNeedsByStatus("open")
	if err != nil {
		return nil, fmt.Errorf("get needs: %w", err)
	}
	if len(needs) == 0 {
		return nil, nil
	}

	best := needs[0] // oldest open need

	job := &Job{
		OfferID:     offer.ID,
		NeedID:      best.ID,
		Carrier:     "manual",
		Status:      "matched",
		MatchScore:  1.0,
		MatchReason: "fifo",
	}

	if err := e.Store.CreateJob(job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	if err := e.Store.UpdateOfferStatus(offer.ID, "matched"); err != nil {
		return nil, fmt.Errorf("update offer status: %w", err)
	}
	if err := e.Store.UpdateNeedStatus(best.ID, "matched"); err != nil {
		return nil, fmt.Errorf("update need status: %w", err)
	}

	log.Printf("[engine] matched offer=%d need=%d (offer-first fifo)", offer.ID, best.ID)

	// Notify both parties
	e.Sender.Send(offer.Phone, MatchDonor(offer.Location))                                              //nolint:errcheck
	e.Sender.Send(best.Phone, MatchRecipient(offer.Description, offer.Location, offer.WindowEnd)) //nolint:errcheck

	return job, nil
}

// MatchForNeed finds the best READY offer for a newly registered need.
// Picks the READY offer expiring soonest (most urgent); falls back to FIFO.
// On match: notifies the donor via async SMS and returns (job, offer) so the
// caller can deliver the recipient notification inline (e.g. as a TwiML reply).
// Returns (nil, nil, nil) when no READY offers exist.
func (e *Engine) MatchForNeed(need *Need) (*Job, *Offer, error) {
	offers, err := e.Store.GetOffersByStatus("ready")
	if err != nil {
		return nil, nil, fmt.Errorf("get ready offers: %w", err)
	}
	if len(offers) == 0 {
		return nil, nil, nil
	}

	// Pick the offer expiring soonest; fall back to FIFO if no window set.
	best := offers[0]
	for _, o := range offers[1:] {
		if !o.WindowEnd.IsZero() && (best.WindowEnd.IsZero() || o.WindowEnd.Before(best.WindowEnd)) {
			best = o
		}
	}

	job := &Job{
		OfferID:     best.ID,
		NeedID:      need.ID,
		Carrier:     "manual",
		Status:      "matched",
		MatchScore:  1.0,
		MatchReason: "fifo",
	}

	if err := e.Store.CreateJob(job); err != nil {
		return nil, nil, fmt.Errorf("create job: %w", err)
	}
	if err := e.Store.UpdateOfferStatus(best.ID, "matched"); err != nil {
		return nil, nil, fmt.Errorf("update offer status: %w", err)
	}
	if err := e.Store.UpdateNeedStatus(need.ID, "matched"); err != nil {
		return nil, nil, fmt.Errorf("update need status: %w", err)
	}

	log.Printf("[engine] matched need=%d offer=%d (need-first)", need.ID, best.ID)

	// Notify the donor asynchronously; caller sends the recipient notification.
	e.Sender.Send(best.Phone, MatchDonor(best.Location)) //nolint:errcheck

	return job, best, nil
}

// RunMatchingPass attempts to match all READY offers to open needs.
// Called by the scheduler on a timer.
func (e *Engine) RunMatchingPass() {
	offers, err := e.Store.GetOffersByStatus("ready")
	if err != nil {
		log.Printf("[engine] get ready offers: %v", err)
		return
	}
	for _, o := range offers {
		if _, err := e.MatchOffer(o); err != nil {
			log.Printf("[engine] match offer=%d: %v", o.ID, err)
		}
	}
}
