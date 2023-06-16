package interchain_accounts

import (
	"context"
	"testing"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	controllertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/suite"
)

func TestInterchainAccountsParamsTestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsParamsTestSuite))
}

type InterchainAccountsParamsTestSuite struct {
	testsuite.E2ETestSuite
}

// QueryControllerParams queries the params for the controller
func (s *InterchainAccountsParamsTestSuite) QueryControllerParams(ctx context.Context, chain ibc.Chain) controllertypes.Params {
	queryClient := s.GetChainGRCPClients(chain).ICAControllerQueryClient
	res, err := queryClient.Params(ctx, &controllertypes.QueryParamsRequest{})
	s.Require().NoError(err)

	return *res.Params
}

func (s *InterchainAccountsParamsTestSuite) TestControllerParams() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	_, _ = s.SetupChainsRelayerAndChannel(ctx)
	chainA, _ := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.FormattedAddress()

	// Assert that default value for enabled is true.
	t.Run("validate the controller is enabled by default", func(t *testing.T) {
		params := s.QueryControllerParams(ctx, chainA)
		s.Require().True(params.ControllerEnabled)
	})

	t.Run("disable controller", func(t *testing.T) {
		if testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion) {
			authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
			s.Require().NoError(err)
			s.Require().NotNil(authority)

			msg := controllertypes.MsgUpdateParams{
				Authority: authority.String(),
				Params:    controllertypes.NewParams(false),
			}
			s.ExecuteGovProposalV1(ctx, &msg, chainA, controllerAccount, 1)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(controllertypes.StoreKey, string(controllertypes.KeyControllerEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteGovProposal(ctx, chainA, controllerAccount, proposal)
		}
	})

	t.Run("validate the param was successfully changed", func(t *testing.T) {
		params := s.QueryControllerParams(ctx, chainA)
		s.Require().False(params.ControllerEnabled)
	})

	t.Run("assert that broadcasting a MsgRegisterInterchainAccount now fails", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)

		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.AssertTxFailure(txResp, controllertypes.ErrControllerSubModuleDisabled)
	})
}
