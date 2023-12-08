//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// QueryControllerParams queries the params for the controller
func (s *InterchainAccountsTestSuite) QueryControllerParams(ctx context.Context, chain ibc.Chain) controllertypes.Params {
	queryClient := s.GetChainGRCPClients(chain).ICAControllerQueryClient
	res, err := queryClient.Params(ctx, &controllertypes.QueryParamsRequest{})
	s.Require().NoError(err)

	return *res.Params
}

// QueryHostParams queries the host chain for the params
func (s *InterchainAccountsTestSuite) QueryHostParams(ctx context.Context, chain ibc.Chain) hosttypes.Params {
	queryClient := s.GetChainGRCPClients(chain).ICAHostQueryClient
	res, err := queryClient.Params(ctx, &hosttypes.QueryParamsRequest{})
	s.Require().NoError(err)

	return *res.Params
}

// TestControllerEnabledParam tests that changing the ControllerEnabled param works as expected
func (s *InterchainAccountsTestSuite) TestControllerEnabledParam() {
	t := s.T()
	t.Parallel()
	ctx := context.TODO()

	chainAVersion := s.chainA.Config().Images[0].Version

	// setup controller account on chainA
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)
	controllerAddress := controllerAccount.FormattedAddress()

	t.Run("ensure the controller is enabled", func(t *testing.T) {
		// setup relayers and connection-0 between two chains
		// channel-0 is a transfer channel but it will not be used in this test case
		_, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
		s.InitGRPCClients(s.chainA)
		s.InitGRPCClients(s.chainB)
		s.Require().NoError(err)
		params := s.QueryControllerParams(ctx, s.chainA)
		s.Require().True(params.ControllerEnabled)
	})

	t.Run("disable the controller", func(t *testing.T) {
		if testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion) {
			authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, s.chainA)
			s.Require().NoError(err)
			s.Require().NotNil(authority)

			msg := controllertypes.MsgUpdateParams{
				Signer: authority.String(),
				Params: controllertypes.NewParams(false),
			}
			s.ExecuteAndPassGovV1Proposal(ctx, &msg, s.chainA, controllerAccount)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(controllertypes.StoreKey, string(controllertypes.KeyControllerEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, s.chainA, controllerAccount, proposal, s.chainB)
		}
	})

	t.Run("ensure controller is disabled", func(t *testing.T) {
		s.InitGRPCClients(s.chainA)
		s.InitGRPCClients(s.chainB)
		params := s.QueryControllerParams(ctx, s.chainA)
		s.Require().False(params.ControllerEnabled)
	})

	t.Run("ensure that broadcasting a MsgRegisterInterchainAccount fails", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)

		txResp := s.BroadcastMessages(ctx, s.chainA, controllerAccount, s.chainB, msgRegisterAccount)
		s.AssertTxFailure(txResp, controllertypes.ErrControllerSubModuleDisabled)
	})
}

func (s *InterchainAccountsTestSuite) TestHostEnabledParam() {
	t := s.T()
	t.Parallel()
	ctx := context.TODO()

	chainBVersion := s.chainB.Config().Images[0].Version

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	chainBUser := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, s.chainB)

	// Assert that default value for enabled is true.
	t.Run("ensure the host is enabled", func(t *testing.T) {
		// setup relayers and connection-0 between two chains
		// channel-0 is a transfer channel but it will not be used in this test case
		_, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
		s.Require().NoError(err)
		s.InitGRPCClients(s.chainA)
		s.InitGRPCClients(s.chainB)

		params := s.QueryHostParams(ctx, s.chainB)
		s.Require().True(params.HostEnabled)
		s.Require().Equal([]string{hosttypes.AllowAllHostMsgs}, params.AllowMessages)
	})

	t.Run("disable the host", func(t *testing.T) {
		if testvalues.SelfParamsFeatureReleases.IsSupported(chainBVersion) {
			authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, s.chainB)
			s.Require().NoError(err)
			s.Require().NotNil(authority)

			msg := hosttypes.MsgUpdateParams{
				Signer: authority.String(),
				Params: hosttypes.NewParams(false, []string{hosttypes.AllowAllHostMsgs}),
			}
			s.ExecuteAndPassGovV1Proposal(ctx, &msg, s.chainB, chainBUser)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(hosttypes.StoreKey, string(hosttypes.KeyHostEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, s.chainB, chainBUser, proposal, s.chainA)
		}
	})

	t.Run("ensure the host is disabled", func(t *testing.T) {
		params := s.QueryHostParams(ctx, s.chainB)
		s.Require().False(params.HostEnabled)
	})
}
