package parent_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/parent"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *ParentTestSuite) TestCreateChildChainProposalHandler() {
	var (
		ctx     sdk.Context
		content govtypes.Content
		err     error
	)

	testCases := []struct {
		name     string
		malleate func(*ParentTestSuite)
		expPass  bool
	}{
		{
			"valid create childchain proposal", func(suite *ParentTestSuite) {
				clientState := ibctmtypes.NewClientState(
					"chainID", ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift,
					clienttypes.NewHeight(0, 1), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath, true, true,
				)
				// ctx blocktime is after proposal's spawn time
				ctx = suite.parentChain.GetContext().WithBlockTime(time.Now().Add(time.Hour))
				content, err = ccv.NewCreateChildChainProposal("title", "description", "chainID", clientState, []byte("gen_hash"), time.Now())
				suite.Require().NoError(err)
			}, true,
		},
		{
			"nil proposal", func(suite *ParentTestSuite) {
				ctx = suite.parentChain.GetContext()
				content = nil
			}, false,
		},
		{
			"unsupported proposal type", func(suite *ParentTestSuite) {
				ctx = suite.parentChain.GetContext()
				content = distributiontypes.NewCommunityPoolSpendProposal(ibctesting.Title, ibctesting.Description, suite.parentChain.SenderAccount.GetAddress(), sdk.NewCoins(sdk.NewCoin("communityfunds", sdk.NewInt(10))))
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate(suite)

			proposalHandler := parent.NewCreateChildChainHandler(suite.parentChain.GetSimApp().ParentKeeper)

			err = proposalHandler(ctx, content)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
