package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestValidateGenesis(t *testing.T) {
	testCases := []struct {
		name     string
		genState types.GenesisState
		expError error
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expError: nil,
		},
		{
			name: "valid genesis",
			genState: types.NewGenesisState(
				[]types.IdentifiedConnection{
					types.NewIdentifiedConnection(connectionID, types.NewConnectionEnd(types.INIT, clientID, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, []*types.Version{ibctesting.ConnectionVersion}, 500)),
				},
				[]types.ConnectionPaths{
					{clientID, []string{connectionID}},
				},
				0,
				types.DefaultParams(),
			),
			expError: nil,
		},
		{
			name: "invalid connection",
			genState: types.NewGenesisState(
				[]types.IdentifiedConnection{
					types.NewIdentifiedConnection(connectionID, types.NewConnectionEnd(types.INIT, "(CLIENTIDONE)", types.Counterparty{clientID, connectionID, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, []*types.Version{ibctesting.ConnectionVersion}, 500)),
				},
				[]types.ConnectionPaths{
					{clientID, []string{connectionID}},
				},
				0,
				types.DefaultParams(),
			),
			expError: host.ErrInvalidID,
		},
		{
			name: "invalid client id",
			genState: types.NewGenesisState(
				[]types.IdentifiedConnection{
					types.NewIdentifiedConnection(connectionID, types.NewConnectionEnd(types.INIT, clientID, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, []*types.Version{ibctesting.ConnectionVersion}, 500)),
				},
				[]types.ConnectionPaths{
					{"(CLIENTIDONE)", []string{connectionID}},
				},
				0,
				types.DefaultParams(),
			),
			expError: host.ErrInvalidID,
		},
		{
			name: "invalid path",
			genState: types.NewGenesisState(
				[]types.IdentifiedConnection{
					types.NewIdentifiedConnection(connectionID, types.NewConnectionEnd(types.INIT, clientID, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, []*types.Version{ibctesting.ConnectionVersion}, 500)),
				},
				[]types.ConnectionPaths{
					{clientID, []string{invalidConnectionID}},
				},
				0,
				types.DefaultParams(),
			),
			expError: host.ErrInvalidID,
		},
		{
			name: "invalid connection identifier",
			genState: types.NewGenesisState(
				[]types.IdentifiedConnection{
					types.NewIdentifiedConnection("conn-0", types.NewConnectionEnd(types.INIT, clientID, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, []*types.Version{ibctesting.ConnectionVersion}, 500)),
				},
				[]types.ConnectionPaths{
					{clientID, []string{connectionID}},
				},
				0,
				types.DefaultParams(),
			),
			expError: host.ErrInvalidID,
		},
		{
			name: "localhost connection identifier",
			genState: types.NewGenesisState(
				[]types.IdentifiedConnection{
					types.NewIdentifiedConnection(exported.LocalhostConnectionID, types.NewConnectionEnd(types.INIT, clientID, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, []*types.Version{ibctesting.ConnectionVersion}, 500)),
				},
				[]types.ConnectionPaths{
					{clientID, []string{connectionID}},
				},
				0,
				types.DefaultParams(),
			),
			expError: nil,
		},
		{
			name: "next connection sequence is not greater than maximum connection identifier sequence provided",
			genState: types.NewGenesisState(
				[]types.IdentifiedConnection{
					types.NewIdentifiedConnection(types.FormatConnectionIdentifier(10), types.NewConnectionEnd(types.INIT, clientID, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, []*types.Version{ibctesting.ConnectionVersion}, 500)),
				},
				[]types.ConnectionPaths{
					{clientID, []string{connectionID}},
				},
				0,
				types.DefaultParams(),
			),
			expError: errors.New("next connection sequence 0 must be greater than maximum sequence used in connection identifier 10"),
		},
		{
			name: "invalid params",
			genState: types.NewGenesisState(
				[]types.IdentifiedConnection{
					types.NewIdentifiedConnection(connectionID, types.NewConnectionEnd(types.INIT, clientID, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, []*types.Version{ibctesting.ConnectionVersion}, 500)),
				},
				[]types.ConnectionPaths{
					{clientID, []string{connectionID}},
				},
				0,
				types.Params{},
			),
			expError: errors.New("MaxExpectedTimePerBlock cannot be zero"),
		},
	}

	for _, tc := range testCases {
		err := tc.genState.Validate()
		if tc.expError == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorContains(t, err, tc.expError.Error())
		}
	}
}
