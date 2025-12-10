package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
)

func TestEncodeDecodeABIGMPPacketData(t *testing.T) {
	testCases := []struct {
		name       string
		packetData *types.GMPPacketData
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
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded, err := types.EncodeABIGMPPacketData(tc.packetData)
			require.NoError(t, err, "encoding should succeed")
			require.NotEmpty(t, encoded, "encoded data should not be empty")

			// Decode
			decoded, err := types.DecodeABIGMPPacketData(encoded)
			require.NoError(t, err, "decoding should succeed")

			// Compare
			require.Equal(t, tc.packetData.Sender, decoded.Sender, "sender mismatch")
			require.Equal(t, tc.packetData.Receiver, decoded.Receiver, "receiver mismatch")
			require.Equal(t, tc.packetData.Salt, decoded.Salt, "salt mismatch")
			require.Equal(t, tc.packetData.Payload, decoded.Payload, "payload mismatch")
			require.Equal(t, tc.packetData.Memo, decoded.Memo, "memo mismatch")
		})
	}
}

func TestDecodeABIGMPPacketData_Invalid(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{
			"empty data",
			[]byte{},
		},
		{
			"invalid abi data",
			[]byte("not valid abi encoded data"),
		},
		{
			"truncated data",
			[]byte{0x00, 0x01, 0x02, 0x03},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := types.DecodeABIGMPPacketData(tc.data)
			require.Error(t, err, "decoding invalid data should fail")
			require.ErrorIs(t, err, types.ErrAbiDecoding)
		})
	}
}

func TestEncodeDecodeABIAcknowledgement(t *testing.T) {
	testCases := []struct {
		name string
		ack  *types.Acknowledgement
	}{
		{
			"success: non-empty result",
			&types.Acknowledgement{
				Result: []byte("success result data"),
			},
		},
		{
			"success: empty result",
			&types.Acknowledgement{
				Result: []byte{},
			},
		},
		{
			"success: large result",
			&types.Acknowledgement{
				Result: make([]byte, 1024), // 1KB result
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded, err := types.EncodeABIAcknowledgement(tc.ack)
			require.NoError(t, err, "encoding should succeed")
			require.NotEmpty(t, encoded, "encoded data should not be empty")

			// Decode
			decoded, err := types.DecodeABIAcknowledgement(encoded)
			require.NoError(t, err, "decoding should succeed")

			// Compare
			require.Equal(t, tc.ack.Result, decoded.Result, "result mismatch")
		})
	}
}

func TestDecodeABIAcknowledgement_Invalid(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{
			"empty data",
			[]byte{},
		},
		{
			"invalid abi data",
			[]byte("not valid abi encoded data"),
		},
		{
			"truncated data",
			[]byte{0x00, 0x01, 0x02, 0x03},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := types.DecodeABIAcknowledgement(tc.data)
			require.Error(t, err, "decoding invalid data should fail")
			require.ErrorIs(t, err, types.ErrAbiDecoding)
		})
	}
}
