package types

import "time"

// TimeDelay is 4 weeks to ensure channel doesn't close on timeout
const TimeDelay = 4 * 7 * 24 * time.Hour

func GetTimeoutTimestamp(currentTime time.Time) time.Time {
	return currentTime.Add(TimeDelay)
}
