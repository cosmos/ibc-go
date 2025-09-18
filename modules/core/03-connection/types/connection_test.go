package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

var (
	connectionID        = "connection-0"
	clientID            = "clientidone"
	connectionID2       = "connectionidtwo"
	clientID2           = "clientidtwo"
	invalidConnectionID = "(invalidConnectionID)"
	clientHeight        = clienttypes.NewHeight(0, 6)
)

func TestConnectionValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		connection types.ConnectionEnd
		expError   error
	}{
		{
			"valid connection",
			types.ConnectionEnd{clientID, []*types.Version{ibctesting.ConnectionVersion}, types.INIT, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, 500},
			nil,
		},
		{
			"invalid client id",
			types.ConnectionEnd{"(clientID1)", []*types.Version{ibctesting.ConnectionVersion}, types.INIT, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, 500},
			host.ErrInvalidID,
		},
		{
			"empty versions",
			types.ConnectionEnd{clientID, nil, types.INIT, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, 500},
			ibcerrors.ErrInvalidVersion,
		},
		{
			"invalid version",
			types.ConnectionEnd{clientID, []*types.Version{{}}, types.INIT, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, 500},
			types.ErrInvalidVersion,
		},
		{
			"invalid counterparty",
			types.ConnectionEnd{clientID, []*types.Version{ibctesting.ConnectionVersion}, types.INIT, types.Counterparty{clientID2, connectionID2, emptyPrefix}, 500},
			types.ErrInvalidCounterparty,
		},
	}

	for i, tc := range testCases {
		err := tc.connection.ValidateBasic()
		if tc.expError == nil {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expError)
		}
	}
}

func TestCounterpartyValidateBasic(t *testing.T) {
	testCases := []struct {
		name         string
		counterparty types.Counterparty
		expError     error
	}{
		{"valid counterparty", types.Counterparty{clientID, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, nil},
		{"invalid client id", types.Counterparty{"(InvalidClient)", connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, host.ErrInvalidID},
		{"invalid connection id", types.Counterparty{clientID, "(InvalidConnection)", commitmenttypes.NewMerklePrefix([]byte("prefix"))}, host.ErrInvalidID},
		{"invalid prefix", types.Counterparty{clientID, connectionID2, emptyPrefix}, types.ErrInvalidCounterparty},
	}

	for i, tc := range testCases {
		err := tc.counterparty.ValidateBasic()
		if tc.expError == nil {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expError)
		}
	}
}

func TestIdentifiedConnectionValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		connection types.IdentifiedConnection
		expError   error
	}{
		{
			"valid connection",
			types.NewIdentifiedConnection(clientID, types.ConnectionEnd{clientID, []*types.Version{ibctesting.ConnectionVersion}, types.INIT, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, 500}),
			nil,
		},
		{
			"invalid connection id",
			types.NewIdentifiedConnection("(connectionIDONE)", types.ConnectionEnd{clientID, []*types.Version{ibctesting.ConnectionVersion}, types.INIT, types.Counterparty{clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))}, 500}),
			host.ErrInvalidID,
		},
	}

	for i, tc := range testCases {
		err := tc.connection.ValidateBasic()
		if tc.expError == nil {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expError)
		}
	}
}
