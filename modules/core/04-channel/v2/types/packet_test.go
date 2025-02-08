package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

// TestValidateBasic tests the ValidateBasic function of Packet
func TestValidateBasic(t *testing.T) {
	var packet types.Packet
	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"success, single payload just below MaxPayloadsSize",
			func() {
				packet.Payloads[0].Value = make([]byte, channeltypesv1.MaximumPayloadsSize-1)
			},
			nil,
		},
		{
			"failure: invalid single payloads size",
			func() {
				// bytes that are larger than MaxPayloadsSize
				packet.Payloads[0].Value = make([]byte, channeltypesv1.MaximumPayloadsSize+1)
			},
			types.ErrInvalidPacket,
		},
		// TODO: add test cases for multiple payloads when enabled (#7008)
		{
			"failure: payloads is nil",
			func() {
				packet.Payloads = nil
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: empty payload",
			func() {
				packet.Payloads = []types.Payload{}
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: invalid payload source port ID",
			func() {
				packet.Payloads[0].SourcePort = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid payload dest port ID",
			func() {
				packet.Payloads[0].DestinationPort = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid source ID",
			func() {
				packet.SourceClient = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid dest ID",
			func() {
				packet.DestinationClient = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid sequence",
			func() {
				packet.Sequence = 0
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: invalid timestamp",
			func() {
				packet.TimeoutTimestamp = 0
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: empty version",
			func() {
				packet.Payloads[0].Version = ""
			},
			types.ErrInvalidPayload,
		},
		{
			"failure: empty encoding",
			func() {
				packet.Payloads[0].Encoding = ""
			},
			types.ErrInvalidPayload,
		},
		{
			"failure: empty value",
			func() {
				packet.Payloads[0].Value = []byte{}
			},
			types.ErrInvalidPayload,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			packet = types.NewPacket(1, ibctesting.FirstChannelID, ibctesting.SecondChannelID, uint64(time.Now().Unix()), types.Payload{
				SourcePort:      ibctesting.MockPort,
				DestinationPort: ibctesting.MockPort,
				Version:         "ics20-v2",
				Encoding:        transfertypes.EncodingProtobuf,
				Value:           mock.MockPacketData,
			})

			tc.malleate()

			err := packet.ValidateBasic()
			if tc.expErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expErr)
			}
		})
	}
}
