package config

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"
)

// State of the config file as of the last successful load. We compare content
// fingerprints (instead of e.g. the modification time) so that we only report
// the file as outdated when its content actually differs from what the
// collector has loaded into memory.
var (
	configFileMu          sync.Mutex
	configFileTracked     bool
	configFileFname       string
	configFileFingerprint string
)

// resetConfigFileTracking - clears the tracked config file state. Called at
// the start of each config read, since the source of the configuration (file
// vs. environment variables) may differ between reads.
func resetConfigFileTracking() {
	configFileMu.Lock()
	defer configFileMu.Unlock()
	configFileTracked = false
	configFileFname = ""
	configFileFingerprint = ""
}

// RecordConfigFile - remembers the content fingerprint of the config file
// that was just loaded, so ConfigFileOutdated can later detect if the file
// on disk was modified.
func RecordConfigFile(filename string) {
	fingerprint, err := computeConfigFileFingerprint(filename)
	if err != nil {
		return
	}
	configFileMu.Lock()
	defer configFileMu.Unlock()
	configFileTracked = true
	configFileFname = filename
	configFileFingerprint = fingerprint
}

// ConfigFileOutdated - reports whether the config file on disk contains
// different content than the version the collector loaded, meaning the
// in-memory configuration is stale and a reload is required to pick up the
// changes.
//
// It returns false when no config file is in use (e.g. when the collector is
// configured purely via environment variables or on Heroku), since there is
// no file that could be out of date.
func ConfigFileOutdated() bool {
	configFileMu.Lock()
	tracked := configFileTracked
	filename := configFileFname
	fingerprint := configFileFingerprint
	configFileMu.Unlock()

	if !tracked {
		return false
	}

	currentFingerprint, err := computeConfigFileFingerprint(filename)
	if err != nil {
		// The file is no longer readable (e.g. it was removed); we don't
		// report it as outdated, since a reload could not recover it either
		return false
	}

	return currentFingerprint != fingerprint
}

func computeConfigFileFingerprint(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
