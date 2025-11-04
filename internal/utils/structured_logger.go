package utils

import (
	"fmt"
	"go-log/internal/config"
	"log"
	"os"
	"strings"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// StructuredLogger provides structured logging capabilities
type StructuredLogger struct {
	logger   *log.Logger
	minLevel LogLevel
}

var defaultLogger *StructuredLogger

func init() {
	defaultLogger = NewStructuredLogger()
}

// NewStructuredLogger creates a new structured logger instance
func NewStructuredLogger() *StructuredLogger {
	minLevel := INFO // Default level
	envConfig := config.GetEnvConfig()
	levelStr := envConfig.LogLevel
	
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		minLevel = DEBUG
	case "INFO":
		minLevel = INFO
	case "WARN", "WARNING":
		minLevel = WARN
	case "ERROR":
		minLevel = ERROR
	case "FATAL":
		minLevel = FATAL
	}

	return &StructuredLogger{
		logger:   log.New(os.Stderr, "", log.LstdFlags),
		minLevel: minLevel,
	}
}

// shouldLog checks if a message should be logged based on level
func (sl *StructuredLogger) shouldLog(level LogLevel) bool {
	return level >= sl.minLevel
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(format string, args ...any) {
	if !sl.shouldLog(DEBUG) {
		return
	}
	message := fmt.Sprintf(format, args...)
	sl.logger.Printf("[%s] %s", DEBUG.String(), message)
}

// Info logs an info message
func (sl *StructuredLogger) Info(format string, args ...any) {
	if !sl.shouldLog(INFO) {
		return
	}
	message := fmt.Sprintf(format, args...)
	sl.logger.Printf("[%s] %s", INFO.String(), message)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(format string, args ...any) {
	if !sl.shouldLog(WARN) {
		return
	}
	message := fmt.Sprintf(format, args...)
	sl.logger.Printf("[%s] %s", WARN.String(), message)
}

// Error logs an error message
func (sl *StructuredLogger) Error(format string, args ...any) {
	if !sl.shouldLog(ERROR) {
		return
	}
	message := fmt.Sprintf(format, args...)
	sl.logger.Printf("[%s] %s", ERROR.String(), message)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(format string, args ...any) {
	if !sl.shouldLog(FATAL) {
		return
	}
	message := fmt.Sprintf(format, args...)
	sl.logger.Printf("[%s] %s", FATAL.String(), message)
	os.Exit(1)
}

// WarnWithContext logs a warning with component context
func (sl *StructuredLogger) WarnWithContext(component, message string, err error) {
	if !sl.shouldLog(WARN) {
		return
	}
	if err != nil {
		sl.logger.Printf("[%s] [%s] %s: %v", WARN.String(), component, message, err)
	} else {
		sl.logger.Printf("[%s] [%s] %s", WARN.String(), component, message)
	}
}

// ErrorWithContext logs an error with component context
func (sl *StructuredLogger) ErrorWithContext(component, message string, err error) {
	if !sl.shouldLog(ERROR) {
		return
	}
	if err != nil {
		sl.logger.Printf("[%s] [%s] %s: %v", ERROR.String(), component, message, err)
	} else {
		sl.logger.Printf("[%s] [%s] %s", ERROR.String(), component, message)
	}
}

// InfoWithContext logs info with component context
func (sl *StructuredLogger) InfoWithContext(component, message string, err error) {
	if !sl.shouldLog(INFO) {
		return
	}
	if err != nil {
		sl.logger.Printf("[%s] [%s] %s: %v", INFO.String(), component, message, err)
	} else {
		sl.logger.Printf("[%s] [%s] %s", INFO.String(), component, message)
	}
}

// Package-level convenience functions using the default logger
func LogDebug(format string, args ...any) {
	defaultLogger.Debug(format, args...)
}

func LogInfo(format string, args ...any) {
	defaultLogger.Info(format, args...)
}

func LogWarn(format string, args ...any) {
	defaultLogger.Warn(format, args...)
}

func LogError(format string, args ...any) {
	defaultLogger.Error(format, args...)
}

func LogFatal(format string, args ...any) {
	defaultLogger.Fatal(format, args...)
}

func LogWarnWithContext(component, message string, err error) {
	defaultLogger.WarnWithContext(component, message, err)
}

func LogErrorWithContext(component, message string, err error) {
	defaultLogger.ErrorWithContext(component, message, err)
}

func LogInfoWithContext(component, message string, err error) {
	defaultLogger.InfoWithContext(component, message, err)
}