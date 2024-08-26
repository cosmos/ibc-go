package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *TypesTestSuite) TestMarshalConsensusStateWithHeight() {
	var cswh types.ConsensusStateWithHeight

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"solo machine client", func() {
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 1)
				cswh = types.NewConsensusStateWithHeight(types.NewHeight(0, soloMachine.Sequence), soloMachine.ConsensusState())
			},
		},
		{
			"tendermint client", func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupClients()

				latestHeight, ok := path.EndpointA.GetClientLatestHeight().(types.Height)
				suite.Require().True(ok)
				consensusState, ok := suite.chainA.GetConsensusState(path.EndpointA.ClientID, latestHeight)
				suite.Require().True(ok)

				cswh = types.NewConsensusStateWithHeight(latestHeight, consensusState)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			cdc := suite.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(&cswh)
			suite.Require().NoError(err)

			// unmarshal message
			newCswh := &types.ConsensusStateWithHeight{}
			err = cdc.UnmarshalJSON(bz, newCswh)
			suite.Require().NoError(err)
		})
	}
}

func TestValidateClientType(t *testing.T) {
	testCases := []struct {
		name       string
		clientType string
		expPass    bool
	}{
		{"valid", "tendermint", true},
		{"valid solomachine", "solomachine-v1", true},
		{"too large", "tenderminttenderminttenderminttenderminttendermintt", false},
		{"too short", "t", false},
		{"blank id", "               ", false},
		{"empty id", "", false},
		{"ends with dash", "tendermint-", false},
	}

	for _, tc := range testCases {
		tc := tc

		err := types.ValidateClientType(tc.clientType)

		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}

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
			"success with multiple element prefix",
			ibctesting.FirstClientID,
			commitmenttypes.NewMerklePath([]byte("ibc"), []byte("address")),
			nil,
		},
		{
			"success with multiple element prefix, last prefix empty",
			ibctesting.FirstClientID,
			commitmenttypes.NewMerklePath([]byte("ibc"), []byte("")),
			nil,
		},
		{
			"success with single empty key prefix",
			ibctesting.FirstClientID,
			commitmenttypes.NewMerklePath([]byte("")),
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
		{
			"failure: empty key in merkle path prefix first element",
			ibctesting.FirstClientID,
			commitmenttypes.NewMerklePath([]byte(""), []byte("ibc")),
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
