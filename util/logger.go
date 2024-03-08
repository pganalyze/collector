package util

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
)

type Logger struct {
	Verbose        bool
	Quiet          bool
	Prefix         *string
	Destination    *log.Logger
	RememberErrors bool
	ErrorMessages  []string
	UseJSON        bool // if true, emit newline delimited JSON
}

func (logger *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{Verbose: logger.Verbose, Quiet: logger.Quiet, Destination: logger.Destination, Prefix: &prefix, UseJSON: logger.UseJSON}
}

func (logger *Logger) WithPrefixAndRememberErrors(prefix string) *Logger {
	return &Logger{Verbose: logger.Verbose, Quiet: logger.Quiet, Destination: logger.Destination, Prefix: &prefix, RememberErrors: true, UseJSON: logger.UseJSON}
}

func (logger *Logger) print(logLevel string, format string, args ...interface{}) {
	if logger.Prefix != nil {
		format = fmt.Sprintf("[%s] %s", *logger.Prefix, format)
	}

	if logger.UseJSON {
		logger.printJSON(logLevel, format, args...)
		return
	}

	format = fmt.Sprintf("%s %s", logLevel, format)
	logger.Destination.Printf(format, args...)
}

func (logger *Logger) printJSON(logLevel string, format string, args ...interface{}) {
	severity := "DEFAULT"
	switch logLevel {
	case "I":
		severity = "INFO"
	case "V":
		severity = "DEBUG"
	case "E":
		severity = "ERROR"
	case "W":
		severity = "WARNING"
	}
	entry := jsonLogEntry{
		Severity: severity,
		Message:  fmt.Sprintf(format, args...),
		Time:     time.Now().Format(time.RFC3339Nano),
	}
	bs, _ := json.Marshal(entry)
	bs = append(bs, '\n')
	_, _ = os.Stderr.Write(bs)
	return
}

var VerboseMarker = color.New(color.FgBlue).Sprintf("V")

func (logger *Logger) PrintVerbose(format string, args ...interface{}) {
	if logger.Quiet || !logger.Verbose {
		return
	}

	logger.print(VerboseMarker, format, args...)
}

// Info stays the default color
var InfoMarker = "I"

func (logger *Logger) PrintInfo(format string, args ...interface{}) {
	if logger.Quiet {
		return
	}

	logger.print(InfoMarker, format, args...)
}

var WarningMarker = color.New(color.FgHiYellow).Sprintf("W")

func (logger *Logger) PrintWarning(format string, args ...interface{}) {
	if logger.Quiet {
		return
	}

	logger.print(WarningMarker, format, args...)
}

var ErrorMarker = color.New(color.FgRed).Sprintf("E")

func (logger *Logger) PrintError(format string, args ...interface{}) {
	if logger.RememberErrors {
		logger.ErrorMessages = append(logger.ErrorMessages, fmt.Sprintf(format, args...))
	}

	logger.print(ErrorMarker, format, args...)
}

type jsonLogEntry struct {
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message,omitempty"`
	Time     string `json:"time,omitempty"`
}
