package state

import (
	"os"
	"strings"
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

	FilterLogSecret []LogSecretKind
}

// LogSecretKind - Enum to classify the kind of log secret identified by a marker
type LogSecretKind int

const (
	_ = iota // Reserve 0 value for nil state

	// CredentialLogSecret - Passwords and other credentials (e.g. private keys)
	CredentialLogSecret

	// ParsingErrorLogSecret - User supplied text during parsing errors - could contain anything, including credentials
	ParsingErrorLogSecret

	// StatementTextLogSecret - All statement texts (which may contain table data if not using bind parameters)
	StatementTextLogSecret

	// StatementParameterLogSecret - Bind parameters for a statement (which may contain table data for INSERT statements)
	StatementParameterLogSecret

	// TableDataLogSecret - Table data contained in constraint violations and COPY errors
	TableDataLogSecret

	// OpsLogSecret - System, network errors, file locations and configured commands (e.g. archive command)
	OpsLogSecret

	// UnidentifiedLogSecret - Text that could not be identified and might contain secrets
	UnidentifiedLogSecret
)

// AllLogSecretKinds - List of all defined secret kinds
var AllLogSecretKinds = []LogSecretKind{
	CredentialLogSecret,
	ParsingErrorLogSecret,
	StatementTextLogSecret,
	StatementParameterLogSecret,
	TableDataLogSecret,
	OpsLogSecret,
	UnidentifiedLogSecret,
}

func ParseFilterLogSecret(input string) (result []LogSecretKind) {
	for _, kind := range strings.Split(input, ",") {
		switch strings.TrimSpace(kind) {
		case "credential":
			result = append(result, CredentialLogSecret)
		case "parsing_error":
			result = append(result, ParsingErrorLogSecret)
		case "statement_text":
			result = append(result, StatementTextLogSecret)
		case "statement_parameter":
			result = append(result, StatementParameterLogSecret)
		case "table_data":
			result = append(result, TableDataLogSecret)
		case "ops":
			result = append(result, OpsLogSecret)
		case "unidentified":
			result = append(result, UnidentifiedLogSecret)
		case "all":
			result = AllLogSecretKinds
		}
	}
	return result
}

// LogSecretMarker - Marks log secrets in a log line
type LogSecretMarker struct {
	ByteStart int // Start of the secret in the log line content
	ByteEnd   int // End of the secret in the log line content
	Kind      LogSecretKind
}

// LogLine - "Line" in a Postgres log file, and the associated analysis metadata
type LogLine struct {
	UUID       uuid.UUID
	ParentUUID uuid.UUID

	ByteStart        int64
	ByteContentStart int64
	ByteEnd          int64

	OccurredAt  time.Time
	Username    string
	Database    string
	Query       string
	Application string

	SchemaName   string
	RelationName string

	// Only used for collector-internal bookkeeping to determine how long to wait
	// for associating related loglines with each other
	CollectedAt time.Time

	LogLevel   pganalyze_collector.LogLineInformation_LogLevel
	BackendPid int32

	// %l in log_line_prefix (or similar syslog equivalents)
	LogLineNumber int32

	Content string

	Classification pganalyze_collector.LogLineInformation_LogClassification

	Details map[string]interface{}

	RelatedPids []int32

	ReviewedForSecrets bool
	SecretMarkers      []LogSecretMarker
}

func (logFile LogFile) Cleanup() {
	os.Remove(logFile.TmpFile.Name())
}

func (ls LogState) Cleanup() {
	for _, logFile := range ls.LogFiles {
		logFile.Cleanup()
	}
}
