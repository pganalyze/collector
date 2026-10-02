package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	filename := filepath.Join(dir, "pganalyze-collector.conf")
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Could not write test config file: %s", err)
	}
	return filename
}

func TestConfigFileOutdatedNoTrackedFile(t *testing.T) {
	resetConfigFileTracking()
	if ConfigFileOutdated() {
		t.Errorf("Expected config file to not be outdated when no file was tracked")
	}
}

func TestConfigFileOutdated(t *testing.T) {
	content := "[pganalyze]\napi_key = test\n\n[default]\ndb_url = postgres://user@localhost/db\n"
	filename := writeTestConfigFile(t, content)

	RecordConfigFile(filename)
	if ConfigFileOutdated() {
		t.Errorf("Expected config file to not be outdated right after loading it")
	}

	// A content change makes the file outdated
	changedContent := content + "# new line\n"
	err := os.WriteFile(filename, []byte(changedContent), 0644)
	if err != nil {
		t.Fatalf("Could not modify test config file: %s", err)
	}
	if !ConfigFileOutdated() {
		t.Errorf("Expected config file to be outdated after its content was changed")
	}

	// Reverting to the original content means the in-memory config matches
	// again, even though the modification time changed
	err = os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Could not revert test config file: %s", err)
	}
	if ConfigFileOutdated() {
		t.Errorf("Expected config file to not be outdated when its content matches the loaded version again")
	}

	// A removed file is not reported as outdated (a reload could not recover it)
	err = os.Remove(filename)
	if err != nil {
		t.Fatalf("Could not remove test config file: %s", err)
	}
	if ConfigFileOutdated() {
		t.Errorf("Expected config file to not be outdated when the file was removed")
	}
}

func TestConfigFileOutdatedUnreadableAtRecordTime(t *testing.T) {
	resetConfigFileTracking()
	RecordConfigFile(filepath.Join(t.TempDir(), "nonexistent.conf"))
	if ConfigFileOutdated() {
		t.Errorf("Expected config file to not be outdated when recording a nonexistent file")
	}
}
