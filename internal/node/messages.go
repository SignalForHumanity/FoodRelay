package node

import (
	"fmt"
	"time"
)

// OfferReceived is sent to a donor after OFFER is parsed.
func OfferReceived(description, location string, windowEnd time.Time) string {
	timeStr := ""
	if !windowEnd.IsZero() {
		timeStr = fmt.Sprintf(" (until %s)", windowEnd.Format("3:04 PM"))
	}
	return fmt.Sprintf(
		"Got it! Offer recorded: %s at %s%s. Reply READY when it's packed and ready for pickup.",
		description, location, timeStr,
	)
}

// OfferReady is sent to a donor after READY is received.
func OfferReady() string {
	return "Your offer is marked READY. We'll notify anyone looking for food and text you when a match is found."
}

// MatchDonor is sent to the donor when a match is made.
func MatchDonor(offerLocation string) string {
	return fmt.Sprintf(
		"MATCH! Someone is on their way to pick up at: %s. Please have your offer ready. Reply CANCEL to abort.",
		offerLocation,
	)
}

// MatchRecipient is sent to the recipient when a match is made.
func MatchRecipient(description, offerLocation string, windowEnd time.Time) string {
	timeStr := ""
	if !windowEnd.IsZero() {
		timeStr = fmt.Sprintf(" Pickup before %s.", windowEnd.Format("3:04 PM"))
	}
	return fmt.Sprintf(
		"Food available: %s at %s.%s Head there now! Reply CANCEL to abort.",
		description, offerLocation, timeStr,
	)
}

// NeedReceived is sent to a recipient after NEED is registered with no immediate match.
func NeedReceived() string {
	return "Got it! We'll text you when food is available nearby."
}

// Canceled is sent after a CANCEL command.
func Canceled(what string) string {
	return fmt.Sprintf("Your latest %s has been canceled.", what)
}

// NothingToCancel is sent when there's nothing active to cancel.
func NothingToCancel() string {
	return "No active offer or need found to cancel."
}

// HelpText is the full help message.
func HelpText() string {
	return `FoodRelay SMS commands:

OFFER <description> <time> <address>
  e.g. OFFER 3 trays pasta until 8pm 123 Main St
  e.g. OFFER hot soup 7-9pm First Baptist Church

READY - mark your offer ready for pickup

NEED [address]
  e.g. NEED
  e.g. NEED 55 Church St

CANCEL - cancel your latest active offer or need

FOOD - show this message`
}

// EscalationAlert is sent to admin phones.
func EscalationAlert(offerID int64, phone string, minutesWaiting int) string {
	return fmt.Sprintf(
		"[ALERT] Offer #%d from %s has been READY for %d min with no match. Manual intervention may be needed.",
		offerID, phone, minutesWaiting,
	)
}

// UnknownCommand is sent when parsing fails.
func UnknownCommand(raw string) string {
	return fmt.Sprintf("Sorry, I didn't understand %q. Reply FOOD for commands.", raw)
}
