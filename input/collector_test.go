package input

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

// TestGetCollectorConfigFileOutdated - verifies that the config_file_outdated
// flag that is submitted with every full snapshot reflects whether the config
// file on disk differs from the version the collector has loaded
func TestGetCollectorConfigFileOutdated(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "pganalyze-collector.conf")
	content := []byte("[pganalyze]\n\n[default]\ndb_url = postgres://user@localhost/db\n")
	if err := os.WriteFile(filename, content, 0644); err != nil {
		t.Fatalf("Could not write config file: %s", err)
	}

	conf, err := config.Read(false, &util.Logger{}, filename)
	if err != nil {
		t.Fatalf("Could not read config file: %s", err)
	}

	if got := getCollectorConfig(conf.Servers[0]).ConfigFileOutdated; got {
		t.Errorf("Expected ConfigFileOutdated to be false right after loading the config file")
	}

	// Modify the file on disk; the snapshot should now report it as outdated
	modifiedContent := append(content, []byte("# changed\n")...)
	if err := os.WriteFile(filename, modifiedContent, 0644); err != nil {
		t.Fatalf("Could not modify config file: %s", err)
	}
	if got := getCollectorConfig(conf.Servers[0]).ConfigFileOutdated; !got {
		t.Errorf("Expected ConfigFileOutdated to be true after the config file was modified on disk")
	}
}
