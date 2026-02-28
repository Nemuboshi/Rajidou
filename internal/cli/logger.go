package cli

import (
	"fmt"
	"os"
)

// Logger writes human-readable status messages to stdout/stderr.
type Logger struct{}

// Info prints an informational message.
func (Logger) Info(msg string) {
	writeLog(os.Stdout, "[INFO]", msg)
}

// Warn prints a warning message.
func (Logger) Warn(msg string) {
	writeLog(os.Stdout, "[WARN]", msg)
}

// Error prints an error message.
func (Logger) Error(msg string) {
	writeLog(os.Stderr, "[ERROR]", msg)
}

// Success prints a success message.
func (Logger) Success(msg string) {
	writeLog(os.Stdout, "[OK]", msg)
}

// Failure prints a failed-operation message.
func (Logger) Failure(msg string) {
	writeLog(os.Stdout, "[FAIL]", msg)
}

func writeLog(stream *os.File, level, msg string) {
	outputMu.Lock()
	defer outputMu.Unlock()
	fmt.Fprintln(stream, level, msg)
}
