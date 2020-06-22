package azure

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func LogTestRun(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) error {
	cctx, cancel := context.WithCancel(context.Background())

	// We're testing one server at a time during the test run for now
	servers := []state.Server{server}

	logTestSucceeded := make(chan bool, 1)
	azureLogStream := make(chan AzurePostgresLogRecord, 500)
	wg := sync.WaitGroup{}
	err := SetupLogSubscriber(cctx, &wg, globalCollectionOpts, logger, servers, azureLogStream)
	if err != nil {
		return err
	}

	logReceiver(cctx, servers, azureLogStream, globalCollectionOpts, logger, logTestSucceeded)

	select {
	case <-logTestSucceeded:
		cancel()
		return nil
	case <-time.After(10 * time.Second):
		cancel()
		return fmt.Errorf("Timeout")
	}
}
