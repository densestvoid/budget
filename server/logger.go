package server

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Logger provides structured logging capabilities
type Logger struct {
	level  LogLevel
	output *log.Logger
}

// NewLogger creates a new structured logger
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:  level,
		output: log.New(os.Stdout, "", 0), // No default formatting, we'll handle it
	}
}

// NewLoggerFromEnv creates a logger from environment variables
func NewLoggerFromEnv() *Logger {
	levelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	if levelStr == "" {
		levelStr = "INFO"
	}

	var level LogLevel
	switch levelStr {
	case "DEBUG":
		level = LogLevelDebug
	case "INFO":
		level = LogLevelInfo
	case "WARN", "WARNING":
		level = LogLevelWarn
	case "ERROR":
		level = LogLevelError
	case "FATAL":
		level = LogLevelFatal
	default:
		level = LogLevelInfo
	}

	return NewLogger(level)
}

// shouldLog determines if a message should be logged based on level
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// log writes a structured log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}, err error) {
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Level:     level.String(),
		Message:   message,
		Timestamp: time.Now().UTC(),
		Fields:    fields,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	jsonBytes, jsonErr := json.Marshal(entry)
	if jsonErr != nil {
		// Fallback to simple logging if JSON marshaling fails
		l.output.Printf("LOG_ERROR: Failed to marshal log entry: %v", jsonErr)
		l.output.Printf("%s: %s", level.String(), message)
		return
	}

	l.output.Println(string(jsonBytes))
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelDebug, message, f, nil)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelInfo, message, f, nil)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelWarn, message, f, nil)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelError, message, f, err)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelFatal, message, f, err)
	os.Exit(1)
}

// WithFields creates a logger with default fields
func (l *Logger) WithFields(fields map[string]interface{}) *LoggerWithFields {
	return &LoggerWithFields{
		logger: l,
		fields: fields,
	}
}

// LoggerWithFields is a logger with predefined fields
type LoggerWithFields struct {
	logger *Logger
	fields map[string]interface{}
}

// mergeFields combines default fields with additional fields
func (lwf *LoggerWithFields) mergeFields(additional map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	
	// Copy default fields
	for k, v := range lwf.fields {
		merged[k] = v
	}
	
	// Add additional fields (they override defaults)
	for k, v := range additional {
		merged[k] = v
	}
	
	return merged
}

// Debug logs a debug message with default fields
func (lwf *LoggerWithFields) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = lwf.mergeFields(fields[0])
	} else {
		f = lwf.fields
	}
	lwf.logger.log(LogLevelDebug, message, f, nil)
}

// Info logs an info message with default fields
func (lwf *LoggerWithFields) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = lwf.mergeFields(fields[0])
	} else {
		f = lwf.fields
	}
	lwf.logger.log(LogLevelInfo, message, f, nil)
}

// Warn logs a warning message with default fields
func (lwf *LoggerWithFields) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = lwf.mergeFields(fields[0])
	} else {
		f = lwf.fields
	}
	lwf.logger.log(LogLevelWarn, message, f, nil)
}

// Error logs an error message with default fields
func (lwf *LoggerWithFields) Error(message string, err error, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = lwf.mergeFields(fields[0])
	} else {
		f = lwf.fields
	}
	lwf.logger.log(LogLevelError, message, f, err)
}

// Global logger instance
var globalLogger = NewLoggerFromEnv()

// Global logging functions
func Debug(message string, fields ...map[string]interface{}) {
	globalLogger.Debug(message, fields...)
}

func Info(message string, fields ...map[string]interface{}) {
	globalLogger.Info(message, fields...)
}

func Warn(message string, fields ...map[string]interface{}) {
	globalLogger.Warn(message, fields...)
}

func Error(message string, err error, fields ...map[string]interface{}) {
	globalLogger.Error(message, err, fields...)
}

func Fatal(message string, err error, fields ...map[string]interface{}) {
	globalLogger.Fatal(message, err, fields...)
}

// WithFields creates a logger with default fields using the global logger
func WithFields(fields map[string]interface{}) *LoggerWithFields {
	return globalLogger.WithFields(fields)
}

// Helper functions for common logging patterns

// LogHTTPRequest logs an HTTP request with standard fields
func LogHTTPRequest(method, path, userAgent, remoteAddr string, duration time.Duration, statusCode int) {
	fields := map[string]interface{}{
		"method":      method,
		"path":        path,
		"user_agent":  userAgent,
		"remote_addr": remoteAddr,
		"duration_ms": duration.Milliseconds(),
		"status_code": statusCode,
	}

	message := fmt.Sprintf("%s %s - %d (%dms)", method, path, statusCode, duration.Milliseconds())
	
	if statusCode >= 500 {
		Error(message, nil, fields)
	} else if statusCode >= 400 {
		Warn(message, fields)
	} else {
		Info(message, fields)
	}
}

// LogDatabaseOperation logs database operations
func LogDatabaseOperation(operation, table string, duration time.Duration, err error, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"operation":   operation,
		"table":       table,
		"duration_ms": duration.Milliseconds(),
	}

	if len(fields) > 0 {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	message := fmt.Sprintf("DB %s on %s (%dms)", operation, table, duration.Milliseconds())
	
	if err != nil {
		Error(message, err, logFields)
	} else {
		Debug(message, logFields)
	}
}

// LogAuthentication logs authentication events
func LogAuthentication(event, email, result string, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"event":  event,
		"email":  email,
		"result": result,
	}

	if len(fields) > 0 {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	message := fmt.Sprintf("Auth %s for %s: %s", event, email, result)
	
	if result == "success" {
		Info(message, logFields)
	} else {
		Warn(message, logFields)
	}
}