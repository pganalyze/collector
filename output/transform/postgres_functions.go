package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformPostgresFunctions(s snapshot.FullSnapshot, newState state.State, diffState state.DiffState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	for _, function := range newState.Functions {
		ref := snapshot.FunctionReference{
			DatabaseIdx:  databaseOidToIdx[function.DatabaseOid],
			SchemaName:   function.SchemaName,
			FunctionName: function.FunctionName,
			Arguments:    function.Arguments,
		}
		idx := int32(len(s.FunctionReferences))
		s.FunctionReferences = append(s.FunctionReferences, &ref)

		// Information
		info := snapshot.FunctionInformation{
			FunctionIdx:     idx,
			Language:        function.Language,
			Source:          function.Source,
			Config:          function.Config,
			Result:          function.Result,
			Aggregate:       function.Aggregate,
			Window:          function.Window,
			SecurityDefiner: function.SecurityDefiner,
			Leakproof:       function.Leakproof,
			Strict:          function.Strict,
			ReturnsSet:      function.ReturnsSet,
			Volatile:        function.Volatile,
		}
		if function.SourceBin.Valid {
			info.SourceBin = function.SourceBin.String
		}

		s.FunctionInformations = append(s.FunctionInformations, &info)

		// Statistic
		stats, exists := diffState.FunctionStats[function.Oid]
		if exists {
			statistic := snapshot.FunctionStatistic{
				FunctionIdx: idx,
				Calls:       stats.Calls,
				TotalTime:   stats.TotalTime,
				SelfTime:    stats.SelfTime,
			}
			s.FunctionStatistics = append(s.FunctionStatistics, &statistic)
		}
	}

	return s
}
