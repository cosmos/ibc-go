package mock_test

import (
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types/v2"
	mocklightclient "github.com/cosmos/ibc-go/v8/modules/light-clients/00-mock"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type MockClientTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chain *ibctesting.TestChain

	ctx sdk.Context
	cdc codec.Codec
}

func (suite *MockClientTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chain = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	suite.ctx = suite.chain.GetContext()
	suite.cdc = suite.chain.App.AppCodec()
}

func TestMockClientTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MockClientTestSuite))
}

func (suite *MockClientTestSuite) TestMockClient() {
	initialClientState := mocklightclient.ClientState{
		LatestHeight: clienttypes.NewHeight(1, 42),
	}
	clientStateBz := suite.cdc.MustMarshal(&initialClientState)
	initialConsensusState := mocklightclient.ConsensusState{
		Timestamp: uint64(time.Now().UnixNano()),
	}
	consensusStateBz := suite.cdc.MustMarshal(&initialConsensusState)
	clientID, err := suite.chain.GetSimApp().GetIBCKeeper().ClientKeeper.CreateClient(suite.ctx, mocklightclient.ModuleName, clientStateBz, consensusStateBz)
	suite.Require().NoError(err)
	suite.Require().Equal("00-mock-0", clientID)

	res, err := suite.chain.GetSimApp().GetIBCKeeper().ClientKeeper.VerifyMembership(suite.ctx, &clienttypes.QueryVerifyMembershipRequest{
		ClientId:    clientID,
		Proof:       []byte("doesntmatter"),
		ProofHeight: clienttypes.NewHeight(1, 10), // Doesnt matter, we dont verify it exists (yet?)
		Value:       []byte("doesntmatter"),
		TimeDelay:   0,
		BlockDelay:  0,
		MerklePath:  commitmenttypesv2.NewMerklePath([]byte("doesntmatter")),
	})
	suite.Require().NoError(err)
	suite.Require().True(res.Success)
}
