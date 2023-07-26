package mock

import "github.com/cometbft/cometbft/libs/log"

var _ log.Logger = (*MockLogger)(nil)

// MockLogger implements the Logger interface and records the messages and params passed
// to the logger methods. It is used for testing.
//
// # Example:
//
//	mockLogger := ibcmock.NewMockLogger()
//	ctx := suite.chainA.GetContext().WithLogger(mockLogger)
//	// ...
//	suite.Require().Equal("Expected debug log.", mockLogger.DebugLogs[0].Message)
type MockLogger struct {
	DebugLogs  []LogEntry
	InfoLogs   []LogEntry
	ErrorLogs  []LogEntry
	WithRecord []interface{}
}

// LogEntry is a struct that contains the message and params passed to the logger methods
type LogEntry struct {
	Message string
	Params  []interface{}
}

// NewMockLogger returns a new MockLogger
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

// Debug appends the passed message and params to the debug logs
func (l *MockLogger) Debug(msg string, params ...interface{}) {
	l.DebugLogs = append(l.DebugLogs, LogEntry{Message: msg, Params: params})
}

// Info appends the passed message and params to the info logs
func (l *MockLogger) Info(msg string, params ...interface{}) {
	l.InfoLogs = append(l.InfoLogs, LogEntry{Message: msg, Params: params})
}

// Error appends the passed message and params to the error logs
func (l *MockLogger) Error(msg string, params ...interface{}) {
	l.ErrorLogs = append(l.ErrorLogs, LogEntry{Message: msg, Params: params})
}

// With sets the WithRecord field to the passed params and returns the logger
func (l *MockLogger) With(params ...interface{}) log.Logger {
	l.WithRecord = params
	return l
}
