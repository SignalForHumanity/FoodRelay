package node

import (
	"log"
	"net/http"
	"time"

	"github.com/foodrelay/foodrelay/internal/common"
)

// TwilioHandler handles inbound Twilio SMS webhooks.
type TwilioHandler struct {
	Store  Store
	Engine *Engine
	Sender *Sender
}

// ServeHTTP handles POST /twilio/sms (application/x-www-form-urlencoded from Twilio).
func (h *TwilioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	from := common.NormalizePhone(r.FormValue("From"))
	body := r.FormValue("Body")
	log.Printf("[sms] from=%s body=%q", from, body)

	reply := h.handle(from, body)

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(common.TwiMLReply(reply))) //nolint:errcheck
}

func (h *TwilioHandler) handle(from, body string) string {
	cmd := Parse(body, time.Now())

	switch cmd.Type {
	case "OFFER":
		cmd.Offer.Phone = from
		if err := h.Store.CreateOffer(cmd.Offer); err != nil {
			log.Printf("[sms] create offer: %v", err)
			return "Sorry, we couldn't record your offer. Please try again."
		}
		return OfferReceived(cmd.Offer.Description, cmd.Offer.Location, cmd.Offer.WindowEnd)

	case "READY":
		offer, err := h.Store.GetLatestOfferByPhone(from)
		if err != nil {
			log.Printf("[sms] get offer: %v", err)
			return "Error retrieving your offer. Please try again."
		}
		if offer == nil || offer.Status != "awaiting_ready" {
			return "No offer awaiting READY found. Send OFFER first."
		}

		if err := h.Store.UpdateOfferStatus(offer.ID, "ready"); err != nil {
			log.Printf("[sms] update offer status: %v", err)
			return "Error updating offer. Please try again."
		}

		// Reload to get fresh timestamps
		offer, _ = h.Store.GetOfferByID(offer.ID)

		// Attempt immediate match
		if _, err := h.Engine.MatchOffer(offer); err != nil {
			log.Printf("[sms] match offer: %v", err)
		}
		return OfferReady()

	case "NEED":
		cmd.Need.Phone = from
		if err := h.Store.CreateNeed(cmd.Need); err != nil {
			log.Printf("[sms] create need: %v", err)
			return "Sorry, we couldn't record your need. Please try again."
		}

		// Reload to get the assigned ID, then try to match immediately.
		need, _ := h.Store.GetLatestNeedByPhone(from)
		if need != nil {
			job, offer, err := h.Engine.MatchForNeed(need)
			if err != nil {
				log.Printf("[sms] match for need: %v", err)
			}
			if job != nil {
				// Return the recipient notification inline as the TwiML reply.
				return MatchRecipient(offer.Description, offer.Location, offer.WindowEnd)
			}
		}
		return NeedReceived()

	case "CANCEL":
		return h.handleCancel(from)

	case "FOOD":
		return HelpText()

	default:
		return UnknownCommand(body)
	}
}

func (h *TwilioHandler) handleCancel(phone string) string {
	offer, err := h.Store.GetLatestOfferByPhone(phone)
	if err == nil && offer != nil && (offer.Status == "awaiting_ready" || offer.Status == "ready") {
		h.Store.UpdateOfferStatus(offer.ID, "canceled") //nolint:errcheck
		return Canceled("offer")
	}

	need, err := h.Store.GetLatestNeedByPhone(phone)
	if err == nil && need != nil && need.Status == "open" {
		h.Store.UpdateNeedStatus(need.ID, "canceled") //nolint:errcheck
		return Canceled("need")
	}

	return NothingToCancel()
}
