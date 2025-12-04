package util

import (
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
)

const TempFilePrefix = "pganalyze_collector_"

// Delete any temp files we may have left behind on an unclean shutdown
func PruneTempFiles(logger *Logger) {
	usr, err := user.Current()
	if err != nil {
		logger.PrintWarning("Could not check current user to prune temp files: %s\n", err)
		return
	}
	uid, err := strconv.ParseUint(usr.Uid, 10, 32)
	if err != nil {
		logger.PrintWarning("Could not parse current user uid to prune temp files: %s\n", err)
		return
	}
	uid32 := uint32(uid)
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
		fi, err := entry.Info()
		if err != nil {
			logger.PrintWarning("Could not check info for file %s in temp dir %s: %s\n", name, tmpdir, err)
			continue
		}
		if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
			if stat.Uid == uid32 {
				err = os.Remove(path.Join(tmpdir, name))
				if err != nil {
					logger.PrintWarning("Could not remove stray temp file %s in temp dir %s: %s\n", name, tmpdir, err)
					continue
				}
				logger.PrintVerbose("Removed stray temp file %s in temp dir %s\n", name, tmpdir)
			}
		}
	}
}
