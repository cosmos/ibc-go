package localhost_test

import (
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	localhost "github.com/cosmos/ibc-go/v8/modules/light-clients/09-localhost"
)

func (suite *LocalhostTestSuite) TestClientType() {
	clientState := localhost.NewClientState(clienttypes.NewHeight(3, 10))
	suite.Require().Equal(exported.Localhost, clientState.ClientType())
}

func (suite *LocalhostTestSuite) TestGetLatestHeight() {
	expectedHeight := clienttypes.NewHeight(3, 10)
	clientState := localhost.NewClientState(expectedHeight)
	suite.Require().Equal(expectedHeight, clientState.LatestHeight)
}

func (suite *LocalhostTestSuite) TestGetTimestampAtHeight() {
	ctx := suite.chain.GetContext()
	clientState := localhost.NewClientState(clienttypes.NewHeight(1, 10))

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
			clientState: localhost.NewClientState(clienttypes.NewHeight(3, 10)),
			expPass:     true,
		},
		{
			name:        "invalid height",
			clientState: localhost.NewClientState(clienttypes.ZeroHeight()),
			expPass:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			err := tc.clientState.Validate()
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
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

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			clientState := localhost.NewClientState(clienttypes.NewHeight(3, 10))
			clientStore := suite.chain.GetSimApp().GetIBCKeeper().ClientKeeper.ClientStore(suite.chain.GetContext(), exported.LocalhostClientID)

			err := clientState.Initialize(suite.chain.GetContext(), suite.chain.Codec, clientStore, tc.consState)

			if tc.expPass {
				suite.Require().NoError(err, "valid testcase: %s failed", tc.name)
			} else {
				suite.Require().Error(err, "invalid testcase: %s passed", tc.name)
			}
		})
		suite.SetupTest()

	}
}

func (suite *LocalhostTestSuite) TestVerifyClientMessage() {
	clientState := localhost.NewClientState(clienttypes.NewHeight(1, 10))
	suite.Require().Error(clientState.VerifyClientMessage(suite.chain.GetContext(), nil, nil, nil))
}

func (suite *LocalhostTestSuite) TestVerifyCheckForMisbehaviour() {
	clientState := localhost.NewClientState(clienttypes.NewHeight(1, 10))
	suite.Require().False(clientState.CheckForMisbehaviour(suite.chain.GetContext(), nil, nil, nil))
}

func (suite *LocalhostTestSuite) TestUpdateState() {
	clientState := localhost.NewClientState(clienttypes.NewHeight(1, uint64(suite.chain.GetContext().BlockHeight())))
	store := suite.chain.GetSimApp().GetIBCKeeper().ClientKeeper.ClientStore(suite.chain.GetContext(), exported.LocalhostClientID)

	suite.coordinator.CommitBlock(suite.chain)

	heights := clientState.UpdateState(suite.chain.GetContext(), suite.chain.Codec, store, nil)

	expHeight := clienttypes.NewHeight(1, uint64(suite.chain.GetContext().BlockHeight()))
	suite.Require().True(heights[0].EQ(expHeight))

	var ok bool
	clientState, ok = suite.chain.GetClientState(exported.LocalhostClientID).(*localhost.ClientState)
	suite.Require().True(ok)
	suite.Require().True(heights[0].EQ(clientState.LatestHeight))
}
