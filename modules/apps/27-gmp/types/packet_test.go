package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestGMPPacketData_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		packetData types.GMPPacketData
		expErr     error
	}{
		{
			"success: valid packet",
			types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", []byte("salt"), []byte("payload"), "memo"),
			nil,
		},
		{
			"success: empty receiver is allowed",
			types.NewGMPPacketData("cosmos1sender", "", []byte("salt"), []byte("payload"), "memo"),
			nil,
		},
		{
			"success: empty salt is allowed",
			types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", []byte{}, []byte("payload"), "memo"),
			nil,
		},
		{
			"success: empty payload is allowed",
			types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", []byte("salt"), []byte{}, "memo"),
			nil,
		},
		{
			"success: empty memo is allowed",
			types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", []byte("salt"), []byte("payload"), ""),
			nil,
		},
		{
			"failure: empty sender",
			types.NewGMPPacketData("", "cosmos1receiver", []byte("salt"), []byte("payload"), "memo"),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: whitespace-only sender",
			types.NewGMPPacketData("   ", "cosmos1receiver", []byte("salt"), []byte("payload"), "memo"),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: receiver too long",
			types.NewGMPPacketData("cosmos1sender", ibctesting.GenerateString(types.MaximumReceiverLength+1), []byte("salt"), []byte("payload"), "memo"),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: payload too long",
			types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", []byte("salt"), make([]byte, types.MaximumPayloadLength+1), "memo"),
			types.ErrInvalidPayload,
		},
		{
			"failure: salt too long",
			types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", make([]byte, types.MaximumSaltLength+1), []byte("payload"), "memo"),
			types.ErrInvalidSalt,
		},
		{
			"failure: memo too long",
			types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", []byte("salt"), []byte("payload"), ibctesting.GenerateString(types.MaximumMemoLength+1)),
			types.ErrInvalidMemo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.packetData.ValidateBasic()
			if tc.expErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expErr)
			}
		})
	}
}

func TestMarshalUnmarshalPacketData(t *testing.T) {
	packetData := &types.GMPPacketData{
		Sender:   "cosmos1sender",
		Receiver: "cosmos1receiver",
		Salt:     []byte("randomsalt"),
		Payload:  []byte("test payload"),
		Memo:     "test memo",
	}

	testCases := []struct {
		name     string
		encoding string
		expErr   error
	}{
		{
			"success: JSON encoding",
			types.EncodingJSON,
			nil,
		},
		{
			"success: Protobuf encoding",
			types.EncodingProtobuf,
			nil,
		},
		{
			"success: ABI encoding",
			types.EncodingABI,
			nil,
		},
		{
			"failure: invalid encoding",
			"invalid-encoding",
			types.ErrInvalidEncoding,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal
			bz, err := types.MarshalPacketData(packetData, types.Version, tc.encoding)
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, bz)

			// Unmarshal
			decoded, err := types.UnmarshalPacketData(bz, types.Version, tc.encoding)
			require.NoError(t, err)

			// Compare
			require.Equal(t, packetData.Sender, decoded.Sender)
			require.Equal(t, packetData.Receiver, decoded.Receiver)
			require.Equal(t, packetData.Salt, decoded.Salt)
			require.Equal(t, packetData.Payload, decoded.Payload)
			require.Equal(t, packetData.Memo, decoded.Memo)
		})
	}
}

func TestUnmarshalPacketData_InvalidEncoding(t *testing.T) {
	_, err := types.UnmarshalPacketData([]byte("data"), types.Version, "invalid-encoding")
	require.ErrorIs(t, err, types.ErrInvalidEncoding)
}

func TestUnmarshalPacketData_InvalidData(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		encoding string
		expErr   error
	}{
		{
			"failure: invalid JSON data",
			[]byte("not valid json"),
			types.EncodingJSON,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: invalid Protobuf data",
			[]byte("not valid protobuf"),
			types.EncodingProtobuf,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: invalid ABI data",
			[]byte("not valid abi"),
			types.EncodingABI,
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := types.UnmarshalPacketData(tc.data, types.Version, tc.encoding)
			require.ErrorIs(t, err, tc.expErr)
		})
	}
}

func TestMsgSendCall_ValidateBasic(t *testing.T) {
	validSender := "cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du"

	testCases := []struct {
		name   string
		msg    *types.MsgSendCall
		expErr error
	}{
		{
			"success: valid message",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				"receiver",
				[]byte("payload"),
				[]byte("salt"),
				1000000000,
				types.EncodingABI,
				"memo",
			),
			nil,
		},
		{
			"success: empty encoding defaults to valid",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				"receiver",
				[]byte("payload"),
				[]byte("salt"),
				1000000000,
				"",
				"memo",
			),
			nil,
		},
		{
			"success: empty receiver is allowed",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				"",
				[]byte("payload"),
				[]byte("salt"),
				1000000000,
				types.EncodingABI,
				"",
			),
			nil,
		},
		{
			"failure: invalid source client ID - too short",
			types.NewMsgSendCall(
				"abc",
				validSender,
				"receiver",
				[]byte("payload"),
				[]byte("salt"),
				1000000000,
				types.EncodingABI,
				"memo",
			),
			host.ErrInvalidID,
		},
		{
			"failure: invalid sender address",
			types.NewMsgSendCall(
				"07-tendermint-0",
				"not-a-bech32-address",
				"receiver",
				[]byte("payload"),
				[]byte("salt"),
				1000000000,
				types.EncodingABI,
				"memo",
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: zero timeout timestamp",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				"receiver",
				[]byte("payload"),
				[]byte("salt"),
				0,
				types.EncodingABI,
				"memo",
			),
			types.ErrInvalidTimeoutTimestamp,
		},
		{
			"failure: invalid encoding",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				"receiver",
				[]byte("payload"),
				[]byte("salt"),
				1000000000,
				"invalid-encoding",
				"memo",
			),
			types.ErrInvalidEncoding,
		},
		{
			"failure: receiver too long",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				strings.Repeat("a", types.MaximumReceiverLength+1),
				[]byte("payload"),
				[]byte("salt"),
				1000000000,
				types.EncodingABI,
				"memo",
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: payload too long",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				"receiver",
				make([]byte, types.MaximumPayloadLength+1),
				[]byte("salt"),
				1000000000,
				types.EncodingABI,
				"memo",
			),
			types.ErrInvalidPayload,
		},
		{
			"failure: salt too long",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				"receiver",
				[]byte("payload"),
				make([]byte, types.MaximumSaltLength+1),
				1000000000,
				types.EncodingABI,
				"memo",
			),
			types.ErrInvalidSalt,
		},
		{
			"failure: memo too long",
			types.NewMsgSendCall(
				"07-tendermint-0",
				validSender,
				"receiver",
				[]byte("payload"),
				[]byte("salt"),
				1000000000,
				types.EncodingABI,
				strings.Repeat("m", types.MaximumMemoLength+1),
			),
			types.ErrInvalidMemo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expErr)
			}
		})
	}
}
