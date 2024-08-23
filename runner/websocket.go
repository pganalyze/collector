package runner

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/proto"
)

func SetupWebsocketForAllServers(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for idx := range servers {
		go func(server *state.Server) {
			logger = logger.WithPrefixAndRememberErrors(server.Config.SectionName)
			for {
				if server.WebSocket == nil {
					connect(ctx, server, globalCollectionOpts, logger)
				}
				time.Sleep(60 * time.Second) // Delay between reconnect attempts
			}
		}(servers[idx])
	}
}

func connect(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	connCtx, cancelConn := context.WithCancel(ctx)
	url, _ := url.Parse(server.Config.APIBaseURL + "/v2/websocket")
	if url.Scheme == "http" {
		url.Scheme = "ws"
	} else {
		url.Scheme = "wss"
	}
	headers := make(map[string][]string)
	headers["Pganalyze-Api-Key"] = []string{server.Config.APIKey}
	headers["Pganalyze-System-Id"] = []string{server.Config.SystemID}
	headers["Pganalyze-System-Type"] = []string{server.Config.SystemType}
	headers["Pganalyze-System-Scope"] = []string{server.Config.SystemScope}
	headers["Pganalyze-System-Id-Fallback"] = []string{server.Config.SystemIDFallback}
	headers["Pganalyze-System-Type-Fallback"] = []string{server.Config.SystemTypeFallback}
	headers["Pganalyze-System-Scope-Fallback"] = []string{server.Config.SystemScopeFallback}
	headers["User-Agent"] = []string{util.CollectorNameAndVersion}
	headers["Sec-WebSocket-Protocol"] = []string{"websocket"}
	conn, response, err := websocket.DefaultDialer.DialContext(connCtx, url.String(), headers)
	if err != nil {
		cancelConn()
		logger.PrintWarning("Error starting websocket: %s %v", err, response)
		return
	}
	server.WebSocket = conn
	server.Pause.Pause = false
	go func() {
		for {
			select {
			case <-connCtx.Done():
				err := server.WebSocket.Close()
				if err != nil {
					logger.PrintWarning("Error closing websocket: %s", err)
					return
				}
				server.WebSocket = nil
				return
			case snapshot := <-server.SnapshotStream:
				logger.PrintVerbose("Uploading snapshot to websocket")
				err = server.WebSocket.WriteMessage(websocket.BinaryMessage, snapshot)
				if err != nil {
					logger.PrintError("Error uploading snapshot: %s", err)
					cancelConn()
					return
				}
			}
		}
	}()
	go func() {
		for {
			_, compressedData, err := conn.ReadMessage()
			if err != nil {
				logger.PrintWarning("Error reading from websocket: %s", err)
				cancelConn()
				return
			}
			var data bytes.Buffer
			r, err := zlib.NewReader(bytes.NewReader(compressedData))
			if err != nil {
				logger.PrintWarning("Error decompressing ServerMessage: %s", err)
				return
			}
			defer r.Close()
			io.Copy(&data, r)
			message := &pganalyze_collector.ServerMessage{}
			err = proto.Unmarshal(data.Bytes(), message)
			if err != nil {
				logger.PrintWarning("Error parsing ServerMessage: %s", err)
			} else if message.GetConfig() != nil {
				server.Grant.Config = *message.GetConfig()
			} else if message.GetPause() != nil {
				server.Pause = *message.GetPause()
			} else if message.GetExplainRun() != nil {
				logger.PrintVerbose("ExplainRun: %v", message.GetExplainRun()) // TODO
			}
		}
	}()
}
