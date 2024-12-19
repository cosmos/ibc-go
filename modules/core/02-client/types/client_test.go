package types_test

import (
	"errors"
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
		expError   error
	}{
		{"valid", "tendermint", nil},
		{"valid solomachine", "solomachine-v1", nil},
		{"too large", "tenderminttenderminttenderminttenderminttendermintt", errors.New("client type results in largest client identifier being invalid")},
		{"too short", "t", errors.New("client type results in smallest client identifier being invalid")},
		{"blank id", "               ", errors.New("client type cannot be blank")},
		{"empty id", "", errors.New("client type cannot be blank")},
		{"ends with dash", "tendermint-", errors.New("invalid client type")},
	}

	for _, tc := range testCases {
		tc := tc

		err := types.ValidateClientType(tc.clientType)

		if tc.expError == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorContains(t, err, tc.expError.Error())
		}
	}
}
