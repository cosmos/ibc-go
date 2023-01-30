package localhost_test

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	localhost "github.com/cosmos/ibc-go/v7/modules/light-clients/09-localhost"
)

func (suite *LocalhostTestSuite) TestStatus() {
	clientState := localhost.NewClientState("chainID", clienttypes.NewHeight(3, 10))
	suite.Require().Equal(exported.Active, clientState.Status(suite.chain.GetContext(), nil, nil))
}

func (suite *LocalhostTestSuite) TestClientType() {
	clientState := localhost.NewClientState("chainID", clienttypes.NewHeight(3, 10))
	suite.Require().Equal(exported.Localhost, clientState.ClientType())
}

func (suite *LocalhostTestSuite) TestGetLatestHeight() {
	expectedHeight := clienttypes.NewHeight(3, 10)
	clientState := localhost.NewClientState("chainID", expectedHeight)
	suite.Require().Equal(expectedHeight, clientState.GetLatestHeight())
}

func (suite *LocalhostTestSuite) TestZeroCustomFields() {
	clientState := localhost.NewClientState("chainID", clienttypes.NewHeight(1, 10))
	suite.Require().Equal(clientState, clientState.ZeroCustomFields())
}

func (suite *LocalhostTestSuite) TestGetTimestampAtHeight() {
	ctx := suite.chain.GetContext()
	clientState := localhost.NewClientState("chainID", clienttypes.NewHeight(1, 10))

	timestamp, err := clientState.GetTimestampAtHeight(ctx, nil, nil, nil)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(ctx.BlockTime().UnixNano()), timestamp)
}

func (suite *LocalhostTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState exported.ClientState
		expPass     bool
	}{
		{
			name:        "valid client",
			clientState: localhost.NewClientState("chainID", clienttypes.NewHeight(3, 10)),
			expPass:     true,
		},
		{
			name:        "invalid chain id",
			clientState: localhost.NewClientState(" ", clienttypes.NewHeight(3, 10)),
			expPass:     false,
		},
		{
			name:        "invalid height",
			clientState: localhost.NewClientState("chainID", clienttypes.ZeroHeight()),
			expPass:     false,
		},
	}

	for _, tc := range testCases {
		err := tc.clientState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}

func (suite *LocalhostTestSuite) TestInitialize() {
	testCases := []struct {
		name      string
		consState exported.ConsensusState
		expPass   bool
	}{
		{
			"valid initialization",
			nil,
			true,
		},
		{
			"invalid consenus state",
			&ibctm.ConsensusState{},
			false,
		},
	}

	clientState := localhost.NewClientState("chainID", clienttypes.NewHeight(3, 10))

	for _, tc := range testCases {
		err := clientState.Initialize(suite.chain.GetContext(), suite.chain.Codec, nil, tc.consState)

		if tc.expPass {
			suite.Require().NoError(err, "valid testcase: %s failed", tc.name)
		} else {
			suite.Require().Error(err, "invalid testcase: %s passed", tc.name)
		}
	}
}
