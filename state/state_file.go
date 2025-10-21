package state

import (
	"encoding/gob"
	"os"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

func WriteStateFile(servers []*Server, opts CollectionOpts, logger *util.Logger) {
	stateOnDisk := StateOnDisk{
		PrevStateByServer:         make(map[config.ServerIdentifier]PersistedState),
		HighFreqPrevStateByServer: make(map[config.ServerIdentifier]PersistedHighFreqState),
		FormatVersion:             StateOnDiskFormatVersion,
	}

	for _, server := range servers {
		// We must hold the relevant mutexes here because reads on structs stored by value are
		// not atomic (see https://go.dev/ref/mem#restrictions), and concurrent runs could cause
		// a corrupted state file.
		server.StateMutex.Lock()
		stateOnDisk.PrevStateByServer[server.Config.Identifier] = server.PrevState
		server.StateMutex.Unlock()
		server.HighFreqStateMutex.Lock()
		stateOnDisk.HighFreqPrevStateByServer[server.Config.Identifier] = server.HighFreqPrevState
		server.HighFreqStateMutex.Unlock()
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

// ReadStateFile - This reads in the state structs from the state file - only run this on initial bootup and SIGHUP!
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
		// We do not hold the state mutexes here since reads of the state file occur at a time
		// when there is no concurrent runs yet (at startup of the collector, or when reloading).
		prevState, exist := stateOnDisk.PrevStateByServer[server.Config.Identifier]
		if exist {
			prefixedLogger := logger.WithPrefix(server.Config.SectionName)
			prefixedLogger.PrintVerbose("Successfully recovered state from on-disk file")
			servers[idx].PrevState = prevState
		}
		prevHighFreqState, exist := stateOnDisk.HighFreqPrevStateByServer[server.Config.Identifier]
		if exist {
			prefixedLogger := logger.WithPrefix(server.Config.SectionName)
			prefixedLogger.PrintVerbose("Successfully recovered high freq state from on-disk file")
			servers[idx].HighFreqPrevState = prevHighFreqState
		}
	}
}
