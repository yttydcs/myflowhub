package enrollment

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
)

func transcriptDigest(initPayload, challengePayload []byte) [32]byte {
	hash := sha256.New()
	_, _ = hash.Write([]byte("MFHE-TRANSCRIPT-V1"))
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(initPayload)))
	_, _ = hash.Write(length[:])
	_, _ = hash.Write(initPayload)
	binary.BigEndian.PutUint32(length[:], uint32(len(challengePayload)))
	_, _ = hash.Write(length[:])
	_, _ = hash.Write(challengePayload)
	var result [32]byte
	copy(result[:], hash.Sum(nil))
	return result
}

func challengeMessage(challenge unsignedChallenge) []byte {
	data, _ := json.Marshal(challenge)
	return append([]byte("MFHE-CHALLENGE-V1\x00"), data...)
}

func signChallenge(privateKey ed25519.PrivateKey, challenge unsignedChallenge) []byte {
	return ed25519.Sign(privateKey, challengeMessage(challenge))
}

func verifyChallenge(publicKey ed25519.PublicKey, challenge unsignedChallenge, signature []byte) bool {
	return len(publicKey) == ed25519.PublicKeySize && len(signature) == ed25519.SignatureSize && ed25519.Verify(publicKey, challengeMessage(challenge), signature)
}

type unsignedChallenge struct {
	Version            int    `json:"version"`
	RequestID          string `json:"request_id"`
	ParentNodeID       string `json:"parent_node_id"`
	ParentPublicKey    string `json:"parent_public_key"`
	AuthorityNodeID    string `json:"authority_node_id"`
	AuthorityPublicKey string `json:"authority_public_key"`
	Nonce              string `json:"nonce"`
	ExpiresAtUnixMS    int64  `json:"expires_at_unix_ms"`
}
