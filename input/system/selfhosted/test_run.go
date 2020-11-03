package selfhosted

import (
	"context"
	"fmt"
	"time"

	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// TestLogTail - Tests the tailing of a log file (without watching it continuously)
// as well as parsing and analyzing the log data
func TestLogTail(server *state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger) error {
	cctx, cancel := context.WithCancel(context.Background())

	logTestSucceeded := make(chan bool, 1)

	logStream := logReceiver(cctx, server, globalCollectionOpts, prefixedLogger, logTestSucceeded)
	err := setupLogLocationTail(cctx, server.Config.LogLocation, logStream, prefixedLogger)
	if err != nil {
		cancel()
		return err
	}

	logs.EmitTestLogMsg(server, globalCollectionOpts, prefixedLogger)

	select {
	case <-logTestSucceeded:
		cancel()
		return nil
	case <-time.After(10 * time.Second):
		cancel()
		return fmt.Errorf("Timeout")
	}
}
