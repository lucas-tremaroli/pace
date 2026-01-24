package task

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/storage"
)

const (
	// DefaultPrefix is used when no prefix is configured
	DefaultPrefix = "task"
	// IDLength is the number of random hex characters after the prefix
	IDLength = 3
	// ConfigKeyPrefix is the config key for the ID prefix
	ConfigKeyPrefix = "id_prefix"
)

// GenerateID creates a new short hash ID like "prefix-a1b"
func GenerateID(prefix string) string {
	bytes := make([]byte, 2) // 2 bytes = 4 hex chars, we'll use 3
	if _, err := rand.Read(bytes); err != nil {
		// Fallback - this should never happen in practice
		panic("failed to generate random bytes: " + err.Error())
	}
	hash := hex.EncodeToString(bytes)[:IDLength]
	return prefix + "-" + hash
}

// GetOrInitPrefix returns the configured prefix, initializing it if needed
func GetOrInitPrefix(db *storage.DB) (string, error) {
	// Try to get existing prefix
	prefix, err := db.GetConfig(ConfigKeyPrefix)
	if err == nil && prefix != "" {
		return prefix, nil
	}

	// If not found, initialize with current directory name or default
	if err == sql.ErrNoRows || prefix == "" {
		prefix = detectPrefix()
		if err := db.SetConfig(ConfigKeyPrefix, prefix); err != nil {
			return "", err
		}
		return prefix, nil
	}

	return "", err
}

// detectPrefix determines the prefix based on the current directory
func detectPrefix() string {
	// Try to get the current working directory name
	cwd, err := os.Getwd()
	if err != nil {
		return DefaultPrefix
	}

	dirName := filepath.Base(cwd)
	// Clean the directory name to make it suitable as an ID prefix
	dirName = strings.ToLower(dirName)
	dirName = strings.ReplaceAll(dirName, " ", "-")
	dirName = strings.ReplaceAll(dirName, "_", "-")

	if dirName == "" || dirName == "." || dirName == "/" {
		return DefaultPrefix
	}

	return dirName
}

// SetPrefix updates the configured prefix
func SetPrefix(db *storage.DB, prefix string) error {
	return db.SetConfig(ConfigKeyPrefix, prefix)
}
