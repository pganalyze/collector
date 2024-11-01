package runner

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
	"golang.org/x/net/http/httpproxy"
	"google.golang.org/protobuf/proto"
)

func SetupWebsocketForAllServers(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	if globalCollectionOpts.ForceEmptyGrant {
		return
	}
	for idx := range servers {
		go func(server *state.Server) {
			logger = logger.WithPrefixAndRememberErrors(server.Config.SectionName)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if server.WebSocket.Load() == nil {
						connect(ctx, server, globalCollectionOpts, logger)
					}
					time.Sleep(3 * 60 * time.Second) // Delay between reconnect attempts
				}
			}
		}(servers[idx])
	}
}

func connect(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	connCtx, cancelConn := context.WithCancel(ctx)
	proxyConfig := httpproxy.Config{
		HTTPProxy:  server.Config.HTTPProxy,
		HTTPSProxy: server.Config.HTTPSProxy,
		NoProxy:    server.Config.NoProxy,
	}
	dialer := websocket.Dialer{
		ReadBufferSize:  10240,
		WriteBufferSize: 10240,
		Proxy: func(req *http.Request) (*url.URL, error) {
			return proxyConfig.ProxyFunc()(req.URL)
		},
	}
	url, err := url.Parse(server.Config.APIBaseURL + "/v2/snapshots/websocket")
	if err != nil {
		logger.PrintWarning("Error parsing websocket URL: %s", err)
		return
	}
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
	if globalCollectionOpts.TestRun {
		headers["Pganalyze-Test-Run"] = []string{"true"}
	}
	headers["User-Agent"] = []string{util.CollectorNameAndVersion}
	conn, response, err := dialer.DialContext(connCtx, url.String(), headers)
	if err != nil {
		cancelConn()
		logger.PrintWarning("Error starting websocket: %s %v", err, response)
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectWebSocket, "error starting WebSocket: %s", err)
		return
	}
	server.WebSocket.Store(conn)
	// Writer goroutine
	go func() {
		for {
			select {
			case <-connCtx.Done():
				socket := server.WebSocket.Swap(nil)
				if socket != nil {
					err = socket.Close()
					if err != nil {
						logger.PrintWarning("Error closing websocket: %s", err)
					}
				}
				return
			case snapshot := <-server.SnapshotStream:
				logger.PrintVerbose("Uploading snapshot to websocket")
				err = conn.WriteMessage(websocket.BinaryMessage, snapshot)
				if err != nil {
					logger.PrintError("Error uploading snapshot: %s", err)
					cancelConn()
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
			message := &pganalyze_collector.ServerMessage{}
			err = proto.Unmarshal(data.Bytes(), message)
			if err != nil {
				logger.PrintWarning("Error parsing ServerMessage: %s", err)
			} else if message.GetConfig() != nil {
				grant := *server.Grant.Load()
				grant.Config = *message.GetConfig()
				server.Grant.Store(&grant)
			} else if message.GetPause() != nil {
				server.Pause.Store(message.GetPause().Pause)
			} else if message.GetQueryRun() != nil {
				q := message.GetQueryRun()
				logger.PrintVerbose("Query run %d received: %s", q.Id, q.QueryText)
				server.QueryRunsMutex.Lock()
				server.QueryRuns = append(server.QueryRuns, state.QueryRun{
					Id:           q.Id,
					Type:         q.Type,
					DatabaseName: q.DatabaseName,
					QueryText:    q.QueryText,
				})
				server.QueryRunsMutex.Unlock()
			}
		}
	}()
}
