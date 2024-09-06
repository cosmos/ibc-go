package logging

import (
	"fmt"

	coreevents "cosmossdk.io/core/event"
)

// SdkEventsToLogArguments converts a given sdk.Events and returns a slice of strings that provide human
// readable values for the event attributes.
func SdkEventsToLogArguments(events coreevents.Events) []string {
	logArgs := []string{"events"}
	for _, e := range events.Events {
		logArgs = append(logArgs, fmt.Sprintf("type=%s", e.Type))
		for _, attr := range e.Attributes {
			if len(attr.Value) == 0 {
				continue
			}
			logArgs = append(logArgs, fmt.Sprintf("%s=%s", attr.Key, attr.Value))
		}
	}
	return logArgs
}
