package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func GenerateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func GenerateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
