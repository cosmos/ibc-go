package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestValidateHop(t *testing.T) {
	tests := []struct {
		name     string
		hop      types.Hop
		expError error
	}{
		{
			"valid hop",
			types.NewHop(types.PortID, ibctesting.FirstChannelID),
			nil,
		},
		{
			"invalid hop with too short port ID",
			types.NewHop(invalidShortPort, ibctesting.FirstChannelID),
			host.ErrInvalidID,
		},
		{
			"invalid hop with too long port ID",
			types.NewHop(invalidLongPort, ibctesting.FirstChannelID),
			host.ErrInvalidID,
		},
		{
			"invalid hop with non-alpha port ID",
			types.NewHop(invalidPort, ibctesting.FirstChannelID),
			host.ErrInvalidID,
		},
		{
			"invalid hop with too long channel ID",
			types.NewHop(types.PortID, invalidLongChannel),
			host.ErrInvalidID,
		},
		{
			"invalid hop with too short channel ID",
			types.NewHop(types.PortID, invalidShortChannel),
			host.ErrInvalidID,
		},
		{
			"invalid hop with non-alpha channel ID",
			types.NewHop(types.PortID, invalidChannel),
			host.ErrInvalidID,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc

			err := tc.hop.Validate()

			if tc.expError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}
