package node

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// Sender sends outbound SMS messages via Twilio.
type Sender struct {
	AccountSID string
	AuthToken  string
	FromPhone  string
}

// Send sends an SMS to `to` with the given body.
// If AccountSID or AuthToken is empty, logs instead of sending (dev mode).
func (s *Sender) Send(to, body string) error {
	if s.AccountSID == "" || s.AuthToken == "" {
		log.Printf("[SMS dev] To=%s Body=%s", to, body)
		return nil
	}

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", s.AccountSID)

	vals := url.Values{}
	vals.Set("To", to)
	vals.Set("From", s.FromPhone)
	vals.Set("Body", body)

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(vals.Encode()))
	if err != nil {
		return fmt.Errorf("build twilio request: %w", err)
	}
	req.SetBasicAuth(s.AccountSID, s.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("twilio request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("twilio returned status %d", resp.StatusCode)
	}
	return nil
}
