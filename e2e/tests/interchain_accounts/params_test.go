//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	testifysuite "github.com/stretchr/testify/suite"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestInterchainAccountsParamsTestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsParamsTestSuite))
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

// QueryHostParams queries the host chain for the params
func (s *InterchainAccountsParamsTestSuite) QueryHostParams(ctx context.Context, chain ibc.Chain) hosttypes.Params {
	queryClient := s.GetChainGRCPClients(chain).ICAHostQueryClient
	res, err := queryClient.Params(ctx, &hosttypes.QueryParamsRequest{})
	s.Require().NoError(err)

	return *res.Params
}

// TestControllerEnabledParam tests that changing the ControllerEnabled param works as expected
func (s *InterchainAccountsParamsTestSuite) TestControllerEnabledParam() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	_, _ = s.SetupChainsRelayerAndChannel(ctx, nil)
	chainA, _ := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	// setup controller account on chainA
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.FormattedAddress()

	t.Run("ensure the controller is enabled", func(t *testing.T) {
		params := s.QueryControllerParams(ctx, chainA)
		s.Require().True(params.ControllerEnabled)
	})

	t.Run("disable the controller", func(t *testing.T) {
		if testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion) {
			authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
			s.Require().NoError(err)
			s.Require().NotNil(authority)

			msg := controllertypes.MsgUpdateParams{
				Signer: authority.String(),
				Params: controllertypes.NewParams(false),
			}
			s.ExecuteAndPassGovV1Proposal(ctx, &msg, chainA, controllerAccount)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(controllertypes.StoreKey, string(controllertypes.KeyControllerEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, chainA, controllerAccount, proposal)
		}
	})

	t.Run("ensure controller is disabled", func(t *testing.T) {
		params := s.QueryControllerParams(ctx, chainA)
		s.Require().False(params.ControllerEnabled)
	})

	t.Run("ensure that broadcasting a MsgRegisterInterchainAccount fails", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)

		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.AssertTxFailure(txResp, controllertypes.ErrControllerSubModuleDisabled)
	})
}

func (s *InterchainAccountsParamsTestSuite) TestHostEnabledParam() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	_, _ = s.SetupChainsRelayerAndChannel(ctx, nil)
	_, chainB := s.GetChains()
	chainBVersion := chainB.Config().Images[0].Version

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	chainBUser := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	// Assert that default value for enabled is true.
	t.Run("ensure the host is enabled", func(t *testing.T) {
		params := s.QueryHostParams(ctx, chainB)
		s.Require().True(params.HostEnabled)
		s.Require().Equal([]string{hosttypes.AllowAllHostMsgs}, params.AllowMessages)
	})

	t.Run("disable the host", func(t *testing.T) {
		if testvalues.SelfParamsFeatureReleases.IsSupported(chainBVersion) {
			authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainB)
			s.Require().NoError(err)
			s.Require().NotNil(authority)

			msg := hosttypes.MsgUpdateParams{
				Signer: authority.String(),
				Params: hosttypes.NewParams(false, []string{hosttypes.AllowAllHostMsgs}),
			}
			s.ExecuteAndPassGovV1Proposal(ctx, &msg, chainB, chainBUser)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(hosttypes.StoreKey, string(hosttypes.KeyHostEnabled), "false"),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, chainB, chainBUser, proposal)
		}
	})

	t.Run("ensure the host is disabled", func(t *testing.T) {
		params := s.QueryHostParams(ctx, chainB)
		s.Require().False(params.HostEnabled)
	})
}
