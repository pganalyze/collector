package runner

import (
	"context"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// createServer initializes a state.Server for the given configuration,
// including its per-server HTTP clients, optional OpenTelemetry tracing
// provider, and per-server context (derived from the passed run context).
func createServer(ctx context.Context, cfg config.ServerConfig, opts state.CollectionOpts, logger *util.Logger) *state.Server {
	prefixedLogger := logger.WithPrefix(cfg.SectionName)
	prefixedLogger.PrintVerbose("Identified as api_system_type: %s, api_system_scope: %s, api_system_id: %s", cfg.SystemType, cfg.SystemScope, cfg.SystemID)

	cfg.HTTPClient = config.CreateHTTPClient(cfg, prefixedLogger, false)
	cfg.HTTPClientWithRetry = config.CreateHTTPClient(cfg, prefixedLogger, true)
	if cfg.OtelExporterOtlpEndpoint != "" {
		var err error
		cfg.OTelTracingProvider, cfg.OTelTracingProviderShutdownFunc, err = config.CreateOTelTracingProvider(ctx, cfg)
		prefixedLogger.PrintVerbose("Initializing OpenTelemetry tracing provider with endpoint: %s", cfg.OtelExporterOtlpEndpoint)
		if err != nil {
			prefixedLogger.PrintError("Failed to initialize OpenTelemetry tracing provider, disabling exports: %s", err)
		}
	}

	server := state.MakeServer(cfg, opts.TestRun)
	server.Ctx, server.CancelCtx = context.WithCancel(ctx)
	return server
}

// activateServer starts the long-running goroutines for a server (WebSocket
// connection, snapshot uploads, query runs). They stop when the server's
// context is canceled.
func activateServer(server *state.Server, opts state.CollectionOpts, logger *util.Logger) {
	SetupWebsocketForServer(server, opts, logger)
	output.SetupSnapshotUploadForServer(server, opts, logger)
	SetupQueryRunnerForServer(server, opts, logger)
}

// deactivateServer stops all long-running goroutines and connections of a
// server that is removed from monitoring at runtime
func deactivateServer(server *state.Server) {
	if server.WebSocket != nil {
		server.WebSocket.Disconnect()
	}
	server.CancelCtx()
	if server.Config.OTelTracingProviderShutdownFunc != nil {
		server.Config.OTelTracingProviderShutdownFunc(context.Background())
	}
}

// RefreshServers updates the monitored server list to match the desired
// configurations. Existing servers (matched by identifier) are kept as-is,
// new servers are created and activated, and servers no longer present are
// stopped (with their state retained in case they come back).
func RefreshServers(ctx context.Context, serverList *state.ServerList, desiredConfigs []config.ServerConfig, opts state.CollectionOpts, logger *util.Logger) {
	current := serverList.Load()
	currentByIdentifier := make(map[config.ServerIdentifier]*state.Server, len(current))
	for _, server := range current {
		currentByIdentifier[server.Config.Identifier] = server
	}

	var next []*state.Server
	seen := make(map[config.ServerIdentifier]bool)
	for _, cfg := range desiredConfigs {
		if seen[cfg.Identifier] {
			// Note this is only printed in verbose mode since RefreshServers may
			// run every discovery interval
			logger.PrintVerbose("Skipping duplicate server %s (identical to another monitored server)", cfg.SectionName)
			continue
		}
		seen[cfg.Identifier] = true

		if server, ok := currentByIdentifier[cfg.Identifier]; ok {
			next = append(next, server)
			continue
		}

		server := createServer(ctx, cfg, opts, logger)
		if prevState, highFreqPrevState, ok := serverList.TakeRetained(cfg.Identifier); ok {
			server.PrevState = prevState
			server.HighFreqPrevState = highFreqPrevState
		}
		next = append(next, server)

		prefixedLogger := logger.WithPrefix(cfg.SectionName)
		err := checkOneInitialCollectionStatus(ctx, server, opts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintVerbose("could not check initial collection status: %s", err)
		}
		activateServer(server, opts, logger)
		prefixedLogger.PrintInfo("Added server to monitoring")
	}

	serverList.Store(next)

	for identifier, server := range currentByIdentifier {
		if seen[identifier] {
			continue
		}
		serverList.Retain(server)
		deactivateServer(server)
		logger.WithPrefix(server.Config.SectionName).PrintInfo("Removed server from monitoring")
	}
}
