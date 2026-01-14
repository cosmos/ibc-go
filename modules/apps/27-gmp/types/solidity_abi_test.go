package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

func TestEncodeDecodeABIGMPPacketData(t *testing.T) {
	testCases := []struct {
		name        string
		packetData  *types.GMPPacketData
		invalidData []byte
		expErr      error
	}{
		{
			"success: all fields populated",
			&types.GMPPacketData{
				Sender:   "cosmos1sender",
				Receiver: "cosmos1receiver",
				Salt:     []byte("randomsalt"),
				Payload:  []byte("some payload data"),
				Memo:     "test memo",
			},
			nil,
			nil,
		},
		{
			"success: empty salt",
			&types.GMPPacketData{
				Sender:   "cosmos1sender",
				Receiver: "cosmos1receiver",
				Salt:     []byte{},
				Payload:  []byte("payload"),
				Memo:     "memo",
			},
			nil,
			nil,
		},
		{
			"success: empty payload",
			&types.GMPPacketData{
				Sender:   "cosmos1sender",
				Receiver: "cosmos1receiver",
				Salt:     []byte("salt"),
				Payload:  []byte{},
				Memo:     "memo",
			},
			nil,
			nil,
		},
		{
			"success: empty memo",
			&types.GMPPacketData{
				Sender:   "cosmos1sender",
				Receiver: "cosmos1receiver",
				Salt:     []byte("salt"),
				Payload:  []byte("payload"),
				Memo:     "",
			},
			nil,
			nil,
		},
		{
			"success: empty receiver",
			&types.GMPPacketData{
				Sender:   "cosmos1sender",
				Receiver: "",
				Salt:     []byte("salt"),
				Payload:  []byte("payload"),
				Memo:     "memo",
			},
			nil,
			nil,
		},
		{
			"success: all optional fields empty",
			&types.GMPPacketData{
				Sender:   "cosmos1sender",
				Receiver: "",
				Salt:     []byte{},
				Payload:  []byte{},
				Memo:     "",
			},
			nil,
			nil,
		},
		{
			"success: large payload",
			&types.GMPPacketData{
				Sender:   "cosmos1sender",
				Receiver: "cosmos1receiver",
				Salt:     []byte("salt"),
				Payload:  make([]byte, 1024), // 1KB payload
				Memo:     "memo",
			},
			nil,
			nil,
		},
		{
			"failure: empty data",
			nil,
			[]byte{},
			types.ErrAbiDecoding,
		},
		{
			"failure: invalid abi data",
			nil,
			[]byte("not valid abi encoded data"),
			types.ErrAbiDecoding,
		},
		{
			"failure: truncated data",
			nil,
			[]byte{0x00, 0x01, 0x02, 0x03},
			types.ErrAbiDecoding,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.invalidData != nil {
				_, err := types.DecodeABIGMPPacketData(tc.invalidData)
				require.ErrorIs(t, err, tc.expErr)
				return
			}

			encoded, err := types.EncodeABIGMPPacketData(tc.packetData)
			require.NoError(t, err)
			require.NotEmpty(t, encoded)

			decoded, err := types.DecodeABIGMPPacketData(encoded)
			require.NoError(t, err)

			require.Equal(t, tc.packetData.Sender, decoded.Sender)
			require.Equal(t, tc.packetData.Receiver, decoded.Receiver)
			require.Equal(t, tc.packetData.Salt, decoded.Salt)
			require.Equal(t, tc.packetData.Payload, decoded.Payload)
			require.Equal(t, tc.packetData.Memo, decoded.Memo)
		})
	}
}

func TestEncodeDecodeABIAcknowledgement(t *testing.T) {
	testCases := []struct {
		name        string
		ack         *types.Acknowledgement
		invalidData []byte
		expErr      error
	}{
		{
			"success: non-empty result",
			&types.Acknowledgement{
				Result: []byte("success result data"),
			},
			nil,
			nil,
		},
		{
			"success: empty result",
			&types.Acknowledgement{
				Result: []byte{},
			},
			nil,
			nil,
		},
		{
			"success: large result",
			&types.Acknowledgement{
				Result: make([]byte, 1024), // 1KB result
			},
			nil,
			nil,
		},
		{
			"failure: empty data",
			nil,
			[]byte{},
			types.ErrAbiDecoding,
		},
		{
			"failure: invalid abi data",
			nil,
			[]byte("not valid abi encoded data"),
			types.ErrAbiDecoding,
		},
		{
			"failure: truncated data",
			nil,
			[]byte{0x00, 0x01, 0x02, 0x03},
			types.ErrAbiDecoding,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.invalidData != nil {
				_, err := types.DecodeABIAcknowledgement(tc.invalidData)
				require.ErrorIs(t, err, tc.expErr)
				return
			}

			encoded, err := types.EncodeABIAcknowledgement(tc.ack)
			require.NoError(t, err)
			require.NotEmpty(t, encoded)

			decoded, err := types.DecodeABIAcknowledgement(encoded)
			require.NoError(t, err)

			require.Equal(t, tc.ack.Result, decoded.Result)
		})
	}
}
