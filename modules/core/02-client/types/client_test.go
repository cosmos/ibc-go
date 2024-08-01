package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
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
