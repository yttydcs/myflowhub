package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func fileIdentity(data []byte) (string, string) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		digest := sha256.Sum256(append(data, byte(len(data))))
		copy(id[:], digest[:16])
	}
	return hex.EncodeToString(id[:]), sha256Hex(data)
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
