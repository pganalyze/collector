package planetscale

import (
	"context"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState returns system information for a PlanetScale Postgres instance.
// PlanetScale doesn't expose detailed system metrics via this API, so we return
// minimal information.
func GetSystemState(ctx context.Context, server *state.Server, logger *util.Logger) state.SystemState {
	system := state.SystemState{}
	system.Info.Type = state.PlanetScaleSystem
	system.Info.SystemID = server.Config.SystemID
	system.Info.SystemScope = server.Config.SystemScope

	// PlanetScale doesn't expose detailed system metrics (CPU, memory, disk) via the logs API.
	// These metrics would need to come from Postgres directly or a future metrics API.
	server.SelfTest.MarkCollectionAspectNotAvailable(state.CollectionAspectSystemStats, "not available on this platform")

	return system
}
