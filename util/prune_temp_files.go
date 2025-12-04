package util

import (
	"os"
	"path"
	"strings"
)

const TempFilePrefix = "pganalyze_collector_"

// Delete any temp files we may have left behind on an unclean shutdown
func PruneTempFiles(logger *Logger) {
	tmpdir := os.TempDir()
	entries, err := os.ReadDir(tmpdir)
	if err != nil {
		logger.PrintWarning("Could not open temp directory to prune temp files: %s\n", err)
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, TempFilePrefix) {
			continue
		}
		err = os.Remove(path.Join(tmpdir, name))
		if err != nil {
			logger.PrintWarning("Could not remove stray temp file %s in temp dir %s: %s\n", name, tmpdir, err)
			continue
		}
		logger.PrintVerbose("Removed stray temp file %s in temp dir %s\n", name, tmpdir)
	}
}
