package types

import (
	"testing"
	"time"

	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/testing/mock"
	"github.com/stretchr/testify/require"
)

// TestValidate tests the Validate function of Packet
func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		payload Payload
		expErr  error
	}{
		{
			"success",
			NewPayload("ics20-v1", "json", mock.MockPacketData),
			nil,
		},
		{
			"failure: empty version",
			NewPayload("", "json", mock.MockPacketData),
			ErrInvalidPayload,
		},
		{
			"failure: empty encoding",
			NewPayload("ics20-v2", "", mock.MockPacketData),
			ErrInvalidPayload,
		},
		{
			"failure: empty value",
			NewPayload("ics20-v1", "json", []byte{}),
			ErrInvalidPayload,
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
	var packet Packet
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
				packet.Data = []PacketData{}
			},
			ErrInvalidPacket,
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
			ErrInvalidPacket,
		},
		{
			"failure: invalid timestamp",
			func() {
				packet.TimeoutTimestamp = 0
			},
			ErrInvalidPacket,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			packet = NewPacket(1, "sourceChannelID", "destChannelID", uint64(time.Now().Unix()), PacketData{
				SourcePort:      "sourcePort",
				DestinationPort: "destPort",
				Payload: Payload{
					Version:  "ics20-v2",
					Encoding: "encoding",
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
