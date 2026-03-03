package node

import "fmt"

// OfferReceived is sent to a donor after OFFER is parsed.
func OfferReceived(qty int, unit string) string {
	return fmt.Sprintf(
		"Got it! We recorded your offer of %d %s. Reply READY when the food is packed and ready for pickup.",
		qty, unit,
	)
}

// OfferReady is sent to a donor after READY is received.
func OfferReady(qty int, unit string) string {
	return fmt.Sprintf(
		"Your %d %s are marked READY. We'll find a match and text you shortly.",
		qty, unit,
	)
}

// MatchDonor is sent to the donor when a match is made.
func MatchDonor(needLocation string, needQty int, unit string) string {
	return fmt.Sprintf(
		"MATCH FOUND! Please prepare %d %s for pickup. The recipient is at: %s. A volunteer or driver will collect. Reply CANCEL to abort.",
		needQty, unit, needLocation,
	)
}

// MatchRecipient is sent to the recipient when a match is made.
func MatchRecipient(offerLocation string, offerQty int, unit string) string {
	return fmt.Sprintf(
		"MATCH FOUND! %d %s are available at: %s. Pickup is being arranged. Reply CANCEL to abort.",
		offerQty, unit, offerLocation,
	)
}

// NoMatchYet is sent when READY has no open needs to match.
func NoMatchYet() string {
	return "No matching needs right now. We'll keep trying and notify you when a match is found."
}

// NeedReceived is sent to a recipient after NEED is parsed.
func NeedReceived(qty int, unit string) string {
	return fmt.Sprintf(
		"Need recorded: %d %s. We'll notify you when a match is found.",
		qty, unit,
	)
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

OFFER <qty> <unit> <time> <location>
  e.g. OFFER 12 meals 9-10pm 123 Main St

READY - mark your offer ready for pickup

NEED <qty> <unit> <time> <location> [priority: N]
  e.g. NEED 20 meals by 8pm 55 Church St

CANCEL - cancel your latest active offer or need

HELP - show this message`
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
	return fmt.Sprintf("Sorry, I didn't understand %q. Reply HELP for commands.", raw)
}
