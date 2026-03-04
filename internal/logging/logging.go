package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// Logger provides structured JSON logging
type Logger struct {
	logger *log.Logger
}

// New creates a new structured logger
func New() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
	}
}

// logEntry is the internal log format
type logEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Fields    interface{} `json:"fields,omitempty"`
}

// Info logs an info-level message with optional fields
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log("info", msg, fields)
}

// Error logs an error-level message with optional fields
func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log("error", msg, fields)
}

// Warn logs a warning-level message with optional fields
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log("warn", msg, fields)
}

func (l *Logger) log(level, msg string, fields map[string]interface{}) {
	entry := logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		Fields:    fields,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf(`{"timestamp":"%s","level":"error","message":"failed to marshal log entry"}`, time.Now().UTC().Format(time.RFC3339))
		return
	}
	l.logger.Print(string(data))
}
