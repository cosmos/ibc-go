package types

// Logger is an interface that defines the required method for a log library used by an implementer
// of the beefy light client.
type Logger interface {
	Fatal(...interface{})
}

var log Logger

// UseLogger sets the log variable to an externally defined Log library
func UseLogger(l Logger) {
	log = l
}

type logg struct{}

// Errorf has no log body, it simply satisfies the logger interface
func (l logg) Fatal(params ...interface{}) {}

func init() {
	// set log to default logger which prints nothing if no logger interface is passed to the UseLogger method
	log = logg{}
}
