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
		name        string
		encoding    string
		invalidData []byte
		expErr      error
	}{
		{"success: JSON encoding", types.EncodingJSON, nil, nil},
		{"success: Protobuf encoding", types.EncodingProtobuf, nil, nil},
		{"success: ABI encoding", types.EncodingABI, nil, nil},
		{"failure: invalid encoding on marshal", "invalid-encoding", nil, types.ErrInvalidEncoding},
		{"failure: invalid encoding on unmarshal", "invalid-encoding", []byte("data"), types.ErrInvalidEncoding},
		{"failure: invalid JSON data", types.EncodingJSON, []byte("not valid json"), ibcerrors.ErrInvalidType},
		{"failure: invalid Protobuf data", types.EncodingProtobuf, []byte("not valid protobuf"), ibcerrors.ErrInvalidType},
		{"failure: invalid ABI data", types.EncodingABI, []byte("not valid abi"), ibcerrors.ErrInvalidType},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.invalidData != nil {
				_, err := types.UnmarshalAcknowledgement(tc.invalidData, types.Version, tc.encoding)
				require.ErrorIs(t, err, tc.expErr)
				return
			}

			bz, err := types.MarshalAcknowledgement(ack, types.Version, tc.encoding)
			if tc.expErr != nil {
				require.ErrorIs(t, err, tc.expErr)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, bz)

			decoded, err := types.UnmarshalAcknowledgement(bz, types.Version, tc.encoding)
			require.NoError(t, err)
			require.Equal(t, ack.Result, decoded.Result)
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
