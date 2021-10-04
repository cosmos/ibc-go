package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *KeeperTestSuite) TestCreateChildChainProposal() {
	var (
		ctx      sdk.Context
		proposal *ccv.CreateChildChainProposal
		ok       bool
	)

	chainID := "chainID"

	clientState := ibctmtypes.NewClientState(
		chainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift,
		clienttypes.NewHeight(0, 1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, true, true,
	)

	testCases := []struct {
		name         string
		malleate     func(*KeeperTestSuite)
		expPass      bool
		spawnReached bool
	}{
		{
			"valid create child chain proposal: spawn time reached", func(suite *KeeperTestSuite) {
				// ctx blocktime is after proposal's spawn time
				ctx = suite.parentChain.GetContext().WithBlockTime(time.Now().Add(time.Hour))
				content, err := ccv.NewCreateChildChainProposal("title", "description", chainID, clientState, []byte("gen_hash"), time.Now())
				suite.Require().NoError(err)
				proposal, ok = content.(*ccv.CreateChildChainProposal)
				suite.Require().True(ok)
			}, true, true,
		},
		{
			"valid proposal: spawn time has not yet been reached", func(suite *KeeperTestSuite) {
				// ctx blocktime is before proposal's spawn time
				ctx = suite.parentChain.GetContext().WithBlockTime(time.Now())
				content, err := ccv.NewCreateChildChainProposal("title", "description", chainID, clientState, []byte("gen_hash"), time.Now().Add(time.Hour))
				suite.Require().NoError(err)
				proposal, ok = content.(*ccv.CreateChildChainProposal)
				suite.Require().True(ok)
			}, true, false,
		},
		{
			"client state unpack failed", func(suite *KeeperTestSuite) {
				ctx = suite.parentChain.GetContext().WithBlockTime(time.Now())
				any, err := clienttypes.PackConsensusState(&ibctmtypes.ConsensusState{})
				suite.Require().NoError(err)

				proposal = &types.CreateChildChainProposal{
					Title:       "title",
					Description: "description",
					ChainId:     chainID,
					ClientState: any,
					GenesisHash: []byte("gen_hash"),
					SpawnTime:   time.Now(),
				}
			}, false, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate(suite)

			err := suite.parentChain.GetSimApp().ParentKeeper.CreateChildChainProposal(ctx, proposal)
			if tc.expPass {
				suite.Require().NoError(err, "error returned on valid case")
				if tc.spawnReached {
					clientId := suite.parentChain.GetSimApp().ParentKeeper.GetChildClient(ctx, chainID)
					suite.Require().NotEqual("", clientId, "child client was not created after spawn time reached")
				} else {
					pendingClient := suite.parentChain.GetSimApp().ParentKeeper.GetPendingClient(ctx, proposal.SpawnTime, chainID)
					suite.Require().Equal(clientState, pendingClient, "pending client not equal to clientstate in proposal")
				}
			} else {
				suite.Require().Error(err, "did not return error on invalid case")
			}
		})
	}
}
