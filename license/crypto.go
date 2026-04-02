package license

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
)

func verifySignature(payload map[string]any, timestamp int64, signature string, publicKey ed25519.PublicKey) bool {
	body := canonicalPayload(payload)
	msg := []byte(fmt.Sprintf("%d.%s", timestamp, body))
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, msg, sig)
}
