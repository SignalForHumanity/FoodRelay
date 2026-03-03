package node

import (
	"log"
	"time"
)

// Scheduler runs periodic background tasks for the node.
type Scheduler struct {
	Store         Store
	Engine        *Engine
	Admin         *Admin
	EscalateAfter time.Duration
}

// Run starts the scheduler loop. Call in a goroutine; blocks until ctx is done.
func (s *Scheduler) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Run once immediately on startup
	s.tick()

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-stop:
			log.Println("[scheduler] stopped")
			return
		}
	}
}

func (s *Scheduler) tick() {
	now := time.Now()

	// 1. Expire stale offers and needs
	if n, err := s.Store.ExpireOffers(now); err != nil {
		log.Printf("[scheduler] expire offers: %v", err)
	} else if n > 0 {
		log.Printf("[scheduler] expired %d offers", n)
	}

	if n, err := s.Store.ExpireNeeds(now); err != nil {
		log.Printf("[scheduler] expire needs: %v", err)
	} else if n > 0 {
		log.Printf("[scheduler] expired %d needs", n)
	}

	// 2. Retry matching for all READY offers
	s.Engine.RunMatchingPass()

	// 3. Escalate offers that have been READY too long without a match
	s.escalateStuckOffers(now)
}

func (s *Scheduler) escalateStuckOffers(now time.Time) {
	if s.EscalateAfter <= 0 {
		return
	}

	offers, err := s.Store.GetOffersByStatus("ready")
	if err != nil {
		log.Printf("[scheduler] get ready offers for escalation: %v", err)
		return
	}

	for _, o := range offers {
		waited := now.Sub(o.UpdatedAt)
		if waited >= s.EscalateAfter {
			msg := EscalationAlert(o.ID, o.Phone, int(waited.Minutes()))
			log.Printf("[scheduler] escalating offer=%d (waited %.0f min)", o.ID, waited.Minutes())
			s.Admin.Alert(msg)
		}
	}
}
