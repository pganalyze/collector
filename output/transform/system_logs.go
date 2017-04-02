package transform

import (
	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformSystemLogs(s snapshot.FullSnapshot, transientState state.TransientState, roleNameToIdx NameToIdx, databaseNameToIdx NameToIdx) snapshot.FullSnapshot {
	for _, logLineIn := range transientState.Logs {
		logLine := transformSystemLogLine(&s, logLineIn, roleNameToIdx, databaseNameToIdx)
		s.Logs = append(s.Logs, &logLine)
	}

	return s
}

func transformSystemLogLine(s *snapshot.FullSnapshot, logLineIn state.LogLine, roleNameToIdx NameToIdx, databaseNameToIdx NameToIdx) snapshot.LogLine {
	occurredAt, _ := ptypes.TimestampProto(logLineIn.OccurredAt)
	roleIdx, hasRoleIdx := roleNameToIdx[logLineIn.Username]
	databaseIdx, hasDatabaseIdx := databaseNameToIdx[logLineIn.Database]

	logLine := snapshot.LogLine{
		ClientHostname: logLineIn.ClientHostname,
		ClientPort:     logLineIn.ClientPort,
		LogLevel:       logLineIn.LogLevel,
		BackendPid:     logLineIn.BackendPid,
		Content:        logLineIn.Content,
		OccurredAt:     occurredAt,
		RoleIdx:        roleIdx,
		HasRoleIdx:     hasRoleIdx,
		DatabaseIdx:    databaseIdx,
		HasDatabaseIdx: hasDatabaseIdx,
	}

	if hasRoleIdx && hasDatabaseIdx && logLineIn.Query != "" {
		logLine.QueryIdx = upsertQueryReferenceAndInformationSimple(
			s,
			roleIdx,
			databaseIdx,
			logLineIn.Query,
		)
		logLine.HasQueryIdx = true
	}

	for _, additionalLineIn := range logLineIn.AdditionalLines {
		additionalLine := transformSystemLogLine(s, additionalLineIn, roleNameToIdx, databaseNameToIdx)
		logLine.AdditionalLines = append(logLine.AdditionalLines, &additionalLine)
	}

	return logLine
}
