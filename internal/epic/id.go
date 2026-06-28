package epic

import (
	"crypto/rand"
	"encoding/hex"
)

const (
	// IDPrefix is the literal prefix used for all epic IDs so they are
	// visually distinct from task IDs (which use a configurable per-project
	// prefix).
	IDPrefix = "epic"
	// IDLength is the number of random hex characters after the prefix
	IDLength = 3
)

// GenerateID creates a new short epic ID like "epic-a1b".
func GenerateID() string {
	bytes := make([]byte, 2)
	if _, err := rand.Read(bytes); err != nil {
		panic("failed to generate random bytes: " + err.Error())
	}
	hash := hex.EncodeToString(bytes)[:IDLength]
	return IDPrefix + "-" + hash
}
