package node

import (
	"fmt"
	"log"
	"time"

	"github.com/foodrelay/foodrelay/internal/common"
)

// Engine performs matching between ready offers and open needs.
type Engine struct {
	Store  Store
	Sender *Sender
}

// MatchOffer attempts to find the best open need for the given READY offer.
// Returns the created Job on success, or nil if no suitable need found.
func (e *Engine) MatchOffer(offer *Offer) (*Job, error) {
	needs, err := e.Store.GetNeedsByStatus("open")
	if err != nil {
		return nil, fmt.Errorf("get needs: %w", err)
	}

	best, bestScore, bestReason := (*Need)(nil), -1.0, ""

	for _, n := range needs {
		score, reason, ok := scoreMatch(offer, n)
		if !ok {
			log.Printf("[engine] offer=%d need=%d rejected: %s", offer.ID, n.ID, reason)
			continue
		}
		if score > bestScore {
			best, bestScore, bestReason = n, score, reason
		}
	}

	if best == nil {
		return nil, nil
	}

	job := &Job{
		OfferID:     offer.ID,
		NeedID:      best.ID,
		Carrier:     "manual",
		Status:      "matched",
		MatchScore:  bestScore,
		MatchReason: bestReason,
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

	log.Printf("[engine] matched offer=%d need=%d score=%.2f reason=%s", offer.ID, best.ID, bestScore, bestReason)

	// Notify both parties
	e.Sender.Send(offer.Phone, MatchDonor(best.Location, best.Qty, best.Unit))   //nolint:errcheck
	e.Sender.Send(best.Phone, MatchRecipient(offer.Location, offer.Qty, offer.Unit)) //nolint:errcheck

	return job, nil
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

// scoreMatch returns (score, reason, ok).
// ok=false means a hard constraint failed.
func scoreMatch(offer *Offer, need *Need) (float64, string, bool) {
	// Hard constraint: time windows must overlap (skip if either is zero)
	if !offer.WindowStart.IsZero() && !offer.WindowEnd.IsZero() &&
		!need.WindowStart.IsZero() && !need.WindowEnd.IsZero() {
		if !common.WindowsOverlap(offer.WindowStart, offer.WindowEnd, need.WindowStart, need.WindowEnd) {
			return 0, "time windows do not overlap", false
		}
	}

	score := 0.0
	reasons := []string{}

	// Soft: recipient priority (higher = better)
	score += float64(need.Priority) * 10
	reasons = append(reasons, fmt.Sprintf("priority=%d", need.Priority))

	// Soft: window overlap quality
	if !offer.WindowStart.IsZero() && !need.WindowStart.IsZero() {
		overlap := common.OverlapDuration(offer.WindowStart, offer.WindowEnd, need.WindowStart, need.WindowEnd)
		score += float64(overlap) / float64(time.Hour)
		if overlap > 0 {
			reasons = append(reasons, fmt.Sprintf("overlap=%.0fm", overlap.Minutes()))
		}
	}

	return score, joinReasons(reasons), true
}

func joinReasons(rs []string) string {
	out := ""
	for i, r := range rs {
		if i > 0 {
			out += ", "
		}
		out += r
	}
	return out
}
