package registry

import (
	"fmt"
	"strconv"
	"time"

	"github.com/foodrelay/foodrelay/internal/common"
)

const maxTimestampSkew = 5 * time.Minute

// AnnounceSignatureMessage returns the canonical message that is signed in an
// announce request: "nodeID|publicPhone|timestamp".
// Covering publicPhone in the signature means a peer registry cannot substitute
// a different phone number while replaying a valid signature.
func AnnounceSignatureMessage(nodeID, publicPhone string, timestamp int64) string {
	return nodeID + "|" + publicPhone + "|" + strconv.FormatInt(timestamp, 10)
}

// VerifyAnnounceSignature checks an ed25519 announce signature with a timestamp
// skew guard. Used for live announce requests arriving directly from a node.
func VerifyAnnounceSignature(pubKeyHex, nodeID, publicPhone string, timestamp int64, signature string) error {
	ts := time.Unix(timestamp, 0)
	skew := time.Since(ts)
	if skew < 0 {
		skew = -skew
	}
	if skew > maxTimestampSkew {
		return fmt.Errorf("timestamp too old or too far in future (skew=%s)", skew)
	}

	msg := AnnounceSignatureMessage(nodeID, publicPhone, timestamp)
	if !common.Verify(pubKeyHex, signature, []byte(msg)) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// VerifySignature checks an ed25519 signature over "nodeID|timestamp".
// Used for heartbeat requests where the phone number is not re-transmitted.
func VerifySignature(pubKeyHex, nodeID string, timestamp int64, signature string) error {
	ts := time.Unix(timestamp, 0)
	skew := time.Since(ts)
	if skew < 0 {
		skew = -skew
	}
	if skew > maxTimestampSkew {
		return fmt.Errorf("timestamp too old or too far in future (skew=%s)", skew)
	}

	msg := nodeID + "|" + strconv.FormatInt(timestamp, 10)
	if !common.Verify(pubKeyHex, signature, []byte(msg)) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}
