package types

import "fmt"

// TODO move to a different ile
// String returns the Hop in the format:
// <portID>/<channelID>
func (h Hop) String() string {
	return fmt.Sprintf("%s/%s", h.PortId, h.ChannelId)
}
