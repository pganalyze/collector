package runner

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"time"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/proto"
)

func SetupWebsocketForAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) {
	if opts.ForceEmptyGrant {
		return
	}
	for idx := range servers {
		logger = logger.WithPrefixAndRememberErrors(servers[idx].Config.SectionName)
		servers[idx].WebSocket = util.NewReconnectingSocket(
			ctx, logger,
			config.CreateWebSocketDialer(servers[idx].Config), servers[idx].Config.WebSocketUrl, config.APIHeaders(servers[idx].Config, opts.TestRun),
			1*time.Minute, 8*time.Minute,
		)

		// Server messages are read in processServerMessages, snapshots are sent via output.SetupSnapshotUploadForAllServers
		go processServerMessages(ctx, servers[idx], logger)
	}
}

func processServerMessages(ctx context.Context, server *state.Server, logger *util.Logger) {
	initialConfig := true
	for {
		select {
		case <-ctx.Done():
			return
		case compressedData := <-server.WebSocket.Read:
			var data bytes.Buffer
			r, err := zlib.NewReader(bytes.NewReader(compressedData))
			if err != nil {
				logger.PrintWarning("Error decompressing websocket data: %s", err)
				continue
			}
			io.Copy(&data, r)
			r.Close()

			message := &pganalyze_collector.ServerMessage{}
			err = proto.Unmarshal(data.Bytes(), message)
			if err != nil {
				logger.PrintWarning("Error parsing ServerMessage: %s", err)
				continue
			}
			if message.GetConfig() != nil {
				grant := *server.Grant.Load()
				grant.Config = *message.GetConfig()
				grant.ValidConfig = true
				server.Grant.Store(&grant)
				if initialConfig {
					close(server.InitialConfigReceived)
					initialConfig = false
				}
			} else if message.GetPause() != nil {
				server.Pause.Store(message.GetPause().Pause)
			} else if message.GetQueryRun() != nil {
				q := message.GetQueryRun()
				logger.PrintVerbose("Query run %d received: %s", q.Id, q.QueryText)
				server.QueryRunsMutex.Lock()
				if _, exists := server.QueryRuns[q.Id]; !exists {
					parameters := []null.String{}
					for _, p := range q.QueryParameters {
						parameters = append(parameters, null.NewString(p.Value, p.Valid))
					}
					server.QueryRuns[q.Id] = &state.QueryRun{
						Id:                  q.Id,
						Type:                q.Type,
						DatabaseName:        q.DatabaseName,
						QueryText:           q.QueryText,
						QueryParameters:     parameters,
						QueryParameterTypes: q.QueryParameterTypes,
						PostgresSettings:    q.PostgresSettings,
					}
				}
				server.QueryRunsMutex.Unlock()
			}
		}
	}
}
