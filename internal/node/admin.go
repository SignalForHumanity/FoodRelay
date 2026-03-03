package node

import "log"

// Admin handles escalation alerts to administrator phone numbers.
type Admin struct {
	Phones []string
	Sender *Sender
}

// Alert sends msg to all configured admin phones.
func (a *Admin) Alert(msg string) {
	if len(a.Phones) == 0 {
		log.Printf("[admin-alert] no admin phones configured. msg=%s", msg)
		return
	}
	for _, phone := range a.Phones {
		if err := a.Sender.Send(phone, msg); err != nil {
			log.Printf("[admin-alert] failed to send to %s: %v", phone, err)
		}
	}
}
