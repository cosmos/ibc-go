package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v11/modules/core/24-host"
)

func TestValidateGenesis(t *testing.T) {
	const channelID = "channel-0"

	testCases := []struct {
		name     string
		genState *types.GenesisState
		expErr   error
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expErr:   nil,
		},
		{
			"valid genesis",
			&types.GenesisState{
				PortId: "portidone",
			},
			nil,
		},
		{
			"invalid client",
			&types.GenesisState{
				PortId: "(INVALIDPORT)",
			},
			host.ErrInvalidID,
		},
		{
			"valid channel escrow",
			&types.GenesisState{
				PortId:        types.PortID,
				TotalEscrowed: sdk.NewCoins(sdk.NewInt64Coin("stake", 10)),
				ChannelEscrows: []types.ChannelEscrow{{
					ChannelOrClientId: "07-tendermint-0", Tokens: sdk.NewCoins(sdk.NewInt64Coin("stake", 10)),
				}},
			},
			nil,
		},
		{
			"mismatched channel escrow total",
			&types.GenesisState{
				PortId:        types.PortID,
				TotalEscrowed: sdk.NewCoins(sdk.NewInt64Coin("stake", 10)),
				ChannelEscrows: []types.ChannelEscrow{{
					ChannelOrClientId: channelID, Tokens: sdk.NewCoins(sdk.NewInt64Coin("stake", 9)),
				}},
			},
			errors.New("total escrowed"),
		},
		{
			"legacy genesis missing channel escrow",
			&types.GenesisState{
				PortId:        types.PortID,
				TotalEscrowed: sdk.NewCoins(sdk.NewInt64Coin("stake", 10)),
			},
			errors.New("total escrowed"),
		},
	}

	for _, tc := range testCases {
		err := tc.genState.Validate()
		if tc.expErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			if errors.Is(tc.expErr, host.ErrInvalidID) {
				require.ErrorIs(t, err, tc.expErr, tc.name)
			} else {
				require.ErrorContains(t, err, tc.expErr.Error(), tc.name)
			}
		}
	}
}
