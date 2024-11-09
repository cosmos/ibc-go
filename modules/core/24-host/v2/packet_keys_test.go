package v2_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
)

func TestPacketAcknowledgementKey(t *testing.T) {
	var (
		channelID = "channel-0"
		sequence  = uint64(1)
	)

	key := hostv2.PacketAcknowledgementKey(channelID, sequence)
	require.Equal(t, "acks/channels/channel-0/sequences/1", string(key))
}
