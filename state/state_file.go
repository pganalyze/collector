package state

import (
	"encoding/gob"
	"os"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

func WriteStateFile(servers []*Server, opts CollectionOpts, logger *util.Logger) {
	stateOnDisk := StateOnDisk{PrevStateByServer: make(map[config.ServerIdentifier]PersistedState), FormatVersion: StateOnDiskFormatVersion}

	for _, server := range servers {
		stateOnDisk.PrevStateByServer[server.Config.Identifier] = server.PrevState
	}

	file, err := os.Create(opts.StateFilename)
	if err != nil {
		logger.PrintWarning("Could not write out state file to %s because of error: %s", opts.StateFilename, err)
		return
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	encoder.Encode(stateOnDisk)
}

// ReadStateFile - This reads in the prevState structs from the state file - only run this on initial bootup and SIGHUP!
func ReadStateFile(servers []*Server, opts CollectionOpts, logger *util.Logger) {
	var stateOnDisk StateOnDisk

	file, err := os.Open(opts.StateFilename)
	if err != nil {
		if !util.IsHeroku() {
			logger.PrintVerbose("Did not open state file: %s", err)
		}
		return
	}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&stateOnDisk)
	if err != nil {
		logger.PrintVerbose("Could not decode state file: %s", err)
		return
	}
	defer file.Close()

	if stateOnDisk.FormatVersion < StateOnDiskFormatVersion {
		logger.PrintVerbose("Ignoring state file since the on-disk format has changed")
		return
	}

	for idx, server := range servers {
		prevState, exist := stateOnDisk.PrevStateByServer[server.Config.Identifier]
		if exist {
			prefixedLogger := logger.WithPrefix(server.Config.SectionName)
			prefixedLogger.PrintVerbose("Successfully recovered state from on-disk file")
			servers[idx].PrevState = prevState
		}
	}
}
