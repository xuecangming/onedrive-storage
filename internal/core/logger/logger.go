package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"
)

// Level represents log level
type Level int

const (
	// DebugLevel for debug messages
	DebugLevel Level = iota
	// InfoLevel for informational messages
	InfoLevel
	// WarnLevel for warning messages
	WarnLevel
	// ErrorLevel for error messages
	ErrorLevel
	// FatalLevel for fatal messages (will exit)
	FatalLevel
)

// String returns string representation of log level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger interface defines logging operations
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	With(fields ...Field) Logger
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

// Config holds logger configuration
type Config struct {
	Level      Level
	Format     string // "json" or "text"
	Output     io.Writer
	TimeFormat string
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      InfoLevel,
		Format:     "json",
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
	}
}

// StandardLogger is the default logger implementation
type StandardLogger struct {
	config *Config
	fields []Field
	logger *log.Logger
}

// New creates a new logger
func New(config *Config) Logger {
	if config == nil {
		config = DefaultConfig()
	}

	return &StandardLogger{
		config: config,
		fields: make([]Field, 0),
		logger: log.New(config.Output, "", 0),
	}
}

// Debug logs a debug message
func (l *StandardLogger) Debug(msg string, fields ...Field) {
	if l.config.Level <= DebugLevel {
		l.log(DebugLevel, msg, fields...)
	}
}

// Info logs an info message
func (l *StandardLogger) Info(msg string, fields ...Field) {
	if l.config.Level <= InfoLevel {
		l.log(InfoLevel, msg, fields...)
	}
}

// Warn logs a warning message
func (l *StandardLogger) Warn(msg string, fields ...Field) {
	if l.config.Level <= WarnLevel {
		l.log(WarnLevel, msg, fields...)
	}
}

// Error logs an error message
func (l *StandardLogger) Error(msg string, fields ...Field) {
	if l.config.Level <= ErrorLevel {
		l.log(ErrorLevel, msg, fields...)
	}
}

// Fatal logs a fatal message and exits
func (l *StandardLogger) Fatal(msg string, fields ...Field) {
	l.log(FatalLevel, msg, fields...)
	os.Exit(1)
}

// With creates a child logger with additional fields
func (l *StandardLogger) With(fields ...Field) Logger {
	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &StandardLogger{
		config: l.config,
		fields: newFields,
		logger: l.logger,
	}
}

// log is the internal logging function
func (l *StandardLogger) log(level Level, msg string, fields ...Field) {
	// Combine logger fields with message fields
	allFields := make([]Field, len(l.fields)+len(fields))
	copy(allFields, l.fields)
	copy(allFields[len(l.fields):], fields)

	if l.config.Format == "json" {
		l.logJSON(level, msg, allFields)
	} else {
		l.logText(level, msg, allFields)
	}
}

// logJSON logs in JSON format
func (l *StandardLogger) logJSON(level Level, msg string, fields []Field) {
	entry := make(map[string]interface{})
	entry["timestamp"] = time.Now().Format(l.config.TimeFormat)
	entry["level"] = level.String()
	entry["message"] = msg

	// Add caller information
	if _, file, line, ok := runtime.Caller(3); ok {
		entry["caller"] = fmt.Sprintf("%s:%d", file, line)
	}

	// Add fields
	for _, field := range fields {
		entry[field.Key] = field.Value
	}

	data, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(data))
}

// logText logs in text format
func (l *StandardLogger) logText(level Level, msg string, fields []Field) {
	timestamp := time.Now().Format(l.config.TimeFormat)
	
	// Build field string
	fieldStr := ""
	if len(fields) > 0 {
		fieldStr = " "
		for i, field := range fields {
			if i > 0 {
				fieldStr += " "
			}
			fieldStr += fmt.Sprintf("%s=%v", field.Key, field.Value)
		}
	}

	l.logger.Printf("[%s] %s: %s%s", timestamp, level.String(), msg, fieldStr)
}

// Helper functions for creating fields

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Global logger instance
var globalLogger Logger

func init() {
	globalLogger = New(DefaultConfig())
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() Logger {
	return globalLogger
}

// Global logging functions

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...Field) {
	globalLogger.Debug(msg, fields...)
}

// Info logs an info message using the global logger
func Info(msg string, fields ...Field) {
	globalLogger.Info(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...Field) {
	globalLogger.Warn(msg, fields...)
}

// ErrorLog logs an error message using the global logger
func ErrorLog(msg string, fields ...Field) {
	globalLogger.Error(msg, fields...)
}

// Fatal logs a fatal message using the global logger
func Fatal(msg string, fields ...Field) {
	globalLogger.Fatal(msg, fields...)
}

// WithContext extracts logger from context or returns global logger
func WithContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value("logger").(Logger); ok {
		return logger
	}
	return globalLogger
}

// ToContext adds logger to context
func ToContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}
