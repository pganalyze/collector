package transform

import (
	"encoding/json"

	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	uuid "github.com/satori/go.uuid"
)

func LogStateToLogSnapshot(logState state.LogState) (snapshot.CompactLogSnapshot, snapshot.CompactSnapshot_BaseRefs) {
	var s snapshot.CompactLogSnapshot
	var r snapshot.CompactSnapshot_BaseRefs
	s, r = transformPostgresQuerySamples(s, r, logState)
	s, r = transformSystemLogs(s, r, logState)
	return s, r
}

func upsertRoleReference(refs []*snapshot.RoleReference, roleName string) (int32, []*snapshot.RoleReference) {
	newRef := snapshot.RoleReference{Name: roleName}

	for idx, ref := range refs {
		if ref.Name == newRef.Name {
			return int32(idx), refs
		}
	}

	idx := int32(len(refs))
	refs = append(refs, &newRef)

	return idx, refs
}

func upsertDatabaseReference(refs []*snapshot.DatabaseReference, databaseName string) (int32, []*snapshot.DatabaseReference) {
	newRef := snapshot.DatabaseReference{Name: databaseName}

	for idx, ref := range refs {
		if ref.Name == newRef.Name {
			return int32(idx), refs
		}
	}

	idx := int32(len(refs))
	refs = append(refs, &newRef)

	return idx, refs
}

func transformPostgresQuerySamples(s snapshot.CompactLogSnapshot, r snapshot.CompactSnapshot_BaseRefs, logState state.LogState) (snapshot.CompactLogSnapshot, snapshot.CompactSnapshot_BaseRefs) {
	for _, sampleIn := range logState.QuerySamples {
		occurredAt, _ := ptypes.TimestampProto(sampleIn.OccurredAt)

		if sampleIn.Username == "" || sampleIn.Database == "" || sampleIn.Query == "" {
			continue
		}

		var roleIdx, databaseIdx, queryIdx int32

		roleIdx, r.RoleReferences = upsertRoleReference(r.RoleReferences, sampleIn.Username)
		databaseIdx, r.DatabaseReferences = upsertDatabaseReference(r.DatabaseReferences, sampleIn.Database)

		queryIdx, r.QueryReferences, r.QueryInformations = upsertQueryReferenceAndInformationSimple(
			r.QueryReferences,
			r.QueryInformations,
			roleIdx,
			databaseIdx,
			sampleIn.Query,
		)

		sample := snapshot.QuerySample{
			QueryIdx:    queryIdx,
			OccurredAt:  occurredAt,
			RuntimeMs:   sampleIn.RuntimeMs,
			LogLineUuid: sampleIn.LogLineUUID.String(),
			QueryText:   sampleIn.Query,
			Parameters:  sampleIn.Parameters,

			HasExplain:    sampleIn.HasExplain,
			ExplainOutput: sampleIn.ExplainOutput,
			ExplainError:  sampleIn.ExplainError,
		}
		s.QuerySamples = append(s.QuerySamples, &sample)
	}

	return s, r
}

func transformSystemLogs(s snapshot.CompactLogSnapshot, r snapshot.CompactSnapshot_BaseRefs, logState state.LogState) (snapshot.CompactLogSnapshot, snapshot.CompactSnapshot_BaseRefs) {
	for _, logFileIn := range logState.LogFiles {
		fileIdx := int32(len(s.LogFileReferences))
		s.LogFileReferences = append(s.LogFileReferences, &snapshot.LogFileReference{
			Uuid:         logFileIn.UUID.String(),
			S3Location:   logFileIn.S3Location,
			S3CekAlgo:    logFileIn.S3CekAlgo,
			S3CmkKeyId:   logFileIn.S3CmkKeyID,
			ByteSize:     logFileIn.ByteSize,
			OriginalName: logFileIn.OriginalName,
		})
		for _, logLineIn := range logFileIn.LogLines {
			logLine := transformSystemLogLine(&r, fileIdx, logLineIn)
			s.LogLineInformations = append(s.LogLineInformations, &logLine)
		}
	}

	return s, r
}

func transformSystemLogLine(r *snapshot.CompactSnapshot_BaseRefs, logFileIdx int32, logLineIn state.LogLine) snapshot.LogLineInformation {
	occurredAt, _ := ptypes.TimestampProto(logLineIn.OccurredAt)

	logLine := snapshot.LogLineInformation{
		LogFileIdx:       logFileIdx,
		Uuid:             logLineIn.UUID.String(),
		ByteStart:        logLineIn.ByteStart,
		ByteContentStart: logLineIn.ByteContentStart,
		ByteEnd:          logLineIn.ByteEnd,
		OccurredAt:       occurredAt,
		BackendPid:       logLineIn.BackendPid,
		Level:            logLineIn.LogLevel,
		Classification:   logLineIn.Classification,
	}

	if logLineIn.ParentUUID != uuid.Nil {
		logLine.ParentUuid = logLineIn.ParentUUID.String()
	}

	if logLineIn.Details != nil {
		detailsJson, err := json.Marshal(logLineIn.Details)
		if err == nil {
			logLine.DetailsJson = string(detailsJson)
		}
	}

	if logLineIn.Username != "" {
		logLine.RoleIdx, r.RoleReferences = upsertRoleReference(r.RoleReferences, logLineIn.Username)
		logLine.HasRoleIdx = true
	}
	if logLineIn.Database != "" {
		logLine.DatabaseIdx, r.DatabaseReferences = upsertDatabaseReference(r.DatabaseReferences, logLineIn.Database)
		logLine.HasDatabaseIdx = true
	}

	if logLine.HasRoleIdx && logLine.HasDatabaseIdx && logLineIn.Query != "" {
		logLine.QueryIdx, r.QueryReferences, r.QueryInformations = upsertQueryReferenceAndInformationSimple(
			r.QueryReferences,
			r.QueryInformations,
			logLine.RoleIdx,
			logLine.DatabaseIdx,
			logLineIn.Query,
		)
		logLine.HasQueryIdx = true
	}

	return logLine
}
