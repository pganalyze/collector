package util

import (
	"fmt"
	"log"
)

type Logger struct {
	Verbose        bool
	Quiet          bool
	Prefix         *string
	Destination    *log.Logger
	RememberErrors bool
	ErrorMessages  []string
}

func (logger *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{Verbose: logger.Verbose, Quiet: logger.Quiet, Destination: logger.Destination, Prefix: &prefix}
}

func (logger *Logger) WithPrefixAndRememberErrors(prefix string) *Logger {
	return &Logger{Verbose: logger.Verbose, Quiet: logger.Quiet, Destination: logger.Destination, Prefix: &prefix, RememberErrors: true}
}

func (logger *Logger) print(logLevel string, format string, args ...interface{}) {
	if logger.Prefix != nil {
		format = fmt.Sprintf("[%s] %s", *logger.Prefix, format)
	}

	format = fmt.Sprintf("%s %s", logLevel, format)

	logger.Destination.Printf(format, args...)
}

func (logger *Logger) PrintVerbose(format string, args ...interface{}) {
	if logger.Quiet || !logger.Verbose {
		return
	}

	logger.print("V", format, args...)
}

func (logger *Logger) PrintInfo(format string, args ...interface{}) {
	if logger.Quiet {
		return
	}

	logger.print("I", format, args...)
}

func (logger *Logger) PrintWarning(format string, args ...interface{}) {
	logger.print("W", format, args...)
}

func (logger *Logger) PrintError(format string, args ...interface{}) {
	if logger.RememberErrors {
		logger.ErrorMessages = append(logger.ErrorMessages, fmt.Sprintf(format, args...))
	}

	logger.print("E", format, args...)
}

//
// Compatibility layer for retryablehttp
//

func (logger *Logger) Info(msg string, keysAndValues ...interface{}) {
	logger.PrintInfo(msg, keysAndValues...)
}

func (logger *Logger) Debug(msg string, keysAndValues ...interface{}) {
	logger.PrintVerbose(msg, keysAndValues...)
}

func (logger *Logger) Error(msg string, keysAndValues ...interface{}) {
	logger.PrintError(msg, keysAndValues...)
}

func (logger *Logger) Warn(msg string, keysAndValues ...interface{}) {
	logger.PrintWarning(msg, keysAndValues...)
}
