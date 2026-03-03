package common

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateKeyPair generates a fresh ed25519 keypair.
// Returns (pubKeyHex, seedHex, error).
// Store seedHex in NODE_PRIVATE_KEY_HEX to persist the identity.
func GenerateKeyPair() (pubKeyHex, seedHex string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return hex.EncodeToString(pub), hex.EncodeToString(priv.Seed()), nil
}

// KeyPairFromSeedHex restores a keypair from a 32-byte hex seed.
func KeyPairFromSeedHex(seedHex string) (pubKeyHex string, priv ed25519.PrivateKey, err error) {
	seed, err := hex.DecodeString(seedHex)
	if err != nil {
		return "", nil, fmt.Errorf("decode seed: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return "", nil, fmt.Errorf("seed must be %d bytes, got %d", ed25519.SeedSize, len(seed))
	}
	priv = ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	return hex.EncodeToString(pub), priv, nil
}

// Sign signs data and returns the hex-encoded signature.
func Sign(priv ed25519.PrivateKey, data []byte) string {
	return hex.EncodeToString(ed25519.Sign(priv, data))
}

// Verify checks a hex-encoded signature against a hex-encoded public key.
func Verify(pubKeyHex, sigHex string, data []byte) bool {
	pub, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	return ed25519.Verify(pub, data, sig)
}
