package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformPostgresFunctions(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	for _, function := range newState.Functions {
		ref := snapshot.FunctionReference{
			DatabaseIdx:  databaseOidToIdx[function.DatabaseOid],
			SchemaName:   function.SchemaName,
			FunctionName: function.FunctionName,
			Arguments:    function.Arguments,
		}
		idx := int32(len(s.FunctionReferences))
		s.FunctionReferences = append(s.FunctionReferences, &ref)

		var kind snapshot.FunctionInformation_FunctionKind
		switch function.Kind {
		case "a":
			kind = snapshot.FunctionInformation_AGGREGATE
		case "w":
			kind = snapshot.FunctionInformation_WINDOW
		case "p":
			kind = snapshot.FunctionInformation_PROCEDURE
		case "f":
			kind = snapshot.FunctionInformation_FUNCTION
		default:
			kind = snapshot.FunctionInformation_UNKNOWN
		}

		// Information
		info := snapshot.FunctionInformation{
			FunctionIdx:     idx,
			Language:        function.Language,
			Source:          function.Source,
			Config:          function.Config,
			Result:          function.Result,
			Kind:            kind,
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
