package state

import (
	"os"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	uuid "github.com/satori/go.uuid"
)

type GrantLogs struct {
	Valid         bool
	Logdata       GrantS3                `json:"logdata"`
	Snapshot      GrantS3                `json:"snapshot"`
	EncryptionKey GrantLogsEncryptionKey `json:"encryption_key"`
}

type GrantLogsEncryptionKey struct {
	CiphertextBlob string `json:"ciphertext_blob"`
	KeyId          string `json:"key_id"`
	Plaintext      string `json:"plaintext"`
}

type LogState struct {
	CollectedAt time.Time

	LogFiles     []LogFile
	QuerySamples []PostgresQuerySample
}

// LogFile - Log file that we are uploading for reference in log line metadata
type LogFile struct {
	LogLines []LogLine

	UUID       uuid.UUID
	S3Location string
	S3CekAlgo  string
	S3CmkKeyID string

	ByteSize     int64
	OriginalName string

	TmpFile *os.File
}

// LogLine - "Line" in a PostgreSQL log file - can be multiple lines if they belong together
type LogLine struct {
	UUID       uuid.UUID
	ParentUUID uuid.UUID

	ByteStart        int64
	ByteContentStart int64
	ByteEnd          int64

	OccurredAt     time.Time
	Username       string
	Database       string
	Query          string
	SchemaName     string
	RelationName   string
	ConstraintName string

	// Only used for collector-internal bookkeeping to determine how long to wait
	// for associating related loglines with each other
	CollectedAt time.Time

	LogLevel   pganalyze_collector.LogLineInformation_LogLevel
	BackendPid int32

	Content string

	Classification pganalyze_collector.LogLineInformation_LogClassification

	Details map[string]interface{}
}

func (logFile LogFile) Cleanup() {
	os.Remove(logFile.TmpFile.Name())
}

func (ls LogState) Cleanup() {
	for _, logFile := range ls.LogFiles {
		logFile.Cleanup()
	}
}
