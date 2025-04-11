package utils

import (
	"fmt"
	"log"
	"os"
)

const (
	// Colors
	Reset      = "\033[0m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Purple     = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
	BoldPurple = "\033[1;35m"
	BoldCyan   = "\033[1;36m"
	BoldWhite  = "\033[1;37m"
)

// Logger represents a colorful logger
type Logger struct {
	*log.Logger
}

// NewLogger creates a new colorful logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", 0),
	}
}

// Info logs an info message in green
func (l *Logger) Info(format string, v ...interface{}) {
	l.Printf("%s%s%s", Green, fmt.Sprintf(format, v...), Reset)
}

// Error logs an error message in red
func (l *Logger) Error(format string, v ...interface{}) {
	l.Printf("%s%s%s", BoldRed, fmt.Sprintf(format, v...), Reset)
}

// Debug logs a debug message in blue
func (l *Logger) Debug(format string, v ...interface{}) {
	l.Printf("%s%s%s", Blue, fmt.Sprintf(format, v...), Reset)
}

// Warn logs a warning message in yellow
func (l *Logger) Warn(format string, v ...interface{}) {
	l.Printf("%s%s%s", Yellow, fmt.Sprintf(format, v...), Reset)
}

// Request logs a request message in cyan
func (l *Logger) Request(format string, v ...interface{}) {
	l.Printf("%s%s%s", Cyan, fmt.Sprintf(format, v...), Reset)
}

// Response logs a response message in purple
func (l *Logger) Response(format string, v ...interface{}) {
	l.Printf("%s%s%s", Purple, fmt.Sprintf(format, v...), Reset)
}

// Proxy logs a proxy message in bold white
func (l *Logger) Proxy(format string, v ...interface{}) {
	l.Printf("%s%s%s", BoldWhite, fmt.Sprintf(format, v...), Reset)
}

// Header logs a header message in bold yellow
func (l *Logger) Header(format string, v ...interface{}) {
	l.Printf("%s%s%s", BoldYellow, fmt.Sprintf(format, v...), Reset)
}

// Body logs a body message in bold cyan
func (l *Logger) Body(format string, v ...interface{}) {
	l.Printf("%s%s%s", BoldCyan, fmt.Sprintf(format, v...), Reset)
}

// Separator logs a separator line in bold white
func (l *Logger) Separator() {
	l.Printf("%s%s%s", BoldWhite, "==========================================", Reset)
}

// StartRequest logs the start of a request
func (l *Logger) StartRequest() {
	l.Info("Starting new request")
}

// EndRequest logs the end of a request
func (l *Logger) EndRequest() {
	l.Info("Request completed")
}
