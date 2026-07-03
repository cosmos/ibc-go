package ratelimiting_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ratelimiting "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting"
	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
)

func TestWriteAcknowledgement_NilAck(t *testing.T) {
	middleware := ratelimiting.NewIBCMiddleware(nil)
	packet := channeltypes.Packet{
		Sequence:           1,
		DestinationChannel: "channel-0",
	}

	var ack ibcexported.Acknowledgement
	err := middleware.WriteAcknowledgement(sdk.Context{}, packet, ack)

	require.ErrorIs(t, err, types.ErrAsyncAckNil)
	require.ErrorContains(t, err, "cannot write nil ack for packet channel-0/1")
}
