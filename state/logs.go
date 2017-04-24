package state

import (
	"os"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
)

// LogFile - Log file that we are uploading for reference in log line metadata
type LogFile struct {
	LogLines []LogLine

	UUID       string
	S3Location string
	S3CekAlgo  string
	S3CmkKeyID string

	ByteSize     int64
	OriginalName string

	TmpFile *os.File
}

// LogLine - "Line" in a PostgreSQL log file - can be multiple lines if they belong together
type LogLine struct {
	ByteStart int64
	ByteEnd   int64

	OccurredAt time.Time
	Username   string
	Database   string
	Query      string

	LogLevel   pganalyze_collector.LogLineInformation_LogLevel
	BackendPid int32

	Content string

	LogClassification pganalyze_collector.LogLineInformation_LogClassification

	AdditionalLines []LogLine
}

func (logFile LogFile) Cleanup() {
	//os.Remove(logFile.TmpFile.Name())
}
