package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

func TestNewAcknowledgement(t *testing.T) {
	result := []byte("test result")
	ack := types.NewAcknowledgement(result)

	require.Equal(t, result, ack.Result)
}

func TestAcknowledgement_ValidateBasic(t *testing.T) {
	ack := types.NewAcknowledgement([]byte("test result"))
	err := ack.ValidateBasic()
	require.NoError(t, err)
}

func TestMarshalUnmarshalAcknowledgement(t *testing.T) {
	ack := &types.Acknowledgement{
		Result: []byte("test result"),
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
			bz, err := types.MarshalAcknowledgement(ack, types.Version, tc.encoding)
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, bz)

			// Unmarshal
			decoded, err := types.UnmarshalAcknowledgement(bz, types.Version, tc.encoding)
			require.NoError(t, err)

			// Compare
			require.Equal(t, ack.Result, decoded.Result)
		})
	}
}

func TestUnmarshalAcknowledgement_InvalidEncoding(t *testing.T) {
	_, err := types.UnmarshalAcknowledgement([]byte("data"), types.Version, "invalid-encoding")
	require.ErrorIs(t, err, types.ErrInvalidEncoding)
}

func TestUnmarshalAcknowledgement_InvalidData(t *testing.T) {
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
			_, err := types.UnmarshalAcknowledgement(tc.data, types.Version, tc.encoding)
			require.ErrorIs(t, err, tc.expErr)
		})
	}
}

func TestMarshalAcknowledgement_EmptyResult(t *testing.T) {
	ack := &types.Acknowledgement{
		Result: []byte{},
	}

	testCases := []struct {
		name            string
		encoding        string
		expectEmptyData bool // Protobuf produces empty data for empty struct
	}{
		{"JSON encoding", types.EncodingJSON, false},
		{"Protobuf encoding", types.EncodingProtobuf, true},
		{"ABI encoding", types.EncodingABI, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bz, err := types.MarshalAcknowledgement(ack, types.Version, tc.encoding)
			require.NoError(t, err)
			if !tc.expectEmptyData {
				require.NotEmpty(t, bz)
			}

			decoded, err := types.UnmarshalAcknowledgement(bz, types.Version, tc.encoding)
			require.NoError(t, err)
			require.Empty(t, decoded.Result)
		})
	}
}
