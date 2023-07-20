package mock

import "github.com/cometbft/cometbft/libs/log"

var _ log.Logger = (*MockLogger)(nil)

// MockLogger implements the Logger interface
type MockLogger struct {
	DebugLogs  []LogEntry
	InfoLogs   []LogEntry
	ErrorLogs  []LogEntry
	WithRecord []interface{}
}

// LogEntry is a struct that contains the message and params passed to the logger
type LogEntry struct {
	Message string
	Params  []interface{}
}

// NewMockLogger returns a new MockLogger
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

// DebugLogs returns the debug logs
func (l *MockLogger) Debug(msg string, params ...interface{}) {
	l.DebugLogs = append(l.DebugLogs, LogEntry{Message: msg, Params: params})
}

// InfoLogs returns the info logs
func (l *MockLogger) Info(msg string, params ...interface{}) {
	l.InfoLogs = append(l.InfoLogs, LogEntry{Message: msg, Params: params})
}

// ErrorLogs returns the error logs
func (l *MockLogger) Error(msg string, params ...interface{}) {
	l.ErrorLogs = append(l.ErrorLogs, LogEntry{Message: msg, Params: params})
}

// With returns the logger with the params
func (l *MockLogger) With(params ...interface{}) log.Logger {
	l.WithRecord = params
	return l
}
