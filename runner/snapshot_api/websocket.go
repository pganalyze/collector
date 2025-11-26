package snapshot_api

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
	"golang.org/x/net/http/httpproxy"
	"google.golang.org/protobuf/proto"
)

func SetupSnapshotAPIForAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) {
	if opts.ForceEmptyGrant {
		return
	}
	for idx := range servers {
		snapshotStreamWebsocket := make(chan []byte)
		go func(server *state.Server) {
			logger = logger.WithPrefixAndRememberErrors(server.Config.SectionName)
			for {
				select {
				case <-ctx.Done():
					return
				case snapshot := <-server.SnapshotStream:
					if isWebSocketConnected(server) {
						snapshotStreamWebsocket <- snapshot.Data
					} else {
						s3Location, err := uploadSnapshot(ctx, server.Config.HTTPClientWithRetry, server.Grant.Load(), logger, snapshot.Data, snapshot.SnapshotUuid)
						if err != nil {
							logger.PrintError("Error uploading snapshot: %s", err)
							continue
						}
						submitSnapshot(ctx, server, opts, logger, s3Location, snapshot.CollectedAt, snapshot.CompactSnapshot)
					}

					if !opts.TestRun && !snapshot.CompactSnapshot {
						logger.PrintInfo("Submitted full snapshot successfully")
					}
				}
			}
		}(servers[idx])

		go func(server *state.Server, snapshotStreamWebsocket chan []byte) {
			logger = logger.WithPrefixAndRememberErrors(server.Config.SectionName)
			var skipConnectUntil time.Time
			for {
				select {
				case <-ctx.Done():
					return
				case <-server.WebSocketStart:
					if !isWebSocketConnected(server) && server.WebSocketRequested.Load() && time.Now().After(skipConnectUntil) {
						connectStatus := connect(ctx, server, opts, logger, snapshotStreamWebsocket)
						if connectStatus >= 400 && connectStatus < 500 {
							skipConnectUntil = time.Now().Add(time.Minute * 8) // Delay reconnect when server responds with 4xx errors
						}
					}
				case <-server.WebSocketShutdown:
					if isWebSocketConnected(server) {
						closeConnection(server, logger)
					}
				// Try reconnecting outside of requested starts in case of disconnects
				case <-time.After(60 * time.Second):
					// This is a problem since this is an unbuffered channel, so we block on ourselves
					server.WebSocketStart <- struct{}{}
				}
			}
		}(servers[idx], snapshotStreamWebsocket)
	}
}

func InitializeGrant(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) error {
	if opts.ForceEmptyGrant {
		return nil
	}

	startWebSocketIfNeeded(server)

	// Accept existing grant data if we're using a WebSocket connection and its
	// up and running and received at least one config message. Note that if we
	// started the websocket just now, this likely won't have happened yet.
	if isWebSocketConnected(server) && server.Grant.Load().Valid {
		return nil
	}

	newGrant, err := getGrant(ctx, server, opts, logger)
	if err != nil {
		return err
	}

	server.Grant.Store(&newGrant)

	return nil
}

func isWebSocketConnected(server *state.Server) bool {
	return server.WebSocket.Load() != nil
}

func startWebSocketIfNeeded(server *state.Server) {
	server.WebSocketRequested.Store(true)
	if !isWebSocketConnected(server) {
		server.WebSocketStart <- struct{}{}
	}
}

func ShutdownWebSocketIfNeeded(server *state.Server) {
	server.WebSocketRequested.Store(false)
	if isWebSocketConnected(server) {
		server.WebSocketShutdown <- struct{}{}
	}
}

func connect(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger, snapshotStreamWebsocket chan []byte) (connectStatus int) {
	connCtx, cancelConn := context.WithCancel(ctx)
	conn, connectStatus, err := openConnection(connCtx, server.Config, opts.TestRun)
	if err != nil {
		cancelConn()
		logger.PrintWarning("Error starting websocket: %s", err)
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectWebSocket, "error starting WebSocket: %s", err)
		return
	}
	server.WebSocket.Store(conn)
	// Writer goroutine
	go func() {
		for {
			select {
			case <-connCtx.Done():
				closeConnection(server, logger)
				return
			case snapshot := <-snapshotStreamWebsocket:
				logger.PrintVerbose("Uploading snapshot to websocket")
				err = conn.WriteMessage(websocket.BinaryMessage, snapshot)
				if err != nil {
					logger.PrintError("Error uploading snapshot: %s", err)
					closeConnection(server, logger)
					return
				}
			}
		}
	}()
	// Reader goroutine
	go func() {
		for {
			_, compressedData, err := conn.ReadMessage()
			if err != nil {
				serverClosed := websocket.IsCloseError(err, websocket.CloseNoStatusReceived) // The server shutdown the websocket
				shutdown := errors.Is(err, net.ErrClosed)                                    // The collector process is shutting down
				if !serverClosed && !shutdown {
					logger.PrintWarning("Error reading from websocket: %s", err)
					server.SelfTest.MarkCollectionAspectError(state.CollectionAspectWebSocket, "error reading from WebSocket: %s", err)
				}
				cancelConn()
				return
			}
			server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectWebSocket)
			var data bytes.Buffer
			r, err := zlib.NewReader(bytes.NewReader(compressedData))
			if err != nil {
				logger.PrintWarning("Error decompressing ServerMessage: %s", err)
				cancelConn()
				return
			}
			io.Copy(&data, r)
			r.Close()
			processServerMessage(server, logger, data)
		}
	}()
	return
}

func processServerMessage(server *state.Server, logger *util.Logger, data bytes.Buffer) {
	message := &pganalyze_collector.ServerMessage{}
	err := proto.Unmarshal(data.Bytes(), message)
	if err != nil {
		logger.PrintWarning("Error parsing ServerMessage: %s", err)
	} else if message.GetConfig() != nil {
		grant := *server.Grant.Load()
		grant.Config = *message.GetConfig()
		// Note: This doesn't set Valid: true itself, but will use the last HTTP grant to set it
		server.Grant.Store(&grant)
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

func openConnection(ctx context.Context, config config.ServerConfig, testRun bool) (conn *websocket.Conn, connectStatus int, err error) {
	proxyConfig := httpproxy.Config{
		HTTPProxy:  config.HTTPProxy,
		HTTPSProxy: config.HTTPSProxy,
		NoProxy:    config.NoProxy,
	}
	dialer := websocket.Dialer{
		ReadBufferSize:  10240,
		WriteBufferSize: 10240,
		Proxy: func(req *http.Request) (*url.URL, error) {
			return proxyConfig.ProxyFunc()(req.URL)
		},
	}
	url, err := url.Parse(config.APIBaseURL + "/v2/snapshots/websocket")
	if err != nil {
		err = fmt.Errorf("Error parsing websocket URL: %s", err)
		return
	}
	if url.Scheme == "http" {
		url.Scheme = "ws"
	} else {
		url.Scheme = "wss"
	}
	headers := make(map[string][]string)
	headers["Pganalyze-Api-Key"] = []string{config.APIKey}
	headers["Pganalyze-System-Id"] = []string{config.SystemID}
	headers["Pganalyze-System-Type"] = []string{config.SystemType}
	headers["Pganalyze-System-Scope"] = []string{config.SystemScope}
	headers["Pganalyze-System-Id-Fallback"] = []string{config.SystemIDFallback}
	headers["Pganalyze-System-Type-Fallback"] = []string{config.SystemTypeFallback}
	headers["Pganalyze-System-Scope-Fallback"] = []string{config.SystemScopeFallback}
	if testRun {
		headers["Pganalyze-Test-Run"] = []string{"true"}
	}
	headers["User-Agent"] = []string{util.CollectorNameAndVersion}
	conn, response, err := dialer.DialContext(ctx, url.String(), headers)
	if response != nil {
		connectStatus = response.StatusCode
	}
	if err != nil {
		err = fmt.Errorf("%s %v", err, response)
	}
	return
}

func closeConnection(server *state.Server, logger *util.Logger) {
	socket := server.WebSocket.Swap(nil)
	if socket != nil {
		err := socket.Close()
		if err != nil {
			logger.PrintWarning("Error closing websocket: %s", err)
		}
	}
}
