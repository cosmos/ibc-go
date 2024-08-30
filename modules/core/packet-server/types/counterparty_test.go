package types_test

import (
	"testing"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/stretchr/testify/require"
)

func TestValidateCounterparty(t *testing.T) {
	testCases := []struct {
		name             string
		clientID         string
		merklePathPrefix commitmenttypes.MerklePath
		expError         error
	}{
		{
			"success",
			ibctesting.FirstClientID,
			commitmenttypes.NewMerklePath([]byte("ibc")),
			nil,
		},
		{
			"failure: invalid client id",
			"",
			commitmenttypes.NewMerklePath([]byte("ibc")),
			host.ErrInvalidID,
		},
		{
			"failure: empty merkle path prefix",
			ibctesting.FirstClientID,
			commitmenttypes.NewMerklePath(),
			types.ErrInvalidCounterparty,
		},
	}

	for _, tc := range testCases {
		tc := tc

		counterparty := types.NewCounterparty(tc.clientID, tc.merklePathPrefix)
		err := counterparty.Validate()

		expPass := tc.expError == nil
		if expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
			require.ErrorIs(t, err, tc.expError)
		}
	}
}
