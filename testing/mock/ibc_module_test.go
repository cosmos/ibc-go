package mock_test

import (
	"testing"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v3/testing/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateCapabilityName(t *testing.T) {
	packet := channeltypes.NewPacket(
		[]byte{},
		1,
		mock.PortID,
		"channel-0",
		mock.PortID,
		"channel-0",
		clienttypes.NewHeight(0, 100),
		0,
	)

	name := mock.GetMockRecvCanaryCapabilityName(packet)
	require.Equal(t, "mock receive canary capability namemock-portIDchannel-01", name)
}
