package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

func TestToLowerOnPacketDirection(t *testing.T) {
	send := types.PACKET_SEND
	lower := strings.ToLower(send.String())
	require.Equal(t, "packet_send", lower)

	recv := types.PACKET_RECV
	lower = strings.ToLower(recv.String())
	require.Equal(t, "packet_recv", lower)
}
