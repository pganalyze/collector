package transform

import (
	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformSystemLogs(s snapshot.FullSnapshot, transientState state.TransientState, roleNameToIdx NameToIdx, databaseNameToIdx NameToIdx) snapshot.FullSnapshot {
	for _, logFileIn := range transientState.LogFiles {
		fileIdx := int32(len(s.LogFileReferences))
		s.LogFileReferences = append(s.LogFileReferences, &snapshot.LogFileReference{
			Uuid:         logFileIn.UUID,
			S3Location:   logFileIn.S3Location,
			S3CekAlgo:    logFileIn.S3CekAlgo,
			S3CmkKeyId:   logFileIn.S3CmkKeyID,
			ByteSize:     logFileIn.ByteSize,
			OriginalName: logFileIn.OriginalName,
		})
		for _, logLineIn := range logFileIn.LogLines {
			logLine := transformSystemLogLine(&s, fileIdx, logLineIn, roleNameToIdx, databaseNameToIdx)
			s.LogLineInformations = append(s.LogLineInformations, &logLine)
		}
	}

	return s
}

func transformSystemLogLine(s *snapshot.FullSnapshot, logFileIdx int32, logLineIn state.LogLine, roleNameToIdx NameToIdx, databaseNameToIdx NameToIdx) snapshot.LogLineInformation {
	occurredAt, _ := ptypes.TimestampProto(logLineIn.OccurredAt)
	roleIdx, hasRoleIdx := roleNameToIdx[logLineIn.Username]
	databaseIdx, hasDatabaseIdx := databaseNameToIdx[logLineIn.Database]

	logLine := snapshot.LogLineInformation{
		LogFileIdx:        logFileIdx,
		ByteStart:         logLineIn.ByteStart,
		ByteEnd:           logLineIn.ByteEnd,
		OccurredAt:        occurredAt,
		BackendPid:        logLineIn.BackendPid,
		RoleIdx:           roleIdx,
		HasRoleIdx:        hasRoleIdx,
		DatabaseIdx:       databaseIdx,
		HasDatabaseIdx:    hasDatabaseIdx,
		LogLevel:          logLineIn.LogLevel,
		LogClassification: logLineIn.LogClassification,
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
		additionalLine := transformSystemLogLine(s, logFileIdx, additionalLineIn, roleNameToIdx, databaseNameToIdx)
		logLine.AdditionalLines = append(logLine.AdditionalLines, &additionalLine)
	}

	return logLine
}
