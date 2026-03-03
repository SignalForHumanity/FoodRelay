package common

import (
	"encoding/xml"
	"fmt"
)

type twiMLResponse struct {
	XMLName  xml.Name       `xml:"Response"`
	Messages []twiMLMessage `xml:"Message,omitempty"`
}

type twiMLMessage struct {
	Body string `xml:",chardata"`
}

// TwiMLReply builds a TwiML XML body with one <Message> per arg.
// Returns the full XML string ready to write as Content-Type: text/xml.
func TwiMLReply(bodies ...string) string {
	resp := twiMLResponse{}
	for _, b := range bodies {
		resp.Messages = append(resp.Messages, twiMLMessage{Body: b})
	}
	out, err := xml.MarshalIndent(resp, "", "  ")
	if err != nil {
		return `<?xml version="1.0"?><Response></Response>`
	}
	return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n%s", string(out))
}
