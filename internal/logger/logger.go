package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger is the main logger structure
type Logger struct {
	debugMode bool
	logger    *log.Logger
}

var defaultLogger *Logger

// Init initializes the default logger
func Init(debugMode bool) {
	defaultLogger = &Logger{
		debugMode: debugMode,
		logger:    log.New(os.Stdout, "", 0),
	}
}

// formatMessage formats a log message with timestamp and level
func (l *Logger) formatMessage(level LogLevel, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var levelStr string
	
	switch level {
	case DEBUG:
		levelStr = "DEBUG"
	case INFO:
		levelStr = "INFO "
	case WARN:
		levelStr = "WARN "
	case ERROR:
		levelStr = "ERROR"
	}
	
	return fmt.Sprintf("[%s] %s - %s", levelStr, timestamp, message)
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	// Skip DEBUG logs if not in debug mode
	if level == DEBUG && !l.debugMode {
		return
	}
	
	message := fmt.Sprintf(format, args...)
	formatted := l.formatMessage(level, message)
	l.logger.Println(formatted)
	
	// Send to Telegram notifier if enabled
	notifier := GetTelegramNotifier()
	if notifier != nil {
		levelStr := l.getLevelString(level)
		notifier.SendLog(levelStr, message)
	}
}

// getLevelString returns the string representation of log level
func (l *Logger) getLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(DEBUG, format, args...)
	}
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(INFO, format, args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(WARN, format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(ERROR, format, args...)
	}
}

// IsDebugMode returns whether debug mode is enabled
func IsDebugMode() bool {
	if defaultLogger != nil {
		return defaultLogger.debugMode
	}
	return false
}

// SendSummaryNotification sends a summary notification to Telegram
func SendSummaryNotification(groupName, summary string) {
	notifier := GetTelegramNotifier()
	if notifier != nil {
		notifier.SendSummary(groupName, summary)
	}
}

// FlushTelegramLogs flushes any buffered Telegram logs
func FlushTelegramLogs() {
	notifier := GetTelegramNotifier()
	if notifier != nil {
		notifier.Flush()
	}
}
