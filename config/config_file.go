package config

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"
)

// State of the config file as of the last successful load
var (
	configFileMu          sync.Mutex
	configFileTracked     bool
	configFileFname       string
	configFileFingerprint string
)

func resetConfigFileTracking() {
	configFileMu.Lock()
	defer configFileMu.Unlock()
	configFileTracked = false
	configFileFname = ""
	configFileFingerprint = ""
}

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

func ConfigFileOutdated() bool {
	configFileMu.Lock()
	tracked := configFileTracked
	filename := configFileFname
	fingerprint := configFileFingerprint
	configFileMu.Unlock()
	if !tracked {
		return false // The config isn't in a file (e.g. environment variables)
	}
	currentFingerprint, err := computeConfigFileFingerprint(filename)
	if err != nil {
		return false // The file isn't readable (e.g. it was removed)
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
