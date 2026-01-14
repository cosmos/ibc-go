package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestDefaultGenesisState(t *testing.T) {
	gs := types.DefaultGenesisState()
	require.NotNil(t, gs)
	require.Empty(t, gs.Ics27Accounts)
}

func TestGenesisState_Validate(t *testing.T) {
	validAddress := ibctesting.TestAccAddress
	validClientID := ibctesting.FirstClientID

	testCases := []struct {
		name     string
		genState *types.GenesisState
		expErr   bool
	}{
		{
			"success: default genesis",
			types.DefaultGenesisState(),
			false,
		},
		{
			"success: valid genesis with account",
			&types.GenesisState{
				Ics27Accounts: []types.RegisteredICS27Account{
					{
						AccountAddress: validAddress,
						AccountId: types.AccountIdentifier{
							ClientId: validClientID,
							Sender:   validAddress,
							Salt:     []byte("salt"),
						},
					},
				},
			},
			false,
		},
		{
			"failure: invalid account address",
			&types.GenesisState{
				Ics27Accounts: []types.RegisteredICS27Account{
					{
						AccountAddress: "invalid",
						AccountId: types.AccountIdentifier{
							ClientId: validClientID,
							Sender:   validAddress,
							Salt:     []byte("salt"),
						},
					},
				},
			},
			true,
		},
		{
			"failure: invalid sender address",
			&types.GenesisState{
				Ics27Accounts: []types.RegisteredICS27Account{
					{
						AccountAddress: validAddress,
						AccountId: types.AccountIdentifier{
							ClientId: validClientID,
							Sender:   "invalid",
							Salt:     []byte("salt"),
						},
					},
				},
			},
			true,
		},
		{
			"failure: invalid client ID",
			&types.GenesisState{
				Ics27Accounts: []types.RegisteredICS27Account{
					{
						AccountAddress: validAddress,
						AccountId: types.AccountIdentifier{
							ClientId: "x",
							Sender:   validAddress,
							Salt:     []byte("salt"),
						},
					},
				},
			},
			true,
		},
		{
			"failure: salt exceeds max length",
			&types.GenesisState{
				Ics27Accounts: []types.RegisteredICS27Account{
					{
						AccountAddress: validAddress,
						AccountId: types.AccountIdentifier{
							ClientId: validClientID,
							Sender:   validAddress,
							Salt:     make([]byte, types.MaximumSaltLength+1),
						},
					},
				},
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
