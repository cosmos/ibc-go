package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

// TestValidate tests the Validate function of Packet
func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		payload types.Payload
		expErr  error
	}{
		{
			"success",
			types.NewPayload("ics20-v1", "json", mock.MockPacketData),
			nil,
		},
		{
			"failure: empty version",
			types.NewPayload("", "json", mock.MockPacketData),
			types.ErrInvalidPayload,
		},
		{
			"failure: empty encoding",
			types.NewPayload("ics20-v2", "", mock.MockPacketData),
			types.ErrInvalidPayload,
		},
		{
			"failure: empty value",
			types.NewPayload("ics20-v1", "json", []byte{}),
			types.ErrInvalidPayload,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.payload.Validate()
			if tc.expErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expErr)
			}
		})
	}
}

// TestValidateBasic tests the ValidateBasic functio of Packet
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
			"failure: empty data",
			func() {
				packet.Data = []types.PacketData{}
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: invalid data source port ID",
			func() {
				packet.Data[0].SourcePort = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid data dest port ID",
			func() {
				packet.Data[0].DestinationPort = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid source channel ID",
			func() {
				packet.SourceId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid dest channel ID",
			func() {
				packet.DestinationId = ""
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
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			packet = types.NewPacket(1, ibctesting.FirstClientID, ibctesting.FirstClientID, uint64(time.Now().Unix()), types.PacketData{
				SourcePort:      ibctesting.MockPort,
				DestinationPort: ibctesting.MockPort,
				Payload: types.Payload{
					Version:  "ics20-v2",
					Encoding: "json",
					Value:    mock.MockPacketData,
				},
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
