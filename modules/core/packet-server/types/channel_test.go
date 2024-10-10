package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func TestValidateChannel(t *testing.T) {
	testCases := []struct {
		name             string
		clientID         string
		channelID        string
		merklePathPrefix commitmenttypes.MerklePath
		expError         error
	}{
		{
			"success",
			ibctesting.FirstClientID,
			ibctesting.FirstChannelID,
			commitmenttypes.NewMerklePath([]byte("ibc")),
			nil,
		},
		{
			"success with multiple element prefix",
			ibctesting.FirstClientID,
			ibctesting.FirstChannelID,
			commitmenttypes.NewMerklePath([]byte("ibc"), []byte("address")),
			nil,
		},
		{
			"success with multiple element prefix, last prefix empty",
			ibctesting.FirstClientID,
			ibctesting.FirstChannelID,
			commitmenttypes.NewMerklePath([]byte("ibc"), []byte("")),
			nil,
		},
		{
			"success with single empty key prefix",
			ibctesting.FirstClientID,
			ibctesting.FirstChannelID,
			commitmenttypes.NewMerklePath([]byte("")),
			nil,
		},
		{
			"failure: invalid client id",
			"",
			ibctesting.FirstChannelID,
			commitmenttypes.NewMerklePath([]byte("ibc")),
			host.ErrInvalidID,
		},
		{
			"failure: invalid channel id",
			ibctesting.FirstClientID,
			"",
			commitmenttypes.NewMerklePath([]byte("ibc")),
			host.ErrInvalidID,
		},
		{
			"failure: empty merkle path prefix",
			ibctesting.FirstClientID,
			ibctesting.FirstChannelID,
			commitmenttypes.NewMerklePath(),
			types.ErrInvalidChannel,
		},
		{
			"failure: empty key in merkle path prefix first element",
			ibctesting.FirstClientID,
			ibctesting.FirstChannelID,
			commitmenttypes.NewMerklePath([]byte(""), []byte("ibc")),
			types.ErrInvalidChannel,
		},
	}

	for _, tc := range testCases {
		tc := tc

		channel := types.NewChannel(tc.clientID, tc.channelID, tc.merklePathPrefix)
		err := channel.Validate()

		expPass := tc.expError == nil
		if expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
			require.ErrorIs(t, err, tc.expError)
		}
	}
}
